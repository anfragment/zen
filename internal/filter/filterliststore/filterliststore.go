package filterliststore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/ZenPrivacy/zen-desktop/internal/filter/filterliststore/diskcache"
)

const defaultExpiry = 24 * time.Hour

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	// headerRegex matches comments prefixed with a hash and [Adblock Plus 2.0]-style headers.
	headerRegex = regexp.MustCompile(`^(?:!|\[|#[^#%@$])`)
)

type FilterListStore struct {
	cache *diskcache.Cache
}

func New() (*FilterListStore, error) {
	cache, err := diskcache.New()
	if err != nil {
		return nil, fmt.Errorf("create cache: %v", err)
	}

	return &FilterListStore{
		cache: cache,
	}, nil
}

func (st *FilterListStore) Get(url string) (io.ReadCloser, error) {
	if content, err := st.cache.Load(url); err != nil {
		log.Printf("failed to load from cache: %v", err)
	} else if content != nil {
		log.Printf("loading %q from cache", url)
		return content, nil
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("non-200 response: %q", resp.Status)
	}

	var teeBuffer bytes.Buffer
	resp.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(resp.Body, &teeBuffer),
		Closer: resp.Body,
	}

	var cacheTTL time.Duration
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		// TODO: move this into a cache-handling goroutine.
		line := scanner.Bytes()

		if len(line) != 0 && !headerRegex.Match(line) {
			// Stop scanning for "! Expires" if we encounter a non-comment line.
			break
		}

		cacheTTL, err = parseExpires(line)
		if err != nil {
			log.Printf("failed to parse expiry timestamp %q, assuming default: %v", line, err)
			break
		} else if cacheTTL != 0 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("read response header comments: %v", err)
	}

	if cacheTTL == 0 {
		// Default to 24 hours if no expiry is found.
		cacheTTL = defaultExpiry
	}
	expiresAt := time.Now().Add(cacheTTL) // time.Now() might deviate from the time the request was received, but it isn't critical.

	var notifyCh <-chan struct{}
	resp.Body, notifyCh = newNotifyReadCloser(resp.Body)

	// Read everything scanned so far from teeBuffer.
	header, _ := io.ReadAll(&teeBuffer) // err is always nil since (*bytes.Buffer).Read only returns io.EOF.

	go func() {
		// The intention here is to make caching non-blocking. Data from the response body is cloned into teeBuffer,
		// and the cache is saved in a separate goroutine.
		// This allows the consumer of Get to start reading the response body without waiting for the entire response to be fetched.

		<-notifyCh

		// Read the remaining content from the response body.
		remaining, _ := io.ReadAll(&teeBuffer)
		cacheContent := make([]byte, len(header)+len(remaining))
		copy(cacheContent, header)
		copy(cacheContent[len(header):], remaining)
		if err := st.cache.Save(url, cacheContent, expiresAt); err != nil {
			log.Printf("failed to store in cache: %v", err)
		}
	}()

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(bytes.NewReader(header), resp.Body),
		Closer: resp.Body,
	}, nil
}
