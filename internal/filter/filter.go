package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/ZenPrivacy/zen-desktop/internal/cfg"
	"github.com/ZenPrivacy/zen-desktop/internal/cosmetic"
	"github.com/ZenPrivacy/zen-desktop/internal/cssrule"
	"github.com/ZenPrivacy/zen-desktop/internal/jsrule"
	"github.com/ZenPrivacy/zen-desktop/internal/logger"
	"github.com/ZenPrivacy/zen-desktop/internal/networkrules/rule"
)

// filterEventsEmitter emits filter events.
type filterEventsEmitter interface {
	OnFilterBlock(method, url, referer string, rules []rule.Rule)
	OnFilterRedirect(method, url, to, referer string, rules []rule.Rule)
	OnFilterModify(method, url, referer string, rules []rule.Rule)
}

type networkRules interface {
	ParseRule(rule string, filterName *string) (isException bool, err error)
	ModifyReq(req *http.Request) (appliedRules []rule.Rule, shouldBlock bool, redirectURL string)
	ModifyRes(req *http.Request, res *http.Response) []rule.Rule
	CreateBlockResponse(req *http.Request) *http.Response
	CreateRedirectResponse(req *http.Request, to string) *http.Response
}

// config provides filter configuration.
type config interface {
	GetFilterLists() []cfg.FilterList
	GetMyRules() []string
}

// scriptletsInjector injects scriptlets into HTML responses.
type scriptletsInjector interface {
	Inject(*http.Request, *http.Response) error
	AddRule(string, bool) error
}

type cosmeticRulesInjector interface {
	Inject(*http.Request, *http.Response) error
	AddRule(string) error
}

type cssRulesInjector interface {
	Inject(*http.Request, *http.Response) error
	AddRule(string) error
}

type jsRuleInjector interface {
	AddRule(rule string) error
	Inject(*http.Request, *http.Response) error
}

type filterListStore interface {
	Get(url string) (io.ReadCloser, error)
}

// Filter is capable of parsing Adblock-style filter lists and hosts rules and matching URLs against them.
//
// Safe for concurrent use.
type Filter struct {
	config                config
	networkRules          networkRules
	scriptletsInjector    scriptletsInjector
	cosmeticRulesInjector cosmeticRulesInjector
	cssRulesInjector      cssRulesInjector
	jsRuleInjector        jsRuleInjector
	eventsEmitter         filterEventsEmitter
	filterListStore       filterListStore
}

var (
	// ignoreLineRegex matches comments and [Adblock Plus 2.0]-style headers.
	ignoreLineRegex = regexp.MustCompile(`^(?:!|\[|#[^#%@$])`)
	// scriptletRegex matches scriptlet rules.
	scriptletRegex = regexp.MustCompile(`(?:#%#\/\/scriptlet)|(?:##\+js)`)
)

// NewFilter creates and initializes a new filter.
func NewFilter(config config, networkRules networkRules, scriptletsInjector scriptletsInjector, cosmeticRulesInjector cosmeticRulesInjector, cssRulesInjector cssRulesInjector, jsRuleInjector jsRuleInjector, eventsEmitter filterEventsEmitter, filterListStore filterListStore) (*Filter, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	if eventsEmitter == nil {
		return nil, errors.New("eventsEmitter is nil")
	}
	if networkRules == nil {
		return nil, errors.New("networkRules is nil")
	}
	if scriptletsInjector == nil {
		return nil, errors.New("scriptletsInjector is nil")
	}
	if cosmeticRulesInjector == nil {
		return nil, errors.New("cosmeticRulesInjector is nil")
	}
	if cssRulesInjector == nil {
		return nil, errors.New("cssRulesInjector is nil")
	}
	if jsRuleInjector == nil {
		return nil, errors.New("jsRuleInjector is nil")
	}
	if filterListStore == nil {
		return nil, errors.New("filterListStore is nil")
	}

	f := &Filter{
		config:                config,
		networkRules:          networkRules,
		scriptletsInjector:    scriptletsInjector,
		cosmeticRulesInjector: cosmeticRulesInjector,
		cssRulesInjector:      cssRulesInjector,
		jsRuleInjector:        jsRuleInjector,
		eventsEmitter:         eventsEmitter,
		filterListStore:       filterListStore,
	}
	f.init()

	return f, nil
}

// init initializes the filter by downloading and parsing the filter lists.
func (f *Filter) init() {
	var wg sync.WaitGroup
	for _, filterList := range f.config.GetFilterLists() {
		if !filterList.Enabled {
			continue
		}
		wg.Add(1)
		go func(filterList cfg.FilterList) {
			defer wg.Done()

			contents, err := f.filterListStore.Get(filterList.URL)
			if err != nil {
				log.Printf("failed to get filter list %q from store: %v", filterList.URL, err)
				return
			}
			rules, exceptions := f.ParseAndAddRules(contents, &filterList.Name, filterList.Trusted)
			if err := contents.Close(); err != nil {
				log.Printf("failed to close filter list: %v", err)
			}

			log.Printf("filter initialization: added %d rules and %d exceptions from %q", rules, exceptions, filterList.URL)
		}(filterList)
	}
	wg.Wait()

	myRules := f.config.GetMyRules()
	filterName := "My rules"

	var ruleCount, exceptionCount int
	for _, rule := range myRules {
		isException, err := f.AddRule(rule, &filterName, true)
		if err != nil {
			log.Printf("failed to add rule from %q: %v", filterName, err)
			continue
		}
		if isException {
			exceptionCount++
		} else {
			ruleCount++
		}
	}

	if len(myRules) > 0 {
		log.Printf("filter initialization: added %d rules and %d exceptions from %q", ruleCount, exceptionCount, filterName)
	}
}

