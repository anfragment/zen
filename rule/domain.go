package rule

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type domainModifier struct {
	entries  []domainModifierEntry
	inverted bool
}

var (
	domainModifierRegex = regexp.MustCompile(`~?((/.*/)|[^|]+)+`)
)

func (m *domainModifier) Parse(modifier string) error {
	eqIndex := strings.IndexByte(modifier, '=')
	if eqIndex == -1 || eqIndex == len(modifier)-1 {
		return errors.New("invalid domain modifier")
	}
	value := modifier[eqIndex+1:]

	m.inverted = strings.HasPrefix(value, "~")
	matches := domainModifierRegex.FindAllString(value, -1)
	m.entries = make([]domainModifierEntry, len(matches))
	for i, entry := range matches {
		if entry == "" {
			return errors.New("empty method modifier entry")
		}
		inverted := strings.HasPrefix(entry, "~")
		if inverted != m.inverted {
			return errors.New("cannot mix inverted and non-inverted method modifiers")
		}
		if inverted {
			entry = entry[1:]
		}

		m.entries[i] = domainModifierEntry{}
		m.entries[i].Parse(entry)
	}
	return nil
}

func (m *domainModifier) ShouldMatch(req *http.Request) bool {
	if referer := req.Header.Get("Referer"); referer == "" {
		return false
	}
	url, err := url.Parse(req.Header.Get("Referer"))
	if err != nil {
		return false
	}
	hostname := url.Hostname()

	matches := false
	for _, entry := range m.entries {
		if entry.MatchDomain(hostname) {
			matches = true
			break
		}
	}
	if m.inverted {
		return !matches
	}
	return matches
}

type domainModifierEntry struct {
	regular string
	tld     string
	regex   *regexp.Regexp
}

func (m *domainModifierEntry) Parse(entry string) error {
	switch {
	case entry[0] == '/' && entry[len(entry)-1] == '/':
		regex, err := regexp.Compile(entry[1 : len(entry)-1])
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", entry, err)
		}
		m.regex = regex
	case entry[len(entry)-1] == '*':
		m.tld = entry[:len(entry)-1]
	default:
		m.regular = entry
	}
	return nil
}

func (m *domainModifierEntry) MatchDomain(domain string) bool {
	switch {
	case m.regular != "":
		return strings.HasSuffix(domain, m.regular)
	case m.tld != "":
		return strings.HasPrefix(domain, m.tld)
	case m.regex != nil:
		return m.regex.MatchString(domain)
	default:
		return false
	}
}
