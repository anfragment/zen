package networkrules

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"

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

func (nr *NetworkRules) ParseRule(rawRule string, filterName *string) (isException bool, err error) {
	if matches := reHosts.FindStringSubmatch(rawRule); matches != nil {
		hostsField := matches[1]
		if commentIndex := strings.IndexByte(hostsField, '#'); commentIndex != -1 {
			hostsField = hostsField[:commentIndex]
		}

		// An IP address may be followed by multiple hostnames.
		//
		// As stated in https://man.freebsd.org/cgi/man.cgi?hosts(5):
		// "Items are separated by any number of blanks and/or tab characters."
		hosts := strings.Fields(hostsField)

		nr.hostsMu.Lock()
		for _, host := range hosts {
			if reHostsIgnore.MatchString(host) {
				continue
			}

			nr.hosts[host] = filterName
		}
		nr.hostsMu.Unlock()

		return false, nil
	}

	if exceptionRegex.MatchString(rawRule) {
		return true, nr.exceptionRuleTree.Add(rawRule[2:], &exceptionrule.ExceptionRule{
			RawRule:    rawRule,
			FilterName: filterName,
		})
	}

	return false, nr.regularRuleTree.Add(rawRule, &rule.Rule{
		RawRule:    rawRule,
		FilterName: filterName,
	})
}

func (nr *NetworkRules) ModifyRes(req *http.Request, res *http.Response) []rule.Rule {
	regularRules := nr.regularRuleTree.FindMatchingRulesRes(req, res)
	if len(regularRules) == 0 {
		return nil
	}

	exceptions := nr.exceptionRuleTree.FindMatchingRulesRes(req, res)
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
	host := req.URL.Hostname()
	nr.hostsMu.RLock()

	if filterName, ok := nr.hosts[host]; ok {
		nr.hostsMu.RUnlock()
		// 0.0.0.0 may not be the actual IP defined in the hosts file,
		// but storing the actual one feels wasteful.
		return []rule.Rule{
			{
				RawRule:    fmt.Sprintf("0.0.0.0 %s", host),
				FilterName: filterName,
			},
		}, false, ""
	}
	nr.hostsMu.RUnlock()

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
