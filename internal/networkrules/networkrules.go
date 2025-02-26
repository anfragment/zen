package networkrules

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"github.com/anfragment/zen/internal/networkrules/exceptionrule"
	"github.com/anfragment/zen/internal/networkrules/rule"
	"github.com/anfragment/zen/internal/ruletree"
)

var (
	// exceptionRegex matches exception rules.
	exceptionRegex = regexp.MustCompile(`^@@`)

	reHosts = regexp.MustCompile(`^(?:0\.0\.0\.0|127\.0\.0\.1) (.+)`)
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
	isHost := reHosts.MatchString(rawRule)
	isException := exceptionRegex.MatchString(rawRule)

	if isException {
		rule := &exceptionrule.ExceptionRule{
			RawRule:    rawRule,
			FilterName: filterName,
		}

		if isHost {
			return nr.exceptionRuleTree.AddHost(rawRule, filterName, rule)
		}
		return nr.exceptionRuleTree.Add(rawRule[2:], filterName, rule)
	}

	rule := &rule.Rule{
		RawRule:    rawRule,
		FilterName: filterName,
	}
	if isHost {
		return nr.regularRuleTree.AddHost(rawRule, filterName, rule)
	}

	return nr.regularRuleTree.Add(rawRule, filterName, rule)
}

func (nr *NetworkRules) ModifyRes(req *http.Request, res *http.Response) []rule.Rule {
	fmt.Println("modify res")

	exceptions := nr.exceptionRuleTree.FindMatchingRulesRes(req, res)
	regularRules := nr.regularRuleTree.FindMatchingRulesRes(req, res)

	if len(exceptions) > 0 {
		fmt.Printf("found %d exceptions", len(exceptions))
	}

	var appliedRules []rule.Rule

	for _, r := range regularRules {
		if r.ModifyRes(res) {
			appliedRules = append(appliedRules, *r)
		}
	}

	fmt.Println(len(appliedRules))

	return appliedRules
}

func (nr *NetworkRules) ModifyReq(res *http.Request) {
	fmt.Println("modify req")
}
