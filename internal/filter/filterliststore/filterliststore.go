package filterliststore

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/anfragment/zen/internal/filter/filterliststore/diskcache"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	ignoreLineRegex = regexp.MustCompile(`^(?:!|\[|#([^#%]|$))`)
)

type FilterListStore struct {
	cache *diskcache.Cache
}

func NewFilterListStore() *FilterListStore {
	cache, err := diskcache.New()
	if err != nil {
		log.Fatalf("failed to init diskcache: %v", err)
	}

	return &FilterListStore{
		cache: cache,
	}
}

func (st *FilterListStore) Get(url string) ([]byte, error) {
	if content, ok := st.cache.Load(url); ok {
		return content, nil
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response: %s", resp.Status)
	}

	var comments [][]byte
	var bodyLines [][]byte

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if ignoreLineRegex.Match(line) {
			comments = append(comments, append([]byte(nil), line...))
		} else {
			bodyLines = append(bodyLines, append([]byte(nil), line...))
		}
	}

	bodyBytes := bytes.Join(bodyLines, []byte("\n"))
	if err := st.cache.Save(url, bodyBytes); err != nil {
		return nil, fmt.Errorf("cache save: %w", err)
	}

	return bodyBytes, nil
}
