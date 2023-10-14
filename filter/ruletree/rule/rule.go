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
	FilterName *string
	// generic is true if the rule is a generic rule.
	generic   bool
	modifiers []modifier
}

type modifier interface {
	Parse(modifier string) error
	ShouldMatch(req *http.Request) bool
}

func (rm *Rule) ParseModifiers(modifiers string) error {
	if len(modifiers) == 0 {
		rm.generic = true
		return nil
	}

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
				rm.modifiers = append(rm.modifiers, dm)
			case "method":
				mm := &methodModifier{}
				if err := mm.Parse(value); err != nil {
					return err
				}
				rm.modifiers = append(rm.modifiers, mm)
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
				rm.modifiers = append(rm.modifiers, ctm)
			case "third-party":
				tpm := &thirdPartyModifier{}
				if err := tpm.Parse(modifier); err != nil {
					return err
				}
				rm.modifiers = append(rm.modifiers, tpm)
			default:
				return fmt.Errorf("unknown modifier %s", modifier)
			}
		}
	}

	return nil
}

type RequestAction struct {
	Type       RequestActionType
	RawRule    string
	FilterName string
}

type RequestActionType int8

const (
	ActionAllow RequestActionType = iota
	ActionBlock
)

func (rm *Rule) HandleRequest(req *http.Request) RequestAction {
	for _, modifier := range rm.modifiers {
		if !modifier.ShouldMatch(req) {
			return RequestAction{Type: ActionAllow}
		}
	}

	action := RequestAction{Type: ActionBlock, RawRule: rm.RawRule}
	if rm.FilterName != nil {
		action.FilterName = *rm.FilterName
	}
	return action
}
