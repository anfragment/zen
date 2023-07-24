package modifiers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
)

// RuleModifiers represents modifiers of a rule.
type RuleModifiers struct {
	rule      string
	generic   bool
	modifiers []modifier
}

type modifier interface {
	Parse(modifier string) error
	ShouldBlock(req *http.Request) bool
}

func (rm *RuleModifiers) Parse(rule string, modifiers string) error {
	rm.rule = rule
	if modifiers == "" {
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
				if err := ctm.Parse(ruleType); err != nil {
					return err
				}
				rm.modifiers = append(rm.modifiers, ctm)
			case "third-party":
				tpm := &thirdPartyModifier{}
				if err := tpm.Parse(ruleType); err != nil {
					return err
				}
				rm.modifiers = append(rm.modifiers, tpm)
			default:
				return fmt.Errorf("unknown modifier %s", ruleType)
			}
		}
	}

	return nil
}

func (rm *RuleModifiers) HandleRequest(req *http.Request) (*http.Request, *http.Response) {
	for _, modifier := range rm.modifiers {
		if !modifier.ShouldBlock(req) {
			return req, nil
		}
	}
	log.Printf("rule %s matched", rm.rule)
	return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "blocked by zen")
}
