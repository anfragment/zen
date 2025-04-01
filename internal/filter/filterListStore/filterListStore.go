package filterListStore

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anfragment/zen/internal/filter/filterListStore/diskcache"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

type FilterListStore struct{}

func NewFilterListStore() *FilterListStore {
	return &FilterListStore{}
}

func (st *FilterListStore) Get(url string) ([]byte, error) {
	if content, ok := diskcache.Load(url); ok {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if err := diskcache.Save(url, body); err != nil {
		return nil, fmt.Errorf("cache save: %w", err)
	}

	return body, nil
}
