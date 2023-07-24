package matcher

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/anfragment/zen/matcher/ruletree"
	"github.com/elazarl/goproxy"
)

// Matcher is trie-based matcher for URLs that is capable of parsing
// Adblock filters and hosts rules and matching URLs against them.
//
// The matcher is safe for concurrent use.
type Matcher struct {
	ruleTree *ruletree.RuleTree
}

func NewMatcher() *Matcher {
	return &Matcher{
		ruleTree: ruletree.NewRuleTree(),
	}
}

var (
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreRule = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
)

func (m *Matcher) AddRules(reader io.Reader) int {
	var count int
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		rule := strings.TrimSpace(scanner.Text())
		if rule == "" || reIgnoreRule.MatchString(rule) {
			continue
		}
		if err := m.ruleTree.AddRule(rule); err != nil {
			// log.Printf("error adding rule %q: %v", rule, err)
			continue
		}
		if strings.Contains(rule, "jscrambler") {
			log.Printf("adding rule %q", rule)
		}
		count++
	}
	return count
}

func (m *Matcher) Middleware(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return m.ruleTree.Middleware(req, ctx)
}
