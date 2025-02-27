package networkrules

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"slices"

	"github.com/anfragment/zen/internal/networkrules/exceptionrule"
	"github.com/anfragment/zen/internal/networkrules/rule"
	"github.com/anfragment/zen/internal/ruletree"
)

var (
	// exceptionRegex matches exception rules.
	exceptionRegex = regexp.MustCompile(`^@@`)

	reHosts       = regexp.MustCompile(`^(?:0\.0\.0\.0|127\.0\.0\.1) (.+)`)
	reHostsIgnore = regexp.MustCompile(`^(?:0\.0\.0\.0|broadcasthost|local|localhost(?:\.localdomain)?|ip6-\w+)$`)
)

type NetworkRules struct {
	regularRuleTree   *ruletree.RuleTree[*rule.Rule]
	exceptionRuleTree *ruletree.RuleTree[*exceptionrule.ExceptionRule]

	hosts   map[string]*string
	hostsMu sync.RWMutex
}

func NewNetworkRules() *NetworkRules {
	regularTree := ruletree.NewRuleTree[*rule.Rule]()
	exceptionTree := ruletree.NewRuleTree[*exceptionrule.ExceptionRule]()

	return &NetworkRules{
		regularRuleTree:   regularTree,
		exceptionRuleTree: exceptionTree,
		hosts:             make(map[string]*string),
	}
}

func (nr *NetworkRules) ParseRule(rawRule string, filterName *string) error {
	if reHosts.MatchString(rawRule) {
		if commentIndex := strings.IndexByte(rawRule, '#'); commentIndex != -1 {
			rawRule = rawRule[:commentIndex]
		}

		host := reHosts.FindStringSubmatch(rawRule)[1]
		if strings.ContainsRune(host, ' ') {
			for _, host := range strings.Split(host, " ") {
				nr.regularRuleTree.Add(fmt.Sprintf("127.0.0.1 %s", host), filterName, &rule.Rule{
					RawRule:    rawRule,
					FilterName: filterName,
				})
			}
			return nil
		}

		if reHostsIgnore.MatchString(host) {
			return nil
		}

		nr.hostsMu.Lock()
		nr.hosts[host] = filterName
		nr.hostsMu.Unlock()

		return nil
	}

	if exceptionRegex.MatchString(rawRule) {
		return nr.exceptionRuleTree.Add(rawRule[2:], filterName, &exceptionrule.ExceptionRule{
			RawRule:    rawRule,
			FilterName: filterName,
		})
	}

	return nr.regularRuleTree.Add(rawRule, filterName, &rule.Rule{
		RawRule:    rawRule,
		FilterName: filterName,
	})
}

func (nr *NetworkRules) ModifyRes(req *http.Request, res *http.Response) []rule.Rule {
	regularRules := nr.regularRuleTree.FindMatchingRulesRes(req, res)
	if len(regularRules) == 0 {
		return nil
	}

	exceptions := nr.exceptionRuleTree.FindMatchingRulesReq(req)
	for _, ex := range exceptions {
		if slices.ContainsFunc(regularRules, ex.Cancels) {
			return nil
		}
	}

	var appliedRules []rule.Rule
	for _, r := range regularRules {
		if r.ModifyRes(res) {
			appliedRules = append(appliedRules, *r)
		}
	}

	return appliedRules
}

func (nr *NetworkRules) ModifyReq(req *http.Request) (appliedRules []rule.Rule, shouldBlock bool, redirectURL string) {
	regularRules := nr.regularRuleTree.FindMatchingRulesReq(req)
	if len(regularRules) == 0 {
		return nil, false, ""
	}

	exceptions := nr.exceptionRuleTree.FindMatchingRulesReq(req)
	for _, ex := range exceptions {
		if slices.ContainsFunc(regularRules, ex.Cancels) {
			return nil, false, ""
		}
	}

	for _, r := range regularRules {
		if r.ShouldBlockReq(req) {
			return []rule.Rule{*r}, true, ""
		}
		if r.ModifyReq(req) {
			appliedRules = append(appliedRules, *r)
		}
	}

	initialURL := req.URL.String()
	finalURL := req.URL.String()

	if initialURL != finalURL {
		return appliedRules, false, finalURL
	}

	return appliedRules, false, ""
}
