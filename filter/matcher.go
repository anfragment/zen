package filter

import (
	"fmt"
	"regexp"
	"sync"
)

// nodeKind is the type of a node in the trie.
type nodeKind int

const (
	nodeKindExactMatch   nodeKind = iota
	nodeKindAddressRoot           // |
	nodeKindHostnameRoot          // hosts.txt
	nodeKindDomain                // ||
	nodeKindWildcard              // *
	nodeKindSeparator             // ^
)

// nodeKey identifies a node in the trie.
// It is a combination of the node kind and the token that the node represents.
// The token is only present for nodes with kind nodeKindExactMatch.
// The other kinds of nodes only represent roots of subtrees.
type nodeKey struct {
	kind  nodeKind
	token string
}

// node is a node in the trie.
type node struct {
	children   map[nodeKey]*node
	childrenMu sync.RWMutex
	// isLeaf is true if the node is a leaf node and should
	// terminate the traversal in case of a match.
	isLeaf bool
}

func (n *node) findOrAddChild(key nodeKey) *node {
	n.childrenMu.RLock()
	child, ok := n.children[key]
	n.childrenMu.RUnlock()
	if ok {
		return child
	}

	n.childrenMu.Lock()
	child = &node{
		children: make(map[nodeKey]*node),
	}
	n.children[key] = child
	n.childrenMu.Unlock()
	return child
}

func (n *node) findChild(key nodeKey) *node {
	n.childrenMu.RLock()
	child := n.children[key]
	n.childrenMu.RUnlock()
	return child
}

// match returns true if the node's subtree matches the given tokens.
func (n *node) match(tokens []string) bool {
	if n == nil {
		return false
	}
	if n.isLeaf {
		return true
	}
	if len(tokens) == 0 {
		return false
	}

	return n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).match(tokens[1:])
}

// Matcher is trie-based matcher for URLs that is capable of parsing
// Adblock filter and hosts rules and matching URLs against them.
//
// The matcher is safe for concurrent use.
type Matcher struct {
	root *node
}

func NewMatcher() *Matcher {
	return &Matcher{
		root: &node{
			children: make(map[nodeKey]*node),
		},
	}
}

var (
	// hostnameCG is a capture group for a hostname.
	hostnameCG    = `((?:[\da-z][\da-z_-]*\.)+[\da-z-]*[a-z])`
	reHosts       = regexp.MustCompile(fmt.Sprintf(`^(?:0\.0\.0\.0|127\.0\.0\.1) %s`, hostnameCG))
	reHostsIgnore = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
)

func (m *Matcher) AddRule(rule string) {
	rootKeyKind := nodeKindExactMatch
	var tokens []string

	if host := reHosts.FindStringSubmatch(rule); host != nil {
		if !reHostsIgnore.MatchString(host[1]) {
			rootKeyKind = nodeKindHostnameRoot
			tokens = tokenize(host[1])
		}
	} else {
		tokens = tokenize(rule)
	}

	// temporary
	if len(tokens) == 0 {
		return
	}

	node := m.root.findOrAddChild(nodeKey{kind: rootKeyKind})
	for i, token := range tokens {
		if i == len(tokens)-1 {
			node = node.findOrAddChild(nodeKey{kind: nodeKindExactMatch, token: token})
			node.isLeaf = true
		} else {
			node = node.findOrAddChild(nodeKey{kind: nodeKindExactMatch, token: token})
		}
	}
}

// Match returns true if the given URL matches any of the rules.
// It expects the URL to be in the fully qualified form.
func (m *Matcher) Match(url string) bool {
	// address root -> hostname root -> domain -> etc.
	tokens := tokenize(url)

	// address root
	if match := m.root.findChild(nodeKey{kind: nodeKindAddressRoot}).match(tokens); match {
		return true
	}
	if match := m.root.match(tokens); match {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	tokens = tokens[1:]

	// protocol separator
	if match := m.root.match(tokens); match {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	tokens = tokens[1:]

	// hostname root
	if match := m.root.findChild(nodeKey{kind: nodeKindHostnameRoot}).match(tokens); match {
		return true
	}

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			if match := m.root.findChild(nodeKey{kind: nodeKindDomain}).match(tokens); match {
				return true
			}
		}
		if match := m.root.match(tokens); match {
			return true
		}
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters, etc.
	for len(tokens) > 0 {
		if match := m.root.findChild(nodeKey{kind: nodeKindExactMatch}).match(tokens); match {
			return true
		}
		tokens = tokens[1:]
	}

	return false
}

var (
	reTokenSep = regexp.MustCompile(`(^https|^http|\.|-|_|:\/\/|\/|\?|=|&)`)
)

func tokenize(s string) []string {
	tokenRanges := reTokenSep.FindAllStringIndex(s, -1)
	// assume that each separator is followed by a token
	// over-allocating is fine, since the token arrays will be short-lived
	tokens := make([]string, 0, len(tokenRanges)*2)

	var nextStartIndex int
	for i, tokenRange := range tokenRanges {
		tokens = append(tokens, s[tokenRange[0]:tokenRange[1]])

		nextStartIndex = tokenRange[1]
		if i < len(tokenRanges)-1 {
			nextEndIndex := tokenRanges[i+1][0]
			if nextStartIndex < nextEndIndex {
				tokens = append(tokens, s[nextStartIndex:nextEndIndex])
			}
		}
	}

	if nextStartIndex < len(s) {
		tokens = append(tokens, s[nextStartIndex:])
	}

	return tokens
}
