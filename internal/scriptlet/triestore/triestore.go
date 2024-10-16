package triestore

import (
	"strings"
	"sync"

	"github.com/anfragment/zen/internal/scriptlet"
)

type node struct {
	children   map[string]*node
	scriptlets []*scriptlet.Scriptlet
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

func (n *node) getMatchingScriptlets(segments []string, canTerminate bool, isWildcard bool) []*scriptlet.Scriptlet {
	if len(segments) == 0 {
		if canTerminate {
			return n.scriptlets
		}
		return nil
	}

	resSet := make(map[*scriptlet.Scriptlet]struct{})
	wildcardChild := n.children["*"]
	if wildcardChild != nil {
		for _, scriptlet := range wildcardChild.getMatchingScriptlets(segments[1:], canTerminate, true) {
			resSet[scriptlet] = struct{}{}
		}
	}
	if isWildcard {
		for _, scriptlet := range n.getMatchingScriptlets(segments[1:], canTerminate, true) {
			resSet[scriptlet] = struct{}{}
		}
	}
	exactChild := n.children[segments[0]]
	if exactChild != nil {
		for _, scriptlet := range exactChild.getMatchingScriptlets(segments[1:], true, false) {
			resSet[scriptlet] = struct{}{}
		}
	}

	var res []*scriptlet.Scriptlet
	for s := range resSet {
		res = append(res, s)
	}
	return res
}

type TrieStore struct {
	mu                  sync.RWMutex
	universalScriptlets []scriptlet.Scriptlet
	root                *node
}

// assert TrieStore implements scriptlet.Store
var _ scriptlet.Store = (*TrieStore)(nil)

func NewTrieStore() *TrieStore {
	return &TrieStore{
		root: newNode(),
	}
}

func (ts *TrieStore) Add(hostnames []string, scriptlet scriptlet.Scriptlet) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if len(hostnames) == 0 {
		ts.universalScriptlets = append(ts.universalScriptlets, scriptlet)
		return
	}

	for _, hostname := range hostnames {
		segments := strings.Split(hostname, ".")

		node := ts.root
		for _, segment := range segments {
			node = node.findOrAddChild(segment)
		}
		node.scriptlets = append(node.scriptlets, &scriptlet)
	}
}

func (ts *TrieStore) Get(hostname string) []*scriptlet.Scriptlet {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	segments := strings.Split(hostname, ".")
	return ts.root.getMatchingScriptlets(segments, false, false)
}
