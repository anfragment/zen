package cosmetic

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// CosmeticRuleRegex matches cosmetic rules.
	CosmeticRuleRegex = regexp.MustCompile(`^(?:([^#$]+?)##|##)(.+)$`)

	errUnsupportedSyntax = errors.New("unsupported syntax")
)

func (inj *Injector) AddRule(rule string) error {
	var rawHostnames string
	var selector string

	if match := CosmeticRuleRegex.FindStringSubmatch(rule); match != nil {
		rawHostnames = match[1]
		selector = match[2]
	} else {
		return errUnsupportedSyntax
	}

	if len(rawHostnames) == 0 {
		inj.store.Add(nil, selector)
		return nil
	}

	hostnames := strings.Split(rawHostnames, ",")
	for _, hostname := range hostnames {
		if len(hostname) == 0 {
			return errors.New("empty hostnames are not allowed")
		}
	}
	inj.store.Add(hostnames, selector)

	return nil
}
