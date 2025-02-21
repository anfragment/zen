package scriptlet

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	adguardPrimaryRegex   = regexp.MustCompile(`(.*)#%#\/\/scriptlet\((.+)\)`)
	adguardExceptionRegex = regexp.MustCompile(`(.*)#@%#\/\/scriptlet\((.+)\)`)
	ublockPrimaryRegex    = regexp.MustCompile(`(.*)##\+js\((.+)\)`)
	ublockExceptionRegex  = regexp.MustCompile(`(.*)#@#\+js\((.+)\)`)
	errUnsupportedSyntax  = errors.New("unsupported syntax")
	// TODO: rethink and reimplement trusted rule handling
	// trustedOnlyScriptlets              = []string{}
	// errTrustedScriptletInUntrustedList = errors.New("trusted scriptlet in untrusted list")
)

func (inj *Injector) AddRule(rule string, filterListTrusted bool) error {
	if match := adguardPrimaryRegex.FindStringSubmatch(rule); match != nil {
		normalized, err := argList(match[2]).Normalize()
		if err != nil {
			return fmt.Errorf("normalize scriptlet body: %w", err)
		}
		inj.store.AddPrimaryRule(match[1], normalized)
	} else if match := adguardExceptionRegex.FindStringSubmatch(rule); match != nil {
		normalized, err := argList(match[2]).Normalize()
		if err != nil {
			return fmt.Errorf("normalize scriptlet body: %w", err)
		}
		inj.store.AddExceptionRule(match[1], normalized)
	} else if match := ublockPrimaryRegex.FindStringSubmatch(rule); match != nil {
		normalized, err := argList(match[2]).ConvertUboToCanonical().Normalize()
		if err != nil {
			return fmt.Errorf("normalize scriptlet body: %w", err)
		}
		inj.store.AddPrimaryRule(match[1], normalized)
	} else if match := ublockExceptionRegex.FindStringSubmatch(rule); match != nil {
		normalized, err := argList(match[2]).ConvertUboToCanonical().Normalize()
		if err != nil {
			return fmt.Errorf("normalize scriptlet body: %w", err)
		}
		inj.store.AddExceptionRule(match[1], normalized)
	} else {
		return errUnsupportedSyntax
	}

	return nil
}
