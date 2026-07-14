package cli

import (
	"strings"
	"testing"
)

func TestMaskKey(t *testing.T) {
	got := maskKey("ABCDEFGHIJKLMNOP")
	if got != "ABC"+strings.Repeat("*", 10)+"NOP" {
		t.Fatalf("maskKey long = %q", got)
	}
	if maskKey("abcdef") != "******" {
		t.Fatalf("short key not fully masked: %q", maskKey("abcdef"))
	}
}

func TestCheckKeyShape(t *testing.T) {
	if err := checkKeyShape(strings.Repeat("A", 34)); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	if err := checkKeyShape("ABC DEF"); err == nil {
		t.Fatal("expected whitespace key to be rejected")
	}
}

func TestEndpointRegistryCoversPromptModules(t *testing.T) {
	got := map[string]int{}
	for _, spec := range endpoints() {
		got[spec.Module]++
	}
	for _, module := range []string{"account", "contract", "transaction", "block", "proxy", "logs", "stats", "token", "gastracker", "nametag", "getapilimit"} {
		if got[module] == 0 {
			t.Fatalf("missing module %s", module)
		}
	}
	if got["proxy"] != 14 {
		t.Fatalf("proxy endpoint count=%d, want 14", got["proxy"])
	}
}

func TestAdvancedFilterCoversTransferActions(t *testing.T) {
	want := map[string]bool{"txlist": true, "txlistinternal": true, "tokentx": true, "tokennfttx": true, "token1155tx": true}
	for _, spec := range endpoints() {
		if !want[spec.Action] {
			continue
		}
		if !spec.AdvancedFilter {
			t.Fatalf("%s should set AdvancedFilter", spec.Action)
		}
		names := map[string]bool{}
		for _, p := range spec.Params {
			names[p.Name] = true
		}
		for _, p := range []string{"from", "to", "fromto_opr"} {
			if !names[p] {
				t.Fatalf("%s missing advanced-filter param %q", spec.Action, p)
			}
		}
	}
}

// TestFilterModeSpecs: the txlist family accepts alternative filters instead of
// a required address (server-verified against the API repo): address stays a
// positional (Arg) but is no longer Required, and RequireOneOf names the
// alternatives.
func TestFilterModeSpecs(t *testing.T) {
	oneOf := map[string][]string{
		"txlist":      {"address", "from", "to"},
		"tokentx":     {"address", "contractaddress", "from", "to"},
		"tokennfttx":  {"address", "contractaddress", "from", "to"},
		"token1155tx": {"address", "contractaddress", "from", "to"},
	}
	seen := 0
	for _, spec := range endpoints() {
		want, ok := oneOf[spec.Action]
		if !ok || spec.Module != "account" {
			continue
		}
		seen++
		if len(spec.RequireOneOf) != len(want) {
			t.Fatalf("%s RequireOneOf = %v, want %v", spec.Action, spec.RequireOneOf, want)
		}
		for i, name := range want {
			if spec.RequireOneOf[i] != name {
				t.Fatalf("%s RequireOneOf = %v, want %v", spec.Action, spec.RequireOneOf, want)
			}
		}
		for _, p := range spec.Params {
			if p.Name == "address" {
				if !p.Arg || p.Required {
					t.Fatalf("%s address must stay positional (Arg) but optional, got %+v", spec.Action, p)
				}
			}
		}
	}
	if seen != len(oneOf) {
		t.Fatalf("only %d of %d filter-mode specs found", seen, len(oneOf))
	}
}

// TestValidateParamsFilterModes: end-to-end through validateParams with the real
// specs — the exact combinations the server accepts and rejects.
func TestValidateParamsFilterModes(t *testing.T) {
	specs := map[string]EndpointSpec{}
	for _, spec := range endpoints() {
		if spec.Module == "account" {
			specs[spec.Action] = spec
		}
	}
	addr := "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97"
	addr2 := "0x504e7319f2257501552d5b412787d183efe5374f"

	// from/to-only txlist (the reported bug) must validate.
	if err := validateParams(specs["txlist"], map[string]string{"from": addr, "to": addr2, "fromto_opr": "and"}); err != nil {
		t.Fatalf("from/to-only txlist rejected: %v", err)
	}
	// address-only still validates.
	if err := validateParams(specs["txlist"], map[string]string{"address": addr}); err != nil {
		t.Fatalf("address-only txlist rejected: %v", err)
	}
	// nothing at all → at-least-one error.
	if err := validateParams(specs["txlist"], map[string]string{}); err == nil || !strings.Contains(err.Error(), "at least one of") {
		t.Fatalf("empty txlist not rejected with at-least-one: %v", err)
	}
	// address combined with from/to → mutual-exclusion error.
	if err := validateParams(specs["txlist"], map[string]string{"address": addr, "from": addr2, "fromto_opr": "or"}); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("address+from not rejected: %v", err)
	}
	// contractaddress-only tokentx is a valid query.
	if err := validateParams(specs["tokentx"], map[string]string{"contractaddress": addr}); err != nil {
		t.Fatalf("contractaddress-only tokentx rejected: %v", err)
	}
	if err := validateParams(specs["tokentx"], map[string]string{}); err == nil {
		t.Fatal("empty tokentx not rejected")
	}
	// txlistinternal keeps its permissive modes (no RequireOneOf).
	if err := validateParams(specs["txlistinternal"], map[string]string{"txhash": "0x" + strings.Repeat("a", 64)}); err != nil {
		t.Fatalf("txhash-only txlistinternal rejected: %v", err)
	}
}

