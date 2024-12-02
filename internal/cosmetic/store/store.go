package store

import (
	"strings"
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

	var selectors []string

	if global := rs.store["*"]; len(global) > 0 {
		selectors = append(selectors, global...)
	}

	h := normalizeDomain(hostname)
	if hostSpecific := rs.store[h]; len(hostSpecific) > 0 {
		selectors = append(selectors, hostSpecific...)
	}

	if len(selectors) == 0 {
		return nil
	}

	return selectors
}

func normalizeDomain(hostname string) string {
	parts := strings.Split(hostname, ".")

	if len(parts) <= 1 {
		return hostname
	}

	if len(parts) >= 3 && len(parts[len(parts)-1]) == 2 {
		return strings.Join(parts[len(parts)-3:], ".")
	}

	return strings.Join(parts[len(parts)-2:], ".")
}
