package matcher

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
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
	modifiers  []*ruleModifiers
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
	// log.Printf("matching %s, current node: children length=%d; modifiers length=%d", strings.Join(tokens, "|"), len(n.children), len(n.modifiers))
	if n.modifiers != nil {
		return n, tokens
	}
	if len(tokens) == 0 {
		if separator := n.findChild(nodeKey{kind: nodeKindSeparator}); separator != nil && separator.modifiers != nil {
			return separator, tokens
		}
		return nil, nil
	}
	// TODO: return multiple matches if they exist
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

func (n *node) traverseAndHandleReq(req *http.Request, tokens []string, shouldUseNode func(*node, []string) bool) (*http.Request, *http.Response) {
	if n == nil {
		return nil, nil
	}
	if shouldUseNode == nil {
		shouldUseNode = func(n *node, tokens []string) bool {
			return true
		}
	}
	if len(tokens) == 0 {
		// end of an address is also accepted as a separator
		// see: https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-special-characters
		if separator := n.findChild(nodeKey{kind: nodeKindSeparator}); separator != nil && separator.modifiers != nil && shouldUseNode(separator, tokens) {
			return separator.handleRequest(req)
		}
		return nil, nil
	}
	if reSeparator.MatchString(tokens[0]) {
		if match, remainingTokens := n.findChild(nodeKey{kind: nodeKindSeparator}).match(tokens[1:]); match != nil && match.modifiers != nil && shouldUseNode(match, remainingTokens) {
			return match.handleRequest(req)
		}
	}
	if wildcard := n.findChild(nodeKey{kind: nodeKindWildcard}); wildcard != nil {
		if match, remainingTokens := wildcard.match(tokens[1:]); match != nil && match.modifiers != nil && shouldUseNode(match, remainingTokens) {
			return match.handleRequest(req)
		}
	}
	if tokens[0] == "yadro" {
		log.Printf("yadro: %+v", n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}))
	}
	if tokens[0] == "." {
		log.Printf("dot: %+v", n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}))
	}
	if tokens[0] == "ru" {
		log.Printf("ru: %+v", n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}))
	}
	return n.findChild(nodeKey{kind: nodeKindExactMatch, token: tokens[0]}).traverseAndHandleReq(req, tokens[1:], shouldUseNode)
}

func (n *node) handleRequest(req *http.Request) (*http.Request, *http.Response) {
	for _, modifier := range n.modifiers {
		req, resp := modifier.handleRequest(req)
		if resp != nil {
			return req, resp
		}
	}
	return req, nil
}

type modifierType int

const (
	modifierTypeNone modifierType = iota
	modifierTypeInclude
	modifierTypeExclude
)

// ruleModifiers represents modifiers of a rule.
type ruleModifiers struct {
	rule    string
	generic bool
	// basic modifiers
	// https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-basic-modifiers
	// domain     string
	// thirdParty optionType
	// header     string
	// important  optionType
	// method     string
	// content type modifiers
	// https://adguard.com/kb/general/ad-filtering/create-own-filters/#content-type-modifiers
	document   modifierType
	font       modifierType
	image      modifierType
	media      modifierType
	script     modifierType
	stylesheet modifierType
	other      modifierType
}

func (m *ruleModifiers) handleRequest(req *http.Request) (*http.Request, *http.Response) {
	blockResponse := goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "blocked by zen")

	if m.generic {
		log.Printf("blocking with rule %s", m.rule)
		return req, blockResponse
	}

	modifiers := map[string]modifierType{
		"document": m.document,
		"font":     m.font,
		"image":    m.image,
		"audio":    m.media,
		"video":    m.media,
		"script":   m.script,
		"style":    m.stylesheet,
	}

	dest := req.Header.Get("Sec-Fetch-Dest")
	if val, ok := modifiers[dest]; ok {
		if val == modifierTypeInclude {
			log.Printf("blocking with rule %s", m.rule)
			return req, blockResponse
		}
	} else if m.other == modifierTypeInclude {
		log.Printf("blocking with rule %s", m.rule)
		return req, blockResponse
	}

	return req, nil
}

func parseModifiers(modifiers string) (*ruleModifiers, error) {
	if modifiers == "" {
		return nil, nil
	}

	m := &ruleModifiers{}
	for _, modifier := range strings.Split(modifiers, ",") {
		if strings.ContainsRune(modifier, '=') {
			// TODO: handle key=value modifiers
			return nil, fmt.Errorf("key=value modifiers are not supported")
		}
		t := modifierTypeInclude
		if modifier[0] == '~' {
			t = modifierTypeExclude
			modifier = modifier[1:]
		}
		switch modifier {
		case "document":
			m.document = t
		case "font":
			m.font = t
		case "image":
			m.image = t
		case "media":
			m.media = t
		case "other":
			m.other = t
		case "script":
			m.script = t
		case "stylesheet":
			m.stylesheet = t
		default:
			// first, do no harm
			// in case an unknown modifier is encountered, ignore the whole rule
			return nil, fmt.Errorf("unknown modifier %q", modifier)
		}
	}
	return m, nil
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
	hostnameCG  = `((?:[\da-z][\da-z_-]*\.)+[\da-z-]*[a-z])`
	urlCG       = `(https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*))`
	modifiersCG = `(?:\$(.+))?`
	// Ignore comments, cosmetic rules, [Adblock Plus 2.0]-style, and, temporarily, exception rules.
	reIgnoreRule           = regexp.MustCompile(`^(?:!|#|\[|@@)|(##|#\?#|#\$#|#@#)`)
	reHosts                = regexp.MustCompile(fmt.Sprintf(`^(?:0\.0\.0\.0|127\.0\.0\.1) %s`, hostnameCG))
	reHostsIgnore          = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
	reDomainName           = regexp.MustCompile(fmt.Sprintf(`^\|\|%s\^%s$`, hostnameCG, modifiersCG))
	reExactAddress         = regexp.MustCompile(fmt.Sprintf(`^\|%s%s$`, urlCG, modifiersCG))
	reAddressPartsModifier = regexp.MustCompile(fmt.Sprintf(`%s$`, modifiersCG))
)

