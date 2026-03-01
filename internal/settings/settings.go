// Package settings provides an interactive TUI for editing dirstral configuration.
package settings

import (
	"fmt"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/config"
	"github.com/alibilge/dirstral-cli/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// viewState tracks the current interaction mode.
type viewState int

const (
	stateBrowsing viewState = iota
	stateEditing
	stateConfirmQuit
)

// model is the bubbletea model for the settings editor.
type model struct {
	cfg       config.Config
	fields    []config.FieldInfo
	cursor    int
	state     viewState
	input     textinput.Model
	dirty     bool
	errMsg    string
	statusMsg string
	width     int
	height    int
	showHelp  bool
}

func initialModel(cfg config.Config) model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60

	fields := config.EffectiveFields(cfg)

	return model{
		cfg:    cfg,
		fields: fields,
		cursor: 0,
		state:  stateBrowsing,
		input:  ti,
	}
}

// Run launches the settings TUI as its own tea.Program.
func Run(cfg config.Config) error {
	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = maxInt(settingsContentWidth(msg.Width)-4, 20)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "?" || msg.String() == "ctrl+k" {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			switch msg.String() {
			case "esc", "q", "?", "ctrl+k":
				m.showHelp = false
			}
			return m, nil
		}
		return m.handleKey(msg)
	}

	// Forward to text input when editing.
	if m.state == stateEditing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateBrowsing:
		return m.handleBrowsingKey(msg)
	case stateEditing:
		return m.handleEditingKey(msg)
	case stateConfirmQuit:
		return m.handleConfirmQuitKey(msg)
	}
	return m, nil
}

func (m model) handleBrowsingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if m.dirty {
			m.state = stateConfirmQuit
			m.errMsg = ""
			m.statusMsg = ""
			return m, nil
		}
		return m, tea.Quit
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.errMsg = ""
	case "down", "j":
		if m.cursor < len(m.fields)-1 {
			m.cursor++
		}
		m.errMsg = ""
	case "enter":
		m.startEditing()
		return m, m.input.Focus()
	case "r":
		m.resetField()
	case "s":
		m.save()
	}
	return m, nil
}

func (m *model) startEditing() {
	f := m.fields[m.cursor]
	m.state = stateEditing
	m.errMsg = ""
	m.statusMsg = ""
	m.input.SetValue(f.Value)
	m.input.CursorEnd()
	if f.Sensitive {
		m.input.EchoMode = textinput.EchoPassword
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}
	m.input.Placeholder = fmt.Sprintf("Enter value for %s", f.Key)
}

func (m model) handleEditingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateBrowsing
		m.errMsg = ""
		m.input.Blur()
		return m, nil
	case "enter":
		return m.commitEdit()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	// Live validation feedback.
	val := m.input.Value()
	if err := config.ValidateField(m.fields[m.cursor].Key, val); err != nil {
		m.errMsg = err.Error()
	} else {
		m.errMsg = ""
	}
	return m, cmd
}

func (m model) commitEdit() (tea.Model, tea.Cmd) {
	key := m.fields[m.cursor].Key
	val := m.input.Value()

	if err := config.ValidateField(key, val); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	// Apply to the in-memory config or track secret change.
	if m.fields[m.cursor].Sensitive {
		m.fields[m.cursor].Value = val
		m.fields[m.cursor].Source = config.SourceDotEnvLocal
	} else {
		config.ApplyField(&m.cfg, key, val)
		m.fields[m.cursor].Value = val
		m.fields[m.cursor].Source = config.SourceConfigFile
	}

	m.dirty = true
	m.state = stateBrowsing
	m.errMsg = ""
	m.statusMsg = fmt.Sprintf("Updated %s", key)
	m.input.Blur()
	return m, nil
}

func (m *model) resetField() {
	f := &m.fields[m.cursor]
	def := config.DefaultValueForField(f.Key)
	if def == "" && f.Sensitive {
		m.errMsg = "No default for secrets"
		return
	}
	if def == "" {
		m.errMsg = "No default value for this field"
		return
	}
	config.ApplyField(&m.cfg, f.Key, def)
	f.Value = def
	f.Source = config.SourceDefault
	m.dirty = true
	m.errMsg = ""
	m.statusMsg = fmt.Sprintf("Reset %s to default", f.Key)
}

func (m *model) save() {
	// Save non-sensitive fields to config.toml.
	if err := config.Save(m.cfg); err != nil {
		m.errMsg = fmt.Sprintf("Save failed: %v", err)
		return
	}

	// Save secrets to .env.local.
	for _, f := range m.fields {
		if f.Sensitive && f.Value != "" {
			envKey := config.EnvVarForField(f.Key)
			if envKey == "" {
				envKey = f.Key
			}
			if err := config.SaveSecret(envKey, f.Value); err != nil {
				m.errMsg = fmt.Sprintf("Save secret %s failed: %v", f.Key, err)
				return
			}
		}
	}

	m.dirty = false
	m.errMsg = ""
	m.statusMsg = "Settings saved"
}

