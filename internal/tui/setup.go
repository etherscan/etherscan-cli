package tui

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrSetupAborted is returned by RunSetup when the user quits without saving a key.
var ErrSetupAborted = errors.New("setup aborted")

// SetupConfig wires the first-launch key screen to the caller's validate+persist
// logic. Save receives the entered key and returns nil once it is stored.
type SetupConfig struct {
	Save func(ctx context.Context, key string) error
}

// RunSetup shows the first-launch API-key screen and blocks until a key is saved
// (nil) or the user quits without one (ErrSetupAborted).
func RunSetup(ctx context.Context, cfg SetupConfig) error {
	m := newSetupModel(ctx, cfg)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	out, err := p.Run()
	if err != nil {
		return err
	}
	if sm, ok := out.(*setupModel); ok && sm.saved {
		return nil
	}
	return ErrSetupAborted
}

type setupSaveMsg struct{ err error }

type setupModel struct {
	ctx    context.Context
	cfg    SetupConfig
	input  textinput.Model
	spin   spinner.Model
	saving bool
	saved  bool
	errMsg string
}

func newSetupModel(ctx context.Context, cfg SetupConfig) setupModel {
	ti := textinput.New()
	ti.Placeholder = "paste your API key"
	ti.Prompt = "› "
	ti.Focus()
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)
	return setupModel{ctx: ctx, cfg: cfg, input: ti, spin: sp}
}

func (m setupModel) Init() tea.Cmd { return textinput.Blink }

func (m *setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case spinner.TickMsg:
		if m.saving {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case setupSaveMsg:
		m.saving = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.input.Focus()
			return m, textinput.Blink
		}
		m.saved = true
		return m, tea.Quit

	case tea.KeyMsg:
		if m.saving {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			key := strings.TrimSpace(m.input.Value())
			if key == "" {
				m.errMsg = "API key is required"
				return m, nil
			}
			m.errMsg = ""
			m.saving = true
			m.input.Blur()
			save, ctx := m.cfg.Save, m.ctx
			return m, tea.Batch(m.spin.Tick, func() tea.Msg {
				return setupSaveMsg{err: save(ctx, key)}
			})
		}
	}
	// Everything else (typed characters, textinput's paste msg) goes to the input.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m setupModel) View() string {
	var b strings.Builder
	b.WriteString(titleSt.Render("◆ Etherscan") + "\n\n")
	b.WriteString(headSt.Render("Set up your API key") + "\n")
	b.WriteString(subSt.Render("An API key is required. Get a free one at https://etherscan.io/apis") + "\n\n")
	if m.saving {
		b.WriteString(m.spin.View() + " validating key…" + "\n")
	} else {
		b.WriteString(m.input.View() + "\n")
		if m.errMsg != "" {
			b.WriteString("\n" + errSt.Render(m.errMsg) + "\n")
		}
	}
	b.WriteString("\n" + footerSt.Render("enter save · esc quit") + "\n")
	b.WriteString(subSt.Render("Prefer the shell? Run 'etherscan login' or set ETHERSCAN_API_KEY."))
	return b.String()
}