func (m *Matcher) AddRule(rule string) {
	if reIgnoreRule.MatchString(rule) {
		return
	}

	var tokens []string
	var modifiers *ruleModifiers
	var err error
	rootKeyKind := nodeKindExactMatch
	if host := reHosts.FindStringSubmatch(rule); host != nil {
		if !reHostsIgnore.MatchString(host[1]) {
			rootKeyKind = nodeKindHostnameRoot
			tokens = tokenize(host[1])
		}
	} else if match := reDomainName.FindStringSubmatch(rule); match != nil {
		rootKeyKind = nodeKindDomain
		tokens = tokenize(match[1])
		if modifiers, err = parseModifiers(match[2]); err != nil {
			return
		}
	} else if match := reExactAddress.FindStringSubmatch(rule); match != nil {
		rootKeyKind = nodeKindAddressRoot
		tokens = tokenize(match[1])
		if modifiers, err = parseModifiers(match[2]); err != nil {
			return
		}
	} else {
		tokens = tokenize(rule)
		if match := reAddressPartsModifier.FindStringSubmatch(rule); match != nil {
			if modifiers, err = parseModifiers(match[1]); err != nil {
				return
			}
		}
	}
	if modifiers == nil {
		modifiers = &ruleModifiers{generic: true}
	}
	modifiers.rule = rule

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
	node.modifiers = append(node.modifiers, modifiers)
}

// AddRemoteFilters parses the rules files at the given URLs and adds them to
// the filter.
func (m *Matcher) AddRemoteFilters(urls []string) error {
	c := 0
	for _, url := range urls {
		file, err := http.Get(url)
		if err != nil {
			log.Printf("failed to download rules file %s: %v", url, err)
		}
		defer file.Body.Close()
		reader := bufio.NewReader(file.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("failed to read line from rules file %s: %v", url, err)
			}
			line = line[:len(line)-1] // strip the trailing newline
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			m.AddRule(line)
			c++
		}
	}
	log.Printf("added %d rules", c)
	return nil
}

func (m *Matcher) Middleware(req *http.Request, ctx *goproxy.ProxyCtx) (endReq *http.Request, endRes *http.Response) {
	defer func() {
		if endRes != nil {
			log.Printf("matched %s -> %s", req.URL.Hostname(), endRes.Status)
		}
	}()

	host := req.URL.Hostname()
	urlWithoutPort := url.URL{
		Scheme:   req.URL.Scheme,
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
		Fragment: req.URL.Fragment,
	}

	url := urlWithoutPort.String()
	// address root -> hostname root -> domain -> etc.
	tokens := tokenize(url)

	// address root
	if req, resp := m.root.findChild(nodeKey{kind: nodeKindAddressRoot}).traverseAndHandleReq(req, tokens, func(n *node, t []string) bool {
		return len(t) == 0
	}); resp != nil {
		return req, resp
	}

	if req, resp := m.root.traverseAndHandleReq(req, tokens, nil); resp != nil {
		return req, resp
	}
	tokens = tokens[1:]

	// protocol separator
	if req, resp := m.root.traverseAndHandleReq(req, tokens, nil); resp != nil {
		return req, resp
	}
	tokens = tokens[1:]

	var hostnameMatcher func(*node, []string) (*http.Request, *http.Response)
	hostnameMatcher = func(rootNode *node, tokens []string) (*http.Request, *http.Response) {
		if req, resp := rootNode.traverseAndHandleReq(req, tokens, func(n *node, t []string) bool {
			return len(t) == 0 || t[0] != "."
		}); resp != nil {
			return req, resp
		}
		if len(tokens) > 2 && tokens[1] == "." {
			// try to match the next domain segment
			tokens = tokens[2:]
			return hostnameMatcher(rootNode, tokens)
		}
		return nil, nil
	}

	// hostname root
	hostnameRootNode := m.root.findChild(nodeKey{kind: nodeKindHostnameRoot})
	if hostnameRootNode != nil {
		if req, resp := hostnameMatcher(hostnameRootNode, tokens); resp != nil {
			return req, resp
		}
	}

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			if match, _ := m.root.findChild(nodeKey{kind: nodeKindDomain}).match(tokens); match != nil {
				req, resp := match.handleRequest(req)
				if resp != nil {
					return req, resp
				}
			}
		}
		if match, _ := m.root.match(tokens); match != nil {
			req, resp := match.handleRequest(req)
			if resp != nil {
				return req, resp
			}
		}
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters, etc.
	for len(tokens) > 0 {
		if match, _ := m.root.findChild(nodeKey{kind: nodeKindExactMatch}).match(tokens); match != nil {
			res, resp := match.handleRequest(req)
			if resp != nil {
				return res, resp
			}
		}
		tokens = tokens[1:]
	}

	return req, nil
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
