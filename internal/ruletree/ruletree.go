package ruletree

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/anfragment/zen/internal/rule"
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
	// matchingPartCG matches the part of a rule that is used to match URLs.
	// Note: the '$' character is excluded due to its use as the separator between the matching part and modifiers.
	// This means that rules containing '$' in the matching part will get disregarded, but I can't think of any other
	// way to reliably distinguish between the matching part and modifiers.
	matchingPartCG = `([^$]+)`
	// modifiersCG matches the modifiers part of a rule.
	modifiersCG    = `(?:\$(.+))`
	reHosts        = regexp.MustCompile(`^(?:0\.0\.0\.0|127\.0\.0\.1) (.+)`)
	reHostsIgnore  = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
	reDomainName   = regexp.MustCompile(fmt.Sprintf(`^\|\|%s%s?$`, matchingPartCG, modifiersCG))
	reExactAddress = regexp.MustCompile(fmt.Sprintf(`^\|%s%s?$`, matchingPartCG, modifiersCG))
	reAddressParts = regexp.MustCompile(fmt.Sprintf(`^%s%s?$`, matchingPartCG, modifiersCG))
	// reGeneric matches rules without a matching part, e.g. `$removeparam=utm_referrer`.
	reGeneric = regexp.MustCompile(fmt.Sprintf(`^%s+$`, modifiersCG))
)

func NewRuleTree() *RuleTree {
	return &RuleTree{
		root:  node{},
		hosts: make(map[string]*string),
	}
}

func (rt *RuleTree) AddRule(rawRule string, filterName *string) error {
	if reHosts.MatchString(rawRule) {
		// Strip the # and any characters after it
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
	} else if match := reAddressParts.FindStringSubmatch(rawRule); match != nil {
		rootKeyKind = nodeKindExactMatch
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reGeneric.FindStringSubmatch(rawRule); match != nil {
		rootKeyKind = nodeKindGeneric
		tokens = []string{}
		modifiers = match[1]
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
		switch token {
		case "^":
			node = node.findOrAddChild(nodeKey{kind: nodeKindSeparator})
		case "*":
			node = node.findOrAddChild(nodeKey{kind: nodeKindWildcard})
		default:
			node = node.findOrAddChild(nodeKey{kind: nodeKindExactMatch, token: token})
		}
	}
	node.rules = append(node.rules, rule)

	return nil
}

// FindMatchingRulesReq finds all rules that match the given request.
func (rt *RuleTree) FindMatchingRulesReq(req *http.Request) (rules []rule.Rule) {
	host := req.URL.Hostname()
	rt.hostsMu.RLock()
	if filterName, ok := rt.hosts[host]; ok {
		rt.hostsMu.RUnlock()
		// 0.0.0.0 may not be the actual IP defined in the hosts file,
		// but storing the actual one feels wasteful.
		return []rule.Rule{
			{
				RawRule:    fmt.Sprintf("0.0.0.0 %s", host),
				FilterName: filterName,
			},
		}
	}
	rt.hostsMu.RUnlock()

	urlWithoutPort := url.URL{
		Scheme:   req.URL.Scheme,
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	url := urlWithoutPort.String()
	tokens := tokenize(url)

	// generic rules -> address root -> hostname root -> domain -> etc.

	// generic rules
	if genericNode := rt.root.FindChild(nodeKey{kind: nodeKindGeneric}); genericNode != nil {
		rules = append(rules, genericNode.FindMatchingRulesReq(req)...)
	}

	// address root
	rules = append(rules, rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseFindMatchingRulesReq(req, tokens, func(_ *node, t []string) bool {
		// address root rules have to match the entire URL
		// TODO: look into whether we can match the rule if the remaining tokens only contain the query
		return len(t) == 0
	})...)

	rules = append(rules, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
	tokens = tokens[1:]

	// protocol separator
	rules = append(rules, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
	tokens = tokens[1:]

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			rules = append(rules, rt.root.FindChild(nodeKey{kind: nodeKindDomain}).TraverseFindMatchingRulesReq(req, tokens, nil)...)
		}
		rules = append(rules, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters
	for len(tokens) > 0 {
		rules = append(rules, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
		tokens = tokens[1:]
	}

	return rules
}

// FindMatchingRulesRes finds all rules that match the given response.
// It assumes that the request that generated the response has already been matched by FindMatchingRulesReq.
func (rt *RuleTree) FindMatchingRulesRes(req *http.Request, res *http.Response) (rules []rule.Rule) {
	urlWithoutPort := url.URL{
		Scheme:   req.URL.Scheme,
		Host:     req.URL.Hostname(),
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	url := urlWithoutPort.String()
	tokens := tokenize(url)

	// generic rules -> address root -> hostname root -> domain -> etc.

	// generic rules
	if genericNode := rt.root.FindChild(nodeKey{kind: nodeKindGeneric}); genericNode != nil {
		rules = append(rules, genericNode.FindMatchingRulesRes(res)...)
	}

	// address root
	rules = append(rules, rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseFindMatchingRulesRes(res, tokens, func(_ *node, t []string) bool {
		return len(t) == 0
	})...)

	rules = append(rules, rt.root.TraverseFindMatchingRulesRes(res, tokens, nil)...)
	tokens = tokens[1:]

	// protocol separator
	rules = append(rules, rt.root.TraverseFindMatchingRulesRes(res, tokens, nil)...)
	tokens = tokens[1:]

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			rules = append(rules, rt.root.FindChild(nodeKey{kind: nodeKindDomain}).TraverseFindMatchingRulesRes(res, tokens, nil)...)
		}
		rules = append(rules, rt.root.TraverseFindMatchingRulesRes(res, tokens, nil)...)
		tokens = tokens[1:]
	}

	// rest of the URL
	for len(tokens) > 0 {
		rules = append(rules, rt.root.TraverseFindMatchingRulesRes(res, tokens, nil)...)
		tokens = tokens[1:]
	}

	return rules
}
