package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

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
		client: client.New(client.Options{BaseURL: srv.URL + "/v2/api", ChainID: "1", RateLimit: 1000}),
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

func TestTuiChainsPreservesSupportedChainMetadata(t *testing.T) {
	all := tuiChains()
	// Tracks the registry count, which excludes the deprecated Moonbeam family.
	if len(all) != 61 || all[0].DisplayName != "Ethereum Mainnet" || all[len(all)-1].DisplayName != "MegaETH Testnet" {
		t.Fatalf("TUI chains do not follow supported-chains order: count=%d first=%q last=%q", len(all), all[0].DisplayName, all[len(all)-1].DisplayName)
	}
	for _, chain := range all {
		if chain.Name != "polygon" {
			continue
		}
		if !slices.Contains(chain.Aliases, "matic") {
			t.Fatalf("polygon aliases missing matic: %v", chain.Aliases)
		}
		if chain.DisplayName != "Polygon Mainnet" || chain.PaidOnly {
			t.Fatalf("polygon metadata incorrect: %+v", chain)
		}
		for _, paid := range all {
			if paid.Name == "base" && !paid.PaidOnly {
				t.Fatalf("base must be marked paid-only: %+v", paid)
			}
		}
		return
	}
	t.Fatal("polygon missing from TUI chains")
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
	contractCount := 0
	for _, e := range list {
		if e.Module == "contract" {
			contractCount++
		}
		if e.Module == "contract" && (e.Action == "verifysourcecode" || e.Action == "verifyproxycontract") {
			t.Fatalf("write action leaked into TUI: %s/%s", e.Module, e.Action)
		}
		if e.Module == "proxy" && e.Action == "eth_sendRawTransaction" {
			t.Fatalf("write action leaked into TUI: %s/%s", e.Module, e.Action)
		}
	}
	// Counted by wire module, which the CLI group split does not change: 3 data
	// reads plus the 2 verification polls.
	if contractCount != 5 {
		t.Fatalf("read-only TUI contract endpoint count = %d, want 5", contractCount)
	}
	// The two polls sit in the verification sidebar group, mirroring the CLI split
	// (under the shorter label the panel can fit), but keep the contract wire
	// module — it drives the executor lookup and the result header.
	polls := map[string]bool{"checkverifystatus": true, "checkproxyverification": true}
	seenPolls := 0
	for _, e := range list {
		switch {
		case polls[e.Action]:
			seenPolls++
			if tuiGroup(e) != "verification" {
				t.Fatalf("poll %s sidebar group = %q, want verification", e.Action, tuiGroup(e))
			}
			if e.Module != "contract" {
				t.Fatalf("poll %s must keep wire module contract, got %q", e.Action, e.Module)
			}
		case e.Module == "contract":
			if tuiGroup(e) != "contract" {
				t.Fatalf("contract read %s sidebar group = %q, want contract", e.Action, tuiGroup(e))
			}
		}
	}
	if seenPolls != 2 {
		t.Fatalf("browsable verification polls = %d, want 2", seenPolls)
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

// TestTuiSidebarLabelsFitPanel guards the MODULES panel layout. The panel is a
// fixed width and lipgloss wraps rather than truncates, so a label longer than
// tui.SidebarLabelMaxLen splits across two rows — which breaks the
// one-row-per-item windowing and pushes the view past its height budget, trimming
// the header off the top. The CLI group "contractverification" is itself too long,
// which is why tuiGroupLabel shortens it to "verification".
func TestTuiSidebarLabelsFitPanel(t *testing.T) {
	list, _ := tuiEndpoints()
	seen := map[string]bool{}
	for _, e := range list {
		label := tuiGroup(e)
		if seen[label] {
			continue
		}
		seen[label] = true
		if len(label) > tui.SidebarLabelMaxLen {
			t.Errorf("sidebar label %q is %d chars, exceeds the %d that fit the panel", label, len(label), tui.SidebarLabelMaxLen)
		}
	}
	if !seen["verification"] {
		t.Fatal("expected a verification sidebar group")
	}
	// Every label must be ordered, or rank() sinks it to the bottom of the sidebar.
	for label := range seen {
		if !slices.Contains(tuiModuleOrder, label) {
			t.Errorf("sidebar label %q is missing from tuiModuleOrder and would sink to the end", label)
		}
	}
}

// TestTuiGroupLabelsCoverCLIGroups: every CLI command group either shares its name
// with a sidebar label that fits the panel, or has an entry in tuiGroupLabel. Without
// this, adding a group with a long name silently breaks the TUI layout again.
func TestTuiGroupLabelsCoverCLIGroups(t *testing.T) {
	for _, spec := range endpoints() {
		group := spec.CommandGroup()
		if _, mapped := tuiGroupLabel[group]; mapped {
			continue
		}
		if len(group) > tui.SidebarLabelMaxLen {
			t.Errorf("CLI group %q is %d chars and has no tuiGroupLabel entry; it would wrap in the sidebar", group, len(group))
		}
	}
}

// TestRootLevelPlacementIsExplicit pins where each endpoint lands in the command
// tree. Placement used to be inferred from the wire module (`spec.Module ==
// "getapilimit"`), which agreed with CommandGroup() only by accident: adding a
// Group to that spec would have left the tree and the group label disagreeing.
// RootLevel now declares the intent, and the two must never both be set — a
// root-level command has no parent group, so a Group on it would be silently
// ignored.
func TestRootLevelPlacementIsExplicit(t *testing.T) {
	root := newRootCommand(BuildInfo{Version: "test"}, &fakeUpdateManager{})

	childNames := func(cmd *cobra.Command) map[string]*cobra.Command {
		out := map[string]*cobra.Command{}
		for _, c := range cmd.Commands() {
			out[c.Name()] = c
		}
		return out
	}
	topLevel := childNames(root)

	// Pin the actual command surface, not just spec/tree agreement. Without these
	// two lines the loop below is self-fulfilling: drop RootLevel and the spec says
	// "grouped" while the tree obligingly groups it, so consistency still holds and
	// nothing catches that `etherscan apilimit` silently became
	// `etherscan getapilimit apilimit`.
	if _, ok := topLevel["apilimit"]; !ok {
		t.Error("apilimit must be a top-level command: `etherscan apilimit`")
	}
	if _, ok := topLevel["getapilimit"]; ok {
		t.Error("a \"getapilimit\" command group exists; apilimit is meant to sit at the root, not under its wire module")
	}

	for _, spec := range endpoints() {
		name := commandWord(spec)
		if spec.RootLevel {
			if spec.Group != "" {
				t.Errorf("%q sets both RootLevel and Group %q; a root-level command is never filed under a group", name, spec.Group)
			}
			if _, ok := topLevel[name]; !ok {
				t.Errorf("RootLevel spec %q is not a direct child of the root command", name)
			}
			if _, grouped := topLevel[spec.CommandGroup()]; grouped {
				t.Errorf("RootLevel spec %q also produced a %q command group", name, spec.CommandGroup())
			}
			continue
		}
		group, ok := topLevel[spec.CommandGroup()]
		if !ok {
			t.Errorf("spec %q expects command group %q, which is not under the root command", name, spec.CommandGroup())
			continue
		}
		if _, ok := childNames(group)[name]; !ok {
			t.Errorf("spec %q is not filed under its command group %q", name, spec.CommandGroup())
		}
	}
}

// commandWord is the first token of a spec's cobra Use string ("verify <address>"
// -> "verify"), which identifies a spec in test failures better than a module and
// action pair that several specs deliberately share.
func commandWord(spec EndpointSpec) string {
	if fields := strings.Fields(spec.Use); len(fields) > 0 {
		return fields[0]
	}
	return spec.Action
}

// TestTuiEndpointIndexKeysAreUnique guards a latent collision in tuiEndpoints: the
// executor index is keyed on Module+"/"+Action, and four contract specs share
// contract/verifysourcecode (verify, verify-zksync, verify-vyper, verify-stylus).
// A map cannot hold all four, so it would keep whichever came last and the TUI
// would dispatch the wrong spec with no error. That never happens today only
// because the Post/Sensitive filter drops all four before the index is built, so
// this pins that dependency: if the filter moves or relaxes, or a browsable spec
// reuses a module/action pair, the count stops matching here rather than silently
// running the wrong endpoint for a user.
func TestTuiEndpointIndexKeysAreUnique(t *testing.T) {
	_, index := tuiEndpoints()

	browsable := 0
	seen := map[string]string{}
	for _, spec := range endpoints() {
		if spec.Post || spec.Sensitive {
			continue
		}
		browsable++
		key := spec.Module + "/" + spec.Action
		if prev, dup := seen[key]; dup {
			t.Errorf("browsable specs %q and %q both key the executor index on %q", prev, commandWord(spec), key)
		}
		seen[key] = commandWord(spec)
	}

	// A smaller index means browsable specs collided on a key; a larger one means
	// tuiEndpoints stopped excluding Post/Sensitive specs, which reintroduces the
	// four-way verifysourcecode collision. Either way the invariant is broken.
	if len(index) != browsable {
		t.Fatalf("executor index holds %d specs but %d specs are browsable: module/action keys collided, or tuiEndpoints changed which specs it excludes", len(index), browsable)
	}
}

// TestVerificationSubmissionsShareOneWireAction documents why the collision above
// stays latent: the source-verification variants intentionally share one wire
// action and differ only by CLI command, so each must be excluded from the
// read-only TUI. A variant added without Post or Sensitive would be browsable and
// would start overwriting its siblings in the executor index.
func TestVerificationSubmissionsShareOneWireAction(t *testing.T) {
	var sharing []string
	for _, spec := range endpoints() {
		if spec.Module != "contract" || spec.Action != "verifysourcecode" {
			continue
		}
		sharing = append(sharing, commandWord(spec))
		if !spec.Post && !spec.Sensitive {
			t.Errorf("%q shares contract/verifysourcecode but is neither Post nor Sensitive, so it reaches the TUI executor index", commandWord(spec))
		}
	}
	if len(sharing) < 2 {
		t.Fatalf("expected several specs sharing contract/verifysourcecode, got %v", sharing)
	}
}
