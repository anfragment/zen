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
	// Try to load from cache first
	if content, err := st.cache.Load(url); err != nil {
		log.Printf("failed to load from cache: %v", err)
	} else if content != nil {
		log.Printf("loaded %q from cache", url)
		return content, nil
	}

	// Make HTTP request to fetch the filter list
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

	var headerBuffer bytes.Buffer
	tee := io.TeeReader(resp.Body, &headerBuffer)
	now := time.Now()
	expiry := now.Add(24 * time.Hour) // Default to 1 day
	scanner := bufio.NewScanner(tee)

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

	header := headerBuffer.Bytes()

	wrappedBody, resCh := newEavesdropReadCloser(resp.Body)

	go func() {
		contents := <-resCh
		if err := st.cache.Save(url, bytes.Join([][]byte{header, contents}, nil), expiry); err != nil {
			log.Printf("failed to store in cache: %v", err)
		}
	}()

	return newReadThenCloseReadCloser(bytes.NewReader(header), wrappedBody), nil
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
