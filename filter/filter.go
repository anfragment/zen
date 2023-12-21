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
	ruleTree          ruletree.RuleTree
	exceptionRuleTree ruletree.RuleTree
}

func NewFilter() *Filter {
	filter := &Filter{
		ruleTree:          ruletree.NewRuleTree(),
		exceptionRuleTree: ruletree.NewRuleTree(),
	}

	return filter
}

// Init initializes the filter by downloading and parsing the filter lists.
func (f *Filter) Init() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, filterList := range config.Config.GetFilterLists() {
		if !filterList.Enabled {
			continue
		}
		wg.Add(1)
		go func(url string, name string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
			rules, exceptions := f.AddRules(resp.Body, &name)
			log.Printf("filter initialization: added %d rules and %d exceptions from %q", rules, exceptions, url)
		}(filterList.Url, filterList.Name)
	}
	wg.Wait()
}

var (
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreLine = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
	reException  = regexp.MustCompile(`^@@`)
)

// AddRules parses the rules from the given reader and adds them to the filter.
func (m *Filter) AddRules(reader io.Reader, filterName *string) (ruleCount, exceptionCount int) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || reIgnoreLine.MatchString(line) {
			continue
		}

		if reException.MatchString(line) {
			if err := m.exceptionRuleTree.AddRule(line[2:], nil); err != nil {
				// filterName is only needed for logging blocked requests
				// log.Printf("error adding exception rule %q: %v", rule, err)
				continue
			}
			exceptionCount++
		} else {
			if err := m.ruleTree.AddRule(line, filterName); err != nil {
				// log.Printf("error adding rule %q: %v", rule, err)
				continue
			}
			ruleCount++
		}
	}

	return ruleCount, exceptionCount
}

func (m *Filter) HandleRequest(req *http.Request) (block bool, rules []rule.Rule) {
	if len(m.exceptionRuleTree.FindMatchingRules(req)) > 0 {
		// TODO: implement precise exception handling
		// https://adguard.com/kb/general/ad-filtering/create-own-filters/#removeheader-modifier (see "Negating $removeheader")
		return false, nil
	}

	rules = m.ruleTree.FindMatchingRules(req)
	for _, r := range rules {
		if r.ShouldBlock(req) {
			return true, []rule.Rule{r}
		}
	}
	return false, rules
}
