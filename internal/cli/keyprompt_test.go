package cli

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/etherscan/etherscan-cli/internal/brand"
)

func testKeyPrompt(t *testing.T, save func(context.Context, string) (string, error)) keyPromptModel {
	t.Helper()
	if save == nil {
		save = func(context.Context, string) (string, error) {
			t.Fatal("save must not be called")
			return "", nil
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return newKeyPromptModel(ctx, cancel, io.Discard, save)
}

// step drives one message through the model the way Bubble Tea would.
func step(t *testing.T, m keyPromptModel, msg tea.Msg) (keyPromptModel, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	got, ok := next.(keyPromptModel)
	if !ok {
		t.Fatalf("Update returned %T, want keyPromptModel", next)
	}
	return got, cmd
}

// runCmd executes a command the way the runtime does, flattening one level of
// tea.Batch, and returns every message produced. Tests must go through this
// rather than reimplementing what the command does, or they end up asserting on
// their own arithmetic instead of the model's.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	out := make([]tea.Msg, 0, len(batch))
	for _, c := range batch {
		if c != nil {
			out = append(out, c())
		}
	}
	return out
}

// savedMsg picks the save outcome out of a command's messages.
func savedMsg(t *testing.T, cmd tea.Cmd) keyPromptSavedMsg {
	t.Helper()
	for _, msg := range runCmd(cmd) {
		if s, ok := msg.(keyPromptSavedMsg); ok {
			return s
		}
	}
	t.Fatal("command produced no keyPromptSavedMsg")
	return keyPromptSavedMsg{}
}

// The whole point of the prompt: the key must never be legible on screen.
func TestKeyPromptMasksInput(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.input.SetValue("SuperSecretApiKey")

	view := m.View()
	if strings.Contains(view, "SuperSecretApiKey") {
		t.Errorf("view leaked the API key:\n%s", view)
	}
	if !strings.Contains(view, strings.Repeat("*", len("SuperSecretApiKey"))) {
		t.Errorf("view should mask the key with asterisks:\n%s", view)
	}
}

func TestKeyPromptSubmitStartsValidation(t *testing.T) {
	var gotKey string
	m := testKeyPrompt(t, func(_ context.Context, key string) (string, error) {
		gotKey = key
		return "abc***xyz", nil
	})
	// Surrounding whitespace from a sloppy paste is trimmed before validation.
	m.input.SetValue("  MyApiKey123  ")

	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.saving {
		t.Fatal("enter should begin validation")
	}
	// Run the command the runtime would run. The key reaching save must be the
	// one the model trimmed, not one the test trimmed for it.
	saved := savedMsg(t, cmd)
	if gotKey != "MyApiKey123" {
		t.Errorf("save received %q; want the trimmed key", gotKey)
	}
	if saved.err != nil || saved.label != "abc***xyz" {
		t.Errorf("save outcome = %+v", saved)
	}

	m, cmd = step(t, m, keyPromptSavedMsg{label: "abc***xyz"})
	if !m.done || m.canceled || m.label != "abc***xyz" {
		t.Fatalf("success not recorded: done=%v canceled=%v label=%q", m.done, m.canceled, m.label)
	}
	if cmd == nil {
		t.Error("a successful save should quit the program")
	}
}

func TestKeyPromptEmptySubmitStaysAndWarns(t *testing.T) {
	m := testKeyPrompt(t, nil)

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.saving || m.done {
		t.Fatal("an empty submit must not start validation")
	}
	if m.errMsg == "" {
		t.Fatal("an empty submit should explain itself")
	}
	if !strings.Contains(m.View(), m.errMsg) {
		t.Error("the error should be visible in the view")
	}
}

// A rejected key keeps the prompt open so it can be corrected in place.
func TestKeyPromptValidationErrorStaysOpen(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.saving = true

	m, _ = step(t, m, keyPromptSavedMsg{err: errors.New("API key validation failed")})
	if m.done || m.canceled {
		t.Fatal("a failed save must not end the prompt")
	}
	if m.saving {
		t.Error("saving flag should clear after a failure")
	}
	if !strings.Contains(m.View(), "API key validation failed") {
		t.Errorf("the failure should be shown:\n%s", m.View())
	}
}

func TestKeyPromptCancelKeys(t *testing.T) {
	for name, key := range map[string]tea.KeyMsg{
		"esc":    {Type: tea.KeyEsc},
		"ctrl+c": {Type: tea.KeyCtrlC},
	} {
		t.Run(name, func(t *testing.T) {
			m := testKeyPrompt(t, nil)
			m.input.SetValue("PartiallyTyped")

			m, cmd := step(t, m, key)
			if !m.canceled || m.done {
				t.Fatalf("%s should cancel: canceled=%v done=%v", name, m.canceled, m.done)
			}
			if m.input.Value() != "" {
				t.Error("cancelling should clear the typed key")
			}
			if cmd == nil {
				t.Error("cancelling should quit the program")
			}
		})
	}
}

// While a key is being validated, stray keystrokes must not race the request —
// only ctrl+c gets through.
func TestKeyPromptIgnoresKeysWhileSaving(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.saving = true
	m.input.SetValue("InFlightKey")

	after, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil || after.input.Value() != "InFlightKey" {
		t.Errorf("keystrokes during validation should be dropped, got value %q", after.input.Value())
	}
	if after.done || after.canceled {
		t.Error("a stray keystroke should not end the prompt")
	}

	after, cmd = step(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if !after.canceled || cmd == nil {
		t.Error("ctrl+c must still abort an in-flight validation")
	}
}

// The branded header stays on screen after the prompt exits so whatever login
// prints next is framed by it.
func TestKeyPromptFinalFrameKeepsHeader(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.done = true

	view := m.View()
	if !strings.Contains(view, "Etherscan CLI") {
		t.Errorf("final frame should keep the branded header:\n%s", view)
	}
	if strings.Contains(view, keyPromptTitle) || strings.Contains(stripANSI(view), plainHints(keyPromptHints)) {
		t.Errorf("final frame should drop the input chrome:\n%s", view)
	}
}

// plainHints is the un-styled form of a hint row, for assertions.
func plainHints(hints []keyHint) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, "["+h.key+"] "+h.action)
	}
	return strings.Join(parts, hintSep)
}

