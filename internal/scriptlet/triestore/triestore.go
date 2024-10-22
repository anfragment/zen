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

// getMatchingScriptlets traverses the trie and returns matching scriptlets.
func (n *node) getMatchingScriptlets(segments []string, isWildcard bool) []*scriptlet.Scriptlet {
	if len(segments) == 0 {
		return n.scriptlets
	}

	var scriptlets []*scriptlet.Scriptlet
	if isWildcard {
		// Wildcards can consume as many segments as possible.
		scriptlets = append(scriptlets, n.getMatchingScriptlets(segments[1:], true)...)
	}
	wildcardChild := n.children["*"]
	if wildcardChild != nil {
		scriptlets = append(scriptlets, wildcardChild.getMatchingScriptlets(segments[1:], true)...)
	}
	exactChild := n.children[segments[0]]
	if exactChild != nil {
		scriptlets = append(scriptlets, exactChild.getMatchingScriptlets(segments[1:], false)...)
	}

	return scriptlets
}

type TrieStore struct {
	mu                  sync.RWMutex
	universalScriptlets []*scriptlet.Scriptlet
	root                *node
}

// assert TrieStore implements scriptlet.Store.
var _ scriptlet.Store = (*TrieStore)(nil)

func NewTrieStore() *TrieStore {
	return &TrieStore{
		root: newNode(),
	}
}

func (ts *TrieStore) Add(hostnames []string, scriptlet *scriptlet.Scriptlet) {
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
		node.scriptlets = append(node.scriptlets, scriptlet)
	}
}

func (ts *TrieStore) Get(hostname string) []*scriptlet.Scriptlet {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	segments := strings.Split(hostname, ".")
	scriptlets := ts.root.getMatchingScriptlets(segments, false)
	scriptlets = append(scriptlets, ts.universalScriptlets...)
	return scriptlets
}
