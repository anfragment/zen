package filter

import (
	"bufio"
	"context"
	"errors"
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

// filterEventsEmitter represents an object that can emit filter events.
type filterEventsEmitter interface {
	OnFilterBlock(method, url, referer string, rules []rule.Rule)
	OnFilterRedirect(method, url, to, referer string, rules []rule.Rule)
	OnFilterModify(method, url, referer string, rules []rule.Rule)
}

// Filter is trie-based filter for URLs that is capable of parsing
// Adblock filters and hosts rules and matching URLs against them.
//
// The filter is safe for concurrent use.
type Filter struct {
	ruleTree          ruletree.RuleTree
	exceptionRuleTree ruletree.RuleTree
	eventsEmitter     filterEventsEmitter
}

func NewFilter(eventsEmitter filterEventsEmitter) (*Filter, error) {
	if eventsEmitter == nil {
		return nil, errors.New("eventsEmitter is nil")
	}

	return &Filter{
		ruleTree:          ruletree.NewRuleTree(),
		exceptionRuleTree: ruletree.NewRuleTree(),
		eventsEmitter:     eventsEmitter,
	}, nil
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
		}(filterList.URL, filterList.Name)
	}
	wg.Wait()
}

var (
	// Ignore comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	reIgnoreLine = regexp.MustCompile(`^(?:!|#|\[)|(##|#\?#|#\$#|#@#)`)
	reException  = regexp.MustCompile(`^@@`)
)

// AddRules parses the rules from the given reader and adds them to the filter.
func (f *Filter) AddRules(reader io.Reader, filterName *string) (ruleCount, exceptionCount int) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || reIgnoreLine.MatchString(line) {
			continue
		}

		if reException.MatchString(line) {
			if err := f.exceptionRuleTree.AddRule(line[2:], nil); err != nil {
				// filterName is only needed for logging blocked requests
				// log.Printf("error adding exception rule %q: %v", rule, err)
				continue
			}
			exceptionCount++
		} else {
			if err := f.ruleTree.AddRule(line, filterName); err != nil {
				// log.Printf("error adding rule %q: %v", rule, err)
				continue
			}
			ruleCount++
		}
	}

	return ruleCount, exceptionCount
}

// HandleRequest handles the given request by matching it against the filter rules.
// If the request should be blocked, it returns a response that blocks the request. If the request should be modified, it modifies it in-place.
func (f *Filter) HandleRequest(req *http.Request) *http.Response {
	if len(f.exceptionRuleTree.FindMatchingRules(req)) > 0 {
		// TODO: implement precise exception handling
		// https://adguard.com/kb/general/ad-filtering/create-own-filters/#removeheader-modifier (see "Negating $removeheader")
		return nil
	}

	matchingRules := f.ruleTree.FindMatchingRules(req)
	if len(matchingRules) == 0 {
		return nil
	}

	var appliedRules []rule.Rule
	initialURL := req.URL.String()

	for _, r := range matchingRules {
		if r.ShouldBlock(req) {
			f.eventsEmitter.OnFilterBlock(req.Method, initialURL, req.Header.Get("Referer"), []rule.Rule{r})
			return f.createBlockResponse(req, r)
		}
		if r.Modify(req) {
			appliedRules = append(appliedRules, r)
		}
	}

	finalURL := req.URL.String()
	if initialURL != finalURL {
		f.eventsEmitter.OnFilterRedirect(req.Method, initialURL, finalURL, req.Header.Get("Referer"), appliedRules)
		return f.createRedirectResponse(req, finalURL)
	}

	if len(appliedRules) > 0 {
		f.eventsEmitter.OnFilterModify(req.Method, initialURL, req.Header.Get("Referer"), appliedRules)
	}

	return nil
}
