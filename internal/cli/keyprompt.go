package cli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/etherscan/etherscan-cli/internal/brand"
)

// errKeyPromptCanceled is returned when the user dismisses a key prompt with esc
// or ctrl+c. Both input paths — this prompt and readSecret's plain fallback —
// return it, so cancelling surfaces identically through main's "error: %v"
// handler whichever one ran.
var errKeyPromptCanceled = errors.New("cancelled")

const (
	keyPromptHeadline = "Etherscan API V2"
	keyPromptTitle    = "Connect your API key"
	keyPromptHint     = "Paste your Etherscan API key to save it for future commands."
	keyPromptGetKey   = "Get a free key at https://etherscan.io/api/pricing"

	// keyPromptGutter is the width the rail itself occupies: one border column
	// plus one column of padding.
	keyPromptGutter = 2
)

// keyHint is one "[key] action" pair in the footer.
type keyHint struct{ key, action string }

// hintSep separates footer hints. Matches the TUI's (internal/tui, renderHints).
const hintSep = " · "

// Footer hints. While a key is being validated only ctrl+c is honoured, so the
// hints change rather than advertising keys that are being ignored.
var (
	keyPromptHints       = []keyHint{{"enter", "save"}, {"esc", "cancel"}}
	keyPromptHintsSaving = []keyHint{{"ctrl+c", "cancel"}}
)

// keyPromptSavedMsg carries the outcome of the injected save closure back into
// the event loop.
type keyPromptSavedMsg struct {
	label string
	err   error
}

// keyPromptModel is the branded API-key prompt used by `etherscan login`. It is
// deliberately rendered inline (no alt screen) so the final frame stays in the
// user's scrollback, and it mirrors the look of the TUI's just-in-time key
// prompt (internal/tui, viewAPIKey) so the two surfaces feel like one product.
type keyPromptModel struct {
	ctx context.Context
	// cancel aborts ctx. It is called synchronously on the cancel paths, before
	// tea.Quit, so an in-flight validation is killed while the program is still
	// running rather than after Run returns. See runKeyPrompt.
	cancel context.CancelFunc
	save   func(context.Context, string) (string, error)
	input  textinput.Model
	spin   spinner.Model

	head    lipgloss.Style
	sub     lipgloss.Style
	bad     lipgloss.Style
	rail    lipgloss.Style
	hintKey lipgloss.Style

	width    int
	saving   bool
	errMsg   string
	label    string
	done     bool
	canceled bool
}

// newKeyPromptModel builds the prompt. cancel must abort ctx: it is what stops an
// in-flight validation when the user backs out mid-save.
func newKeyPromptModel(ctx context.Context, cancel context.CancelFunc, out io.Writer, save func(context.Context, string) (string, error)) keyPromptModel {
	// Styles are bound to the prompt's own writer (stderr). lipgloss's package
	// default resolves its colour profile against stdout, which would silently
	// drop colour whenever stdout is redirected but the terminal is still there.
	r := lipgloss.NewRenderer(out)

	ti := textinput.New()
	ti.Placeholder = "paste your API key"
	ti.Prompt = "› "
	// Mask the key for the same reason the TUI does: it would otherwise sit in
	// scrollback and in any screenshot. The value is validated against the API
	// before being saved, so it never needs to be read back.
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = r.NewStyle().Foreground(lipgloss.Color(brand.AccentHex))

	return keyPromptModel{
		ctx:     ctx,
		cancel:  cancel,
		save:    save,
		input:   ti,
		spin:    sp,
		head:    r.NewStyle().Bold(true).Foreground(lipgloss.Color(brand.AccentHex)),
		sub:     r.NewStyle().Foreground(lipgloss.Color(brand.DimHex)),
		bad:     r.NewStyle().Foreground(lipgloss.Color(brand.ErrorHex)),
		hintKey: r.NewStyle().Bold(true).Foreground(lipgloss.Color(brand.AccentHex)),
		// A grey rail down the left edge groups the prompt into one visual block
		// and separates it from surrounding shell output, which matters because
		// this renders inline rather than on an alt screen.
		rail: r.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(brand.DimHex)).
			PaddingLeft(1),
	}
}

func (m keyPromptModel) Init() tea.Cmd { return textinput.Blink }

