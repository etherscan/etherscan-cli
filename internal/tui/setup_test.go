package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestSetupEmptySubmitShowsError(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{Save: func(ctx context.Context, key string) error {
		t.Fatal("Save must not be called on empty submit")
		return nil
	}})
	m.Update(keyMsg("enter"))
	if m.errMsg == "" {
		t.Fatal("expected an error message on empty submit")
	}
	if m.saving || m.saved {
		t.Fatalf("unexpected state: saving=%v saved=%v", m.saving, m.saved)
	}
}

func TestSetupSaveCalledWithTrimmedKey(t *testing.T) {
	var got string
	m := newSetupModel(context.Background(), SetupConfig{Save: func(ctx context.Context, key string) error {
		got = key
		return nil
	}})
	m.input.SetValue("  MYKEY  ")
	_, cmd := m.Update(keyMsg("enter"))
	if !m.saving {
		t.Fatal("expected saving state after submit")
	}
	if cmd == nil {
		t.Fatal("expected a save command")
	}
	// Run the batched commands the way Bubble Tea would; one yields setupSaveMsg.
	runSetupCmd(t, &m, cmd)
	if got != "MYKEY" {
		t.Fatalf("Save called with %q, want trimmed MYKEY", got)
	}
	if !m.saved || m.saving {
		t.Fatalf("expected saved state, got saving=%v saved=%v", m.saving, m.saved)
	}
}

func TestSetupSaveErrorReturnsToInput(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{})
	m.saving = true
	m.Update(setupSaveMsg{err: errString("API key validation failed")})
	if m.saving || m.saved {
		t.Fatalf("unexpected state: saving=%v saved=%v", m.saving, m.saved)
	}
	if !strings.Contains(m.errMsg, "validation failed") {
		t.Fatalf("save error not surfaced: %q", m.errMsg)
	}
	if !strings.Contains(m.View(), "validation failed") {
		t.Fatal("error message missing from view")
	}
}

func TestSetupSaveSuccessQuits(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{})
	m.saving = true
	_, cmd := m.Update(setupSaveMsg{})
	if !m.saved {
		t.Fatal("expected saved=true")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.Quit after successful save")
	}
}

func TestSetupEscQuitsUnsaved(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{})
	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.Quit on esc")
	}
	if m.saved {
		t.Fatal("esc must not mark the model saved (RunSetup maps this to ErrSetupAborted)")
	}
}

func TestSetupTypingReachesInput(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{})
	m.Update(keyMsg("A"))
	m.Update(keyMsg("B"))
	if m.input.Value() != "AB" {
		t.Fatalf("typed runes not in input: %q", m.input.Value())
	}
}

func TestSetupViewDoesNotPanic(t *testing.T) {
	m := newSetupModel(context.Background(), SetupConfig{})
	_ = m.View()
	m.saving = true
	_ = m.View()
	m.saving = false
	m.errMsg = "boom"
	if !strings.Contains(m.View(), "boom") {
		t.Fatal("view missing error message")
	}
}

// runSetupCmd executes a command tree (Batch or single) synchronously and feeds
// the resulting setupSaveMsg back into the model. Spinner ticks are dropped:
// feeding them back would schedule ticks forever.
func runSetupCmd(t *testing.T, m *setupModel, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	switch v := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range v {
			runSetupCmd(t, m, c)
		}
	case setupSaveMsg:
		m.Update(v)
	}
}
