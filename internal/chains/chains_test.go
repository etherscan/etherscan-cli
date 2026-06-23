package chains

import "testing"

func TestResolveChain(t *testing.T) {
	tests := map[string]string{
		"ethereum": "1",
		"eth":      "1",
		"8453":     "8453",
		"base":     "8453",
		"arb":      "42161",
	}
	for input, want := range tests {
		got, err := Resolve(input)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", input, err)
		}
		if got.ID != want {
			t.Fatalf("Resolve(%q)=%s, want %s", input, got.ID, want)
		}
	}
	if _, err := Resolve("not-a-chain"); err == nil {
		t.Fatal("expected unknown chain error")
	}
}

func TestRegistryNoDuplicatesAndResolvable(t *testing.T) {
	seen := map[string]string{} // id/name/alias -> owning chain name
	claim := func(key, owner string) {
		if prev, ok := seen[key]; ok {
			t.Fatalf("duplicate identifier %q used by %q and %q", key, prev, owner)
		}
		seen[key] = owner
	}
	for _, c := range registry {
		if c.ID == "" || c.Name == "" || c.Explorer == "" {
			t.Fatalf("chain %q missing id/name/explorer", c.Name)
		}
		claim(c.ID, c.Name)
		claim(c.Name, c.Name)
		for _, a := range c.Aliases {
			claim(a, c.Name)
		}
		// every chain must resolve by its own id and canonical name
		if got, err := Resolve(c.ID); err != nil || got.ID != c.ID {
			t.Fatalf("Resolve(%q) failed for %q", c.ID, c.Name)
		}
		if got, err := Resolve(c.Name); err != nil || got.ID != c.ID {
			t.Fatalf("Resolve(%q) failed", c.Name)
		}
	}
}
