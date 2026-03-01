package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem describes a single menu entry.
type MenuItem struct {
	Label       string
	Description string
	Value       string
	// Badge is optional text rendered after the label (e.g. "[running]").
	Badge string
	// BadgeStyle is the lipgloss style for the badge. If nil, styleSubtle is used.
	BadgeStyle *lipgloss.Style
}

// MenuConfig holds static data for a menu screen.
type MenuConfig struct {
	Title    string
	Intro    []string
	Items    []MenuItem
	ShowLogo bool
	Controls string
}

// MenuModel is a bubbletea Model for an interactive menu.
type MenuModel struct {
	config        MenuConfig
	cursor        int
	chosen        string
	width         int
	height        int
	quitted       bool
	revealedCount int  // how many items have been revealed (-1 = all)
	animate       bool // whether staggered reveal is active
}

// NewMenuModel creates a MenuModel from a config.
func NewMenuModel(cfg MenuConfig) MenuModel {
	if cfg.Controls == "" {
		cfg.Controls = "arrows navigate · enter select · q/esc back"
	}
	animate := animationsEnabled()
	revealed := -1 // show all by default
	if animate {
		revealed = 0
	}
	return MenuModel{
		config:        cfg,
		width:         DefaultTerminalWidth,
		animate:       animate,
		revealedCount: revealed,
	}
}

// Chosen returns the selected value after the model quits.
func (m MenuModel) Chosen() string { return m.chosen }

// Quitted returns whether the user quit/escaped.
func (m MenuModel) Quitted() bool { return m.quitted }

// SetItems replaces the menu items (used for dynamic updates).
func (m *MenuModel) SetItems(items []MenuItem) {
	m.config.Items = items
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m MenuModel) Init() tea.Cmd {
	if m.animate {
		return tickReveal()
	}
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case revealTickMsg:
		if m.revealedCount >= 0 && m.revealedCount < len(m.config.Items) {
			m.revealedCount++
			if m.revealedCount < len(m.config.Items) {
				return m, tickReveal()
			}
			// All revealed — mark complete.
			m.revealedCount = -1
		}
		return m, nil

	case tea.KeyMsg:
		// If still revealing, skip to showing all items on any key press.
		if m.revealedCount >= 0 {
			m.revealedCount = -1
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.config.Items) - 1
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.config.Items) {
				m.cursor = 0
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.config.Items) {
				m.chosen = m.config.Items[m.cursor].Value
			}
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quitted = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder

	// Logo
	if m.config.ShowLogo {
		b.WriteString(RenderLogo(m.width))
		b.WriteByte('\n')
		b.WriteByte('\n')
	}

	// Collect body lines for centering
	var body []string

	// Title
	if m.config.Title != "" {
		body = append(body, styleBrandStrong.Render(m.config.Title))
	}

	// Intro
	for _, line := range m.config.Intro {
		body = append(body, styleMuted.Render(line))
	}
	body = append(body, "") // spacer

	// Determine how many items to show.
	showCount := len(m.config.Items)
	if m.revealedCount >= 0 {
		showCount = m.revealedCount
	}

	// Menu items
	for i, item := range m.config.Items {
		if i >= showCount {
			break
		}

		cursor := "  "
		labelStyle := styleMuted
		descStyle := styleDescription
		if i == m.cursor {
			cursor = styleBrandStrong.Render("> ")
			labelStyle = styleSelected
			descStyle = styleSelectedDesc
		}

		line := cursor + labelStyle.Render(item.Label)

		// Badge (e.g. "[running]")
		if item.Badge != "" {
			bs := styleSubtle
			if item.BadgeStyle != nil {
				bs = *item.BadgeStyle
			}
			line += " " + bs.Render(fmt.Sprintf("[%s]", item.Badge))
		}

		if item.Description != "" {
			line += "  " + descStyle.Render(item.Description)
		}
		body = append(body, line)
	}

	body = append(body, "") // spacer
	body = append(body, styleSubtle.Render(m.config.Controls))

	// Center the body block
	tier := ChooseTier(m.width)
	if tier == LogoCompact {
		for _, line := range body {
			b.WriteString(padLine(line, compactLeftPad))
			b.WriteByte('\n')
		}
	} else {
		for _, line := range centerBlockLines(body, m.width) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	return b.String()
}
