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

// matchingModifier is a modifier that defines whether a rule matches a request.
type matchingModifier interface {
	Parse(modifier string) error
	ShouldMatch(req *http.Request) bool
}

// modifyingModifier is a modifier that defines how a rule modifies a request.
type modifyingModifier interface {
	Parse(modifier string) error
	Modify(req *http.Request)
}

func (rm *Rule) ParseModifiers(modifiers string) error {
	for _, modifier := range strings.Split(modifiers, ",") {
		if len(modifier) == 0 {
			return fmt.Errorf("empty modifier")
		}

		if eqIndex := strings.IndexByte(modifier, '='); eqIndex != -1 {
			rule, value := modifier[:eqIndex], modifier[eqIndex+1:]
			switch rule {
			case "domain":
				dm := &domainModifier{}
				if err := dm.Parse(value); err != nil {
					return err
				}
				rm.matchingModifiers = append(rm.matchingModifiers, dm)
			case "method":
				mm := &methodModifier{}
				if err := mm.Parse(value); err != nil {
					return err
				}
				rm.matchingModifiers = append(rm.matchingModifiers, mm)
			default:
				return fmt.Errorf("unknown modifier %s", rule)
			}
		} else {
			ruleType := modifier
			if ruleType[0] == '~' {
				ruleType = ruleType[1:]
			}
			switch ruleType {
			case "document", "xmlhttprequest", "font", "subdocument", "image", "object", "script", "stylesheet", "media", "websocket":
				ctm := &contentTypeModifier{}
				if err := ctm.Parse(modifier); err != nil {
					return err
				}
				rm.matchingModifiers = append(rm.matchingModifiers, ctm)
			case "third-party":
				tpm := &thirdPartyModifier{}
				if err := tpm.Parse(modifier); err != nil {
					return err
				}
				rm.matchingModifiers = append(rm.matchingModifiers, tpm)
			case "all":
				// TODO: this should act as a $popup modifier once it gets implemented
			default:
				return fmt.Errorf("unknown modifier %s", modifier)
			}
		}
	}

	return nil
}

func (rm *Rule) ShouldMatch(req *http.Request) bool {
	for _, modifier := range rm.matchingModifiers {
		if !modifier.ShouldMatch(req) {
			return false
		}
	}

	return true
}

func (rm *Rule) ShouldBlock(req *http.Request) bool {
	return len(rm.modifyingModifiers) == 0
}