func (m model) handleConfirmQuitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.save()
		if m.errMsg != "" {
			// Save failed, stay in confirm state.
			return m, nil
		}
		return m, tea.Quit
	case "n", "N":
		return m, tea.Quit
	case "esc", "c":
		m.state = stateBrowsing
		m.statusMsg = ""
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

var (
	settingsTitleStyle = lipgloss.NewStyle().
				Foreground(ui.ClrBrand).
				Bold(true).
				Underline(true)

	settingsMutedStyle = lipgloss.NewStyle().
				Foreground(ui.ClrMuted)

	settingsSubtleStyle = lipgloss.NewStyle().
				Foreground(ui.ClrSubtle)

	settingsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.ClrSubtle).
				Padding(1, 2).
				MarginTop(1).
				MarginBottom(1)

	settingsSelectedMarkerStyle = lipgloss.NewStyle().
					Foreground(ui.ClrBrand).
					Bold(true)

	settingsSelectedKeyStyle = lipgloss.NewStyle().
					Background(ui.ClrBrand).
					Foreground(lipgloss.Color("0")).
					Bold(true)

	settingsKeyStyle = lipgloss.NewStyle().
				Foreground(ui.ClrMuted)

	settingsValueStyle = lipgloss.NewStyle().
				Foreground(ui.ClrSubtle)

	settingsSelectedValueStyle = lipgloss.NewStyle().
					Foreground(ui.ClrBrand).
					Italic(true)

	settingsSourceStyle = lipgloss.NewStyle().
				Foreground(ui.ClrMuted).
				Italic(true)

	settingsSelectedSourceStyle = lipgloss.NewStyle().
					Foreground(ui.ClrYellow).
					Italic(true)
)

func (m model) View() string {
	viewWidth := m.width
	if viewWidth <= 0 {
		viewWidth = 100
	}

	tinyHeight := m.height > 0 && m.height < 14
	contentWidth := settingsContentWidth(viewWidth)
	panelWidth := settingsPanelWidth(viewWidth, contentWidth)

	title := settingsTitleStyle.MaxWidth(contentWidth).Render("Settings")
	intro := settingsMutedStyle.MaxWidth(contentWidth).Render("Edit config and API defaults for Dirstral.")
	header := joinVerticalNonEmpty(
		lipgloss.Center,
		lipgloss.PlaceHorizontal(panelWidth, lipgloss.Center, title),
		lipgloss.PlaceHorizontal(panelWidth, lipgloss.Center, intro),
	)

	panelLines := m.fieldRows(contentWidth)
	stateLines := m.stateLines()
	if len(stateLines) > 0 {
		panelLines = append(panelLines, "")
		panelLines = append(panelLines, stateLines...)
	}
	panel := settingsPanelStyle.MaxWidth(panelWidth).Render(strings.Join(panelLines, "\n"))

	body := joinVerticalNonEmpty(lipgloss.Center, header, panel)
	if m.showHelp {
		help := settingsPanelStyle.MaxWidth(panelWidth).Render(settingsHelpText(contentWidth, tinyHeight))
		body = joinVerticalNonEmpty(lipgloss.Center, body, help)
	}

	footer := settingsSubtleStyle.MaxWidth(contentWidth).Render(truncateText(m.controlsHint(), contentWidth))
	content := composeWithPinnedFooter(body, footer, m.height)

	if m.height <= 0 {
		return content
	}
	vAlign := lipgloss.Center
	if tinyHeight {
		vAlign = lipgloss.Top
	}
	return lipgloss.Place(viewWidth, m.height, lipgloss.Center, vAlign, content)
}

func (m model) fieldRows(contentWidth int) []string {
	maxRows := m.visibleRows()
	start, end := m.visibleRange(maxRows)

	keyWidth := clampInt(contentWidth/3, 18, 30)
	sourceWidth := clampInt(contentWidth/6, 10, 18)
	valueWidth := maxInt(contentWidth-keyWidth-sourceWidth-10, 12)

	lines := make([]string, 0, end-start+2)
	for i := start; i < end; i++ {
		f := m.fields[i]
		marker := settingsSubtleStyle.Render(" ")

		keyText := fitText(f.Key, keyWidth)
		valueText := fitText(fieldDisplayValue(f), valueWidth)
		sourceText := fitText("("+string(f.Source)+")", sourceWidth)

		keyCell := settingsKeyStyle.Width(keyWidth).Render(keyText)
		valueCell := settingsValueStyle.Width(valueWidth).Render(valueText)
		if i == m.cursor {
			marker = settingsSelectedMarkerStyle.Render(">")
			keyCell = settingsSelectedKeyStyle.Width(keyWidth).Render(keyText)
			valueCell = settingsSelectedValueStyle.Width(valueWidth).Render(valueText)
		}
		sourceCell := settingsSourceStyle.Width(sourceWidth).Render(sourceText)
		if i == m.cursor {
			sourceCell = settingsSelectedSourceStyle.Width(sourceWidth).Render(sourceText)
		}
		lines = append(lines, fmt.Sprintf("  %s %s  %s  %s", marker, keyCell, valueCell, sourceCell))
	}

	if len(lines) == 0 {
		lines = append(lines, settingsSubtleStyle.Render("  (no settings fields)"))
	}

	if len(m.fields) > maxRows {
		lines = append(lines, "")
		lines = append(lines, settingsSubtleStyle.Render(fmt.Sprintf("  [%d/%d]", m.cursor+1, len(m.fields))))
	}

	return lines
}

