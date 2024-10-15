package scriptlet

import "testing"

func TestTree(t *testing.T) {
	t.Parallel()

	t.Run("matches single hostname", func(t *testing.T) {
		store := NewTreeStore()
		store.Add([]string{"example.org"}, scriptlet{})

		retrieved := store.Get("example.org")
		if len(retrieved) == 0 {
			t.Error("scriptlet not retrieved for example.org")
		}

		retrieved = store.Get("example.com")
		if len(retrieved) != 0 {
			t.Error("example.com should not match")
		}
	})

	t.Run("matches wildcard hostname (wildcard at the beginning)", func(t *testing.T) {
		store := NewTreeStore()
		store.Add([]string{"*.example.org"}, scriptlet{})

		retrieved := store.Get("mail.example.org")
		if len(retrieved) == 0 {
			t.Error("scriptlet not retrieved for mail.example.org")
		}

		retrieved = store.Get("imap.mail.example.org")
		if len(retrieved) == 0 {
			t.Error("scriptlet not retrieved for imap.mail.example.org")
		}

		retrieved = store.Get("example.org")
		if len(retrieved) != 0 {
			t.Error("example.org should not match")
		}
	})

	t.Run("matches wildcard hostname (wildcard at the end)", func(t *testing.T) {
		store := NewTreeStore()
		store.Add([]string{"example.*"}, scriptlet{})

		retrieved := store.Get("example.org")
		if len(retrieved) == 0 {
			t.Error("scriptlet not retrieved for example.org")
		}

		retrieved = store.Get("example.co.uk")
		if len(retrieved) == 0 {
			t.Error("scriptlet not retrieved for example.co.uk")
		}

		retrieved = store.Get("test.com")
		if len(retrieved) != 0 {
			t.Error("test.com should not match")
		}
	})

	t.Run("handles multiple rules and deduplication", func(t *testing.T) {
		store := NewTreeStore()
		store.Add([]string{"example.*", "*.co.uk"}, scriptlet{Name: "test1"})
		store.Add([]string{"mail.example.co.uk"}, scriptlet{Name: "test2"})

		retrieved := store.Get("example.co.uk")
		if len(retrieved) != 1 || retrieved[0].Name != "test1" {
			t.Error("expected test1")
		}

		retrieved = store.Get("mail.example.co.uk")
		if len(retrieved) != 2 || retrieved[0].Name == retrieved[1].Name {
			t.Error("expected 2 results")
		}
	})
}
