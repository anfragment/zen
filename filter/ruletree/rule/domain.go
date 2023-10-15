package rule

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type domainModifier struct {
	entries []domainModifierEntry
}

func (m *domainModifier) Parse(modifier string) error {
	entries := strings.Split(modifier, "|")
	m.entries = make([]domainModifierEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("empty domain modifier entry")
		}
		dme := domainModifierEntry{}
		if err := dme.Parse(entry); err != nil {
			return err
		}
		m.entries = append(m.entries, dme)
	}
	return nil
}

func (m *domainModifier) ShouldMatch(req *http.Request) bool {
	referer := req.Header.Get("Referer")
	if url, err := url.Parse(referer); err == nil {
		hostname := url.Hostname()
		for _, entry := range m.entries {
			if entry.MatchDomain(hostname) {
				return true
			}
		}
	}
	return false
}

func (m *domainModifier) RedirectTo(req *http.Request) string {
	return ""
}

type domainModifierEntry struct {
	invert  bool
	regular string
	tld     string
	regex   *regexp.Regexp
}

func (m *domainModifierEntry) Parse(entry string) error {
	if entry[0] == '~' {
		m.invert = true
		entry = entry[1:]
	}
	if entry[0] == '/' && entry[len(entry)-1] == '/' {
		regex, err := regexp.Compile(entry[1 : len(entry)-1])
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", entry, err)
		}
		m.regex = regex
	} else if entry[len(entry)-1] == '*' {
		m.tld = entry[:len(entry)-1]
	} else {
		m.regular = entry
	}
	return nil
}

func (m *domainModifierEntry) MatchDomain(domain string) bool {
	matches := false
	if m.regular != "" {
		matches = strings.HasSuffix(domain, m.regular)
	} else if m.tld != "" {
		matches = strings.HasPrefix(domain, m.tld)
	} else if m.regex != nil {
		matches = m.regex.MatchString(domain)
	}
	if m.invert {
		return !matches
	}
	return matches
}
