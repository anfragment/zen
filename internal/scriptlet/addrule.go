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

	_, err := parseAdguardScriptlet(match[2])
	if err != nil {
		return fmt.Errorf("parse adguard scriptlet: %w", err)
	}

	return nil
}

func parseAdguardScriptlet(scriptletBody string) (Scriptlet, error) {
	if len(scriptletBody) == 0 {
		return Scriptlet{}, errors.New("scriptletBody is empty")
	}

	bodyParams := strings.Split(scriptletBody, ",")

	s := Scriptlet{}
	var err error
	s.Name, err = extractQuotedString(bodyParams[0])
	if err != nil {
		return Scriptlet{}, fmt.Errorf("extract quoted string from %q: %w", bodyParams[0], err)
	}
	s.Name = snakeToCamel(s.Name)

	if len(bodyParams) > 1 {
		s.Args = bodyParams[1:]
		for i := range s.Args {
			s.Args[i], err = extractQuotedString(s.Args[i])
			if err != nil {
				return Scriptlet{}, fmt.Errorf("extract quoted string from %q: %w", s.Args[i], err)
			}
		}
	}

	return s, nil
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
