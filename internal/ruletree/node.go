package ruletree

import (
	"net/http"
	"regexp"
	"sync"
)

// nodeKind defines the type of a node in the trie.
type nodeKind int8

const (
	nodeKindExactMatch  nodeKind = iota
	nodeKindAddressRoot          // |
	nodeKindDomain               // ||
	nodeKindWildcard             // *
	nodeKindSeparator            // ^
	// nodeKindGeneric is a kind of node that matches any URL.
	nodeKindGeneric
)

// nodeKey uniquely identifies a node within the trie.
// It comprises the node's kind and the token that the node represents.
// The token is included only for nodes of the type 'nodeKindExactMatch'.
// Nodes of other kinds represent the roots of subtrees without including a token.
type nodeKey struct {
	kind  nodeKind
	token string
}

// arrNode is a node in the trie that is stored in an array.
type arrNode[T Data] struct {
	key  nodeKey
	node *node[T]
}

// nodeChildrenMaxArrSize specifies the maximum size for the array of child nodes.
// When the array's size exceeds this value, it is converted into a map.
// This aims to optimize memory usage since most nodes have only a few children.
// In Go, an empty map occupies 48 bytes of memory on 64-bit systems.
// See: https://go.dev/src/runtime/map.go
const nodeChildrenMaxArrSize = 8

// node represents a node in the rule trie.
// Nodes can be both vertices that only represent a subtree and leaves that represent a rule.
type node[T Data] struct {
	childrenArr []arrNode[T]
	childrenMap map[nodeKey]*node[T]
	// childrenMu protects childrenArr and childrenMap from concurrent access.
	childrenMu sync.RWMutex
	// data holds the rules associated with this node.
	data   []T
	dataMu sync.RWMutex
}

// findOrAddChild finds or adds a child node with the given key.
func (n *node[T]) findOrAddChild(key nodeKey) *node[T] {
	n.childrenMu.Lock()
	defer n.childrenMu.Unlock()

	if n.childrenMap == nil {
		for _, arrNode := range n.childrenArr {
			if arrNode.key == key {
				return arrNode.node
			}
		}
		if len(n.childrenArr) < nodeChildrenMaxArrSize {
			newNode := &node[T]{}
			n.childrenArr = append(n.childrenArr, arrNode[T]{key: key, node: newNode})
			return newNode
		}
		n.childrenMap = make(map[nodeKey]*node[T])
		for _, arrNode := range n.childrenArr {
			n.childrenMap[arrNode.key] = arrNode.node
		}
		n.childrenArr = nil
	}

	if child, ok := n.childrenMap[key]; ok {
		return child
	}

	newNode := &node[T]{}
	n.childrenMap[key] = newNode
	return newNode
}

// FindChild returns the child node with the given key.
func (n *node[T]) FindChild(key nodeKey) *node[T] {
	n.childrenMu.RLock()
	defer n.childrenMu.RUnlock()

	if n.childrenMap == nil {
		for _, arrNode := range n.childrenArr {
			if arrNode.key == key {
				return arrNode.node
			}
		}
		return nil
	}
	return n.childrenMap[key]
}

var (
	// reSeparator is a regular expression that matches the separator token.
	// According to https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters:
	// "Separator character is any character, but a letter, a digit, or one of the following: _ - . %. ... The end of the address is also accepted as separator.".
	reSeparator = regexp.MustCompile(`[^a-zA-Z0-9_\-\.%]`)
)

// TraverseFindMatchingRulesReq traverses the trie and returns the rules that match the given request.
func (n *node[T]) TraverseFindMatchingRulesReq(req *http.Request, tokens []string, shouldUseNode func(*node[T], []string) bool) (rules []T) {
	if n == nil {
		return rules
	}
	if shouldUseNode == nil {
		shouldUseNode = func(*node[T], []string) bool {
			return true
		}
	}

	if shouldUseNode(n, tokens) {
		// Check the node itself
		rules = append(rules, n.FindMatchingRulesReq(req)...)
	}

	if len(tokens) == 0 {
		// End of an address is a valid separator, see:
		// https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters.
		rules = append(rules, n.FindChild(nodeKey{kind: nodeKindSeparator}).TraverseFindMatchingRulesReq(req, tokens, shouldUseNode)...)
		return rules
	}
	if reSeparator.MatchString(tokens[0]) {
		rules = append(rules, n.FindChild(nodeKey{kind: nodeKindSeparator}).TraverseFindMatchingRulesReq(req, tokens[1:], shouldUseNode)...)
	}
	rules = append(rules, n.FindChild(nodeKey{kind: nodeKindWildcard}).TraverseFindMatchingRulesReq(req, tokens[1:], shouldUseNode)...)
	rules = append(rules, n.FindChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).TraverseFindMatchingRulesReq(req, tokens[1:], shouldUseNode)...)

	return rules
}

// TraverseFindMatchingRulesRes traverses the trie and returns the rules that match the given response.
func (n *node[T]) TraverseFindMatchingRulesRes(res *http.Response, tokens []string, shouldUseNode func(*node[T], []string) bool) (rules []T) {
	if n == nil {
		return rules
	}
	if shouldUseNode == nil {
		shouldUseNode = func(*node[T], []string) bool {
			return true
		}
	}

	if shouldUseNode(n, tokens) {
		// Check the node itself
		rules = append(rules, n.FindMatchingRulesRes(res)...)
	}

	if len(tokens) == 0 {
		// End of an address is a valid separator, see:
		// https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters.
		rules = append(rules, n.FindChild(nodeKey{kind: nodeKindSeparator}).TraverseFindMatchingRulesRes(res, tokens, shouldUseNode)...)
		return rules
	}
	if reSeparator.MatchString(tokens[0]) {
		rules = append(rules, n.FindChild(nodeKey{kind: nodeKindSeparator}).TraverseFindMatchingRulesRes(res, tokens[1:], shouldUseNode)...)
	}
	rules = append(rules, n.FindChild(nodeKey{kind: nodeKindWildcard}).TraverseFindMatchingRulesRes(res, tokens[1:], shouldUseNode)...)
	rules = append(rules, n.FindChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).TraverseFindMatchingRulesRes(res, tokens[1:], shouldUseNode)...)

	return rules
}

// FindMatchingRulesReq returns the rules that match the given request.
func (n *node[T]) FindMatchingRulesReq(req *http.Request) []T {
	n.dataMu.RLock()
	defer n.dataMu.RUnlock()

	var matchingRules []T
	for _, r := range n.data {
		if r.ShouldMatchReq(req) {
			matchingRules = append(matchingRules, r)
		}
	}
	return matchingRules
}

// FindMatchingRulesRes returns the rules that match the given response.
func (n *node[T]) FindMatchingRulesRes(res *http.Response) (rules []T) {
	n.dataMu.RLock()
	defer n.dataMu.RUnlock()

	for _, r := range n.data {
		if r.ShouldMatchRes(res) {
			rules = append(rules, r)
		}
	}
	return rules
}
