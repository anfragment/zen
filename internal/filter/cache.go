package filter

import (
	"sync"
	"time"
)

const cacheTTL = 24 * time.Hour

type cacheEntry struct {
	content   []byte
	fetchedAt time.Time
}

type FilterListCache struct {
	mu   sync.RWMutex
	data map[string]*cacheEntry
}

func NewFilterListCache() *FilterListCache {
	return &FilterListCache{
		data: make(map[string]*cacheEntry),
	}
}

func (c *FilterListCache) Get(url string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[url]
	if !ok || time.Since(entry.fetchedAt) > cacheTTL {
		return nil, false
	}
	return entry.content, true
}

func (c *FilterListCache) Set(url string, content []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[url] = &cacheEntry{
		content:   content,
		fetchedAt: time.Now(),
	}
}
