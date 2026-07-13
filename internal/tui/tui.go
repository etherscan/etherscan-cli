// Package tui implements the interactive, full-screen Etherscan explorer that is
// launched on a bare `etherscan` invocation at an interactive terminal. It is a
// read-only browser over the endpoint registry: pick a module, pick an endpoint,
// fill any required parameters, and view the result. It never handles write or
// sensitive actions (those stay CLI-only) and is fully decoupled from the cli
// package — the caller injects the endpoint list and an executor closure.
package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/etherscan/etherscan-cli/internal/output"
)

// Param describes a single input the user must supply for an endpoint.
type Param struct {
	Name     string
	Label    string
	Required bool
}

// Endpoint is a tui-local view of a registry endpoint spec, built by the caller.
type Endpoint struct {
	Module    string
	Action    string
	Title     string
	Desc      string
	Params    []Param
	Columns   []string
	Paginated bool
	// Bare marks an endpoint with no module/action on the wire (chainlist): Module
	// then only places it in the sidebar, and result headers show the Action alone.
	Bare bool
	// Group is an optional sidebar group label (docs nav group name) for when it
	// differs from the wire module (e.g. "usage" for getapilimit). Empty means
	// group by Module. It never appears in result headers or requests.
	Group string
}

// group returns the sidebar group label for the endpoint.
func (e Endpoint) group() string {
	if e.Group != "" {
		return e.Group
	}
	return e.Module
}

// Exec runs one endpoint and returns its raw JSON result. The caller wires this
// to the CLI's existing client/runtime path.
type Exec func(ctx context.Context, module, action string, params map[string]string) (json.RawMessage, error)

// Config is everything Run needs to drive the explorer.
type Config struct {
	Endpoints []Endpoint
	Exec      Exec
	ChainName string
	ChainID   string
	KeyLabel  string // masked key, or "none"
}

// Run launches the full-screen explorer and blocks until the user quits.
func Run(ctx context.Context, cfg Config) error {
	m := newModel(ctx, cfg)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

const pageSize = 25

type viewState int

const (
	stateBrowse viewState = iota
	stateForm
	stateFetching
	stateResult
)

type focusCol int

const (
	focusModules focusCol = iota
	focusEndpoints
)

type resultMsg struct {
	raw json.RawMessage
	err error
}

var (
	accent   = lipgloss.Color("#5A8DEE")
	dim      = lipgloss.Color("#8A8A8A")
	errColor = lipgloss.Color("#E06C75")
	titleSt  = lipgloss.NewStyle().Bold(true).Foreground(accent)
	logoSt   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	subSt    = lipgloss.NewStyle().Foreground(dim)
	footerSt = lipgloss.NewStyle().Foreground(dim)
	panelSt  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim).Padding(0, 1)
	headSt   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	selSt    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent)
	itemSt   = lipgloss.NewStyle()
	descSt   = lipgloss.NewStyle().Foreground(dim)
	errSt    = lipgloss.NewStyle().Foreground(errColor)
	labelSt  = lipgloss.NewStyle().Foreground(accent)
	keyOnSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	keyOffSt = lipgloss.NewStyle().Foreground(errColor)
)

type model struct {
	ctx context.Context
	cfg Config

	modules  []string
	byModule map[string][]Endpoint

	state  viewState
	focus  focusCol
	modIdx int
	epIdx  int

	// form
	current  Endpoint
	inputs   []textinput.Model
	inParams []Param
	inputIdx int
	formErr  string

	// fetch / result
	spin        spinner.Model
	vp          viewport.Model
	params      map[string]string
	page        int
	resultTitle string

	width, height int
	ready         bool
}

func newModel(ctx context.Context, cfg Config) model {
	byModule := map[string][]Endpoint{}
	var modules []string
	for _, ep := range cfg.Endpoints {
		if _, ok := byModule[ep.group()]; !ok {
			modules = append(modules, ep.group())
		}
		byModule[ep.group()] = append(byModule[ep.group()], ep)
	}
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)
	return model{
		ctx:      ctx,
		cfg:      cfg,
		modules:  modules,
		byModule: byModule,
		spin:     sp,
		focus:    focusModules,
		page:     1,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) selectedModule() string {
	if len(m.modules) == 0 {
		return ""
	}
	return m.modules[clamp(m.modIdx, 0, len(m.modules)-1)]
}

