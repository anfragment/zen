package scriptlet

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	reAdguardScriptlet = regexp.MustCompile(`(.*)#%#\/\/scriptlet\((.+)\)`)
	errNotQuotedString = errors.New("not a quoted string")
)

func (i *Injector) AddRule(rule string) error {
	match := reAdguardScriptlet.FindStringSubmatch(rule)
	if match == nil {
		return errors.New("unsupported syntax")
	}

	scriptlet, err := parseAdguardScriptlet(match[2])
	if err != nil {
		return fmt.Errorf("parse adguard scriptlet: %w", err)
	}

	if len(match[1]) == 0 {
		i.store.Add(nil, scriptlet)
		return nil
	}

	hostnames := strings.Split(match[1], ",")
	for i, hostname := range hostnames {
		if len(hostname) == 0 {
			return errors.New("empty hostnames are not allowed")
		}
		if !strings.HasPrefix(hostname, "*.") {
			// Match subdomains as well. Not sure whether this is the correct behavior. FIXME
			hostnames[i] = "*." + hostname
		}
	}
	i.store.Add(hostnames, scriptlet)

	return nil
}

func parseAdguardScriptlet(scriptletBody string) (*Scriptlet, error) {
	if len(scriptletBody) == 0 {
		return nil, errors.New("scriptletBody is empty")
	}

	bodyParams := strings.Split(scriptletBody, ",")

	scriptlet := Scriptlet{}
	var err error
	scriptlet.Name, err = extractQuotedString(bodyParams[0])
	if err != nil {
		return nil, fmt.Errorf("extract quoted string from %q: %w", bodyParams[0], err)
	}
	scriptlet.Name = snakeToCamel(scriptlet.Name)

	if len(bodyParams) > 1 {
		scriptlet.Args = bodyParams[1:]
		for i := range scriptlet.Args {
			scriptlet.Args[i], err = extractQuotedString(scriptlet.Args[i])
			if err != nil {
				return nil, fmt.Errorf("extract quoted string from %q: %w", scriptlet.Args[i], err)
			}
		}
	}

	return &scriptlet, nil
}

func extractQuotedString(quoted string) (string, error) {
	quoted = strings.TrimSpace(quoted)
	if len(quoted) < 2 {
		return "", errNotQuotedString
	}
	if (quoted[0] == '\'' && quoted[len(quoted)-1] == '\'') || (quoted[0] == '"' && quoted[len(quoted)-1] == '"') {
		return quoted[1 : len(quoted)-1], nil
	}
	return "", errNotQuotedString
}

func snakeToCamel(snake string) string {
	words := strings.Split(snake, "-")
	for i := range words {
		if i > 0 {
			words[i] = strings.Title(words[i])
		}
	}
	return strings.Join(words, "")
}
