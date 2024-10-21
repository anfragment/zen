package scriptlet

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	// reAdguardScriptlet detects and extracts key data from AdGuard-style scriptlets.
	reAdguardScriptlet = regexp.MustCompile(`(.*)#%#\/\/scriptlet\((.+)\)`)
	// adguardToCanonical maps AdGuard scriptlet names to their respective implementations inside the scriptlet bundle.
	adguardToCanonical = map[string]string{
		"set-local-storage-item":   "setLocalStorageItem",
		"set-session-storage-item": "setSessionStorageItem",
		"nowebrtc":                 "nowebrtc",
		"prevent-fetch":            "preventFetch",
		"prevent-xhr":              "preventXHR",
		"set-constant":             "setConstant",
	}
	// reUblockScriptlet detects and extracts key data from uBlock Origin-style scriptlets.
	reUblockScriptlet = regexp.MustCompile(`(.*)##\+js\((.+)\)`)
	// ublockToCanonical maps uBlock Origin scriptlet names to their respective implementations inside the scriptlet bundle.
	ublockToCanonical = map[string]string{
		// TODO: manually check ublock syntax compatibility
		"set-local-storage-item": "setLocalStorageItem",
		"no-xhr-if":              "preventXHR",
		"no-fetch-if":            "preventFetch",
		"nowebrtc":               "nowebrtc",
		"set-constant":           "setConstant",
	}
	errNotQuotedString    = errors.New("not a quoted string")
	errUnsupportedSyntax  = errors.New("unsupported syntax")
	errEmptyScriptletBody = errors.New("scriptlet body is empty")
)

func (i *Injector) AddRule(rule string) error {
	var rawHostnames string
	var scriptlet *Scriptlet
	var err error
	if match := reAdguardScriptlet.FindStringSubmatch(rule); match != nil {
		rawHostnames = match[1]
		scriptlet, err = parseAdguardScriptlet(match[2])
		if err != nil {
			return fmt.Errorf("parse adguard scriptlet: %w", err)
		}
	} else if match := reUblockScriptlet.FindStringSubmatch(rule); match != nil {
		rawHostnames = match[1]
		scriptlet, err = parseUblockScriptlet(match[2])
		if err != nil {
			return fmt.Errorf("parse ublock origin scriptlet: %w", err)
		}
	} else {
		return errUnsupportedSyntax
	}

	if len(rawHostnames) == 0 {
		i.store.Add(nil, scriptlet)
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
	i.store.Add(hostnames, scriptlet)
	i.store.Add(subdomainHostnames, scriptlet)

	return nil
}

func parseAdguardScriptlet(scriptletBody string) (*Scriptlet, error) {
	if len(scriptletBody) == 0 {
		return nil, errEmptyScriptletBody
	}

	bodyParams := strings.Split(scriptletBody, ",")

	adguardName, err := extractQuotedString(bodyParams[0])
	if err != nil {
		return nil, fmt.Errorf("extract quoted string from %q: %w", bodyParams[0], err)
	}
	canonicalName, ok := adguardToCanonical[adguardName]
	if !ok {
		return nil, fmt.Errorf("%q is not a known AdGuard scriptlet", adguardName)
	}

	scriptlet := Scriptlet{
		Name: canonicalName,
	}
	if len(bodyParams) > 1 {
		scriptlet.Args = bodyParams[1:]
		for i := range scriptlet.Args {
			scriptlet.Args[i] = strings.TrimSpace(scriptlet.Args[i])
			scriptlet.Args[i], err = extractQuotedString(scriptlet.Args[i])
			if err != nil {
				return nil, fmt.Errorf("extract quoted string from %q: %w", scriptlet.Args[i], err)
			}
		}
	}

	return &scriptlet, nil
}

func parseUblockScriptlet(scriptletBody string) (*Scriptlet, error) {
	if len(scriptletBody) == 0 {
		return nil, errEmptyScriptletBody
	}

	bodyParams := strings.Split(scriptletBody, ",")

	canonicalName, ok := ublockToCanonical[bodyParams[0]]
	if !ok {
		return nil, fmt.Errorf("%q is not a known uBlock Origin scriptlet", bodyParams[0])
	}

	scriptlet := Scriptlet{
		Name: canonicalName,
	}
	if len(bodyParams) > 1 {
		scriptlet.Args = bodyParams[1:]
		for i := range scriptlet.Args {
			scriptlet.Args[i] = strings.TrimSpace(scriptlet.Args[i])
		}
	}

	return &scriptlet, nil
}

func extractQuotedString(quoted string) (string, error) {
	if len(quoted) < 2 {
		return "", errNotQuotedString
	}
	if (quoted[0] == '\'' && quoted[len(quoted)-1] == '\'') || (quoted[0] == '"' && quoted[len(quoted)-1] == '"') {
		return quoted[1 : len(quoted)-1], nil
	}
	return "", errNotQuotedString
}
