package ruletree

import (
	"net/http"
	"regexp"
	"sync"

	"github.com/anfragment/zen/filter/ruletree/rule"
)

// nodeKind defines the type of a node in the trie.
type nodeKind int8

const (
	nodeKindExactMatch  nodeKind = iota
	nodeKindAddressRoot          // |
	nodeKindDomain               // ||
	nodeKindWildcard             // *
	nodeKindSeparator            // ^
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
type arrNode struct {
	key  nodeKey
	node *node
}

// nodeChildrenMaxArrSize specifies the maximum size for the array of child nodes.
// When the array's size exceeds this value, it is converted into a map.
// This aims to optimize memory usage since most nodes have only a few children.
// In Go, an empty map occupies 48 bytes of memory on 64-bit systems.
// See: https://go.dev/src/runtime/map.go
const nodeChildrenMaxArrSize = 8

// node represents a node in the rule trie.
// Nodes can be both vertices that only represent a subtree and leaves that represent a rule.
type node struct {
	childrenArr []arrNode
	childrenMap map[nodeKey]*node
	childrenMu  sync.RWMutex
	// rules is the list of rules that match the node's subtree.
	// The list is empty for nodes that do not represent a rule.
	rules []rule.Rule
}

// findOrAddChild finds or adds a child node with the given key.
func (n *node) findOrAddChild(key nodeKey) *node {
	n.childrenMu.Lock()
	defer n.childrenMu.Unlock()

	if n.childrenMap == nil {
		for _, arrNode := range n.childrenArr {
			if arrNode.key == key {
				return arrNode.node
			}
		}
		if len(n.childrenArr) < nodeChildrenMaxArrSize {
			newNode := &node{}
			n.childrenArr = append(n.childrenArr, arrNode{key: key, node: newNode})
			return newNode
		}
		n.childrenMap = make(map[nodeKey]*node)
		for _, arrNode := range n.childrenArr {
			n.childrenMap[arrNode.key] = arrNode.node
		}
		n.childrenArr = nil
	}

	if child, ok := n.childrenMap[key]; ok {
		return child
	}

	newNode := &node{}
	n.childrenMap[key] = newNode
	return newNode
}

// FindChild returns the child node with the given key.
func (n *node) FindChild(key nodeKey) *node {
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
	// "Separator character is any character, but a letter, a digit, or one of the following: _ - . %. ... The end of the address is also accepted as separator."
	reSeparator = regexp.MustCompile(`[^a-zA-Z0-9]|[_\-.%]`)
)

// Match returns the node that matches the given tokens along with the remaining tokens.
// If no node matches, it returns nil.
func (n *node) Match(tokens []string) (*node, []string) {
	if n == nil {
		return nil, nil
	}
	if len(n.rules) > 0 {
		return n, tokens
	}
	if len(tokens) == 0 {
		if separator := n.FindChild(nodeKey{kind: nodeKindSeparator}); separator != nil && len(separator.rules) > 0 {
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

// TraverseAndHandleReq traverses the trie and returns the action to take for the given request.
func (n *node) TraverseAndHandleReq(req *http.Request, tokens []string, shouldUseNode func(*node, []string) bool) rule.RequestAction {
	if n == nil {
		return rule.RequestAction{Type: rule.ActionAllow}
	}
	if shouldUseNode == nil {
		shouldUseNode = func(n *node, tokens []string) bool {
			return true
		}
	}
	if len(n.rules) > 0 && shouldUseNode(n, tokens) {
		if action := n.HandleRequest(req); action.Type != rule.ActionAllow {
			return action
		}
	}
	if len(tokens) == 0 {
		// end of an address is also accepted as a separator
		// see: https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters
		if separator := n.FindChild(nodeKey{kind: nodeKindSeparator}); separator != nil && len(separator.rules) > 0 && shouldUseNode(separator, tokens) {
			if action := separator.HandleRequest(req); action.Type != rule.ActionAllow {
				return action
			}
		}
		return n.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseAndHandleReq(req, tokens, shouldUseNode)
	}
	if reSeparator.MatchString(tokens[0]) {
		if match, remainingTokens := n.FindChild(nodeKey{kind: nodeKindSeparator}).Match(tokens[1:]); match != nil && len(match.rules) > 0 && shouldUseNode(match, remainingTokens) {
			if action := match.HandleRequest(req); action.Type != rule.ActionAllow {
				return action
			}
		}
	}
	if wildcard := n.FindChild(nodeKey{kind: nodeKindWildcard}); wildcard != nil {
		if match, remainingTokens := wildcard.Match(tokens[1:]); match != nil && len(match.rules) > 0 && shouldUseNode(match, remainingTokens) {
			if action := match.HandleRequest(req); action.Type != rule.ActionAllow {
				return action
			}
		}
	}

	return n.FindChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).TraverseAndHandleReq(req, tokens[1:], shouldUseNode)
}

// HandleRequest returns the action to take for the given request.
func (n *node) HandleRequest(req *http.Request) rule.RequestAction {
	for _, r := range n.rules {
		action := r.HandleRequest(req)
		if action.Type != rule.ActionAllow {
			return action
		}
	}
	return rule.RequestAction{Type: rule.ActionAllow}
}