func (m keyPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Track the width so the descriptive lines wrap inside the rail and the
		// input scrolls horizontally instead of spilling past the right edge.
		// Unlike the full-screen explorer the prompt never blocks on this: an
		// unknown width just means "do not wrap".
		m.width = msg.Width
		// lipgloss.Width, not len: the "› " prompt is 4 bytes but 2 cells, and
		// byte length would size the input 2 columns short on every terminal.
		if inner := msg.Width - keyPromptGutter - lipgloss.Width(m.input.Prompt); inner > 0 {
			m.input.Width = inner
		}
		return m, nil

	case keyPromptSavedMsg:
		m.saving = false
		if msg.err != nil {
			// Stay in the prompt so a mistyped key can be corrected without
			// re-running the command.
			m.errMsg = msg.err.Error()
			return m, m.input.Focus()
		}
		m.label = msg.label
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		if !m.saving {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.saving {
			// Only ctrl+c is honoured mid-validation; everything else would race
			// the in-flight request.
			if msg.String() == "ctrl+c" {
				return m.abort()
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return m.abort()
		case "enter":
			key := strings.TrimSpace(m.input.Value())
			if key == "" {
				m.errMsg = "API key is required"
				return m, nil
			}
			m.errMsg = ""
			m.saving = true
			m.input.Blur()
			save, ctx := m.save, m.ctx
			return m, tea.Batch(m.spin.Tick, func() tea.Msg {
				label, err := save(ctx, key)
				return keyPromptSavedMsg{label: label, err: err}
			})
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// abort clears the typed key and ends the prompt. Cancelling the context here
// rather than letting runKeyPrompt's deferred cancel do it is what stops a
// validation that is already in flight: the HTTP request fails immediately, so
// the save closure returns an error long before it would reach config.Save.
func (m keyPromptModel) abort() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}
	m.input.SetValue("")
	m.canceled = true
	return m, tea.Quit
}

func (m keyPromptModel) View() string {
	// The header is kept on screen after the program exits so the branded line
	// frames whatever login prints next. It and the footer sit outside the rail,
	// which brackets only the part the user is being asked to act on. Both are
	// indented by the rail's own width so their text sits in the same column as
	// the railed text.
	indent := strings.Repeat(" ", keyPromptGutter)
	header := "\n" + indent + m.head.Render(quickStartBullet+" Etherscan CLI") + "  " + m.sub.Render("·  "+keyPromptHeadline) + "\n"

	if m.done || m.canceled {
		return header + "\n"
	}

	// The two descriptive lines are the only long ones; wrapping just those keeps
	// the rail intact on a narrow terminal without lipgloss re-wrapping the input
	// line, whose cursor escapes must not be broken up.
	sub := m.sub
	if inner := m.width - keyPromptGutter; m.width > 0 && inner > 0 {
		sub = sub.Width(inner)
	}

	var body strings.Builder
	body.WriteString(m.head.Render(keyPromptTitle) + "\n")
	body.WriteString(sub.Render(keyPromptHint) + "\n")
	body.WriteString(sub.Render(keyPromptGetKey) + "\n\n")
	hints := keyPromptHints
	if m.saving {
		body.WriteString(m.spin.View() + " validating key…")
		hints = keyPromptHintsSaving
	} else {
		body.WriteString(m.input.View())
		// The error belongs to the input, so it stays inside the rail.
		if m.errMsg != "" {
			body.WriteString("\n\n" + m.bad.Render(m.errMsg))
		}
	}

	return header + "\n" + m.rail.Render(body.String()) + "\n\n" + indent + m.renderHints(hints) + "\n"
}

// renderHints formats footer hints as "[enter] save · [esc] cancel", with the
// key in brand accent inside dim brackets so it reads as a key rather than
// blending into its label.
func (m keyPromptModel) renderHints(hints []keyHint) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, m.sub.Render("[")+m.hintKey.Render(h.key)+m.sub.Render("] "+h.action))
	}
	return strings.Join(parts, m.sub.Render(hintSep))
}

// runKeyPrompt drives the branded prompt to completion and returns the masked
// label produced by save. It renders inline to out (stderr) so stdout stays
// clean for the success line and for redirection.
//
// save runs against a context the prompt itself can cancel (see abort), so
// backing out mid-validation kills the request instead of letting it finish and
// write config behind the user's back. The parent ctx — not the derived one —
// drives the program, so an outside interrupt still tears the whole thing down
// while an internal cancel only abandons the save.
//
// The guarantee is "the request is aborted", not "nothing can possibly be
// written": a save that has already passed its own ctx.Err() check will still
// complete. That window is microseconds wide and only reachable when validation
// had already succeeded.
func runKeyPrompt(ctx context.Context, in io.Reader, out io.Writer, save func(context.Context, string) (string, error)) (string, error) {
	saveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	final, err := tea.NewProgram(
		newKeyPromptModel(saveCtx, cancel, out, save),
		tea.WithContext(ctx),
		tea.WithInput(in),
		tea.WithOutput(out),
	).Run()
	if err != nil {
		return "", err
	}
	m, ok := final.(keyPromptModel)
	if !ok || m.canceled || !m.done {
		return "", errKeyPromptCanceled
	}
	return m.label, nil
}
