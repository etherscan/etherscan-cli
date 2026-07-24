// Package tui implements the interactive, full-screen Etherscan explorer that is
// launched by the `etherscan tui` command at an interactive terminal. It is a
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

	"github.com/etherscan/etherscan-cli/internal/brand"
	"github.com/etherscan/etherscan-cli/internal/output"
)

// Param describes a single input the user must supply for an endpoint. Name is
// the API param name from the docs and labels the form field; Label is the human
// explanation shown as the input's placeholder hint.
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
	// Validate optionally checks a submitted form's params before any fetch so
	// errors surface inline in the form (cross-field rules, value kinds). Nil
	// skips the check; the executor is expected to guard regardless.
	Validate  func(module, action string, params map[string]string) error
	ChainName string
	ChainID   string
	KeyLabel  string // masked key, or "none"
	HasAPIKey bool
	// SaveAPIKey validates and persists a key, returning its masked display label.
	// When provided, API-backed endpoints open an in-TUI setup prompt if HasAPIKey
	// is false. Bare endpoints remain available without credentials.
	SaveAPIKey func(ctx context.Context, key string) (label string, err error)
	// Chains is the list offered by the in-TUI chain switcher; SwitchChain applies a
	// selection (rebinding the client) and returns the resolved display name/id. Both are
	// optional — a nil SwitchChain disables the switcher entirely.
	Chains      []ChainInfo
	SwitchChain func(nameOrID string) (name, id string, err error)
}

