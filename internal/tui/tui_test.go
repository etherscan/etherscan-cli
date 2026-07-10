package tui

import (
	"context"
	"encoding/json"
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
	out := renderBanner(bannerWidth+10, minLandingHeight+10)
	if !strings.ContainsAny(out, blockRunes) {
		t.Fatalf("big banner missing block art: %q", out)
	}
	if !strings.Contains(out, bannerTagline) {
		t.Fatalf("big banner missing tagline: %q", out)
	}
	if lines := strings.Count(out, "\n") + 1; lines < bannerHeight {
		t.Fatalf("big banner has %d lines, want >= %d", lines, bannerHeight)
	}
}

func TestRenderBannerFallback(t *testing.T) {
	// Too narrow, and too short: both must drop to the compact one-liner so the
	// landing is never blank and the panels stay on screen.
	for _, tc := range []struct{ w, h int }{
		{40, minLandingHeight + 10},              // narrow
		{bannerWidth + 10, minLandingHeight - 1}, // short
	} {
		out := renderBanner(tc.w, tc.h)
		if strings.ContainsAny(out, blockRunes) {
			t.Fatalf("fallback (w=%d h=%d) should not contain block art: %q", tc.w, tc.h, out)
		}
		if !strings.Contains(out, "Etherscan") || !strings.Contains(out, bannerTagline) {
			t.Fatalf("fallback (w=%d h=%d) missing wordmark/tagline: %q", tc.w, tc.h, out)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
