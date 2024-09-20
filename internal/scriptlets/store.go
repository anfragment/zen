package scriptlets

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Store stores and matches scriptlet rules.
type Store struct {
	universalScriptlets []Scriptlet
	scriptletMap        map[string][]*Scriptlet
}

var (
	reAdguardScriptlet = regexp.MustCompile(`(.*)#%#\/\/scriptlet\((.+)\)`)
	errNotQuotedString = errors.New("not a quoted string")
)

func NewStore() *Store {
	return &Store{
		scriptletMap: make(map[string][]*Scriptlet),
	}
}

func (s *Store) AddRule(rule string) error {
	matches := reAdguardScriptlet.FindStringSubmatch(rule)
	if matches == nil {
		return errors.New("unsupported syntax")
	}

	bodyParams := strings.Split(matches[2], ",")

	scriptlet := Scriptlet{}
	var err error
	scriptlet.Name, err = extractQuotedString(bodyParams[0])
	if err != nil {
		return fmt.Errorf("extracting quoted string from %q: %w", bodyParams[0], err)
	}
	scriptlet.Name = snakeToCamel(scriptlet.Name)

	if len(bodyParams) > 1 {
		scriptlet.Args = make([]string, 0, len(bodyParams)-1)
		for i := 1; i < len(bodyParams); i++ {
			param, err := extractQuotedString(bodyParams[i])
			if err != nil {
				return fmt.Errorf("extracting quoted string from %q: %w", bodyParams[0], err)
			}
			scriptlet.Args = append(scriptlet.Args, param)
		}
	}

	if len(matches[1]) == 0 {
		s.universalScriptlets = append(s.universalScriptlets, scriptlet)
		return nil
	}

	domains := strings.Split(matches[1], ",")
	for _, domain := range domains {
		s.scriptletMap[domain] = append(s.scriptletMap[domain], &scriptlet)
	}
	return nil
}

func (s *Store) CreateInjection(hostname string) []byte {
	var buf []byte

	buf = append(buf, []byte("\n<script>\n")...)

	for _, s := range s.universalScriptlets {
		buf = append(buf, s.GenerateInjection()...)
	}
	hostnameScriptlets := s.scriptletMap[hostname]

	for _, s := range hostnameScriptlets {
		buf = append(buf, s.GenerateInjection()...)
	}
	if strings.HasPrefix(hostname, "www.") {
		wwwScriptlets := s.scriptletMap[strings.TrimPrefix(hostname, "www.")]
		for _, s := range wwwScriptlets {
			buf = append(buf, s.GenerateInjection()...)
		}
	}
	buf = append(buf, []byte("\n</script>\n")...)

	return buf
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

func extractQuotedString(quoted string) (string, error) {
	quoted = strings.TrimSpace(quoted)
	if len(quoted) < 3 {
		return "", errNotQuotedString
	}
	if (quoted[0] == '\'' && quoted[len(quoted)-1] == '\'') || (quoted[0] == '"' && quoted[len(quoted)-1] == '"') {
		return quoted[1 : len(quoted)-1], nil
	}
	return "", errNotQuotedString
}