// railed returns the rendered rows that carry the grey rail, with the rail and
// its padding stripped, plus the unrailed rows.
func railed(view string) (inside, outside []string) {
	for _, line := range strings.Split(view, "\n") {
		plain := strings.TrimRight(stripANSI(line), " ")
		if rest, ok := strings.CutPrefix(plain, "│"); ok {
			inside = append(inside, strings.TrimSpace(rest))
			continue
		}
		if plain != "" {
			outside = append(outside, strings.TrimSpace(plain))
		}
	}
	return inside, outside
}

// The rail brackets only what the user is being asked to act on: it starts at
// the title and ends at the input. The branded header and the key hints sit
// outside it.
func TestKeyPromptRailCoversBodyOnly(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.input.SetValue("ApiKey123")

	inside, outside := railed(m.View())
	if len(inside) == 0 {
		t.Fatal("no railed rows")
	}
	if inside[0] != keyPromptTitle {
		t.Errorf("rail should start at %q, got %q", keyPromptTitle, inside[0])
	}
	if last := inside[len(inside)-1]; !strings.Contains(last, "*") {
		t.Errorf("rail should end at the input row, got %q", last)
	}
	for _, row := range inside {
		if strings.Contains(row, plainHints(keyPromptHints)) || strings.Contains(row, "Etherscan CLI") {
			t.Errorf("header/footer must stay outside the rail, found %q", row)
		}
	}
	if len(outside) != 2 || !strings.Contains(outside[0], "Etherscan CLI") || outside[1] != plainHints(keyPromptHints) {
		t.Errorf("expected an unrailed header and footer, got %q", outside)
	}
}

// A validation error belongs to the input, so it extends the rail rather than
// breaking out of it.
func TestKeyPromptRailIncludesError(t *testing.T) {
	m := testKeyPrompt(t, nil)
	m.errMsg = "API key validation failed"

	inside, _ := railed(m.View())
	if len(inside) == 0 || inside[len(inside)-1] != m.errMsg {
		t.Errorf("the error should be the last railed row, got %q", inside)
	}
}

// While validating, the hints must not advertise keys that are being ignored.
func TestKeyPromptFooterTracksState(t *testing.T) {
	m := testKeyPrompt(t, nil)
	footer := func(m keyPromptModel) string {
		t.Helper()
		_, outside := railed(m.View())
		if len(outside) == 0 {
			t.Fatal("no unrailed rows; the footer should be one of them")
		}
		return outside[len(outside)-1]
	}
	if got, want := footer(m), plainHints(keyPromptHints); got != want {
		t.Errorf("idle footer = %q; want %q", got, want)
	}
	m.saving = true
	if got, want := footer(m), plainHints(keyPromptHintsSaving); got != want {
		t.Errorf("saving footer = %q; want %q", got, want)
	}
}

