package store

import (
	"testing"
)

func TestStore(t *testing.T) {
	t.Parallel()

	t.Run("store is empty", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		if rs == nil {
			t.Errorf("store is nil")
		}

		if len(*&rs.store) != 0 {
			t.Errorf("expected 0 rules, got %d", len(*&rs.store))
		}
	})

	t.Run("rule is added for all hostnames", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		rs.Add([]string{"example.org", "ex.org"}, ".rule")

		if len(*&rs.store) != 2 {
			t.Errorf("expected 2 rules, got %d", len(*&rs.store))
		}

		if rs.store["example.org"][0] != ".rule" {
			t.Errorf("expected .rule, got %s", rs.store["example.org"][0])
		}

		if rs.store["ex.org"][0] != ".rule" {
			t.Errorf("expected .rule, got %s", rs.store["ex.org"][0])
		}
	})

	t.Run("multiple rules on same hostname", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		rs.Add([]string{"example.org"}, ".rule1")
		rs.Add([]string{"example.org"}, ".rule2")

		if len(*&rs.store) != 1 {
			t.Errorf("expected 1 rule, got %d", len(*&rs.store))
		}
	})

	t.Run("add global rule", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		rs.Add([]string{}, "div.ticket")

		if len(*&rs.store) != 1 {
			t.Errorf("expected 1 rule, got %d", len(*&rs.store))
		}

		if rs.store["*"][0] != "div.ticket" {
			t.Errorf("expected div.ticket, got %s", rs.store[""][0])
		}
	})

	t.Run("get rule for hostname", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		rs.Add([]string{"example.org"}, ".rule")
		rs.Add([]string{"example.org"}, ".rule3")

		rules := rs.Get("example.org")

		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}

		if rules[0] != ".rule" {
			t.Errorf("expected .rule, got %s", rules[0])
		}

		if rules[1] != ".rule3" {
			t.Errorf("expected .rule3, got %s", rules[1])
		}
	})

	t.Run("should return both global and hostname specific rules", func(t *testing.T) {
		t.Parallel()

		rs := NewStore()
		rs.Add([]string{}, ".rule")
		rs.Add([]string{"example.org"}, ".rule2")

		rules := rs.Get("example.org")

		if len(rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(rules))
		}

		if rules[0] != ".rule" {
			t.Errorf("expected .rule, got %s", rules[0])
		}
	})
}
