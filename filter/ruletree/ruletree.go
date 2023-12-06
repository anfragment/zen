package ruletree

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/anfragment/zen/filter/ruletree/rule"
)

// RuleTree is a trie-based filter that is capable of parsing
// Adblock-style and hosts rules and matching URLs against them.
//
// The filter is safe for concurrent use.
type RuleTree struct {
	// root is the root node of the trie that stores the rules.
	root node
	// hosts maps hostnames to filter names.
	hosts   map[string]*string
	hostsMu sync.RWMutex
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

func NewRuleTree() RuleTree {
	return RuleTree{
		root:  node{},
		hosts: make(map[string]*string),
	}
}

func (rt *RuleTree) AddRule(rawRule string, filterName *string) error {
	if reIgnoreRule.MatchString(rawRule) {
		return nil
	}

	if reHosts.MatchString(rawRule) {
		// strip the # and any characters after it
		if commentIndex := strings.IndexByte(rawRule, '#'); commentIndex != -1 {
			rawRule = rawRule[:commentIndex]
		}

		host := reHosts.FindStringSubmatch(rawRule)[1]
		if strings.ContainsRune(host, ' ') {
			for _, host := range strings.Split(host, " ") {
				rt.AddRule(fmt.Sprintf("127.0.0.1 %s", host), filterName)
			}
			return nil
		}
		if reHostsIgnore.MatchString(host) {
			return nil
		}

		rt.hostsMu.Lock()
		rt.hosts[host] = filterName
		rt.hostsMu.Unlock()

		return nil
	}

	var tokens []string
	var modifiers string
	var rootKeyKind nodeKind
	if match := reDomainName.FindStringSubmatch(rawRule); match != nil {
		rootKeyKind = nodeKindDomain
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reExactAddress.FindStringSubmatch(rawRule); match != nil {
		rootKeyKind = nodeKindAddressRoot
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reGeneric.FindStringSubmatch(rawRule); match != nil {
		rootKeyKind = nodeKindExactMatch
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else {
		return fmt.Errorf("unknown rule format")
	}

	rule := rule.Rule{
		RawRule:    rawRule,
		FilterName: filterName,
	}
	if err := rule.ParseModifiers(modifiers); err != nil {
		// log.Printf("failed to parse modifiers for rule %q: %v", rule, err)
		return fmt.Errorf("parse modifiers: %w", err)
	}

	var node *node
	if rootKeyKind == nodeKindExactMatch {
		node = &rt.root
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
	node.rules = append(node.rules, rule)

	return nil
}

func (rt *RuleTree) HandleRequest(req *http.Request) rule.RequestAction {
	host := req.URL.Hostname()
	rt.hostsMu.RLock()
	if filterName, ok := rt.hosts[host]; ok {
		rt.hostsMu.RUnlock()
		// 0.0.0.0 may not be the actual IP address defined in the hosts file,
		// but storing the actual one feels wasteful.
		return rule.RequestAction{Type: rule.ActionBlock, RawRule: fmt.Sprintf("0.0.0.0 %s", host), FilterName: *filterName}
	}
	rt.hostsMu.RUnlock()

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
	if action := rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseAndHandleReq(req, tokens, func(n *node, t []string) bool {
		// address root rules have to match the entire URL
		return len(t) == 0
	}); action.Type != rule.ActionAllow {
		return action
	}

	if action := rt.root.TraverseAndHandleReq(req, tokens, nil); action.Type != rule.ActionAllow {
		return action
	}
	tokens = tokens[1:]

	// protocol separator
	if action := rt.root.TraverseAndHandleReq(req, tokens, nil); action.Type != rule.ActionAllow {
		return action
	}
	tokens = tokens[1:]

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			if action := rt.root.FindChild(nodeKey{kind: nodeKindDomain}).TraverseAndHandleReq(req, tokens, nil); action.Type != rule.ActionAllow {
				return action
			}
		}
		if action := rt.root.TraverseAndHandleReq(req, tokens, nil); action.Type != rule.ActionAllow {
			return action
		}
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters, etc.
	for len(tokens) > 0 {
		if action := rt.root.TraverseAndHandleReq(req, tokens, nil); action.Type != rule.ActionAllow {
			return action
		}
		tokens = tokens[1:]
	}

	return rule.RequestAction{Type: rule.ActionAllow}
}