// The keys must be visually distinct from their labels, not just bracketed.
//
// Colour cannot be asserted directly: under `go test` lipgloss resolves to the
// Ascii profile and strips every escape, including bold. Instead the key style
// is swapped for a marker, which proves renderHints routes the key — and only
// the key — through it.
func TestKeyPromptHintKeysAreAccented(t *testing.T) {
	m := testKeyPrompt(t, nil)
	if got := m.renderHints(keyPromptHints); got != "[enter] save · [esc] cancel" {
		t.Errorf("hint text = %q", got)
	}

	m.hintKey = m.hintKey.Transform(func(s string) string { return "<key>" + s + "</key>" })
	got := m.renderHints([]keyHint{{"enter", "save"}})
	if want := "[<key>enter</key>] save"; got != want {
		t.Errorf("renderHints = %q; want %q — the key style must wrap the key alone", got, want)
	}

	// And the style it uses is the brand accent, so the marker above is standing
	// in for something actually visible.
	if !m.hintKey.GetBold() {
		t.Error("hint keys should be bold")
	}
	if got, want := m.hintKey.GetForeground(), lipgloss.Color(brand.AccentHex); got != want {
		t.Errorf("hint key colour = %v; want the brand accent %v", got, want)
	}
	if m.sub.GetForeground() == m.hintKey.GetForeground() {
		t.Error("keys and labels must not share a colour, or the brackets are the only cue")
	}
}

// The final frame is the bare branded header: no rail, no input chrome.
func TestKeyPromptFinalFrameHasNoRail(t *testing.T) {
	for _, done := range []bool{true, false} {
		m := testKeyPrompt(t, nil)
		m.done, m.canceled = done, !done
		if inside, _ := railed(m.View()); len(inside) != 0 {
			t.Errorf("done=%v: final frame should carry no rail, got %q", done, inside)
		}
	}
}

// On a narrow terminal the descriptive lines must wrap inside the rail rather
// than spilling past it, and the input must not be re-wrapped (its cursor
// escapes would be split).
func TestKeyPromptWrapsNarrowTerminal(t *testing.T) {
	m := testKeyPrompt(t, nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = next.(keyPromptModel)
	m.input.SetValue("ApiKey123")

	// The input gets every column the rail and the "› " prompt leave behind.
	// Measured in cells, not bytes: "› " is 4 bytes but 2 cells, and using len
	// here would silently accept an input sized 2 columns short.
	if want := 40 - keyPromptGutter - lipgloss.Width(m.input.Prompt); m.input.Width != want {
		t.Errorf("input width = %d; want %d", m.input.Width, want)
	}

	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if w := visibleWidth(strings.TrimRight(stripANSI(line), " ")); w > 40 {
			t.Errorf("row exceeds the terminal width: %q (%d cols)", stripANSI(line), w)
		}
	}
	if inside, _ := railed(view); len(inside) == 0 {
		t.Error("the rail should survive a narrow layout")
	}
	if !strings.Contains(stripANSI(view), strings.Repeat("*", len("ApiKey123"))) {
		t.Error("masked input should survive the narrow layout intact")
	}
}

// Backing out while a key is being validated must kill the in-flight request.
// Without this the save goroutine keeps a live context, finishes validating, and
// writes the key to disk while login reports "cancelled" — the user would end up
// with a stored key they were told was not saved.
func TestKeyPromptCancelAbortsInFlightSave(t *testing.T) {
	for name, key := range map[string]tea.KeyMsg{
		"ctrl+c while saving": {Type: tea.KeyCtrlC},
		"esc while idle":      {Type: tea.KeyEsc},
	} {
		t.Run(name, func(t *testing.T) {
			m := testKeyPrompt(t, nil)
			m.saving = key.Type == tea.KeyCtrlC

			if err := m.ctx.Err(); err != nil {
				t.Fatalf("context already cancelled before the test acted: %v", err)
			}
			after, cmd := step(t, m, key)
			if !after.canceled || cmd == nil {
				t.Fatalf("%s should cancel and quit", name)
			}
			if err := m.ctx.Err(); err == nil {
				t.Error("cancelling must abort the context the save closure runs against")
			}
		})
	}
}
