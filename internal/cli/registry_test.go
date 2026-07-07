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

func TestValidateAdvancedFilter(t *testing.T) {
	cases := []struct {
		name    string
		params  map[string]string
		wantErr bool
		wantOpr string
	}{
		{name: "none set", params: map[string]string{}, wantErr: false},
		{name: "from without opr", params: map[string]string{"from": "0x1"}, wantErr: true},
		{name: "and needs both", params: map[string]string{"from": "0x1", "fromto_opr": "and"}, wantErr: true},
		{name: "or with one ok, normalized", params: map[string]string{"from": "0x1", "fromto_opr": "or"}, wantErr: false, wantOpr: "OR"},
		{name: "and with both ok, normalized", params: map[string]string{"from": "0x1", "to": "0x2", "fromto_opr": "AnD"}, wantErr: false, wantOpr: "AND"},
		{name: "bad opr", params: map[string]string{"to": "0x2", "fromto_opr": "xor"}, wantErr: true},
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
