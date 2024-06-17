package rule

import (
	"fmt"
	"net/http"
	"strings"
)

// Rule represents modifiers of a rule.
type Rule struct {
	// string representation
	RawRule string
	// FilterName is the name of the filter that the rule belongs to.
	FilterName         *string
	matchingModifiers  []matchingModifier
	modifyingModifiers []modifyingModifier
}

// modifier is a modifier of a rule.
type modifier interface {
	Parse(modifier string) error
}

// matchingModifier defines whether a rule matches a request.
type matchingModifier interface {
	modifier
	ShouldMatchReq(req *http.Request) bool
	ShouldMatchRes(res *http.Response) bool
}

// modifyingModifier modifies a request.
type modifyingModifier interface {
	modifier
	ModifyReq(req *http.Request) (modified bool)
	ModifyRes(res *http.Response) (modified bool)
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
		var modifier modifier
		switch {
		case isKind("domain"):
			modifier = &domainModifier{}
		case isKind("method"):
			modifier = &methodModifier{}
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
			modifier = &contentTypeModifier{}
		case isKind("third-party"):
			modifier = &thirdPartyModifier{}
		case isKind("removeparam"):
			modifier = &removeParamModifier{}
		case isKind("header"):
			modifier = &headerModifier{}
		case isKind("removeheader"):
			modifier = &removeHeaderModifier{}
		case isKind("all"):
			// TODO: should act as "popup" modifier once it gets implemented
			continue
		default:
			return fmt.Errorf("unknown modifier %s", m)
		}

		if err := modifier.Parse(m); err != nil {
			return err
		}

		if matchingModifier, ok := modifier.(matchingModifier); ok {
			rm.matchingModifiers = append(rm.matchingModifiers, matchingModifier)
		} else if modifyingModifier, ok := modifier.(modifyingModifier); ok {
			rm.modifyingModifiers = append(rm.modifyingModifiers, modifyingModifier)
		} else {
			panic(fmt.Sprintf("got unknown modifier type %T for modifier %s", modifier, m))
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
