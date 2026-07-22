package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testModel(exec Exec) *model {
	cfg := Config{
		Endpoints: []Endpoint{
			{Module: "account", Action: "balance", Title: "balance", Desc: "Get native balance",
				Params: []Param{{Name: "address", Label: "address", Required: true}, {Name: "tag", Label: "block tag"}}},
			{Module: "stats", Action: "ethprice", Title: "ethprice", Desc: "Get ether price"},
		},
		Exec:      exec,
		ChainName: "ethereum",
		ChainID:   "1",
		KeyLabel:  "none",
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return &m
}

func TestBrowseNoParamEndpointFetches(t *testing.T) {
	var gotModule, gotAction string
	exec := func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		gotModule, gotAction = module, action
		return json.RawMessage(`{"ethusd":"1000"}`), nil
	}
	m := testModel(exec)

	// Select the "stats" module (index 1) and its only endpoint (ethprice).
	m.modIdx = 1
	m.focus = focusEndpoints
	m.epIdx = 0
	if _, cmd := m.openSelected(); cmd == nil {
		t.Fatal("expected a fetch command")
	}
	if m.state != stateFetching {
		t.Fatalf("expected stateFetching, got %v", m.state)
	}

	// Run the executor the way Bubble Tea would and feed the result back.
	msg, ok := m.fetchCmd()().(resultMsg)
	if !ok {
		t.Fatal("expected resultMsg")
	}
	if gotModule != "stats" || gotAction != "ethprice" {
		t.Fatalf("exec called with %s/%s", gotModule, gotAction)
	}
	m.setResult(msg.raw, msg.err)
	if m.state != stateResult {
		t.Fatalf("expected stateResult, got %v", m.state)
	}
	if !strings.Contains(m.vp.View(), "1000") {
		t.Fatalf("result view missing value: %q", m.vp.View())
	}
}

