package certmanager

import (
	"crypto/tls"
	"sync"
)

// maxSize is the maximum number of certificates to cache.
//
// tls.Certificate takes about 1.7KB of memory, so 5800 certs including the doubly-linked list should fit in about 10MB of memory.
const maxSize = 5800

type lruList struct {
	host string
	// prev and next are the previous and next (older and newer) entries in the list.
	prev, next *lruList
}

type CertLRU struct {
	sync.Mutex

	// head and tail are the head and tail of the doubly-linked list of hosts.
	head, tail *lruList

	// cache is the map of host to certificate.
	cache map[string]*tls.Certificate
}

func NewCertLRU() *CertLRU {
	return &CertLRU{
		cache: make(map[string]*tls.Certificate),
	}
}

func (lru *CertLRU) Get(host string) *tls.Certificate {
	lru.Lock()
	defer lru.Unlock()

	cert, ok := lru.cache[host]
	if !ok {
		return nil
	}

	lru.promote(host)

	return cert
}

func (lru *CertLRU) Put(host string, cert *tls.Certificate) {
	lru.Lock()
	defer lru.Unlock()

	if len(lru.cache) >= maxSize {
		lru.evict()
	}

	lru.promote(host)
	lru.cache[host] = cert
}

// promote moves the host to the front of the list.
// It is assumed that the caller holds lru.Lock().
func (lru *CertLRU) promote(host string) {
	for e := lru.head; e != nil; e = e.prev {
		if e.host != host {
			continue
		}

		if e.next != nil {
			e.next.prev = e.prev
		}
		if e.prev != nil {
			e.prev.next = e.next
		}
		if lru.tail == e {
			lru.tail = e.next
		}

		break
	}

	e := &lruList{
		host: host,
		prev: lru.head,
		next: nil,
	}

	if lru.head != nil {
		lru.head.next = e
	}

	lru.head = e
	if lru.tail == nil {
		lru.tail = e
	}
}

// evict removes the oldest entry from the list and cache.
// It is assumed that the caller holds lru.Lock().
func (lru *CertLRU) evict() {

}
