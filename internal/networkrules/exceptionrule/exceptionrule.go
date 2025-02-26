package exceptionrule

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/anfragment/zen/internal/networkrules/rulemodifiers"
)

type ExceptionRule struct {
	RawRule    string
	FilterName *string

	modifiers []exceptionModifier
}

type modifier interface {
	Parse(modifier string) error
}

type exceptionModifier interface {
	Cancels(modifier)
	ShouldMatchReq(req *http.Request) bool
	ShouldMatchRes(res *http.Response) bool
}

func (er *ExceptionRule) ShouldMatchReq(req *http.Request) bool {
	return false
}

func (er *ExceptionRule) ShouldMatchRes(req *http.Response) bool {
	return false
}

func (er *ExceptionRule) ParseModifiers(modifiers string) error {
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
		var modifier modifier
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

		// if matchingModifier, ok := modifier.(matchingModifier); ok {
		// 	er.matchingModifiers = append(er.matchingModifiers, matchingModifier)
		// } else if modifyingModifier, ok := modifier.(modifyingModifier); ok {
		// 	er.modifyingModifiers = append(er.modifyingModifiers, modifyingModifier)
		// } else {
		// 	panic(fmt.Sprintf("got unknown modifier type %T for modifier %s", modifier, m))
		// }

		// QA: Is it enough to cast only "matchingModifiers" in exception rules cause we dont have "modifyingModifiers" here?
		if matchingModifier, ok := modifier.(exceptionModifier); ok {
			er.modifiers = append(er.modifiers, matchingModifier)
		} else {
			// QA: commment for now, cause not every modifier implements Cancels() func yet.
			// panic(fmt.Sprintf("got unknown modifier type %T for modifier %s", modifier, m))
		}

	}

	return nil
}

// ShouldMatchReq returns true if the rule should match the request.
// func (er *ExceptionRule) ShouldMatchReq(req *http.Request) bool {
// 	for _, modifier := range er.modifiers {
// 		if !modifier.ShouldMatchReq(req) {
// 			return false
// 		}
// 	}

// 	return true
// }

// // ShouldMatchRes returns true if the rule should match the response.
// func (er *ExceptionRule) ShouldMatchRes(res *http.Response) bool {
// 	for _, modifier := range er.modifiers {
// 		if !modifier.ShouldMatchRes(res) {
// 			return false
// 		}
// 	}

// 	return true
// }