// TestValidatePagination: the API only paginates when both page and offset are
// present, so the CLI rejects a half-specified pair on the single-call path.
func TestValidatePagination(t *testing.T) {
	specs := map[string]EndpointSpec{}
	for _, spec := range endpoints() {
		if spec.Module == "account" {
			specs[spec.Action] = spec
		}
	}
	txlist := specs["txlist"] // Paginated, declares both page and offset
	if !txlist.Paginated || !hasParam(txlist, "page") || !hasParam(txlist, "offset") {
		t.Fatalf("txlist spec precondition failed: %+v", txlist)
	}
	balance := specs["balance"] // not Paginated
	if balance.Paginated {
		t.Fatalf("balance unexpectedly Paginated")
	}
	// Paginated spec that only declares offset (no page) — the hasParam guard must
	// skip it so a lone offset is allowed through.
	offsetOnlySpec := EndpointSpec{Paginated: true, Params: []ParamSpec{p("offset", "limit", KindUint)}}

	cases := []struct {
		name    string
		spec    EndpointSpec
		params  map[string]string
		wantErr bool
	}{
		{name: "offset only", spec: txlist, params: map[string]string{"offset": "10"}, wantErr: true},
		{name: "page only", spec: txlist, params: map[string]string{"page": "2"}, wantErr: true},
		{name: "both set", spec: txlist, params: map[string]string{"page": "1", "offset": "10"}, wantErr: false},
		{name: "neither set", spec: txlist, params: map[string]string{}, wantErr: false},
		{name: "blank counts as unset", spec: txlist, params: map[string]string{"offset": "10", "page": "  "}, wantErr: true},
		{name: "non-paginated with stray offset", spec: balance, params: map[string]string{"offset": "10"}, wantErr: false},
		{name: "paginated without page param", spec: offsetOnlySpec, params: map[string]string{"offset": "10"}, wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePagination(tc.spec, tc.params)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v got err=%v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateAdvancedFilter(t *testing.T) {
	cases := []struct {
		name    string
		params  map[string]string
		wantErr bool
		wantOpr string
	}{
		{name: "none set", params: map[string]string{}, wantErr: false},
		{name: "address only ok", params: map[string]string{"address": "0x1"}, wantErr: false},
		{name: "from without opr", params: map[string]string{"from": "0x1"}, wantErr: true},
		{name: "and needs both", params: map[string]string{"from": "0x1", "fromto_opr": "and"}, wantErr: true},
		{name: "or with one ok, normalized", params: map[string]string{"from": "0x1", "fromto_opr": "or"}, wantErr: false, wantOpr: "OR"},
		{name: "and with both ok, normalized", params: map[string]string{"from": "0x1", "to": "0x2", "fromto_opr": "AnD"}, wantErr: false, wantOpr: "AND"},
		{name: "bad opr", params: map[string]string{"to": "0x2", "fromto_opr": "xor"}, wantErr: true},
		// The server rejects address combined with the from/to filter mode.
		{name: "address with from rejected", params: map[string]string{"address": "0x1", "from": "0x2", "fromto_opr": "or"}, wantErr: true},
		{name: "address with to rejected", params: map[string]string{"address": "0x1", "to": "0x2", "fromto_opr": "or"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAdvancedFilter(tc.params)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v got err=%v", tc.wantErr, err)
			}
			if tc.wantOpr != "" && tc.params["fromto_opr"] != tc.wantOpr {
				t.Fatalf("operator not normalized: got %q want %q", tc.params["fromto_opr"], tc.wantOpr)
			}
		})
	}
}
