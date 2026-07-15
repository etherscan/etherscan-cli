package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/chains"
	"github.com/etherscan/etherscan-cli/internal/client"
	"github.com/etherscan/etherscan-cli/internal/tui"
)

// TestTuiExecValidatesBeforeCall guards the fix for the TUI bypassing CLI param
// validation. tuiExec is built with a nil client on purpose: if validation did
// not run first, call() would dereference the nil client and panic. A clean
// validation error therefore proves the guard runs before any request.
func TestTuiExecValidatesBeforeCall(t *testing.T) {
	_, index := tuiEndpoints()
	rt := resolvedRuntime{chain: chains.Chain{ID: "1", Name: "ethereum"}}
	exec := tuiExec(&rt, index)

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
	exec := tuiExec(&rt, index)

	_, err := exec(context.Background(), "stats", "ethsupply2", map[string]string{})
	if err == nil {
		t.Fatal("expected a mainnet-only error on a non-mainnet chain")
	}
	if !strings.Contains(err.Error(), "only supported on Ethereum mainnet") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTuiValidate: the form's inline-validation hook applies the same guards as
// the executor — kind checks, the advanced-filter cross-field rule — and lets
// chainlist (no wire params) through trivially.
func TestTuiValidate(t *testing.T) {
	_, index := tuiEndpoints()
	rt := resolvedRuntime{chain: chains.Chain{ID: "1", Name: "ethereum"}}
	validate := tuiValidate(&rt, index)

	valid := "0x80f3950a4d371c43360f292a4170624abd9eed03"
	if err := validate("account", "txlist", map[string]string{"address": valid, "sort": "up"}); err == nil || !strings.Contains(err.Error(), "sort must be asc or desc") {
		t.Fatalf("bad sort not rejected: %v", err)
	}
	if err := validate("account", "txlist", map[string]string{"from": valid}); err == nil || !strings.Contains(err.Error(), "fromto") {
		t.Fatalf("advanced-filter rule not enforced: %v", err)
	}
	if err := validate("account", "txlist", map[string]string{"address": valid, "from": valid, "fromto_opr": "or"}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("address+from mutual exclusion not enforced: %v", err)
	}
	if err := validate("account", "txlist", map[string]string{"address": valid, "sort": "desc"}); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	if err := validate("getapilimit", "chainlist", nil); err != nil {
		t.Fatalf("chainlist must validate trivially: %v", err)
	}
	if err := validate("account", "nosuch", nil); err == nil {
		t.Fatal("unknown endpoint must be rejected")
	}
}

// TestTuiValidateReflectsChainSwitch proves the pointer wiring behind the chain
// switcher: tuiValidate captures &rt, so mutating rt.chain (as switchChain does)
// re-gates MainnetOnly endpoints without rebuilding the closure.
func TestTuiValidateReflectsChainSwitch(t *testing.T) {
	_, index := tuiEndpoints()
	rt := resolvedRuntime{chain: chains.Chain{ID: "1", Name: "ethereum"}}
	validate := tuiValidate(&rt, index)

	if err := validate("stats", "ethsupply2", map[string]string{}); err != nil {
		t.Fatalf("mainnet-only endpoint rejected on mainnet: %v", err)
	}
	// Simulate a chain switch (switchChain assigns a new rt through &rt).
	rt.chain = chains.Chain{ID: "56", Name: "bsc"}
	if err := validate("stats", "ethsupply2", map[string]string{}); err == nil || !strings.Contains(err.Error(), "only supported on Ethereum mainnet") {
		t.Fatalf("mainnet-only guard did not follow the chain switch: %v", err)
	}
}

// tuiGroup mirrors the sidebar grouping rule: the docs nav group label when set,
// else the wire module.
func tuiGroup(e tui.Endpoint) string {
	if e.Group != "" {
		return e.Group
	}
	return e.Module
}

// TestTuiEndpointsDocsOrderAndActionNames: the TUI is an API explorer, so its
// sidebar follows the Etherscan docs module order and its endpoint titles are
// the real API action names, not the friendly CLI aliases.
func TestTuiEndpointsDocsOrderAndActionNames(t *testing.T) {
	list, _ := tuiEndpoints()

	// First-seen group order must match tuiModuleOrder (for groups present).
	var seen []string
	for _, e := range list {
		if len(seen) == 0 || seen[len(seen)-1] != tuiGroup(e) {
			seen = append(seen, tuiGroup(e))
		}
	}
	want := 0
	for _, mod := range seen {
		found := -1
		for i := want; i < len(tuiModuleOrder); i++ {
			if tuiModuleOrder[i] == mod {
				found = i
				break
			}
		}
		if found < 0 {
			t.Fatalf("module %q out of docs order (or repeated); first-seen order: %v", mod, seen)
		}
		want = found + 1
	}
	if seen[0] != "account" {
		t.Fatalf("sidebar must start with account, got %v", seen)
	}

	// Titles are API action names, never CLI aliases.
	titles := map[string]string{} // module/title -> present
	for _, e := range list {
		titles[e.Module+"/"+e.Title] = e.Title
	}
	for _, mustHave := range []string{"block/getblockreward", "block/getblockcountdown", "logs/getLogs", "gastracker/gasoracle", "transaction/getstatus"} {
		if _, ok := titles[mustHave]; !ok {
			t.Fatalf("expected API action title %q in TUI list", mustHave)
		}
	}
	for _, alias := range []string{"block/reward", "block/countdown", "logs/get", "gastracker/oracle", "transaction/status"} {
		if _, ok := titles[alias]; ok {
			t.Fatalf("CLI alias %q leaked into TUI titles", alias)
		}
	}
}

// TestTuiExecChainList: the chainlist entry routes to client.ChainList (dedicated
// /v2/chainlist URL, no module/action/apikey params) and never touches the
// endpoint-spec index.
func TestTuiExecChainList(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		fmt.Fprint(w, `{"comments":"ok","totalcount":1,"result":[{"chainname":"Ethereum Mainnet","chainid":"1","blockexplorer":"https://etherscan.io","status":1}]}`)
	}))
	defer srv.Close()

	rt := resolvedRuntime{
		client: client.New(client.Options{BaseURL: srv.URL + "/v2/api", APIKey: "k", ChainID: "1", RateLimit: 1000}),
		chain:  chains.Chain{ID: "1", Name: "ethereum"},
	}
	// Empty index on purpose: chainlist must not need a spec entry.
	exec := tuiExec(&rt, map[string]EndpointSpec{})
	raw, err := exec(context.Background(), "getapilimit", "chainlist", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v2/chainlist" || gotQuery != "" {
		t.Fatalf("wrong request: path=%q query=%q", gotPath, gotQuery)
	}
	if !strings.Contains(string(raw), "Ethereum Mainnet") {
		t.Fatalf("unexpected result: %s", raw)
	}
}