func (m model) stateLines() []string {
	lines := make([]string, 0, 4)

	switch m.state {
	case stateEditing:
		lines = append(lines, settingsMutedStyle.Render("Editing "+m.fields[m.cursor].Key))
		lines = append(lines, "  "+m.input.View())
		if m.errMsg != "" {
			lines = append(lines, ui.Red.Render("  "+m.errMsg))
		}
	case stateConfirmQuit:
		lines = append(lines, ui.Yellow.Render("Unsaved changes. Save before quitting?"))
		if m.errMsg != "" {
			lines = append(lines, ui.Red.Render(m.errMsg))
		}
	default:
		if m.errMsg != "" {
			lines = append(lines, ui.Red.Render(m.errMsg))
		}
		if m.statusMsg != "" {
			lines = append(lines, ui.Green.Render(m.statusMsg))
		}
		if m.dirty {
			lines = append(lines, ui.Yellow.Render("Unsaved changes"))
		}
	}

	return lines
}

func settingsHelpText(contentWidth int, tinyHeight bool) string {
	if tinyHeight {
		return settingsSubtleStyle.Render(truncateText("Keys: up/down or j/k move, enter edit, r reset, s save, esc/q back, ? toggle help", contentWidth))
	}

	lines := []string{
		settingsTitleStyle.Render("Settings Keymap"),
		settingsMutedStyle.Render("up/down or j/k  move selection"),
		settingsMutedStyle.Render("enter           edit selected value"),
		settingsMutedStyle.Render("r               reset selected value"),
		settingsMutedStyle.Render("s               save changes"),
		settingsMutedStyle.Render("esc/q           back/quit"),
		settingsMutedStyle.Render("? or ctrl+k     toggle this help"),
	}
	return lipgloss.NewStyle().MaxWidth(maxInt(contentWidth-2, 12)).Render(strings.Join(lines, "\n"))
}

func (m model) controlsHint() string {
	switch m.state {
	case stateEditing:
		return "type value · enter confirm · esc cancel · ? help"
	case stateConfirmQuit:
		return "y save & quit · n discard & quit · c/esc cancel · ? help"
	default:
		return "up/down or j/k move · enter edit · r reset · s save · esc/q back · ? help"
	}
}

func fieldDisplayValue(f config.FieldInfo) string {
	if f.Sensitive {
		if strings.TrimSpace(f.Value) == "" {
			return "(not set)"
		}
		return "****"
	}
	if strings.TrimSpace(f.Value) == "" {
		return "(not set)"
	}
	return f.Value
}

func (m model) visibleRows() int {
	maxRows := m.height - 12
	if m.showHelp {
		maxRows -= 4
	}
	if maxRows < 5 {
		maxRows = 5
	}
	return maxRows
}

func (m model) visibleRange(maxRows int) (int, int) {
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.fields) {
		end = len(m.fields)
	}
	return start, end
}

func settingsContentWidth(viewWidth int) int {
	menuHorizChrome := 6
	if viewWidth >= 56 {
		menuHorizChrome = 10
	}
	return clampInt(viewWidth-menuHorizChrome, 28, 108)
}

func settingsPanelWidth(viewWidth, contentWidth int) int {
	menuHorizChrome := 6
	if viewWidth >= 56 {
		menuHorizChrome = 10
	}
	return contentWidth + menuHorizChrome - 2
}

func composeWithPinnedFooter(body, footer string, height int) string {
	if height <= 0 {
		return joinVerticalNonEmpty(lipgloss.Left, body, footer)
	}
	bodyLines := splitLines(body)
	footerLines := splitLines(footer)
	if len(footerLines) >= height {
		return strings.Join(footerLines[:height], "\n")
	}
	bodyBudget := height - len(footerLines)
	if bodyBudget < len(bodyLines) {
		if bodyBudget <= 0 {
			bodyLines = nil
		} else {
			bodyLines = bodyLines[:bodyBudget]
		}
	}
	lines := make([]string, 0, len(bodyLines)+len(footerLines))
	lines = append(lines, bodyLines...)
	lines = append(lines, footerLines...)
	return strings.Join(lines, "\n")
}

func joinVerticalNonEmpty(pos lipgloss.Position, items ...string) string {
	nonEmpty := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		nonEmpty = append(nonEmpty, item)
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(pos, nonEmpty...)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func truncateText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= width {
		return string(r)
	}
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}

func fitText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= width {
		return string(r) + strings.Repeat(" ", width-len(r))
	}
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
