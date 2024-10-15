package scriptlet

import (
	"strings"
	"sync"
)

type node struct {
	children   map[string]*node
	scriptlets []*scriptlet
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

func (n *node) getMatchingScriptlets(segments []string, canTerminate bool, isWildcard bool) []*scriptlet {
	if len(segments) == 0 {
		if canTerminate {
			return n.scriptlets
		}
		return nil
	}

	resSet := make(map[*scriptlet]struct{})
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

	var res []*scriptlet
	for s := range resSet {
		res = append(res, s)
	}
	return res
}

type TreeStore struct {
	mu                  sync.RWMutex
	universalScriptlets []scriptlet
	root                *node
}

func NewTreeStore() *TreeStore {
	return &TreeStore{
		root: newNode(),
	}
}

func (ts *TreeStore) Add(hostnames []string, scriptlet scriptlet) {
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

func (ts *TreeStore) Get(hostname string) []*scriptlet {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	segments := strings.Split(hostname, ".")
	return ts.root.getMatchingScriptlets(segments, false, false)
}
