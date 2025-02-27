package rulemodifiers

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type MethodModifier struct {
	entries  []methodModifierEntry
	inverted bool
}

var _ MatchingModifier = (*MethodModifier)(nil)

func (m *MethodModifier) Parse(modifier string) error {
	eqIndex := strings.IndexByte(modifier, '=')
	if eqIndex == -1 {
		return fmt.Errorf("invalid method modifier")
	}
	value := modifier[eqIndex+1:]

	m.inverted = strings.HasPrefix(value, "~")
	entries := strings.Split(value, "|")
	m.entries = make([]methodModifierEntry, len(entries))
	for i, entry := range entries {
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

		m.entries[i] = methodModifierEntry{}
		m.entries[i].Parse(entry)
	}
	return nil
}

func (m *MethodModifier) ShouldMatchReq(req *http.Request) bool {
	matches := false
	for _, entry := range m.entries {
		if entry.MatchesMethod(req.Method) {
			matches = true
			break
		}
	}
	if m.inverted {
		return !matches
	}
	return matches
}

func (m *MethodModifier) ShouldMatchRes(_ *http.Response) bool {
	return false
}

type methodModifierEntry struct {
	// method is the method to match. It is expected to be uppercase.
	method string
}

func (m *methodModifierEntry) Parse(modifier string) {
	m.method = strings.ToUpper(modifier)
}

// MatchesMethod returns true if the method matches the entry.
// The method is expected to be uppercase.
func (m *methodModifierEntry) MatchesMethod(method string) bool {
	return m.method == method
}

func (mm *MethodModifier) Cancels(m Modifier) bool {
	rm, ok := m.(*MethodModifier)
	if !ok {
		return false
	}

	for _, v := range mm.entries {
		if !slices.Contains(rm.entries, v) {
			return false
		}
	}

	return rm.inverted == mm.inverted
}
