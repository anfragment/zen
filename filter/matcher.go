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
	// isRule is true if the node represents a rule.
	//
	// Nodes with isRule set to true might have children and
	// therefore should not necessarily terminate the matching process.
	isRule bool
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

var (
	// reSeparator is a regular expression that matches the separator token.
	// according to the https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters
	// "Separator character is any character, but a letter, a digit, or one of the following: _ - . %. ... The end of the address is also accepted as separator."
	reSeparator = regexp.MustCompile(`[^a-zA-Z0-9]|[_\-.%]`)
)

// match returns true if the node's subtree matches the given tokens.
//
// If a matching rule is found, it is returned along with the remaining tokens.
// If no matching rule is found, nil is returned.
func (n *node) match(tokens []string) (*node, []string) {
	if n == nil {
		return nil, nil
	}
	if n.isRule {
		return n, tokens
	}
	if len(tokens) == 0 {
		if separator := n.findChild(nodeKey{kind: nodeKindSeparator}); separator != nil && separator.isRule {
			return separator, tokens
		}
		return nil, nil
	}
	if reSeparator.MatchString(tokens[0]) {
		if match, _ := n.findChild(nodeKey{kind: nodeKindSeparator}).match(tokens[1:]); match != nil {
			return match, tokens
		}
	}
	if wildcard := n.findChild(nodeKey{kind: nodeKindWildcard}); wildcard != nil {
		if match, _ := wildcard.match(tokens[1:]); match != nil {
			return match, tokens
		}
	}

	return n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).match(tokens[1:])
}

// Matcher is trie-based matcher for URLs that is capable of parsing
// Adblock filters and hosts rules and matching URLs against them.
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
	hostnameCG     = `((?:[\da-z][\da-z_-]*\.)+[\da-z-]*[a-z])`
	urlCG          = `https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*)`
	reHosts        = regexp.MustCompile(fmt.Sprintf(`^(?:0\.0\.0\.0|127\.0\.0\.1) %s`, hostnameCG))
	reHostsIgnore  = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
	reDomainName   = regexp.MustCompile(fmt.Sprintf(`^\|\|%s\^`, hostnameCG))
	reExactAddress = regexp.MustCompile(fmt.Sprintf(`^\|%s$`, urlCG))
)

func (m *Matcher) AddRule(rule string) {
	rootKeyKind := nodeKindExactMatch
	var tokens []string

	if host := reHosts.FindStringSubmatch(rule); host != nil {
		if !reHostsIgnore.MatchString(host[1]) {
			rootKeyKind = nodeKindHostnameRoot
			tokens = tokenize(host[1])
		}
	} else if match := reDomainName.MatchString(rule); match {
		rootKeyKind = nodeKindDomain
		tokens = tokenize(rule[2 : len(rule)-1]) // 2:len(rule)-1 to skip the leading || and trailing ^
	} else if match := reExactAddress.MatchString(rule); match {
		rootKeyKind = nodeKindAddressRoot
		tokens = tokenize(rule[1:]) // 1: to skip the leading |
	} else {
		tokens = tokenize(rule)
	}

	if len(tokens) == 0 {
		return
	}

	node := m.root.findOrAddChild(nodeKey{kind: rootKeyKind})
	for _, token := range tokens {
		if token == "^" {
			node = node.findOrAddChild(nodeKey{kind: nodeKindSeparator})
		} else if token == "*" {
			node = node.findOrAddChild(nodeKey{kind: nodeKindWildcard})
		} else {
			node = node.findOrAddChild(nodeKey{kind: nodeKindExactMatch, token: token})
		}
	}
	node.isRule = true
}

// Match returns true if the given URL matches any of the rules.
// It expects the URL to be in the fully qualified form.
func (m *Matcher) Match(url string) bool {
	// address root -> hostname root -> domain -> etc.
	tokens := tokenize(url)

	// address root
	if match, remaingTokens := m.root.findChild(nodeKey{kind: nodeKindAddressRoot}).match(tokens); match != nil && len(remaingTokens) == 0 {
		return true
	}
	if match, _ := m.root.match(tokens); match != nil {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	tokens = tokens[1:]

	// protocol separator
	if match, _ := m.root.match(tokens); match != nil {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	tokens = tokens[1:]

	var hostnameMatcher func(*node, []string) bool
	hostnameMatcher = func(node *node, tokens []string) bool {
		if match, remainingTokens := node.match(tokens); match != nil {
			if len(remainingTokens) == 0 || remainingTokens[0] != "." {
				// hostname matched the entire hostname
				return true
			}
			if remainingTokens[0] == "." {
				return hostnameMatcher(match.findChild(nodeKey{kind: nodeKindExactMatch, token: "."}), remainingTokens[1:])
			}
		}
		return false
	}

	// hostname root
	hostnameRootNode := m.root.findChild(nodeKey{kind: nodeKindHostnameRoot})
	if hostnameRootNode != nil && hostnameMatcher(hostnameRootNode, tokens) {
		return true
	}

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			if match, _ := m.root.findChild(nodeKey{kind: nodeKindDomain}).match(tokens); match != nil {
				return true
			}
		}
		if match, _ := m.root.match(tokens); match != nil {
			return true
		}
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters, etc.
	for len(tokens) > 0 {
		if match, _ := m.root.findChild(nodeKey{kind: nodeKindExactMatch}).match(tokens); match != nil {
			return true
		}
		tokens = tokens[1:]
	}

	return false
}

var (
	reTokenSep = regexp.MustCompile(`(^https|^http|\.|-|_|:\/\/|\/|\?|=|&|:|\^)`)
)

func tokenize(s string) []string {
	tokenRanges := reTokenSep.FindAllStringIndex(s, -1)
	// assume that each separator is followed by a token
	// over-allocating is fine, since the token arrays will be short-lived
	tokens := make([]string, 0, len(tokenRanges)+1)

	// check if the first range doesn't start at the beginning of the string
	// if it doesn't, then the first token is the substring from the beginning
	// of the string to the start of the first range
	if len(tokenRanges) > 0 && tokenRanges[0][0] > 0 {
		tokens = append(tokens, s[:tokenRanges[0][0]])
	}

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
