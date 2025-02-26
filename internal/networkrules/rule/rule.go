package rule

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/anfragment/zen/internal/networkrules/rulemodifiers"
)

// Rule represents modifiers of a rule.
type Rule struct {
	// string representation
	RawRule string
	// FilterName is the name of the filter that the rule belongs to.
	FilterName         *string
	matchingModifiers  []rulemodifiers.MatchingModifier
	modifyingModifiers []rulemodifiers.ModifyingModifier
}

type ExRule struct {
	// string representation
	RawRule string
	// FilterName is the name of the filter that the rule belongs to.
	FilterName *string

	Modifiers []exceptionModifier
}

func (ee *ExRule) Cancels(r Rule) bool {
	return true
}

func (ee *ExRule) ParseModifiers(modifiers string) error {
	return nil
}

func (ee *ExRule) ShouldMatchReq(req *http.Request) bool {
	return true
}

func (ee *ExRule) ShouldMatchRes(req *http.Response) bool {
	return true
}

type exceptionModifier interface {
	Cancels(rulemodifiers.Modifier) bool
}

func (rm *Rule) ParseModifiers(modifiers string) error {
	if len(modifiers) == 0 {
		return nil
	}

	for _, m := range strings.Split(modifiers, ",") {
		if len(m) == 0 {
			return fmt.Errorf("empty modifier")
		}

		isKind := func(kind string) bool {
			if len(m) > 0 && m[0] == '~' {
				return strings.HasPrefix(m[1:], kind)
			}
			return strings.HasPrefix(m, kind)
		}
		var modifier rulemodifiers.Modifier
		switch {
		case isKind("domain"):
			modifier = &rulemodifiers.DomainModifier{}
		case isKind("method"):
			modifier = &rulemodifiers.MethodModifier{}
		case isKind("document"),
			isKind("doc"),
			isKind("xmlhttprequest"),
			isKind("xhr"),
			isKind("font"),
			isKind("subdocument"),
			isKind("image"),
			isKind("object"),
			isKind("script"),
			isKind("stylesheet"),
			isKind("media"),
			isKind("other"):
			modifier = &rulemodifiers.ContentTypeModifier{}
		case isKind("third-party"):
			modifier = &rulemodifiers.ThirdPartyModifier{}
		case isKind("removeparam"):
			modifier = &rulemodifiers.RemoveParamModifier{}
		case isKind("header"):
			modifier = &rulemodifiers.HeaderModifier{}
		case isKind("removeheader"):
			modifier = &rulemodifiers.RemoveHeaderModifier{}
		case isKind("all"):
			// TODO: should act as "popup" modifier once it gets implemented
			continue
		default:
			return fmt.Errorf("unknown modifier %s", m)
		}

		if err := modifier.Parse(m); err != nil {
			return err
		}

		if matchingModifier, ok := modifier.(rulemodifiers.MatchingModifier); ok {
			rm.matchingModifiers = append(rm.matchingModifiers, matchingModifier)
		} else if modifyingModifier, ok := modifier.(rulemodifiers.ModifyingModifier); ok {
			rm.modifyingModifiers = append(rm.modifyingModifiers, modifyingModifier)
		} else {
			// QA: commment for now, cause not every modifier implements Cancels() func yet.
			// panic(fmt.Sprintf("got unknown modifier type %T for modifier %s", modifier, m))
		}
	}

	return nil
}

// ShouldMatchReq returns true if the rule should match the request.
func (rm *Rule) ShouldMatchReq(req *http.Request) bool {
	for _, modifier := range rm.matchingModifiers {
		if !modifier.ShouldMatchReq(req) {
			return false
		}
	}

	return true
}

// ShouldMatchRes returns true if the rule should match the response.
func (rm *Rule) ShouldMatchRes(res *http.Response) bool {
	for _, modifier := range rm.matchingModifiers {
		if !modifier.ShouldMatchRes(res) {
			return false
		}
	}

	return true
}

// ShouldBlockReq returns true if the request should be blocked.
func (rm *Rule) ShouldBlockReq(*http.Request) bool {
	return len(rm.modifyingModifiers) == 0
}

// ModifyReq modifies a request. Returns true if the request was modified.
func (rm *Rule) ModifyReq(req *http.Request) (modified bool) {
	for _, modifier := range rm.modifyingModifiers {
		if modifier.ModifyReq(req) {
			modified = true
		}
	}

	return modified
}

// ModifyRes modifies a response. Returns true if the response was modified.
func (rm *Rule) ModifyRes(res *http.Response) (modified bool) {
	for _, modifier := range rm.modifyingModifiers {
		if modifier.ModifyRes(res) {
			modified = true
		}
	}

	return modified
}
