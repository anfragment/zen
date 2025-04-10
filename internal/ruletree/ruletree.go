package ruletree

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

type Data interface {
	ShouldMatchRes(res *http.Response) bool
	ShouldMatchReq(req *http.Request) bool
	ParseModifiers(modifiers string) error
}

// RuleTree is a trie-based matcher that is capable of parsing
// Adblock-style and hosts rules and matching URLs against them.
//
// It is safe for concurrent use.
type RuleTree[T Data] struct {
	// root is the root node of the trie that stores the rules.
	root node[T]
}

var (
	// matchingPartCG matches the part of a rule that is used to match URLs.
	// Note: the '$' character is excluded due to its use as the separator between the matching part and modifiers.
	// This means that rules containing '$' in the matching part will get disregarded, but I can't think of any other
	// way to reliably distinguish between the matching part and modifiers.
	matchingPartCG = `([^$]+)`
	// modifiersCG matches the modifiers part of a rule.
	modifiersCG    = `(?:\$(.+))`
	reDomainName   = regexp.MustCompile(fmt.Sprintf(`^\|\|%s%s?$`, matchingPartCG, modifiersCG))
	reExactAddress = regexp.MustCompile(fmt.Sprintf(`^\|%s%s?$`, matchingPartCG, modifiersCG))
	reAddressParts = regexp.MustCompile(fmt.Sprintf(`^%s%s?$`, matchingPartCG, modifiersCG))
	// reGeneric matches rules without a matching part, e.g. `$removeparam=utm_referrer`.
	reGeneric = regexp.MustCompile(fmt.Sprintf(`^%s+$`, modifiersCG))
)

func NewRuleTree[T Data]() *RuleTree[T] {
	return &RuleTree[T]{
		root: node[T]{},
	}
}

func (rt *RuleTree[T]) Add(urlPattern string, data T) error {
	var tokens []string
	var modifiers string
	var rootKeyKind nodeKind
	if match := reDomainName.FindStringSubmatch(urlPattern); match != nil {
		rootKeyKind = nodeKindDomain
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reExactAddress.FindStringSubmatch(urlPattern); match != nil {
		rootKeyKind = nodeKindAddressRoot
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reAddressParts.FindStringSubmatch(urlPattern); match != nil {
		rootKeyKind = nodeKindExactMatch
		tokens = tokenize(match[1])
		modifiers = match[2]
	} else if match := reGeneric.FindStringSubmatch(urlPattern); match != nil {
		rootKeyKind = nodeKindGeneric
		tokens = []string{}
		modifiers = match[1]
	} else {
		return errors.New("unknown rule format")
	}

	if modifiers != "" {
		if err := data.ParseModifiers(modifiers); err != nil {
			// log.Printf("failed to parse modifiers for rule %q: %v", rule, err)
			return fmt.Errorf("parse modifiers: %w", err)
		}
	}

	var node *node[T]
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

	node.dataMu.Lock()
	node.data = append(node.data, data)
	node.dataMu.Unlock()

	return nil
}

// FindMatchingRulesReq finds all rules that match the given request.
func (rt *RuleTree[T]) FindMatchingRulesReq(req *http.Request) (data []T) {
	host := req.URL.Hostname()
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
		data = append(data, genericNode.FindMatchingRulesReq(req)...)
	}

	// address root
	data = append(data, rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseFindMatchingRulesReq(req, tokens, func(_ *node[T], t []string) bool {
		// address root rules have to match the entire URL
		// TODO: look into whether we can match the rule if the remaining tokens only contain the query
		return len(t) == 0
	})...)

	data = append(data, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
	tokens = tokens[1:]

	// protocol separator
	data = append(data, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
	tokens = tokens[1:]

	// domain segments
	for len(tokens) > 0 {
		if tokens[0] == "/" {
			break
		}
		if tokens[0] != "." {
			data = append(data, rt.root.FindChild(nodeKey{kind: nodeKindDomain}).TraverseFindMatchingRulesReq(req, tokens, nil)...)
		}
		data = append(data, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
		tokens = tokens[1:]
	}

	// rest of the URL
	// TODO: handle query parameters
	for len(tokens) > 0 {
		data = append(data, rt.root.TraverseFindMatchingRulesReq(req, tokens, nil)...)
		tokens = tokens[1:]
	}

	return data
}

// FindMatchingRulesRes finds all rules that match the given response.
// It assumes that the request that generated the response has already been matched by FindMatchingRulesReq.
func (rt *RuleTree[T]) FindMatchingRulesRes(req *http.Request, res *http.Response) (rules []T) {
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
	rules = append(rules, rt.root.FindChild(nodeKey{kind: nodeKindAddressRoot}).TraverseFindMatchingRulesRes(res, tokens, func(_ *node[T], t []string) bool {
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
