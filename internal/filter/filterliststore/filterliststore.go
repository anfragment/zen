package filterliststore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ZenPrivacy/zen-desktop/internal/filter/filterliststore/diskcache"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	// expiresRegex matches lines like "! Expires: 4 days", supporting formats such as: "4 days", "12 hours", "5d", and "18h".
	expiresRegex = regexp.MustCompile(`(?i)^! Expires:\s*(\d+)\s*(days?|hours?|d|h)?$`)
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
		log.Printf("loaded %q from cache", url)
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

	now := time.Now()
	expiry := now.Add(24 * time.Hour) // Default to 1 day
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 && line[0] != '!' {
			break
		}

		if !expiresRegex.Match(line) {
			continue
		}

		expiry, err = extractExpiryTimestamp(line, now)
		if err != nil {
			log.Printf("failed to parse expiry timestamp, assuming default: %v", err)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("read response header comments: %v", err)
	}

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
		if err := st.cache.Save(url, cacheContent, expiry); err != nil {
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

func extractExpiryTimestamp(line []byte, now time.Time) (time.Time, error) {
	matches := expiresRegex.FindSubmatch(line)
	amount, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid amount: %v", err)
	}

	unit := "days"
	if len(matches) >= 3 {
		unit = strings.ToLower(strings.TrimSpace(string(matches[2])))
	}

	switch unit {
	case "day", "days", "d":
		return now.Add(time.Duration(amount) * 24 * time.Hour), nil
	case "hour", "hours", "h":
		return now.Add(time.Duration(amount) * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid unit: %q", unit)
	}
}
