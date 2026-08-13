package chains

import (
	"slices"
	"testing"
)

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
	seenDisplay := map[string]string{}
	claim := func(key, owner string) {
		if prev, ok := seen[key]; ok {
			t.Fatalf("duplicate identifier %q used by %q and %q", key, prev, owner)
		}
		seen[key] = owner
	}
	for _, c := range registry {
		if c.ID == "" || c.Name == "" || c.DisplayName == "" || c.Explorer == "" {
			t.Fatalf("chain %q missing id/name/display name/explorer", c.Name)
		}
		if previous, ok := seenDisplay[c.DisplayName]; ok {
			t.Fatalf("duplicate display name %q used by %q and %q", c.DisplayName, previous, c.Name)
		}
		seenDisplay[c.DisplayName] = c.Name
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

func TestRegistryMatchesSupportedChainsOrderAndTiers(t *testing.T) {
	all := All()
	// 61 after dropping the deprecated Moonbeam family (1284, 1285, 1287). Update
	// this alongside the registry whenever Etherscan adds or removes a chain.
	if len(all) != 61 {
		t.Fatalf("supported chain count = %d, want 61", len(all))
	}
	if all[0].DisplayName != "Ethereum Mainnet" || all[len(all)-1].DisplayName != "MegaETH Testnet" {
		t.Fatalf("registry is not in supported-chains order: first=%q last=%q", all[0].DisplayName, all[len(all)-1].DisplayName)
	}

	var paidOnly []string
	for _, chain := range all {
		if !chain.FreeTier {
			paidOnly = append(paidOnly, chain.ID)
		}
	}
	wantPaidOnly := []string{"56", "97", "8453", "84532", "10", "11155420", "43114", "43113"}
	if !slices.Equal(paidOnly, wantPaidOnly) {
		t.Fatalf("paid-only chain IDs = %v, want %v", paidOnly, wantPaidOnly)
	}
}
