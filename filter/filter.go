package filter

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anfragment/zen/config"
	"github.com/anfragment/zen/filter/ruletree"
	"github.com/anfragment/zen/filter/ruletree/rule"
)

// Filter is trie-based filter for URLs that is capable of parsing
// Adblock filters and hosts rules and matching URLs against them.
//
// The filter is safe for concurrent use.
type Filter struct {
	ruleTree          *ruletree.RuleTree
	exceptionRuleTree *ruletree.RuleTree
}

func NewFilter() *Filter {
	filter := &Filter{
		ruleTree:          ruletree.NewRuleTree(),
		exceptionRuleTree: ruletree.NewRuleTree(),
	}
	var wg sync.WaitGroup
	wg.Add(len(config.Config.Filter.FilterLists))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, filterList := range config.Config.Filter.FilterLists {
		go func(filterList string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, filterList, nil)
			if err != nil {
				log.Printf("filter initialization: %v", err)
				return
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("filter initialization: %v", err)
				return
			}
			defer resp.Body.Close()
			rules, exceptions := filter.AddRules(resp.Body)
			log.Printf("filter initialization: added %d rules and %d exceptions from %q", rules, exceptions, filterList)
		}(filterList)
	}
	wg.Wait()

	return filter
}

var (
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreRule = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
	reException  = regexp.MustCompile(`^@@`)
)

func (m *Filter) AddRules(reader io.Reader) (ruleCount, exceptionCount int) {
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
			exceptionCount++
		} else {
			if err := m.ruleTree.AddRule(rule); err != nil {
				// log.Printf("error adding rule %q: %v", rule, err)
				continue
			}
			ruleCount++
		}
	}

	return
}

func (m *Filter) HandleRequest(req *http.Request) rule.RequestAction {
	exceptionAction := m.exceptionRuleTree.HandleRequest(req)
	if exceptionAction == rule.ActionBlock {
		return rule.ActionAllow
	}

	return m.ruleTree.HandleRequest(req)
}