// ParseAndAddRules parses the rules from the given reader and adds them to the filter.
func (f *Filter) ParseAndAddRules(reader io.Reader, filterListName *string, filterListTrusted bool) (ruleCount, exceptionCount int) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || ignoreLineRegex.MatchString(line) {
			continue
		}

		if isException, err := f.AddRule(line, filterListName, filterListTrusted); err != nil { // nolint:revive
			// log.Printf("error adding rule: %v", err)
		} else if isException {
			exceptionCount++
		} else {
			ruleCount++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading rules: %v", err)
	}

	return ruleCount, exceptionCount
}

// AddRule adds a new rule to the filter. It returns true if the rule is an exception, false otherwise.
func (f *Filter) AddRule(rule string, filterListName *string, filterListTrusted bool) (isException bool, err error) {
	/*
		The order of operations is crucial here.
		jsRule.RuleRegex also matches scriptlet rules.
		Therefore, we must first check for a scriptlet rule match before checking for a JS rule match.
	*/
	switch {
	case scriptletRegex.MatchString(rule):
		if err := f.scriptletsInjector.AddRule(rule, filterListTrusted); err != nil {
			return false, fmt.Errorf("add scriptlet: %w", err)
		}
	case cosmetic.RuleRegex.MatchString(rule):
		if err := f.cosmeticRulesInjector.AddRule(rule); err != nil {
			return false, fmt.Errorf("add cosmetic rule: %w", err)
		}
	case filterListTrusted && cssrule.RuleRegex.MatchString(rule):
		if err := f.cssRulesInjector.AddRule(rule); err != nil {
			return false, fmt.Errorf("add css rule: %w", err)
		}
	case filterListTrusted && jsrule.RuleRegex.MatchString(rule):
		if err := f.jsRuleInjector.AddRule(rule); err != nil {
			return false, fmt.Errorf("add js rule: %w", err)
		}
	default:
		isExceptionRule, err := f.networkRules.ParseRule(rule, filterListName)
		if err != nil {
			return false, fmt.Errorf("parse network rule: %w", err)
		}
		return isExceptionRule, nil
	}

	return false, nil
}

// HandleRequest handles the given request by matching it against the filter rules.
// If the request should be blocked, it returns a response that blocks the request. If the request should be modified, it modifies it in-place.
func (f *Filter) HandleRequest(req *http.Request) *http.Response {
	initialURL := req.URL.String()

	appliedRules, shouldBlock, redirectURL := f.networkRules.ModifyReq(req)
	if shouldBlock {
		f.eventsEmitter.OnFilterBlock(req.Method, initialURL, req.Header.Get("Referer"), appliedRules)
		return f.networkRules.CreateBlockResponse(req)
	}

	if redirectURL != "" {
		f.eventsEmitter.OnFilterRedirect(req.Method, initialURL, redirectURL, req.Header.Get("Referer"), appliedRules)
		return f.networkRules.CreateRedirectResponse(req, redirectURL)
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
			// This and the following injection errors are recoverable, so we log them and continue processing the response.
			log.Printf("error injecting scriptlets for %q: %v", logger.Redacted(req.URL), err)
		}

		if err := f.cosmeticRulesInjector.Inject(req, res); err != nil {
			log.Printf("error injecting cosmetic rules for %q: %v", logger.Redacted(req.URL), err)
		}
		if err := f.cssRulesInjector.Inject(req, res); err != nil {
			log.Printf("error injecting css rules for %q: %v", logger.Redacted(req.URL), err)
		}
		if err := f.jsRuleInjector.Inject(req, res); err != nil {
			log.Printf("error injecting js rules for %q: %v", logger.Redacted(req.URL), err)
		}
	}

	appliedRules := f.networkRules.ModifyRes(req, res)
	if len(appliedRules) > 0 {
		f.eventsEmitter.OnFilterModify(req.Method, req.URL.String(), req.Header.Get("Referer"), appliedRules)
	}

	return nil
}

func isDocumentNavigation(req *http.Request, res *http.Response) bool {
	// Sec-Fetch-Dest: document indicates that the destination is a document (HTML or XML),
	// and the request is the result of a user-initiated top-level navigation (e.g. resulting from a user clicking a link).
	// Reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-Dest#document
	// Note: Although not explicitly stated in the spec, Fetch Metadata Request Headers are only included in requests sent to HTTPS endpoints.
	if req.URL.Scheme == "https" && req.Header.Get("Sec-Fetch-Dest") != "document" {
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