func TestAPIBackedEndpointPromptsForKeyAndResumes(t *testing.T) {
	called := false
	cfg := Config{
		Endpoints: []Endpoint{{Module: "stats", Action: "ethprice", Title: "ethprice"}},
		Exec: func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
			called = true
			return json.RawMessage(`{"ethusd":"1000"}`), nil
		},
		ChainName: "ethereum",
		ChainID:   "1",
		KeyLabel:  "none",
		SaveAPIKey: func(ctx context.Context, key string) (string, error) {
			if key != "TESTKEY" {
				t.Fatalf("unexpected key: %q", key)
			}
			return "TEST…TKEY", nil
		},
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.focus = focusEndpoints

	if _, cmd := m.openSelected(); cmd == nil {
		t.Fatal("expected key-input command")
	}
	if m.state != stateAPIKey || called {
		t.Fatalf("expected key setup without an API call, state=%v called=%v", m.state, called)
	}
	if !strings.Contains(m.View(), "keep exploring") {
		t.Fatalf("setup view does not explain cancellation:\n%s", m.View())
	}

	m.keyInput.SetValue("TESTKEY")
	if _, cmd := m.keyAPIKey(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil || !m.keySaving {
		t.Fatal("enter should begin asynchronous key validation")
	}
	label, err := m.cfg.SaveAPIKey(context.Background(), "TESTKEY")
	if err != nil {
		t.Fatal(err)
	}
	_, cmd := m.Update(apiKeySavedMsg{label: label})
	if !m.cfg.HasAPIKey || m.cfg.KeyLabel != "TEST…TKEY" {
		t.Fatalf("saved key state not reflected: has=%v label=%q", m.cfg.HasAPIKey, m.cfg.KeyLabel)
	}
	if m.state != stateFetching || cmd == nil {
		t.Fatalf("pending request did not resume: state=%v cmd=%v", m.state, cmd)
	}
	m.fetchCmd()()
	if !called {
		t.Fatal("resumed request did not call executor")
	}
}

func TestAPIKeyPromptCancelReturnsToExistingForm(t *testing.T) {
	cfg := Config{
		Endpoints: []Endpoint{{
			Module: "account", Action: "balance", Title: "balance",
			Params: []Param{{Name: "address", Label: "address", Required: true}},
		}},
		Exec:       func(context.Context, string, string, map[string]string) (json.RawMessage, error) { return nil, nil },
		ChainName:  "ethereum",
		ChainID:    "1",
		KeyLabel:   "none",
		SaveAPIKey: func(context.Context, string) (string, error) { return "", nil },
	}
	m := newModel(context.Background(), cfg)
	m.focus = focusEndpoints
	m.openSelected()
	m.inputs[0].SetValue("0x80f3950a4d371c43360f292a4170624abd9eed03")
	m.submitForm()
	if m.state != stateAPIKey || m.keyReturn != stateForm {
		t.Fatalf("expected key prompt returning to form, state=%v return=%v", m.state, m.keyReturn)
	}
	m.keyInput.SetValue("sensitive")
	m.keyAPIKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateForm {
		t.Fatalf("cancel should return to form, got %v", m.state)
	}
	if got := m.inputs[0].Value(); !strings.HasPrefix(got, "0x80f3") {
		t.Fatalf("form input was not preserved: %q", got)
	}
	if m.keyInput.Value() != "" {
		t.Fatal("cancel should clear key input")
	}
}

func TestBareEndpointRunsWithoutAPIKey(t *testing.T) {
	called := false
	cfg := Config{
		Endpoints: []Endpoint{{Module: "getapilimit", Action: "chainlist", Title: "chainlist", Bare: true}},
		Exec: func(context.Context, string, string, map[string]string) (json.RawMessage, error) {
			called = true
			return json.RawMessage(`[]`), nil
		},
		ChainName:  "ethereum",
		ChainID:    "1",
		KeyLabel:   "none",
		SaveAPIKey: func(context.Context, string) (string, error) { return "", nil },
	}
	m := newModel(context.Background(), cfg)
	m.focus = focusEndpoints
	if _, cmd := m.openSelected(); cmd == nil || m.state != stateFetching {
		t.Fatalf("bare endpoint should fetch directly, state=%v cmd=%v", m.state, cmd)
	}
	m.fetchCmd()()
	if !called {
		t.Fatal("bare endpoint did not call executor")
	}
}

func TestAPIKeyValidationErrorStaysInPrompt(t *testing.T) {
	cfg := Config{
		Endpoints:  []Endpoint{{Module: "stats", Action: "ethprice", Title: "ethprice"}},
		ChainName:  "ethereum",
		ChainID:    "1",
		KeyLabel:   "none",
		SaveAPIKey: func(context.Context, string) (string, error) { return "", errString("invalid API key") },
	}
	m := newModel(context.Background(), cfg)
	m.focus = focusEndpoints
	m.openSelected()
	m.Update(apiKeySavedMsg{err: errString("invalid API key")})
	if m.state != stateAPIKey || m.keySaving || !strings.Contains(m.keyErr, "invalid API key") {
		t.Fatalf("validation error not retained in setup: state=%v saving=%v err=%q", m.state, m.keySaving, m.keyErr)
	}
}

// TestGroupLabelDrivesSidebar: endpoints sharing a Group land in one sidebar
// group under the group label, while the exec call and result header keep the
// wire module.
func TestGroupLabelDrivesSidebar(t *testing.T) {
	var gotModule, gotAction string
	exec := func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		gotModule, gotAction = module, action
		return json.RawMessage(`"ok"`), nil
	}
	cfg := Config{
		Endpoints: []Endpoint{
			{Module: "account", Action: "balance", Title: "balance"},
			{Module: "getapilimit", Action: "getapilimit", Title: "getapilimit", Group: "usage"},
			{Module: "chainlist-wire", Action: "chainlist", Title: "chainlist", Group: "usage"},
		},
		Exec:      exec,
		ChainName: "ethereum",
		ChainID:   "1",
		KeyLabel:  "none",
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if len(m.modules) != 2 || m.modules[0] != "account" || m.modules[1] != "usage" {
		t.Fatalf("sidebar groups wrong: %v", m.modules)
	}
	if got := m.byModule["usage"]; len(got) != 2 {
		t.Fatalf("usage group should hold both endpoints, got %+v", got)
	}
	m.modIdx = 1
	if v := m.View(); !strings.Contains(v, "ENDPOINTS · USAGE") {
		t.Fatalf("endpoints header should show the group label, got:\n%s", v)
	}

	// Opening the first usage endpoint must call exec with the wire module.
	m.focus = focusEndpoints
	m.epIdx = 0
	m.openSelected()
	if msg, ok := m.fetchCmd()().(resultMsg); !ok || msg.err != nil {
		t.Fatalf("fetch failed: %+v", msg)
	}
	if gotModule != "getapilimit" || gotAction != "getapilimit" {
		t.Fatalf("exec called with %s/%s, want getapilimit/getapilimit", gotModule, gotAction)
	}
	if m.resultTitle != "getapilimit/getapilimit" {
		t.Fatalf("result title must keep the wire pair, got %q", m.resultTitle)
	}
}

func TestFormRequiredParamValidation(t *testing.T) {
	called := false
	exec := func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		called = true
		if params["address"] != "0xABC" {
			t.Fatalf("address param not forwarded: %v", params)
		}
		if _, ok := params["tag"]; ok {
			t.Fatalf("empty optional param must be omitted: %v", params)
		}
		return json.RawMessage(`[]`), nil
	}
	m := testModel(exec)

	// Open account/balance (module 0, endpoint 0) -> form with the required
	// address input and the optional tag input.
	m.modIdx = 0
	m.focus = focusEndpoints
	m.epIdx = 0
	m.openSelected()
	if m.state != stateForm || len(m.inputs) != 2 {
		t.Fatalf("expected form with 2 inputs, got state=%v inputs=%d", m.state, len(m.inputs))
	}

	// Submitting empty must fail with a form error and not fetch.
	m.submitForm()
	if m.formErr == "" {
		t.Fatal("expected form error on empty required field")
	}
	if m.state != stateForm {
		t.Fatal("should stay on form when required field empty")
	}

	// Fill only the required field and submit -> fetch without the optional.
	m.inputs[0].SetValue("0xABC")
	if _, cmd := m.submitForm(); cmd == nil {
		t.Fatal("expected a fetch command after valid submit")
	}
	if m.state != stateFetching {
		t.Fatalf("expected stateFetching, got %v", m.state)
	}
	m.fetchCmd()() // trigger exec
	if !called {
		t.Fatal("exec was not called")
	}
}