// ChainInfo is one selectable chain in the switcher.
type ChainInfo struct {
	Name        string // stable CLI slug passed to SwitchChain
	DisplayName string // official Etherscan supported-chains name
	ID          string
	Aliases     []string
	Testnet     bool
	PaidOnly    bool
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
	stateChainPicker
	stateAPIKey
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

type apiKeySavedMsg struct {
	label string
	err   error
}

var (
	accent   = lipgloss.Color(brand.AccentHex)
	dim      = lipgloss.Color(brand.DimHex)
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
	keyOnSt  = lipgloss.NewStyle().Foreground(lipgloss.Color(brand.GreenHex))
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

	// chain switcher
	chainIdx    int
	chainFilter string
	chainErr    string
	chainReturn viewState

	// just-in-time API-key setup
	keyInput  textinput.Model
	keySaving bool
	keyErr    string
	keyReturn viewState

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
		if m.state == stateFetching || (m.state == stateAPIKey && m.keySaving) {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case resultMsg:
		m.setResult(msg.raw, msg.err)
		return m, nil

	case apiKeySavedMsg:
		m.keySaving = false
		if msg.err != nil {
			m.keyErr = msg.err.Error()
			m.keyInput.Focus()
			return m, textinput.Blink
		}
		m.cfg.HasAPIKey = true
		m.cfg.KeyLabel = msg.label
		m.keyInput.SetValue("")
		return m.startFetch()

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
	case stateAPIKey:
		if !m.keySaving {
			var cmd tea.Cmd
			m.keyInput, cmd = m.keyInput.Update(msg)
			return m, cmd
		}
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
	case stateChainPicker:
		return m.keyChainPicker(msg)
	case stateAPIKey:
		return m.keyAPIKey(msg)
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
	case "c":
		return m.openChainPicker()
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

	// Collect params into an input form; if none, fetch immediately. page is
	// omitted on paginated endpoints — the n/p pager owns it (fetchCmd would
	// overwrite a typed value anyway).
	m.inParams = nil
	for _, pr := range m.current.Params {
		if m.current.Paginated && pr.Name == "page" {
			continue
		}
		m.inParams = append(m.inParams, pr)
	}
	if len(m.inParams) == 0 {
		m.params = map[string]string{}
		return m.startFetch()
	}
	m.inputs = make([]textinput.Model, len(m.inParams))
	for i, pr := range m.inParams {
		ti := textinput.New()
		ti.Placeholder = pr.Label
		if m.current.Paginated && pr.Name == "offset" {
			ti.Placeholder = fmt.Sprintf("%s (default %d)", pr.Label, pageSize)
		}
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
	if m.cfg.Validate != nil {
		if err := m.cfg.Validate(m.current.Module, m.current.Action, params); err != nil {
			m.formErr = err.Error()
			return m, nil
		}
	}
	m.formErr = ""
	m.params = params
	return m.startFetch()
}

func (m *model) startFetch() (tea.Model, tea.Cmd) {
	if !m.current.Bare && !m.cfg.HasAPIKey && m.cfg.SaveAPIKey != nil {
		return m.openAPIKey()
	}
	m.state = stateFetching
	m.resultTitle = m.current.Module + "/" + m.current.Action
	if m.current.Bare {
		m.resultTitle = m.current.Action
	}
	return m, tea.Batch(m.spin.Tick, m.fetchCmd())
}

func (m *model) openAPIKey() (tea.Model, tea.Cmd) {
	m.keyReturn = m.state
	m.keyErr = ""
	m.keySaving = false
	ti := textinput.New()
	ti.Placeholder = "paste your API key"
	ti.Prompt = "› "
	ti.Focus()
	m.keyInput = ti
	m.state = stateAPIKey
	return m, textinput.Blink
}

func (m *model) keyAPIKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.keySaving {
		if msg.String() == "ctrl+c" {
			m.keyInput.SetValue("")
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		m.keyInput.SetValue("")
		return m, tea.Quit
	case "esc":
		m.keyInput.SetValue("")
		m.keyErr = ""
		m.state = m.keyReturn
		return m, nil
	case "enter":
		key := strings.TrimSpace(m.keyInput.Value())
		if key == "" {
			m.keyErr = "API key is required"
			return m, nil
		}
		m.keyErr = ""
		m.keySaving = true
		m.keyInput.Blur()
		save, ctx := m.cfg.SaveAPIKey, m.ctx
		return m, tea.Batch(m.spin.Tick, func() tea.Msg {
			label, err := save(ctx, key)
			return apiKeySavedMsg{label: label, err: err}
		})
	}
	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
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
	case "c":
		return m.openChainPicker()
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

func (m *model) openChainPicker() (tea.Model, tea.Cmd) {
	if m.cfg.SwitchChain == nil || len(m.cfg.Chains) == 0 {
		return m, nil
	}
	m.chainReturn = m.state
	m.state = stateChainPicker
	m.chainFilter = ""
	m.chainErr = ""
	m.chainIdx = 0
	return m, nil
}

// filteredChains returns the chains matching the current filter (case-insensitive
// substring on name, or a prefix/substring on the numeric id).
func (m model) filteredChains() []ChainInfo {
	if m.chainFilter == "" {
		return m.cfg.Chains
	}
	needle := strings.ToLower(m.chainFilter)
	var out []ChainInfo
	for _, c := range m.cfg.Chains {
		matched := strings.Contains(strings.ToLower(c.Name), needle) ||
			strings.Contains(strings.ToLower(c.DisplayName), needle) || strings.Contains(c.ID, needle)
		for _, alias := range c.Aliases {
			matched = matched || strings.Contains(strings.ToLower(alias), needle)
		}
		if matched {
			out = append(out, c)
		}
	}
	return out
}

func (m *model) keyChainPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.filteredChains()
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.state = m.chainReturn
		m.chainFilter = ""
		m.chainErr = ""
		return m, nil
	case tea.KeyUp:
		m.chainIdx = clamp(m.chainIdx-1, 0, max(0, len(list)-1))
		m.chainErr = ""
		return m, nil
	case tea.KeyDown:
		m.chainIdx = clamp(m.chainIdx+1, 0, max(0, len(list)-1))
		m.chainErr = ""
		return m, nil
	case tea.KeyEnter:
		if len(list) == 0 {
			return m, nil
		}
		sel := list[clamp(m.chainIdx, 0, len(list)-1)]
		name, id, err := m.cfg.SwitchChain(sel.Name)
		if err != nil {
			m.chainErr = err.Error()
			return m, nil
		}
		m.cfg.ChainName, m.cfg.ChainID = name, id
		m.chainFilter, m.chainErr = "", ""
		m.state = stateBrowse
		return m, nil
	case tea.KeyBackspace:
		if m.chainFilter != "" {
			runes := []rune(m.chainFilter)
			m.chainFilter = string(runes[:len(runes)-1])
			m.chainIdx = 0
			m.chainErr = ""
		}
		return m, nil
	case tea.KeyRunes:
		m.chainFilter += string(msg.Runes)
		m.chainIdx = 0
		m.chainErr = ""
		return m, nil
	}
	return m, nil
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
	case stateChainPicker:
		return m.viewChainPicker()
	case stateAPIKey:
		return m.viewAPIKey()
	default:
		return m.viewBrowse()
	}
}

func (m model) viewAPIKey() string {
	var b strings.Builder
	b.WriteString(headSt.Render("Connect your API key") + "\n")
	b.WriteString(subSt.Render("An API key is needed to run this endpoint. You can keep exploring without one.") + "\n")
	b.WriteString(subSt.Render("Get a free key at https://etherscan.io/api/pricing") + "\n\n")
	if m.keySaving {
		b.WriteString(m.spin.View() + " validating key…\n")
	} else {
		b.WriteString(m.keyInput.View() + "\n")
		if m.keyErr != "" {
			b.WriteString("\n" + errSt.Render(m.keyErr) + "\n")
		}
	}
	return join(m.header(), "", b.String(), m.footer("enter save & continue · esc keep exploring"))
}

func (m model) viewChainPicker() string {
	list := m.filteredChains()
	var b strings.Builder
	b.WriteString(headSt.Render("Switch chain") + "\n")
	b.WriteString(descSt.Render(fmt.Sprintf("current: %s (%s)", m.cfg.ChainName, m.cfg.ChainID)) + "\n\n")
	b.WriteString(labelSt.Render("filter: ") + m.chainFilter + "\n\n")
	if len(list) == 0 {
		b.WriteString(descSt.Render("(no matching chains)") + "\n")
	} else {
		visible := 10
		if m.height > 0 {
			if v := m.height - 12; v >= 1 {
				visible = v
			} else {
				visible = 1
			}
		}
		idx := clamp(m.chainIdx, 0, len(list)-1)
		start, end := windowIndices(len(list), visible, idx)
		if start > 0 {
			b.WriteString(descSt.Render(fmt.Sprintf("… %d above", start)) + "\n")
		}
		for i := start; i < end; i++ {
			c := list[i]
			displayName := c.DisplayName
			if displayName == "" {
				displayName = c.Name
			}
			line := fmt.Sprintf("%s (%s)", displayName, c.ID)
			suffix := ""
			if c.PaidOnly {
				suffix = " (paid only)"
			}
			if i == idx {
				b.WriteString(selSt.Render("› "+line+suffix) + "\n")
			} else {
				b.WriteString("  " + line + descSt.Render(suffix) + "\n")
			}
		}
		if end < len(list) {
			b.WriteString(descSt.Render(fmt.Sprintf("… %d more", len(list)-end)) + "\n")
		}
	}
	if m.chainErr != "" {
		b.WriteString("\n" + errSt.Render(m.chainErr) + "\n")
	}
	foot := m.footer("type to filter · ↑/↓ move · enter switch · esc cancel")
	return join(m.header(), "", b.String(), foot)
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
	keys := "↑/↓ move · ←/→ pane · enter open · esc/q quit"
	if m.cfg.SwitchChain != nil {
		keys = "↑/↓ move · ←/→ pane · enter open · c chain · esc/q quit"
	}
	return join(m.header(), "", banner, "", body, "", m.footer(keys))
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
	// Window the fields to the terminal height, keeping the focused input in
	// view — Bubble Tea trims an overflowing view from the top, which would eat
	// the header first (same failure class viewBrowse guards against). Each
	// field renders 3 lines; the fixed chrome (header, title, desc, blanks,
	// footer, error line, edge markers) budgets 10.
	start, end := 0, len(m.inParams)
	if m.height > 0 {
		visible := (m.height - 10) / 3
		if visible < 1 {
			visible = 1
		}
		start, end = windowIndices(len(m.inParams), visible, m.inputIdx)
	}
	if start > 0 {
		b.WriteString(descSt.Render(fmt.Sprintf("… %d above", start)) + "\n")
	}
	for i := start; i < end; i++ {
		// Fields are labeled by the API param name (docs alignment); the human
		// explanation is the input's placeholder.
		label := labelSt.Render(m.inParams[i].Name)
		if !m.inParams[i].Required {
			label += descSt.Render(" (optional)")
		}
		b.WriteString(label + "\n")
		b.WriteString(m.inputs[i].View() + "\n\n")
	}
	if end < len(m.inParams) {
		b.WriteString(descSt.Render(fmt.Sprintf("… %d more", len(m.inParams)-end)) + "\n")
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
	if m.cfg.SwitchChain != nil {
		keys += " · c chain"
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
