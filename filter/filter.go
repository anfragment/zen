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

	"github.com/anfragment/zen/cfg"
	"github.com/anfragment/zen/rule"
)

// filterEventsEmitter is an interface that can emit filter events.
type filterEventsEmitter interface {
	OnFilterBlock(method, url, referer string, rules []rule.Rule)
	OnFilterRedirect(method, url, to, referer string, rules []rule.Rule)
	OnFilterModify(method, url, referer string, rules []rule.Rule)
}

// ruleMatcher is an interface that can match requests against rules.
type ruleMatcher interface {
	AddRule(rule string, filterName *string) error
	FindMatchingRulesReq(req *http.Request) []rule.Rule
}

// config is an interface that provides filter configuration.
type config interface {
	GetFilterLists() []cfg.FilterList
}

// Filter is a filter for URLs that is capable of Adblock-style filter lists and hosts rules and matching URLs against them.
//
// The filter is safe for concurrent use.
type Filter struct {
	config               config
	ruleMatcher          ruleMatcher
	exceptionRuleMatcher ruleMatcher
	eventsEmitter        filterEventsEmitter
}

// NewFilter creates and initializes a new filter.
func NewFilter(config config, ruleMatcher ruleMatcher, exceptionRuleMatcher ruleMatcher, eventsEmitter filterEventsEmitter) (*Filter, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	if eventsEmitter == nil {
		return nil, errors.New("eventsEmitter is nil")
	}
	if ruleMatcher == nil {
		return nil, errors.New("ruleMatcher is nil")
	}
	if exceptionRuleMatcher == nil {
		return nil, errors.New("exceptionRuleMatcher is nil")
	}

	f := &Filter{
		config:               config,
		ruleMatcher:          ruleMatcher,
		exceptionRuleMatcher: exceptionRuleMatcher,
		eventsEmitter:        eventsEmitter,
	}
	f.init()

	return f, nil
}

// init initializes the filter by downloading and parsing the filter lists.
func (f *Filter) init() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, filterList := range f.config.GetFilterLists() {
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
			resp, err := http.DefaultClient.Do(req) // FIXME: use a custom client with a timeout
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
			if err := f.exceptionRuleMatcher.AddRule(line[2:], filterName); err != nil {
				// log.Printf("error adding exception rule %q: %v", rule, err)
				continue
			}
			exceptionCount++
		} else {
			if err := f.ruleMatcher.AddRule(line, filterName); err != nil {
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
	if len(f.exceptionRuleMatcher.FindMatchingRulesReq(req)) > 0 {
		// TODO: implement precise exception handling
		// https://adguard.com/kb/general/ad-filtering/create-own-filters/#removeheader-modifier (see "Negating $removeheader")
		return nil
	}

	matchingRules := f.ruleMatcher.FindMatchingRulesReq(req)
	if len(matchingRules) == 0 {
		return nil
	}

	var appliedRules []rule.Rule
	initialURL := req.URL.String()

	for _, r := range matchingRules {
		if r.ShouldBlockReq(req) {
			f.eventsEmitter.OnFilterBlock(req.Method, initialURL, req.Header.Get("Referer"), []rule.Rule{r})
			return f.createBlockResponse(req)
		}
		if r.ModifyReq(req) {
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

// HandleResponse handles the given response by matching it against the filter rules.
// If the response should be modified, it modifies it in-place.
//
// As of April 2024, there are no response-only rules that can block or redirect responses.
// For that reason, this method does not return a blocking or redirecting response itself.
func (f *Filter) HandleResponse(req *http.Request, res *http.Response) {
	if len(f.exceptionRuleMatcher.FindMatchingRulesReq(req)) > 0 {
		return
	}

	matchingRules := f.ruleMatcher.FindMatchingRulesReq(req)
	if len(matchingRules) == 0 {
		return
	}

	var appliedRules []rule.Rule

	for _, r := range matchingRules {
		if r.ModifyRes(res) {
			appliedRules = append(appliedRules, r)
		}
	}

	if len(appliedRules) > 0 {
		f.eventsEmitter.OnFilterModify(req.Method, req.URL.String(), req.Header.Get("Referer"), appliedRules)
	}

	return
}