// TestFormOptionalParamsAndPager: the form shows optional params but never
// `page` on paginated endpoints (the n/p pager owns it); a user-set page size
// (offset) is forwarded and survives paging, and an empty one falls back to the
// default injected by fetchCmd.
func TestFormOptionalParamsAndPager(t *testing.T) {
	var got map[string]string
	exec := func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		got = params
		return json.RawMessage(`[]`), nil
	}
	cfg := Config{
		Endpoints: []Endpoint{{
			Module: "account", Action: "txlist", Title: "txlist",
			Params: []Param{
				{Name: "address", Label: "address", Required: true},
				{Name: "page", Label: "page number"},
				{Name: "offset", Label: "page size"},
				{Name: "sort", Label: "asc or desc"},
			},
			Paginated: true,
		}},
		Exec:      exec,
		ChainName: "ethereum", ChainID: "1", KeyLabel: "none",
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.focus = focusEndpoints
	m.openSelected()

	var names []string
	for _, pr := range m.inParams {
		names = append(names, pr.Name)
	}
	if len(names) != 3 || names[0] != "address" || names[1] != "offset" || names[2] != "sort" {
		t.Fatalf("form fields wrong (page must be excluded): %v", names)
	}
	if !strings.Contains(m.inputs[1].Placeholder, "default 25") {
		t.Fatalf("offset placeholder should note the default: %q", m.inputs[1].Placeholder)
	}
	// Fields are labeled by the API param name; the usage text stays in the
	// placeholder only.
	view := m.View()
	for _, name := range []string{"offset", "sort", "address"} {
		if !strings.Contains(view, name) {
			t.Fatalf("form must label fields by param name %q:\n%s", name, view)
		}
	}
	if !strings.Contains(view, "(optional)") {
		t.Fatalf("optional fields must be marked:\n%s", view)
	}

	m.inputs[0].SetValue("0xABC")
	m.inputs[1].SetValue("5")
	m.inputs[2].SetValue("desc")
	m.submitForm()
	m.fetchCmd()()
	if got["address"] != "0xABC" || got["offset"] != "5" || got["sort"] != "desc" || got["page"] != "1" {
		t.Fatalf("submitted params wrong: %v", got)
	}
	// Paging keeps the user's page size.
	m.page = 2
	m.fetchCmd()()
	if got["page"] != "2" || got["offset"] != "5" {
		t.Fatalf("paging lost user params: %v", got)
	}
	// An empty offset falls back to the injected default.
	m.openSelected()
	m.inputs[0].SetValue("0xABC")
	m.submitForm()
	m.fetchCmd()()
	if got["offset"] != "25" {
		t.Fatalf("default page size not injected: %v", got)
	}
}

