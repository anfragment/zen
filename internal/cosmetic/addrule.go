package cosmetic

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

var (
	// RuleRegex matches cosmetic rules.
	RuleRegex = regexp.MustCompile(`^(?:([^#$]+?)##|##)(.+)$`)

	errUnsupportedSyntax = errors.New("unsupported syntax")
	errNotAllowed        = errors.New("rule contains </style> which is not allowed")
)

func (inj *Injector) AddRule(rule string) error {
	if strings.Contains(rule, "</style>") {
		return errNotAllowed
	}

	var rawHostnames string
	var selector string

	if match := RuleRegex.FindStringSubmatch(rule); match != nil {
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
	subdomainHostnames := make([]string, 0, len(hostnames))
	for _, hostname := range hostnames {
		if len(hostname) == 0 {
			return errors.New("empty hostnames are not allowed")
		}

		if net.ParseIP(hostname) == nil && !strings.HasPrefix(hostname, "*.") {
			subdomainHostnames = append(subdomainHostnames, "*."+hostname)
		}
	}
	inj.store.Add(hostnames, selector)
	inj.store.Add(subdomainHostnames, selector)

	return nil
}
