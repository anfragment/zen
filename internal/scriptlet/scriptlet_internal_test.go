package scriptlet

import (
	"reflect"
	"testing"
)

func TestInjectorInternal(t *testing.T) {
	t.Parallel()

	t.Run("parses Adguard-style rule", func(t *testing.T) {
		spyStore := &spyScriptletStore{}
		injector, err := NewInjector(spyStore)
		if err != nil {
			t.Fatalf("failed to create injector: %v", err)
		}

		injector.AddRule("example.org#%#//scriptlet('set-constant', 'first', 'false')")

		if len(spyStore.Entries) != 2 {
			t.Errorf("expected exactly two entries to be collected, got %d", len(spyStore.Entries))
		}

		hostnameSet := map[string]struct{}{}
		for _, entry := range spyStore.Entries {
			for _, hostname := range entry.Hostnames {
				hostnameSet[hostname] = struct{}{}
			}
		}

		if !reflect.DeepEqual(hostnameSet, map[string]struct{}{"example.org": {}, "*.example.org": {}}) {
			t.Errorf("expected hostnames to be collected, got %v", hostnameSet)
		}

		if spyStore.Entries[0].Scriptlet.Name != "setConstant" {
			t.Errorf("expected first scriptlet to be setConstant, got %q", spyStore.Entries[0].Scriptlet.Name)
		}
		if !reflect.DeepEqual(spyStore.Entries[0].Scriptlet.Args, []string{"first", "false"}) {
			t.Errorf("expected first scriptlet args to be [\"first\", \"false\"], got %v", spyStore.Entries[0].Scriptlet.Args)
		}

		if spyStore.Entries[1].Scriptlet.Name != "setConstant" {
			t.Errorf("expected second scriptlet to be setConstant, got %q", spyStore.Entries[1].Scriptlet.Name)
		}
		if !reflect.DeepEqual(spyStore.Entries[1].Scriptlet.Args, []string{"first", "false"}) {
			t.Errorf("expected second scriptlet args to be [\"first\", \"false\"], got %v", spyStore.Entries[1].Scriptlet.Args)
		}
	})

	t.Run("parses uBlock Origin-style rule", func(t *testing.T) {
		spyStore := &spyScriptletStore{}
		injector, err := NewInjector(spyStore)
		if err != nil {
			t.Fatalf("failed to create injector: %v", err)
		}

		injector.AddRule("example.com##+js(set-local-storage-item, player.live.current.mute, false)")

		if len(spyStore.Entries) != 2 {
			t.Errorf("expected exactly two entries to be collected, got %d", len(spyStore.Entries))
		}
		hostnameSet := map[string]struct{}{}
		for _, entry := range spyStore.Entries {
			for _, hostname := range entry.Hostnames {
				hostnameSet[hostname] = struct{}{}
			}
		}
		if !reflect.DeepEqual(hostnameSet, map[string]struct{}{"example.com": {}, "*.example.com": {}}) {
			t.Errorf("expected hostnames to be collected, got %v", hostnameSet)
		}

		if spyStore.Entries[0].Scriptlet.Name != "setLocalStorageItem" {
			t.Errorf("expected first scriptlet to be setConstant, got %q", spyStore.Entries[0].Scriptlet.Name)
		}
		if !reflect.DeepEqual(spyStore.Entries[0].Scriptlet.Args, []string{"player.live.current.mute", "false"}) {
			t.Errorf("expected first scriptlet args to be [\"player.live.current.mute\", \"false\"], got %v", spyStore.Entries[0].Scriptlet.Args)
		}
		if spyStore.Entries[1].Scriptlet.Name != "setLocalStorageItem" {
			t.Errorf("expected second scriptlet to be setConstant, got %q", spyStore.Entries[1].Scriptlet.Name)
		}
		if !reflect.DeepEqual(spyStore.Entries[1].Scriptlet.Args, []string{"player.live.current.mute", "false"}) {
			t.Errorf("expected second scriptlet args to be [\"player.live.current.mute\", \"false\"], got %v", spyStore.Entries[1].Scriptlet.Args)
		}
	})
}

type spyScriptletStore struct {
	Entries []spyScriptletStoreEntry
}

type spyScriptletStoreEntry struct {
	Hostnames []string
	Scriptlet *Scriptlet
}

func (s *spyScriptletStore) Add(hostnames []string, scriptlet *Scriptlet) {
	s.Entries = append(s.Entries, spyScriptletStoreEntry{
		Hostnames: hostnames,
		Scriptlet: scriptlet,
	})
}

func (s *spyScriptletStore) Get(hostname string) []*Scriptlet {
	return nil
}