// TestFormValidateHookInline: a Config.Validate error keeps the user in the form
// with the message inline and typed values intact; fixing the value submits.
func TestFormValidateHookInline(t *testing.T) {
	cfg := Config{
		Endpoints: []Endpoint{{
			Module: "account", Action: "txlist", Title: "txlist",
			Params: []Param{{Name: "address", Label: "address", Required: true}, {Name: "sort", Label: "asc or desc"}},
		}},
		Exec: func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
			return json.RawMessage(`[]`), nil
		},
		Validate: func(module, action string, params map[string]string) error {
			if s := params["sort"]; s != "" && s != "asc" && s != "desc" {
				return fmt.Errorf("sort must be asc or desc")
			}
			return nil
		},
		ChainName: "ethereum", ChainID: "1", KeyLabel: "none",
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.focus = focusEndpoints
	m.openSelected()

	m.inputs[0].SetValue("0xABC")
	m.inputs[1].SetValue("up")
	m.submitForm()
	if m.state != stateForm {
		t.Fatalf("validation error must keep the form, got state=%v", m.state)
	}
	if !strings.Contains(m.formErr, "sort must be asc or desc") {
		t.Fatalf("inline error missing: %q", m.formErr)
	}
	if m.inputs[1].Value() != "up" {
		t.Fatal("typed value lost on validation error")
	}

	m.inputs[1].SetValue("desc")
	if _, cmd := m.submitForm(); cmd == nil {
		t.Fatal("expected a fetch command after fixing the value")
	}
	if m.state != stateFetching || m.formErr != "" {
		t.Fatalf("expected clean fetch, got state=%v err=%q", m.state, m.formErr)
	}
}

