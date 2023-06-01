package filter

import (
	"regexp"
	"sync"
)

type matcherKey struct {
	// start is true if this node should only match the start of a string.
	start bool
	// token is the token to match.
	token string
}

type matcherNode struct {
	children   map[matcherKey]*matcherNode
	childrenMu sync.RWMutex
	leaf       bool
}

type Matcher struct {
	root *matcherNode
}

func NewMatcher() *Matcher {
	return &Matcher{
		root: &matcherNode{
			children: make(map[matcherKey]*matcherNode),
		},
	}
}

func (m *Matcher) AddRule(rule string) {
	tokens := tokenize(rule)
	node := m.root
	for i, token := range tokens {
		key := matcherKey{
			start: false,
			token: token,
		}
		node.childrenMu.RLock()
		child, ok := node.children[key]
		node.childrenMu.RUnlock()
		if ok {
			node = child
			if i == len(tokens)-1 {
				node.leaf = true
			}
		} else {
			child = &matcherNode{
				leaf: i == len(tokens)-1,
			}
			node.childrenMu.Lock()
			if node.children == nil {
				node.children = make(map[matcherKey]*matcherNode, 1)
			}
			node.children[key] = child
			node.childrenMu.Unlock()
			node.leaf = false
			node = child
		}
	}
}

func (m *Matcher) Match(url string) bool {
	tokens := tokenize(url)
	for i := range tokens {
		node := m.root
		for j := i; j < len(tokens); j++ {
			key := matcherKey{
				start: false,
				token: tokens[j],
			}
			node.childrenMu.RLock()
			child, ok := node.children[key]
			node.childrenMu.RUnlock()
			if !ok {
				break
			}
			node = child
			if node.leaf {
				return true
			}
		}
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
