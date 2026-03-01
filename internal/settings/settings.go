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
		return m, nil
	case tea.KeyMsg:
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
			if err := config.SaveSecret(f.Key, f.Value); err != nil {
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

// ── View ──────────────────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.ClrBrand).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.ClrBrand)

	sourceStyle = lipgloss.NewStyle().
			Foreground(ui.ClrMuted).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(ui.ClrMuted).
			MarginTop(1)
)

func (m model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("⚙  Settings"))
	b.WriteString("\n")

	// Visible rows (leave room for header, footer, status).
	maxRows := m.height - 8
	if maxRows < 5 {
		maxRows = 5
	}

	// Scrolling window.
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.fields) {
		end = len(m.fields)
	}

	for i := start; i < end; i++ {
		f := m.fields[i]
		cursor := "  "
		if i == m.cursor {
			cursor = ui.Brand.Render("▸ ")
		}

		displayVal := f.Value
		if f.Sensitive && f.Value != "" {
			displayVal = "****"
		}
		if f.Value == "" {
			displayVal = ui.Muted.Render("(not set)")
		}

		keyStr := f.Key
		if i == m.cursor {
			keyStr = selectedStyle.Render(f.Key)
		}

		src := sourceStyle.Render(fmt.Sprintf("(%s)", f.Source))

		b.WriteString(fmt.Sprintf("%s%-25s %s %s\n", cursor, keyStr, displayVal, src))
	}

	// Scroll indicator.
	if len(m.fields) > maxRows {
		b.WriteString(ui.Muted.Render(fmt.Sprintf("  [%d/%d]\n", m.cursor+1, len(m.fields))))
	}

	b.WriteString("\n")

	// State-specific footer.
	switch m.state {
	case stateEditing:
		b.WriteString(fmt.Sprintf("  Editing %s:\n", ui.Brand.Render(m.fields[m.cursor].Key)))
		b.WriteString("  " + m.input.View() + "\n")
		if m.errMsg != "" {
			b.WriteString("  " + ui.Red.Render(m.errMsg) + "\n")
		}
		b.WriteString(footerStyle.Render("  enter: confirm  esc: cancel"))
	case stateConfirmQuit:
		b.WriteString(ui.Yellow.Render("  You have unsaved changes. Save before quitting?") + "\n")
		if m.errMsg != "" {
			b.WriteString("  " + ui.Red.Render(m.errMsg) + "\n")
		}
		b.WriteString(footerStyle.Render("  y: save & quit  n: discard & quit  c/esc: cancel"))
	default:
		if m.errMsg != "" {
			b.WriteString("  " + ui.Red.Render(m.errMsg) + "\n")
		}
		if m.statusMsg != "" {
			b.WriteString("  " + ui.Green.Render(m.statusMsg) + "\n")
		}
		b.WriteString(footerStyle.Render("  ↑/↓: navigate  enter: edit  r: reset  s: save  q: quit"))
	}

	return b.String()
}
