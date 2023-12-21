package rule

import (
	"fmt"
	"net/http"
	"strings"
)

type methodModifier struct {
	entries []methodModifierEntry
}

func (m *methodModifier) Parse(modifier string) error {
	entries := strings.Split(modifier, "|")
	m.entries = make([]methodModifierEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			return fmt.Errorf("empty method modifier entry")
		}
		mme := methodModifierEntry{}
		if err := mme.Parse(entry); err != nil {
			return err
		}
		m.entries = append(m.entries, mme)
	}
	return nil
}

func (m *methodModifier) ShouldMatch(req *http.Request) bool {
	method := req.Method
	for _, entry := range m.entries {
		if entry.MatchesMethod(method) {
			return true
		}
	}
	return false
}

type methodModifierEntry struct {
	method   string
	inverted bool
}

func (m *methodModifierEntry) Parse(modifier string) error {
	if modifier[0] == '~' {
		m.inverted = true
		modifier = modifier[1:]
	}
	m.method = modifier
	return nil
}

func (m *methodModifierEntry) MatchesMethod(method string) bool {
	if strings.ToLower(method) == m.method {
		return !m.inverted
	}
	return m.inverted
}
