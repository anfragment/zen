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
	ShouldMatch(req *http.Request) bool
}

// modifyingModifier modifies a request.
type modifyingModifier interface {
	modifier
	Modify(req *http.Request) (modified bool)
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
			return strings.HasPrefix(m, kind)
		}
		var modifier modifier
		switch {
		case isKind("domain"):
			modifier = &domainModifier{}
		case isKind("method"):
			modifier = &methodModifier{}
		case isKind("document"), isKind("xmlhttprequest"), isKind("font"), isKind("subdocument"), isKind("image"), isKind("object"), isKind("script"), isKind("stylesheet"), isKind("media"), isKind("websocket"):
			modifier = &contentTypeModifier{}
		case isKind("third-party"):
			modifier = &thirdPartyModifier{}
		case isKind("removeparam"):
			modifier = &removeParamModifier{}
		case isKind("all"):
			// TODO: should act as "popup" modifier once it gets implemented
		default:
			return fmt.Errorf("unknown modifier %s", modifier)
		}

		if err := modifier.Parse(m); err != nil {
			return err
		}

		if matchingModifier, ok := modifier.(matchingModifier); ok {
			rm.matchingModifiers = append(rm.matchingModifiers, matchingModifier)
		}
		if modifyingModifier, ok := modifier.(modifyingModifier); ok {
			rm.modifyingModifiers = append(rm.modifyingModifiers, modifyingModifier)
		}
	}

	return nil
}

// ShouldMatch returns true if the rule should match the request.
func (rm *Rule) ShouldMatch(req *http.Request) bool {
	for _, modifier := range rm.matchingModifiers {
		if !modifier.ShouldMatch(req) {
			return false
		}
	}

	return true
}

// ShouldBlock returns true if the request should be blocked.
func (rm *Rule) ShouldBlock(req *http.Request) bool {
	return len(rm.modifyingModifiers) == 0
}

// Modify modifies a request. Returns true if the request was modified.
func (rm *Rule) Modify(req *http.Request) (modified bool) {
	for _, modifier := range rm.modifyingModifiers {
		if modifier.Modify(req) {
			modified = true
		}
	}

	return modified
}
