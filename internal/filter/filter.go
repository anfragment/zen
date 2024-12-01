package filter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anfragment/zen/internal/cfg"
	"github.com/anfragment/zen/internal/logger"
	"github.com/anfragment/zen/internal/rule"
)

// filterEventsEmitter emits filter events.
type filterEventsEmitter interface {
	OnFilterBlock(method, url, referer string, rules []rule.Rule)
	OnFilterRedirect(method, url, to, referer string, rules []rule.Rule)
	OnFilterModify(method, url, referer string, rules []rule.Rule)
}

// ruleMatcher matches requests against rules.
type ruleMatcher interface {
	AddRule(rule string, filterName *string) error
	FindMatchingRulesReq(*http.Request) []rule.Rule
	FindMatchingRulesRes(*http.Request, *http.Response) []rule.Rule
}

// config provides filter configuration.
type config interface {
	GetFilterLists() []cfg.FilterList
	GetMyRules() []string
}

// scriptletsInjector injects scriptlets into HTML responses.
type scriptletsInjector interface {
	Inject(*http.Request, *http.Response) error
	AddRule(string) error
}

// Filter is capable of parsing Adblock-style filter lists and hosts rules and matching URLs against them.
//
// Safe for concurrent use.
type Filter struct {
	config               config
	ruleMatcher          ruleMatcher
	exceptionRuleMatcher ruleMatcher
	scriptletsInjector   scriptletsInjector
	eventsEmitter        filterEventsEmitter
}

var (
	// ignoreLineRegex matches comments, cosmetic rules and [Adblock Plus 2.0]-style headers.
	ignoreLineRegex = regexp.MustCompile(`^(?:!|#|\[)`)
	// exceptionRegex matches exception rules.
	exceptionRegex = regexp.MustCompile(`^@@`)
	// scriptletRegex matches scriptlet rules.
	scriptletRegex = regexp.MustCompile(`(?:#%#\/\/scriptlet)|(?:##\+js)`)
	// cosmeticRuleRegex matches cosmetic rules.
	cosmeticRuleRegex = regexp.MustCompile(`^(.*##.+)$`)
)

// NewFilter creates and initializes a new filter.
func NewFilter(config config, ruleMatcher ruleMatcher, exceptionRuleMatcher ruleMatcher, scriptletsInjector scriptletsInjector, eventsEmitter filterEventsEmitter) (*Filter, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	if eventsEmitter == nil {
		return nil, errors.New("eventsEmitter is nil")
	}
	if ruleMatcher == nil {
		return nil, errors.New("ruleMatcher is nil")
	}
	if scriptletsInjector == nil {
		return nil, errors.New("scriptletsInjector is nil")
	}
	if exceptionRuleMatcher == nil {
		return nil, errors.New("exceptionRuleMatcher is nil")
	}

	f := &Filter{
		config:               config,
		ruleMatcher:          ruleMatcher,
		exceptionRuleMatcher: exceptionRuleMatcher,
		scriptletsInjector:   scriptletsInjector,
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
			rules, exceptions := f.ParseAndAddRules(resp.Body, &name)
			log.Printf("filter initialization: added %d rules and %d exceptions from %q", rules, exceptions, url)
		}(filterList.URL, filterList.Name)
	}
	wg.Wait()

	myRules := f.config.GetMyRules()
	filterName := "My rules"
	for _, rule := range myRules {
		f.AddRule(rule, &filterName)
	}
}

// ParseAndAddRules parses the rules from the given reader and adds them to the filter.
func (f *Filter) ParseAndAddRules(reader io.Reader, filterName *string) (ruleCount, exceptionCount int) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || ignoreLineRegex.MatchString(line) {
			continue
		}

		if isException, err := f.AddRule(line, filterName); err != nil { // nolint:revive
			// log.Printf("error adding rule: %v", err)
		} else if isException {
			exceptionCount++
		} else {
			ruleCount++
		}
	}

	return ruleCount, exceptionCount
}

// AddRule adds a new rule to the filter. It returns true if the rule is an exception, false otherwise.
func (f *Filter) AddRule(rule string, filterName *string) (isException bool, err error) {
	if scriptletRegex.MatchString(rule) {
		if err := f.scriptletsInjector.AddRule(rule); err != nil {
			return false, fmt.Errorf("add scriptlet: %w", err)
		}
		return false, nil
	}
	// add cosmetic rules here.
	if cosmeticRuleRegex.MatchString(rule) {
		fmt.Println("cosmetic rule: ", rule)
	}

	if exceptionRegex.MatchString(rule) {
		if err := f.exceptionRuleMatcher.AddRule(rule[2:], filterName); err != nil {
			return true, fmt.Errorf("add exception: %w", err)
		}
		return true, nil
	}
	if err := f.ruleMatcher.AddRule(rule, filterName); err != nil {
		return false, fmt.Errorf("add rule: %w", err)
	}
	return false, nil
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
func (f *Filter) HandleResponse(req *http.Request, res *http.Response) error {
	if isDocumentNavigation(req, res) {
		if err := f.scriptletsInjector.Inject(req, res); err != nil {
			// The error is recoverable, so we log it and continue processing the response.
			log.Printf("error injecting scriptlets for %q: %v", logger.Redacted(req.URL), err)
		}
	}

	if len(f.exceptionRuleMatcher.FindMatchingRulesRes(req, res)) > 0 {
		return nil
	}

	matchingRules := f.ruleMatcher.FindMatchingRulesRes(req, res)
	if len(matchingRules) == 0 {
		return nil
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

	return nil
}

func isDocumentNavigation(req *http.Request, res *http.Response) bool {
	// Sec-Fetch-Dest: document indicates that the destination is a document (HTML or XML),
	// and the request is the result of a user-initiated top-level navigation (e.g. resulting from a user clicking a link).
	// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-Dest#document
	if req.Header.Get("Sec-Fetch-Dest") != "document" {
		return false
	}

	contentType := res.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	if mediaType != "text/html" {
		return false
	}

	return true
}
