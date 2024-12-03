package triestore

import (
	"testing"
)

func TestStore(t *testing.T) {
	t.Parallel()

	t.Run("rule is added for all hostnames", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()

		s.Add([]string{"example.org", "ex.org"}, ".rule")

		if len(s.root.children) != 2 {
			t.Errorf("expected 2 children, got %d", len(s.root.children))
		}
	})

	t.Run("multiple rules on same hostname", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()

		s.Add([]string{"example.org"}, ".rule")
		s.Add([]string{"example.org"}, ".rule2")

		if len(s.root.children) != 1 {
			t.Errorf("expected 1 child, got %d", len(s.root.children))
		}
	})

	t.Run("add global rule", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()
		s.Add([]string{}, "div.ticket")
		s.Add(nil, "div.container")

		rules := s.Get("example.org")
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}

		rules = s.Get("example.com")
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}
	})

	t.Run("get rule for hostname", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()
		s.Add([]string{"example.org"}, ".rule")

		rules := s.Get("example.org")

		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("match wildcard hostname", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()
		s.Add([]string{"*.example.org"}, ".rule")

		rules := s.Get("mail.example.org")

		if len(rules) != 1 {
			t.Error("expected 1 rule")
		}
	})

	t.Run("match top level domain wildcard", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()
		s.Add([]string{"example.*"}, ".rule")

		rules := s.Get("example.org")
		if len(rules) != 1 {
			t.Error("expected 1 rule")
		}

		rules = s.Get("example.co.uk")
		if len(rules) != 1 {
			t.Error("expected 1 rule")
		}
	})

	t.Run("match multiple rules", func(t *testing.T) {
		t.Parallel()

		s := NewTrieStore()
		s.Add([]string{"example.*", "*.co.uk"}, ".rule")
		s.Add([]string{"mail.example.co.uk"}, ".rule2")

		rules := s.Get("mail.example.co.uk")
		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}
	})
}