func (m model) endpointsForSelected() []Endpoint {
	return m.byModule[m.selectedModule()]
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		bodyH := msg.Height - 9
		if bodyH < 3 {
			bodyH = 3
		}
		if !m.ready {
			m.vp = viewport.New(msg.Width-4, bodyH)
			m.ready = true
		} else {
			m.vp.Width = msg.Width - 4
			m.vp.Height = bodyH
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == stateFetching {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case resultMsg:
		m.setResult(msg.raw, msg.err)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Route any other message to the active component. textinput's paste (ctrl+v)
	// returns an unexported pasteMsg that must reach the input to insert the text;
	// without this it is silently dropped.
	switch m.state {
	case stateForm:
		if len(m.inputs) > 0 {
			var cmd tea.Cmd
			m.inputs[m.inputIdx], cmd = m.inputs[m.inputIdx].Update(msg)
			return m, cmd
		}
	case stateResult:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateBrowse:
		return m.keyBrowse(msg)
	case stateForm:
		return m.keyForm(msg)
	case stateFetching:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	case stateResult:
		return m.keyResult(msg)
	}
	return m, nil
}

func (m *model) keyBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.focus == focusModules {
			m.modIdx = clamp(m.modIdx-1, 0, len(m.modules)-1)
			m.epIdx = 0
		} else {
			m.epIdx = clamp(m.epIdx-1, 0, len(m.endpointsForSelected())-1)
		}
	case "down", "j":
		if m.focus == focusModules {
			m.modIdx = clamp(m.modIdx+1, 0, len(m.modules)-1)
			m.epIdx = 0
		} else {
			m.epIdx = clamp(m.epIdx+1, 0, len(m.endpointsForSelected())-1)
		}
	case "left", "h":
		m.focus = focusModules
	case "esc":
		// Back out of the endpoints pane; from the top level (modules) esc quits.
		if m.focus == focusEndpoints {
			m.focus = focusModules
			return m, nil
		}
		return m, tea.Quit
	case "right", "l":
		m.focus = focusEndpoints
		m.epIdx = clamp(m.epIdx, 0, len(m.endpointsForSelected())-1)
	case "enter":
		if m.focus == focusModules {
			m.focus = focusEndpoints
			m.epIdx = clamp(m.epIdx, 0, len(m.endpointsForSelected())-1)
			return m, nil
		}
		return m.openSelected()
	}
	return m, nil
}

func (m *model) openSelected() (tea.Model, tea.Cmd) {
	eps := m.endpointsForSelected()
	if len(eps) == 0 {
		return m, nil
	}
	m.current = eps[clamp(m.epIdx, 0, len(eps)-1)]
	m.formErr = ""
	m.page = 1

	// Collect required params into an input form; if none, fetch immediately.
	m.inParams = nil
	for _, pr := range m.current.Params {
		if pr.Required {
			m.inParams = append(m.inParams, pr)
		}
	}
	if len(m.inParams) == 0 {
		m.params = map[string]string{}
		return m.startFetch()
	}
	m.inputs = make([]textinput.Model, len(m.inParams))
	for i, pr := range m.inParams {
		ti := textinput.New()
		ti.Placeholder = pr.Label
		ti.Prompt = "› "
		if i == 0 {
			ti.Focus()
		}
		m.inputs[i] = ti
	}
	m.inputIdx = 0
	m.state = stateForm
	return m, textinput.Blink
}

func (m *model) keyForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.state = stateBrowse
		return m, nil
	case "tab", "down":
		m.focusInput(m.inputIdx + 1)
		return m, nil
	case "shift+tab", "up":
		m.focusInput(m.inputIdx - 1)
		return m, nil
	case "enter":
		return m.submitForm()
	}
	var cmd tea.Cmd
	m.inputs[m.inputIdx], cmd = m.inputs[m.inputIdx].Update(msg)
	return m, cmd
}

