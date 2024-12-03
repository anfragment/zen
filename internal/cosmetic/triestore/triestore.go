package triestore

import (
	"strings"
	"sync"
)

type node struct {
	children  map[string]*node
	selectors []string
}

func newNode() *node {
	return &node{
		children: make(map[string]*node),
	}
}

func (n *node) findOrAddChild(segment string) *node {
	child := n.children[segment]
	if child != nil {
		return child
	}

	child = newNode()
	n.children[segment] = child
	return child
}

// getMatchingSelectors traverses the trie and returns matching selectors.
func (n *node) getMatchingSelectors(segments []string, isWildcard bool) []string {
	if len(segments) == 0 {
		return n.selectors
	}

	var selectors []string
	if isWildcard {
		// Wildcards can consume as many segments as possible.
		selectors = append(selectors, n.getMatchingSelectors(segments[1:], true)...)
	}
	wildcardChild := n.children["*"]
	if wildcardChild != nil {
		selectors = append(selectors, wildcardChild.getMatchingSelectors(segments[1:], true)...)
	}
	exactChild := n.children[segments[0]]
	if exactChild != nil {
		selectors = append(selectors, exactChild.getMatchingSelectors(segments[1:], false)...)
	}

	return selectors
}

type TrieStore struct {
	mu                 sync.RWMutex
	universalSelectors []string
	root               *node
}

func NewTrieStore() *TrieStore {
	return &TrieStore{
		root: newNode(),
	}
}

func (ts *TrieStore) Add(hostnames []string, selector string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(hostnames) == 0 {
		ts.universalSelectors = append(ts.universalSelectors, selector)
		return
	}

	for _, hostname := range hostnames {
		segments := strings.Split(hostname, ".")

		node := ts.root
		for _, segment := range segments {
			node = node.findOrAddChild(segment)
		}
		node.selectors = append(node.selectors, selector)
	}
}

func (ts *TrieStore) Get(hostname string) []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	segments := strings.Split(hostname, ".")
	selectors := ts.root.getMatchingSelectors(segments, false)
	selectors = append(selectors, ts.universalSelectors...)
	return selectors
}