// TestTuiChainListEntry: chainlist appears in the usage group after getapilimit,
// marked Bare (no module/action on the wire).
func TestTuiChainListEntry(t *testing.T) {
	list, index := tuiEndpoints()
	if _, ok := index["getapilimit/chainlist"]; ok {
		t.Fatal("chainlist must not be in the endpoint-spec index")
	}
	var usage []tui.Endpoint
	for _, e := range list {
		if tuiGroup(e) == "usage" {
			usage = append(usage, e)
		}
	}
	if len(usage) != 2 || usage[0].Action != "getapilimit" || usage[1].Action != "chainlist" {
		t.Fatalf("usage group wrong: %+v", usage)
	}
	// The sidebar label is the docs group, but the wire module must stay intact —
	// it drives the executor lookup and the result header.
	for _, e := range usage {
		if e.Module != "getapilimit" {
			t.Fatalf("usage endpoint %s must keep wire module getapilimit, got %q", e.Action, e.Module)
		}
	}
	if !usage[1].Bare {
		t.Fatal("chainlist must be marked Bare")
	}
	if len(usage[1].Params) != 0 || usage[1].Paginated {
		t.Fatal("chainlist takes no params and is not paginated")
	}
}

// TestTuiEndpointsDocsActionOrder: within each module, actions listed in
// tuiActionOrder appear in exactly that relative order (docs sidebar order), and
// unlisted actions come after all listed ones.
func TestTuiEndpointsDocsActionOrder(t *testing.T) {
	list, _ := tuiEndpoints()
	perModule := map[string][]string{}
	for _, e := range list {
		perModule[tuiGroup(e)] = append(perModule[tuiGroup(e)], e.Action)
	}

	for module, order := range tuiActionOrder {
		got := perModule[module]
		pos := map[string]int{}
		for i, a := range got {
			pos[a] = i
		}
		prev := -1
		for _, a := range order {
			i, ok := pos[a]
			if !ok {
				continue // listed but excluded from the TUI (e.g. write actions) or not in registry
			}
			if i < prev {
				t.Fatalf("%s: action %q out of docs order; got %v", module, a, got)
			}
			prev = i
		}
		// Unlisted actions must all rank after listed ones.
		listed := map[string]bool{}
		for _, a := range order {
			listed[a] = true
		}
		lastListed := -1
		for i, a := range got {
			if listed[a] {
				lastListed = i
			}
		}
		for i, a := range got {
			if !listed[a] && i < lastListed {
				t.Fatalf("%s: unlisted action %q appears before listed ones; got %v", module, a, got)
			}
		}
	}

	// Spot-check the exact regressions reported against the docs sidebar.
	for module, wantPrefix := range map[string][]string{
		"account":    {"balance", "balancemulti", "balancehistory", "txlist", "tokentx"},
		"block":      {"getblockreward", "getblocktxnscount", "getblockcountdown", "getblocknobytime"},
		"gastracker": {"gasestimate", "gasoracle"},
		"proxy":      {"eth_blockNumber", "eth_getBlockByNumber", "eth_getUncleByBlockNumberAndIndex"},
		"stats":      {"ethsupply", "ethsupply2", "ethprice", "chainsize", "nodecount", "dailytxnfee"},
	} {
		got := perModule[module]
		if len(got) < len(wantPrefix) {
			t.Fatalf("%s: only %d actions in TUI", module, len(got))
		}
		for i, want := range wantPrefix {
			if got[i] != want {
				t.Fatalf("%s: position %d = %q, want %q (full: %v)", module, i, got[i], want, got)
			}
		}
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
			// Optional params come through too (the form now collects them).
			if len(e.Params) < 2 || e.Params[1].Name != "tag" || e.Params[1].Required {
				t.Fatalf("account/balance optional tag param not adapted: %+v", e.Params)
			}
		}
	}
	if !found {
		t.Fatal("account/balance not present in TUI endpoints")
	}
}