func chainPickerModel(switchChain func(string) (string, string, error)) *model {
	cfg := Config{
		Endpoints: []Endpoint{{Module: "account", Action: "balance", Title: "balance"}},
		Exec: func(context.Context, string, string, map[string]string) (json.RawMessage, error) {
			return json.RawMessage(`[]`), nil
		},
		ChainName: "ethereum", ChainID: "1", KeyLabel: "none",
		Chains: []ChainInfo{
			{Name: "ethereum", DisplayName: "Ethereum Mainnet", ID: "1"},
			{Name: "polygon", DisplayName: "Polygon Mainnet", ID: "137", Aliases: []string{"matic", "pol"}},
			{Name: "sepolia", DisplayName: "Sepolia Testnet", ID: "11155111", Testnet: true},
		},
		SwitchChain: switchChain,
	}
	m := newModel(context.Background(), cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return &m
}

func typeRunes(m *model, s string) {
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// TestChainPickerSwitch: open the picker, filter, select — the SwitchChain callback
// receives the choice and the displayed chain updates, returning to browse.
func TestChainPickerSwitch(t *testing.T) {
	var gotArg string
	m := chainPickerModel(func(nameOrID string) (string, string, error) {
		gotArg = nameOrID
		return "Polygon Mainnet", "137", nil
	})

	typeRunes(m, "c") // open from browse
	if m.state != stateChainPicker {
		t.Fatalf("expected chain picker, got %v", m.state)
	}
	typeRunes(m, "polyg") // "pol" alone would also match sePOLia
	if got := m.filteredChains(); len(got) != 1 || got[0].Name != "polygon" {
		t.Fatalf("filter did not narrow to polygon: %+v", got)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if gotArg != "polygon" {
		t.Fatalf("SwitchChain called with %q, want polygon", gotArg)
	}
	if m.cfg.ChainName != "Polygon Mainnet" || m.cfg.ChainID != "137" {
		t.Fatalf("displayed chain not updated: %s (%s)", m.cfg.ChainName, m.cfg.ChainID)
	}
	if m.state != stateBrowse {
		t.Fatalf("expected return to browse after switch, got %v", m.state)
	}
}

// TestChainPickerFilterAndCancel: a non-matching filter empties the list and enter
// is a no-op; esc cancels without switching.
func TestChainPickerFilterAndCancel(t *testing.T) {
	called := false
	m := chainPickerModel(func(string) (string, string, error) { called = true; return "", "", nil })

	typeRunes(m, "c")
	typeRunes(m, "zzz")
	if len(m.filteredChains()) != 0 {
		t.Fatalf("expected no matches for 'zzz', got %d", len(m.filteredChains()))
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEnter}) // empty selection → no-op
	if called {
		t.Fatal("SwitchChain called on an empty selection")
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyBackspace}) // trims one char, still no match
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if called {
		t.Fatal("SwitchChain called after cancel")
	}
	if m.state != stateBrowse || m.chainFilter != "" {
		t.Fatalf("esc should restore browse and clear filter: state=%v filter=%q", m.state, m.chainFilter)
	}
}

func TestChainPickerCancelRestoresResult(t *testing.T) {
	m := chainPickerModel(func(string) (string, string, error) { return "", "", nil })
	m.state = stateResult

	m.openChainPicker()
	if m.state != stateChainPicker {
		t.Fatalf("expected chain picker, got %v", m.state)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateResult {
		t.Fatalf("cancel returned to %v, want result", m.state)
	}
}

func TestChainPickerAliasesAndErrorReset(t *testing.T) {
	m := chainPickerModel(func(string) (string, string, error) {
		return "", "", fmt.Errorf("switch failed")
	})

	m.openChainPicker()
	typeRunes(m, "matic")
	if got := m.filteredChains(); len(got) != 1 || got[0].Name != "polygon" {
		t.Fatalf("alias filter did not find polygon: %+v", got)
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.chainErr == "" {
		t.Fatal("expected switch error")
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	if m.chainErr != "" {
		t.Fatalf("navigation did not clear stale error: %q", m.chainErr)
	}

	m.chainErr = "switch failed"
	typeRunes(m, "x")
	if m.chainErr != "" {
		t.Fatalf("filter edit did not clear stale error: %q", m.chainErr)
	}

	m.chainFilter = "é"
	m.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.chainFilter != "" {
		t.Fatalf("backspace must remove one rune, got %q", m.chainFilter)
	}
}

func TestChainPickerOfficialNameAndPaidOnlyLabel(t *testing.T) {
	m := chainPickerModel(func(string) (string, string, error) { return "", "", nil })
	m.cfg.Chains[1].PaidOnly = true
	m.openChainPicker()
	m.chainFilter = "Polygon Mainnet"

	if got := m.filteredChains(); len(got) != 1 || got[0].Name != "polygon" {
		t.Fatalf("official display-name filter did not find polygon: %+v", got)
	}
	view := m.viewChainPicker()
	if !strings.Contains(view, "Polygon Mainnet (137)") || !strings.Contains(view, "(paid only)") {
		t.Fatalf("picker missing official name or tier label:\n%s", view)
	}
}

// TestChainPickerDisabledWithoutCallback: 'c' is inert when no SwitchChain is wired.
func TestChainPickerDisabledWithoutCallback(t *testing.T) {
	m := testModel(func(context.Context, string, string, map[string]string) (json.RawMessage, error) {
		return json.RawMessage(`[]`), nil
	})
	typeRunes(m, "c")
	if m.state == stateChainPicker {
		t.Fatal("picker opened despite nil SwitchChain")
	}
}

// TestFormViewFitsTerminal: a many-field form (getLogs-sized) must window its
// inputs to the terminal height — Bubble Tea trims overflow from the top, which
// would delete the header — while keeping the focused input visible.
func TestFormViewFitsTerminal(t *testing.T) {
	params := make([]Param, 14)
	for i := range params {
		params[i] = Param{Name: fmt.Sprintf("p%02d", i), Label: fmt.Sprintf("param %02d", i)}
	}
	params[0].Required = true
	cfg := Config{
		Endpoints: []Endpoint{{Module: "logs", Action: "getLogs", Title: "getLogs", Params: params}},
		ChainName: "ethereum", ChainID: "1", KeyLabel: "none",
	}
	m := newModel(context.Background(), cfg)
	m.focus = focusEndpoints
	m.openSelected()
	m.focusInput(7)
	for h := 15; h <= 50; h++ {
		m.Update(tea.WindowSizeMsg{Width: 100, Height: h})
		v := m.View()
		lines := strings.Split(v, "\n")
		if len(lines) > h {
			t.Fatalf("height %d: form view has %d lines", h, len(lines))
		}
		if !strings.Contains(lines[0], "Etherscan") {
			t.Fatalf("height %d: header not on first line: %q", h, lines[0])
		}
		if !strings.Contains(v, "p07") {
			t.Fatalf("height %d: focused field not visible", h)
		}
	}
}

func TestResultSurfacesError(t *testing.T) {
	m := testModel(nil)
	m.current = m.cfg.Endpoints[1]
	m.setResult(nil, errString("invalid api key"))
	if m.state != stateResult {
		t.Fatalf("expected stateResult, got %v", m.state)
	}
	if !strings.Contains(m.vp.View(), "invalid api key") {
		t.Fatalf("error not surfaced: %q", m.vp.View())
	}
}

func TestViewsDoNotPanic(t *testing.T) {
	m := testModel(func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		return json.RawMessage(`[]`), nil
	})
	for _, st := range []viewState{stateBrowse, stateForm, stateFetching, stateResult, stateAPIKey} {
		m.state = st
		if st == stateForm {
			m.modIdx = 0
			m.focus = focusEndpoints
			m.epIdx = 0
			m.openSelected()
		} else if st == stateAPIKey {
			m.openAPIKey()
		}
		_ = m.View()
	}
}

// blockRunes are the half-block glyphs the art is built from.
const blockRunes = "█▀▄"

func TestBannerArtSane(t *testing.T) {
	if len(bannerLogo) == 0 {
		t.Fatal("bannerLogo is empty")
	}
	if bannerWidth <= 0 {
		t.Fatalf("bannerWidth = %d, want > 0", bannerWidth)
	}
	// bannerWidth must be the true max row width, and no row may exceed it. Rows are
	// intentionally ragged-right (left-aligned block art), so widths need not match.
	max := 0
	for i, row := range bannerLogo {
		w := lipgloss.Width(row)
		if w > bannerWidth {
			t.Fatalf("row %d width %d exceeds bannerWidth %d", i, w, bannerWidth)
		}
		if w > max {
			max = w
		}
	}
	if max != bannerWidth {
		t.Fatalf("bannerWidth = %d, but widest row is %d", bannerWidth, max)
	}
}

func TestRenderBannerBigForm(t *testing.T) {
	out := renderBanner(bannerWidth+10, bigBannerRows)
	if !strings.ContainsAny(out, blockRunes) {
		t.Fatalf("big banner missing block art: %q", out)
	}
	if !strings.Contains(out, bannerTagline) {
		t.Fatalf("big banner missing tagline: %q", out)
	}
	if lines := strings.Count(out, "\n") + 1; lines != bigBannerRows {
		t.Fatalf("big banner has %d lines, want %d (bigBannerRows)", lines, bigBannerRows)
	}
}

func TestRenderBannerFallback(t *testing.T) {
	// Too narrow, and too small a row budget: both must drop to the compact
	// one-liner so the banner never pushes the panels (or header) off-screen.
	for _, tc := range []struct{ w, rows int }{
		{40, bigBannerRows + 10},              // narrow
		{bannerWidth + 10, bigBannerRows - 1}, // budget one row too small
		{bannerWidth + 10, 0},                 // no budget at all
	} {
		out := renderBanner(tc.w, tc.rows)
		if strings.ContainsAny(out, blockRunes) {
			t.Fatalf("fallback (w=%d rows=%d) should not contain block art: %q", tc.w, tc.rows, out)
		}
		if !strings.Contains(out, "Etherscan") || !strings.Contains(out, bannerTagline) {
			t.Fatalf("fallback (w=%d rows=%d) missing wordmark/tagline: %q", tc.w, tc.rows, out)
		}
		if lines := strings.Count(out, "\n") + 1; lines != 1 {
			t.Fatalf("compact banner should be one line, got %d", lines)
		}
	}
}

// TestBrowseViewFitsTerminal guards the header-trimmed-off-screen bug: the browse
// view must never render more lines than the terminal has, at any height, because
// Bubble Tea trims an overflowing view from the top — deleting the header first.
func TestBrowseViewFitsTerminal(t *testing.T) {
	// One big module (mirrors account's 19 endpoints) and one small one.
	eps := make([]Endpoint, 0, 20)
	for i := 0; i < 19; i++ {
		eps = append(eps, Endpoint{Module: "account", Action: fmt.Sprintf("action%02d", i),
			Title: fmt.Sprintf("action%02d", i), Desc: "desc"})
	}
	eps = append(eps, Endpoint{Module: "stats", Action: "ethprice", Title: "ethprice"})
	m := newModel(context.Background(), Config{Endpoints: eps, ChainName: "ethereum", ChainID: "1", KeyLabel: "none"})

	for _, h := range []int{15, 20, 25, 30, 34, 37, 40, 50} {
		m.Update(tea.WindowSizeMsg{Width: 120, Height: h})
		out := m.View()
		if lines := strings.Count(out, "\n") + 1; lines > h {
			t.Fatalf("height %d: view is %d lines — header would be trimmed off-screen", h, lines)
		}
		first := strings.SplitN(out, "\n", 2)[0]
		if !strings.Contains(first, "Etherscan") {
			t.Fatalf("height %d: first line is not the header: %q", h, first)
		}
	}
}

// TestBrowseWindowKeepsSelectionVisible: with more endpoints than fit, the list is
// windowed around the selection and truncation is marked.
func TestBrowseWindowKeepsSelectionVisible(t *testing.T) {
	eps := make([]Endpoint, 0, 19)
	for i := 0; i < 19; i++ {
		eps = append(eps, Endpoint{Module: "account", Action: fmt.Sprintf("action%02d", i),
			Title: fmt.Sprintf("action%02d", i), Desc: "desc"})
	}
	m := newModel(context.Background(), Config{Endpoints: eps, ChainName: "ethereum", ChainID: "1", KeyLabel: "none"})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	m.focus = focusEndpoints
	m.epIdx = 18
	out := m.View()
	if !strings.Contains(out, "action18") {
		t.Fatal("selected endpoint not visible in windowed list")
	}
	if !strings.Contains(out, "above") {
		t.Fatal("expected an '… N above' truncation marker")
	}
	if lines := strings.Count(out, "\n") + 1; lines > 20 {
		t.Fatalf("view is %d lines, exceeds terminal height 20", lines)
	}
}

// TestBannerStableAcrossModules guards the "logo disappears on some modules" bug:
// the banner must not change form when moving the selection between a long module
// (account, 19 endpoints) and a short one (stats) — the logo has priority and the
// lists window themselves into the remaining space instead.
func TestBannerStableAcrossModules(t *testing.T) {
	var eps []Endpoint
	for i := 0; i < 19; i++ {
		eps = append(eps, Endpoint{Module: "account", Action: fmt.Sprintf("a%02d", i), Title: fmt.Sprintf("a%02d", i)})
	}
	for i := 0; i < 8; i++ {
		eps = append(eps, Endpoint{Module: "stats", Action: fmt.Sprintf("s%02d", i), Title: fmt.Sprintf("s%02d", i)})
	}
	for _, h := range []int{20, 26, 30, 36, 44} {
		m := newModel(context.Background(), Config{Endpoints: eps, ChainName: "ethereum", ChainID: "1", KeyLabel: "none"})
		m.Update(tea.WindowSizeMsg{Width: 120, Height: h})
		m.modIdx = 0 // account
		accountArt := strings.ContainsAny(m.View(), blockRunes)
		m.modIdx = 1 // stats
		statsArt := strings.ContainsAny(m.View(), blockRunes)
		if accountArt != statsArt {
			t.Fatalf("height %d: banner form differs between modules (account art=%v, stats art=%v)", h, accountArt, statsArt)
		}
		if h >= 26 && !statsArt {
			t.Fatalf("height %d: expected the big logo (fits with windowed lists)", h)
		}
		for _, out := range []string{m.View()} {
			if lines := strings.Count(out, "\n") + 1; lines > h {
				t.Fatalf("height %d: view is %d lines", h, lines)
			}
		}
	}
}

func TestWindowIndices(t *testing.T) {
	for _, tc := range []struct{ total, visible, idx, wantStart, wantEnd int }{
		{5, 10, 0, 0, 5},    // fits entirely
		{19, 10, 0, 0, 10},  // top
		{19, 10, 18, 9, 19}, // bottom
		{19, 10, 9, 4, 14},  // centred
	} {
		start, end := windowIndices(tc.total, tc.visible, tc.idx)
		if start != tc.wantStart || end != tc.wantEnd {
			t.Fatalf("windowIndices(%d,%d,%d) = %d,%d want %d,%d",
				tc.total, tc.visible, tc.idx, start, end, tc.wantStart, tc.wantEnd)
		}
		if tc.idx < start || tc.idx >= end {
			t.Fatalf("selection %d outside window [%d,%d)", tc.idx, start, end)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
