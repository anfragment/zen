package matcher

import (
	"bufio"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/anfragment/zen/matcher/ruletree"
	"github.com/anfragment/zen/matcher/ruletree/rule"
	"github.com/elazarl/goproxy"
)

// Matcher is trie-based matcher for URLs that is capable of parsing
// Adblock filters and hosts rules and matching URLs against them.
//
// The matcher is safe for concurrent use.
type Matcher struct {
	ruleTree          *ruletree.RuleTree
	exceptionRuleTree *ruletree.RuleTree
}

func NewMatcher() *Matcher {
	return &Matcher{
		ruleTree:          ruletree.NewRuleTree(),
		exceptionRuleTree: ruletree.NewRuleTree(),
	}
}

var (
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreRule = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
	reException  = regexp.MustCompile(`^@@`)
)

func (m *Matcher) AddRules(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		rule := strings.TrimSpace(scanner.Text())
		if rule == "" || reIgnoreRule.MatchString(rule) {
			continue
		}
		if reException.MatchString(rule) {
			if err := m.exceptionRuleTree.AddRule(rule[2:]); err != nil {
				// log.Printf("error adding exception rule %q: %v", rule, err)
				continue
			}
		} else {
			if err := m.ruleTree.AddRule(rule); err != nil {
				// log.Printf("error adding rule %q: %v", rule, err)
				continue
			}
		}
	}
}

func (m *Matcher) Middleware(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	exceptionAction := m.exceptionRuleTree.HandleRequest(req, ctx)
	if exceptionAction == rule.ActionBlock {
		return req, nil
	}

	action := m.ruleTree.HandleRequest(req, ctx)
	switch action {
	case rule.ActionBlock:
		return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "")
	case rule.ActionAllow:
		return req, nil
	default:
		return req, nil
	}
}
