package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/chains"
)

// TestTuiExecValidatesBeforeCall guards the fix for the TUI bypassing CLI param
// validation. tuiExec is built with a nil client on purpose: if validation did
// not run first, call() would dereference the nil client and panic. A clean
// validation error therefore proves the guard runs before any request.
func TestTuiExecValidatesBeforeCall(t *testing.T) {
	_, index := tuiEndpoints()
	rt := resolvedRuntime{chain: chains.Chain{ID: "1", Name: "ethereum"}}
	exec := tuiExec(rt, index)

	valid := "0x80f3950a4d371c43360f292a4170624abd9eed03"
	_, err := exec(context.Background(), "account", "balancemulti", map[string]string{"address": valid + ",," + valid})
	if err == nil {
		t.Fatal("expected a validation error for an empty comma-list entry")
	}
	if !strings.Contains(err.Error(), "must not contain empty entries") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTuiExecMainnetOnlyGuard proves the mainnet-only guard also runs before
// call(): a MainnetOnly endpoint on a non-mainnet chain is rejected without ever
// reaching the (nil) client.
func TestTuiExecMainnetOnlyGuard(t *testing.T) {
	_, index := tuiEndpoints()
	rt := resolvedRuntime{chain: chains.Chain{ID: "56", Name: "bsc"}}
	exec := tuiExec(rt, index)

	_, err := exec(context.Background(), "stats", "ethsupply2", map[string]string{})
	if err == nil {
		t.Fatal("expected a mainnet-only error on a non-mainnet chain")
	}
	if !strings.Contains(err.Error(), "only supported on Ethereum mainnet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTuiEndpointsExcludeWriteActions(t *testing.T) {
	list, index := tuiEndpoints()
	if len(list) == 0 {
		t.Fatal("expected some browsable endpoints")
	}
	// Write/sensitive actions must never appear in the read-only explorer.
	for _, e := range list {
		if e.Module == "contract" && (e.Action == "verifysourcecode" || e.Action == "verifyproxycontract") {
			t.Fatalf("write action leaked into TUI: %s/%s", e.Module, e.Action)
		}
		if e.Module == "proxy" && e.Action == "eth_sendRawTransaction" {
			t.Fatalf("write action leaked into TUI: %s/%s", e.Module, e.Action)
		}
	}
	if _, ok := index["proxy/eth_sendRawTransaction"]; ok {
		t.Fatal("excluded action should not be in the executor index")
	}
	if _, ok := index["account/balance"]; !ok {
		t.Fatal("account/balance missing from executor index")
	}

	// A known read endpoint carries its required param through the adapter.
	var found bool
	for _, e := range list {
		if e.Module == "account" && e.Action == "balance" {
			found = true
			if len(e.Params) == 0 || e.Params[0].Name != "address" || !e.Params[0].Required {
				t.Fatalf("account/balance address param not adapted: %+v", e.Params)
			}
		}
	}
	if !found {
		t.Fatal("account/balance not present in TUI endpoints")
	}
}
