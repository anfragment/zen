package cosmetic

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	// RuleRegex matches cosmetic rules.
	RuleRegex = regexp.MustCompile(`^(?:([^#$]+?)##|##)(.+)$`)

	errUnsupportedSyntax = errors.New("unsupported syntax")
)

func (inj *Injector) AddRule(rule string) error {

	var rawHostnames string
	var selector string

	if match := RuleRegex.FindStringSubmatch(rule); match != nil {
		rawHostnames = match[1]
		selector = match[2]
	} else {
		return errUnsupportedSyntax
	}

	sanitizedSelector, err := sanitizeCSSSelector(selector)
	if err != nil {
		return fmt.Errorf("failed to sanitize selector: %w", err)
	}

	if len(rawHostnames) == 0 {
		inj.store.Add(nil, sanitizedSelector)
		return nil
	}

	hostnames := strings.Split(rawHostnames, ",")
	subdomainHostnames := make([]string, 0, len(hostnames))
	for _, hostname := range hostnames {
		if len(hostname) == 0 {
			return errors.New("empty hostnames are not allowed")
		}

		if net.ParseIP(hostname) == nil && !strings.HasPrefix(hostname, "*.") {
			subdomainHostnames = append(subdomainHostnames, "*."+hostname)
		}
	}
	inj.store.Add(hostnames, sanitizedSelector)
	inj.store.Add(subdomainHostnames, sanitizedSelector)

	return nil
}
