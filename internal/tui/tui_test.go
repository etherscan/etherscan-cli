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
				Params: []Param{{Name: "address", Label: "address", Required: true}}},
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

func TestFormRequiredParamValidation(t *testing.T) {
	called := false
	exec := func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error) {
		called = true
		if params["address"] != "0xABC" {
			t.Fatalf("address param not forwarded: %v", params)
		}
		return json.RawMessage(`[]`), nil
	}
	m := testModel(exec)

	// Open account/balance (module 0, endpoint 0) -> form with one required input.
	m.modIdx = 0
	m.focus = focusEndpoints
	m.epIdx = 0
	m.openSelected()
	if m.state != stateForm || len(m.inputs) != 1 {
		t.Fatalf("expected form with 1 input, got state=%v inputs=%d", m.state, len(m.inputs))
	}

	// Submitting empty must fail with a form error and not fetch.
	m.submitForm()
	if m.formErr == "" {
		t.Fatal("expected form error on empty required field")
	}
	if m.state != stateForm {
		t.Fatal("should stay on form when required field empty")
	}

	// Fill the field and submit -> fetch.
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
	for _, st := range []viewState{stateBrowse, stateForm, stateFetching, stateResult} {
		m.state = st
		if st == stateForm {
			m.modIdx = 0
			m.focus = focusEndpoints
			m.epIdx = 0
			m.openSelected()
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