func (m *model) focusInput(idx int) {
	idx = clamp(idx, 0, len(m.inputs)-1)
	for i := range m.inputs {
		if i == idx {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	m.inputIdx = idx
}

func (m *model) submitForm() (tea.Model, tea.Cmd) {
	params := map[string]string{}
	for i, pr := range m.inParams {
		v := strings.TrimSpace(m.inputs[i].Value())
		if pr.Required && v == "" {
			m.formErr = fmt.Sprintf("%s is required", pr.Name)
			m.focusInput(i)
			return m, nil
		}
		if v != "" {
			params[pr.Name] = v
		}
	}
	m.params = params
	return m.startFetch()
}

func (m *model) startFetch() (tea.Model, tea.Cmd) {
	m.state = stateFetching
	m.resultTitle = m.current.Module + "/" + m.current.Action
	if m.current.Bare {
		m.resultTitle = m.current.Action
	}
	return m, tea.Batch(m.spin.Tick, m.fetchCmd())
}

func (m *model) fetchCmd() tea.Cmd {
	ep := m.current
	params := map[string]string{}
	for k, v := range m.params {
		params[k] = v
	}
	if ep.Paginated {
		params["page"] = strconv.Itoa(m.page)
		if _, ok := params["offset"]; !ok {
			params["offset"] = strconv.Itoa(pageSize)
		}
	}
	ctx := m.ctx
	exec := m.cfg.Exec
	mod, act := ep.Module, ep.Action
	return func() tea.Msg {
		raw, err := exec(ctx, mod, act, params)
		return resultMsg{raw: raw, err: err}
	}
}

func (m *model) keyResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "left", "h":
		m.state = stateBrowse
		return m, nil
	case "n":
		if m.current.Paginated {
			m.page++
			return m.startFetch()
		}
	case "p":
		if m.current.Paginated && m.page > 1 {
			m.page--
			return m.startFetch()
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *model) setResult(raw json.RawMessage, err error) {
	m.state = stateResult
	if err != nil {
		m.setContent(errSt.Render("error: " + err.Error()))
		return
	}
	rows, scalar, rerr := output.Rows(raw)
	if rerr != nil {
		m.setContent(errSt.Render("error: " + rerr.Error()))
		return
	}
	if len(rows) == 0 {
		if strings.TrimSpace(scalar) != "" {
			m.setContent(scalar)
		} else {
			m.setContent(descSt.Render("(no data)"))
		}
		return
	}
	var buf bytes.Buffer
	if werr := output.WriteRows(&buf, rows, output.Table, m.current.Columns); werr != nil {
		m.setContent(errSt.Render("error: " + werr.Error()))
		return
	}
	m.setContent(strings.TrimRight(buf.String(), "\n"))
}

func (m *model) setContent(s string) {
	if m.ready {
		m.vp.SetContent(s)
		m.vp.GotoTop()
	}
}

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	switch m.state {
	case stateForm:
		return m.viewForm()
	case stateFetching:
		return m.viewFetching()
	case stateResult:
		return m.viewResult()
	default:
		return m.viewBrowse()
	}
}

func (m model) header() string {
	left := titleSt.Render("◆ Etherscan")
	key := keyOnSt.Render(m.cfg.KeyLabel)
	if m.cfg.KeyLabel == "" || m.cfg.KeyLabel == "none" {
		key = keyOffSt.Render("none")
	}
	right := subSt.Render(fmt.Sprintf("chain %s (%s)   key ", m.cfg.ChainName, m.cfg.ChainID)) + key
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m model) footer(keys string) string {
	return footerSt.Render(keys)
}

func (m model) viewBrowse() string {
	// Height budget. The view is join(header, "", banner, "", body, "", footer):
	// banner + body + 5 chrome lines, and each panel adds 4 lines (heading, blank,
	// two border rows) around its list. The banner gets priority: it is sized
	// against a fixed minimum list height — NOT the selected module's endpoint
	// count — so it never flickers between big and compact while moving across
	// modules. The lists are then windowed into whatever remains, because Bubble
	// Tea trims an overflowing view from the top — which would eat the header first.
	const chromeRows = 5
	const panelRows = 4
	const minListRows = 5
	eps := m.endpointsForSelected()
	banner := renderBanner(m.width, m.height-chromeRows-panelRows-minListRows)
	visible := m.height - chromeRows - panelRows - lipgloss.Height(banner)
	if visible < 3 {
		visible = 3
	}

	// Modules panel.
	var mod strings.Builder
	mod.WriteString(headSt.Render("MODULES") + "\n\n")
	mStart, mEnd := windowIndices(len(m.modules), visible, m.modIdx)
	for i := mStart; i < mEnd; i++ {
		name := m.modules[i]
		line := itemSt.Render("  " + name)
		if i == m.modIdx {
			if m.focus == focusModules {
				line = selSt.Render(" " + name + " ")
			} else {
				line = labelSt.Render("▸ " + name)
			}
		}
		mod.WriteString(line + "\n")
	}
	modLines := markTruncation(strings.TrimRight(mod.String(), "\n"), len(m.modules), mStart, mEnd)
	modPanel := panelSt.Width(18).Render(modLines)

	// Endpoints panel.
	var ep strings.Builder
	ep.WriteString(headSt.Render("ENDPOINTS · "+strings.ToUpper(m.selectedModule())) + "\n\n")
	nameW := 0
	for _, e := range eps {
		if len(e.Title) > nameW {
			nameW = len(e.Title)
		}
	}
	eStart, eEnd := windowIndices(len(eps), visible, m.epIdx)
	for i := eStart; i < eEnd; i++ {
		e := eps[i]
		name := padRight(e.Title, nameW+2)
		row := name + descSt.Render(e.Desc)
		if m.focus == focusEndpoints && i == m.epIdx {
			row = selSt.Render(" " + name + e.Desc + " ")
		}
		ep.WriteString(row + "\n")
	}
	epLines := markTruncation(strings.TrimRight(ep.String(), "\n"), len(eps), eStart, eEnd)
	epW := m.width - lipgloss.Width(modPanel) - 4
	if epW < 20 {
		epW = 20
	}
	epPanel := panelSt.Width(epW).Render(epLines)

	body := lipgloss.JoinHorizontal(lipgloss.Top, modPanel, " ", epPanel)
	foot := m.footer("↑/↓ move · ←/→ pane · enter open · esc/q quit")
	return join(m.header(), "", banner, "", body, "", foot)
}

// windowIndices returns the [start, end) slice of a total-item list that fits in
// visible rows while keeping idx in view, biased to centre the selection.
func windowIndices(total, visible, idx int) (int, int) {
	if total <= visible {
		return 0, total
	}
	start := clamp(idx-visible/2, 0, total-visible)
	return start, start + visible
}

// markTruncation replaces the first/last rendered list row with a dim "… N more"
// marker when the window hides items above/below. The selection never occupies
// those rows: windowIndices centres it, so an edge row is selected only when the
// window is flush against that end of the list (i.e. nothing is hidden there).
func markTruncation(list string, total, start, end int) string {
	if start == 0 && end == total {
		return list
	}
	lines := strings.Split(list, "\n")
	// The first two lines are the panel heading and its blank spacer.
	if start > 0 && len(lines) > 2 {
		lines[2] = descSt.Render(fmt.Sprintf("  … %d above", start))
	}
	if end < total && len(lines) > 2 {
		lines[len(lines)-1] = descSt.Render(fmt.Sprintf("  … %d more", total-end))
	}
	return strings.Join(lines, "\n")
}

func (m model) viewForm() string {
	var b strings.Builder
	b.WriteString(headSt.Render(m.current.Module+"/"+m.current.Action) + "\n")
	b.WriteString(descSt.Render(m.current.Desc) + "\n\n")
	for i, pr := range m.inParams {
		b.WriteString(labelSt.Render(pr.Label) + "\n")
		b.WriteString(m.inputs[i].View() + "\n\n")
	}
	if m.formErr != "" {
		b.WriteString(errSt.Render(m.formErr) + "\n")
	}
	foot := m.footer("tab next field · enter submit · esc back")
	return join(m.header(), "", b.String(), foot)
}

func (m model) viewFetching() string {
	body := m.spin.View() + " fetching…"
	return join(m.header(), "", headSt.Render(m.resultTitle), "", body)
}

func (m model) viewResult() string {
	keys := "↑/↓ scroll · esc back · q quit"
	if m.current.Paginated {
		keys = fmt.Sprintf("↑/↓ scroll · n/p page (%d) · esc back · q quit", m.page)
	}
	return join(m.header(), "", headSt.Render(m.resultTitle), "", m.vp.View(), "", m.footer(keys))
}

func join(parts ...string) string {
	return strings.Join(parts, "\n")
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
