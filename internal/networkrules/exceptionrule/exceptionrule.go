package exceptionrule

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/anfragment/zen/internal/networkrules/rule"
	"github.com/anfragment/zen/internal/networkrules/rulemodifiers"
)

type ExceptionRule struct {
	RawRule    string
	FilterName *string

	Modifiers ExceptionModifiers
}

type ExceptionModifiers struct {
	AndModifiers []exceptionModifier
	OrModifiers  []exceptionModifier
}

type exceptionModifier interface {
	Cancels(rulemodifiers.Modifier) bool
	ShouldMatchReq(req *http.Request) bool
	ShouldMatchRes(res *http.Response) bool
}

func (er *ExceptionRule) Cancels(r *rule.Rule) bool {
	if len(er.Modifiers.AndModifiers) == 0 && len(er.Modifiers.OrModifiers) == 0 {
		return true
	}

	for _, exc := range er.Modifiers.AndModifiers {
		found := false
		for _, basic := range r.MatchingModifiers.AndModifiers {
			if exc.Cancels(basic) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(er.Modifiers.OrModifiers) > 0 {
		found := false
		for _, exc := range er.Modifiers.OrModifiers {
			for _, basic := range r.MatchingModifiers.OrModifiers {
				if exc.Cancels(basic) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (er *ExceptionRule) ParseModifiers(modifiers string) error {
	if len(modifiers) == 0 {
		return nil
	}

	for _, m := range strings.Split(modifiers, ",") {
		modifier, err := rulemodifiers.ParseModifier(m)
		if err != nil {
			return fmt.Errorf("parse modifier: %w", err)
		}

		if matchingModifier, ok := modifier.(exceptionModifier); ok {
			switch matchingModifier.(type) {
			case *rulemodifiers.ContentTypeModifier:
				er.Modifiers.OrModifiers = append(er.Modifiers.OrModifiers, matchingModifier)
			default:
				er.Modifiers.AndModifiers = append(er.Modifiers.AndModifiers, matchingModifier)
			}
		} else {
			panic(fmt.Sprintf("got unknown modifier type %T for modifier %s", modifier, m))
		}

	}

	return nil
}

// ShouldMatchReq returns true if the rule should match the request.
func (er *ExceptionRule) ShouldMatchReq(req *http.Request) bool {
	// AndModifiers: All must match.
	for _, m := range er.Modifiers.AndModifiers {
		if !m.ShouldMatchReq(req) {
			return false
		}
	}

	// OrModifiers: At least one must match.
	if len(er.Modifiers.OrModifiers) > 0 {
		for _, m := range er.Modifiers.OrModifiers {
			if m.ShouldMatchReq(req) {
				return true
			}
		}
		return false
	}

	return true
}

// ShouldMatchRes returns true if the rule should match the response.
func (er *ExceptionRule) ShouldMatchRes(res *http.Response) bool {
	for _, m := range er.Modifiers.AndModifiers {
		if !m.ShouldMatchRes(res) {
			return false
		}
	}

	if len(er.Modifiers.OrModifiers) > 0 {
		for _, m := range er.Modifiers.OrModifiers {
			if m.ShouldMatchRes(res) {
				return true
			}
		}
		return false
	}

	return true
}
