package ruletree

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/anfragment/zen/matcher/ruletree/modifiers"
	"github.com/anfragment/zen/matcher/ruletree/tokenize"
	"github.com/elazarl/goproxy"
)

// RuleTree is a trie-based matcher that is capable of parsing
// Adblock-style and hosts rules and matching URLs against thert.
//
// The matcher is safe for concurrent use.
type RuleTree struct {
	root *node
}

var (
	modifiersCG = `(?:\$(.+))?`
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreRule   = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
	reHosts        = regexp.MustCompile(`^(?:0\.0\.0\.0|127\.0\.0\.1) (.+)`)
	reHostsIgnore  = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
	reDomainName   = regexp.MustCompile(fmt.Sprintf(`^\|\|(.+?)%s$`, modifiersCG))
	reExactAddress = regexp.MustCompile(fmt.Sprintf(`^\|(.+?)%s$`, modifiersCG))
	reGeneric      = regexp.MustCompile(fmt.Sprintf(`^(.+?)%s$`, modifiersCG))
)

func NewRuleTree() *RuleTree {
	return &RuleTree{
		root: &node{},
	}
}

func (rt *RuleTree) AddRule(rule string) error {
	if reIgnoreRule.MatchString(rule) {
		return nil
	}

	var tokens []string
	var modifiersStr string
	var rootKeyKind nodeKind
	if host := reHosts.FindStringSubmatch(rule); host != nil {
		if !reHostsIgnore.MatchString(host[1]) {
			rootKeyKind = nodeKindHostnameRoot
			tokens = tokenize.Tokenize(host[1])
		}
	} else if match := reDomainName.FindStringSubmatch(rule); match != nil {
		rootKeyKind = nodeKindDomain
		tokens = tokenize.Tokenize(match[1])
		modifiersStr = match[2]
	} else if match := reExactAddress.FindStringSubmatch(rule); match != nil {
		rootKeyKind = nodeKindAddressRoot
		tokens = tokenize.Tokenize(match[1])
		modifiersStr = match[2]
	} else if match := reGeneric.FindStringSubmatch(rule); match != nil {
		rootKeyKind = nodeKindExactMatch
		tokens = tokenize.Tokenize(match[1])
		modifiersStr = match[2]
	} else {
		return fmt.Errorf("unknown rule format")
	}

	modifiers := &modifiers.RuleModifiers{}
	if err := modifiers.Parse(rule, modifiersStr); err != nil {
		// log.Printf("failed to parse modifiers for rule %q: %v", rule, err)
		return fmt.Errorf("failed to parse modifiers: %w", err)
	}

	var node *node
	if rootKeyKind == nodeKindExactMatch {
		node = rt.root
	} else {
		node = rt.root.findOrAddChild(nodeKey{kind: rootKeyKind})
	}
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

	return nil
}

func (rt *RuleTree) Middleware(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
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
	tokens := tokenize.Tokenize(url)

	// address root
	if req, resp := rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseAndHandleReq(req, tokens, func(n *node, t []string) bool {
		return len(t) == 0
	}); resp != nil {
		return req, resp
	}

	if req, resp := rt.root.TraverseAndHandleReq(req, tokens, nil); resp != nil {
		return req, resp
	}
	tokens = tokens[1:]

	// protocol separator
	if req, resp := rt.root.TraverseAndHandleReq(req, tokens, nil); resp != nil {
		return req, resp
	}
	tokens = tokens[1:]

	var hostnameMatcher func(*node, []string) (*http.Request, *http.Response)
	hostnameMatcher = func(rootNode *node, tokens []string) (*http.Request, *http.Response) {
		if req, resp := rootNode.TraverseAndHandleReq(req, tokens, func(n *node, t []string) bool {
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
	hostnameRootNode := rt.root.FindChild(nodeKey{kind: nodeKindHostnameRoot})
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
			if req, resp := rt.root.FindChild(nodeKey{kind: nodeKindDomain}).TraverseAndHandleReq(req, tokens, nil); resp != nil {
				return req, resp
			}
		}
		if req, resp := rt.root.TraverseAndHandleReq(req, tokens, nil); resp != nil {
			return req, resp
		}
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters, etc.
	for len(tokens) > 0 {
		if req, resp := rt.root.TraverseAndHandleReq(req, tokens, nil); resp != nil {
			return req, resp
		}
		tokens = tokens[1:]
	}

	return req, nil
}
