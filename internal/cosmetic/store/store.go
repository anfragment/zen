package store

import (
	"sync"
)

type RuleStore struct {
	mu    sync.RWMutex
	store map[string][]string
}

func NewStore() *RuleStore {
	return &RuleStore{
		store: make(map[string][]string),
	}
}

func (rs *RuleStore) Add(hostnames []string, selector string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if len(hostnames) == 0 {
		rs.store["*"] = append(rs.store["*"], selector)
		return
	}

	for _, hostname := range hostnames {
		rs.store[hostname] = append(rs.store[hostname], selector)
	}
}

func (rs *RuleStore) Get(hostname string) []string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	selectors, ok := rs.store[hostname]
	if !ok {
		return nil
	}

	return selectors
}
