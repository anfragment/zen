package ruletree

import (
	"net/http"
	"regexp"
	"sync"

	"github.com/anfragment/zen/matcher/ruletree/modifiers"
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
// Other kinds of nodes only represent roots of subtrees.
type nodeKey struct {
	kind  nodeKind
	token string
}

// node is a node in the trie.
type node struct {
	children   map[nodeKey]*node
	childrenMu sync.RWMutex
	modifiers  []*modifiers.RuleModifiers
}

func (n *node) findOrAddChild(key nodeKey) *node {
	n.childrenMu.RLock()
	child, ok := n.children[key]
	n.childrenMu.RUnlock()
	if ok {
		return child
	}

	n.childrenMu.Lock()
	child = &node{}
	if n.children == nil {
		n.children = make(map[nodeKey]*node)
	}
	n.children[key] = child
	n.childrenMu.Unlock()
	return child
}

func (n *node) FindChild(key nodeKey) *node {
	n.childrenMu.RLock()
	child := n.children[key]
	n.childrenMu.RUnlock()
	return child
}

var (
	// reSeparator is a regular expression that matches the separator token.
	// according to the https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters
	// "Separator character is any character, but a letter, a digit, or one of the following: _ - . %. ... The end of the address is also accepted as separator."
	reSeparator = regexp.MustCompile(`[^a-zA-Z0-9]|[_\-.%]`)
)

// Match returns true if the node's subtree matches the given tokens.
//
// If a matching rule is found, it is returned along with the remaining tokens.
// If no matching rule is found, nil is returned.
func (n *node) Match(tokens []string) (*node, []string) {
	if n == nil {
		return nil, nil
	}
	if n.modifiers != nil {
		return n, tokens
	}
	if len(tokens) == 0 {
		if separator := n.FindChild(nodeKey{kind: nodeKindSeparator}); separator != nil && separator.modifiers != nil {
			return separator, tokens
		}
		return nil, nil
	}
	if reSeparator.MatchString(tokens[0]) {
		if match, _ := n.FindChild(nodeKey{kind: nodeKindSeparator}).Match(tokens[1:]); match != nil {
			return match, tokens
		}
	}
	if wildcard := n.FindChild(nodeKey{kind: nodeKindWildcard}); wildcard != nil {
		if match, _ := wildcard.Match(tokens[1:]); match != nil {
			return match, tokens
		}
	}

	return n.FindChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).Match(tokens[1:])
}

func (n *node) TraverseAndHandleReq(req *http.Request, tokens []string, shouldUseNode func(*node, []string) bool) (*http.Request, *http.Response) {
	if n == nil {
		return nil, nil
	}
	if shouldUseNode == nil {
		shouldUseNode = func(n *node, tokens []string) bool {
			return true
		}
	}
	if n.modifiers != nil && shouldUseNode(n, tokens) {
		return n.HandleRequest(req)
	}
	if len(tokens) == 0 {
		// end of an address is also accepted as a separator
		// see: https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters
		if separator := n.FindChild(nodeKey{kind: nodeKindSeparator}); separator != nil && separator.modifiers != nil && shouldUseNode(separator, tokens) {
			return separator.HandleRequest(req)
		}
		return nil, nil
	}
	if reSeparator.MatchString(tokens[0]) {
		if match, remainingTokens := n.FindChild(nodeKey{kind: nodeKindSeparator}).Match(tokens[1:]); match != nil && match.modifiers != nil && shouldUseNode(match, remainingTokens) {
			return match.HandleRequest(req)
		}
	}
	if wildcard := n.FindChild(nodeKey{kind: nodeKindWildcard}); wildcard != nil {
		if match, remainingTokens := wildcard.Match(tokens[1:]); match != nil && match.modifiers != nil && shouldUseNode(match, remainingTokens) {
			return match.HandleRequest(req)
		}
	}

	return n.FindChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).TraverseAndHandleReq(req, tokens[1:], shouldUseNode)
}

func (n *node) HandleRequest(req *http.Request) (*http.Request, *http.Response) {
	for _, modifier := range n.modifiers {
		req, resp := modifier.HandleRequest(req)
		if resp != nil {
			return req, resp
		}
	}
	return req, nil
}
