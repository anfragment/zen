package triestore

import (
	"testing"

	"github.com/anfragment/zen/internal/scriptlet"
)

func TestTrie(t *testing.T) {
	t.Parallel()

	t.Run("matches single hostname", func(t *testing.T) {
		t.Parallel()

		store := NewTrieStore()
		store.Add([]string{"example.org"}, &scriptlet.Scriptlet{})

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
		t.Parallel()

		store := NewTrieStore()
		store.Add([]string{"*.example.org"}, &scriptlet.Scriptlet{})

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
		t.Parallel()

		store := NewTrieStore()
		store.Add([]string{"example.*"}, &scriptlet.Scriptlet{})

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

	t.Run("returns multiple matching rules", func(t *testing.T) {
		t.Parallel()

		store := NewTrieStore()
		store.Add([]string{"example.*", "*.co.uk"}, &scriptlet.Scriptlet{Name: "test1"})
		store.Add([]string{"mail.example.co.uk"}, &scriptlet.Scriptlet{Name: "test2"})

		retrieved := store.Get("mail.example.co.uk")
		if len(retrieved) != 2 {
			t.Errorf("expected 2 scriptlets for mail.example.co.uk, got %d", len(retrieved))
		}
		if retrieved[0].Name == retrieved[1].Name {
			t.Error("expected 2 distinct scriptlets for mail.example.co.uk")
		}
	})
}
