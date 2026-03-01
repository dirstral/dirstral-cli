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

// SetIntro replaces the intro lines shown above the menu.
func (m *MenuModel) SetIntro(intro []string) {
	m.config.Intro = intro
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
	if m.width == 0 {
		return ""
	}

	var headerItems []string
	if m.config.Title != "" {
		headerItems = append(headerItems, styleTitle.Render(m.config.Title))
	}
	for _, line := range m.config.Intro {
		headerItems = append(headerItems, styleMuted.Render(line))
	}
	header := lipgloss.JoinVertical(lipgloss.Center, headerItems...)

	var menuLines []string
	maxLabelWidth := 0
	for _, item := range m.config.Items {
		l := lipgloss.Width(item.Label)
		if item.Badge != "" {
			l += lipgloss.Width(" [" + item.Badge + "]")
		}
		if l > maxLabelWidth {
			maxLabelWidth = l
		}
	}

	showCount := len(m.config.Items)
	if m.revealedCount >= 0 {
		showCount = m.revealedCount
	}

	for i, item := range m.config.Items {
		if i >= showCount {
			break
		}

		isSelected := i == m.cursor

		badgeStr := ""
		if item.Badge != "" {
			if isSelected {
				badgeStr = " [" + item.Badge + "]"
			} else {
				bs := styleSubtle
				if item.BadgeStyle != nil {
					bs = *item.BadgeStyle
				}
				badgeStr = " " + bs.Render("["+item.Badge+"]")
			}
		}

		padding := maxLabelWidth - lipgloss.Width(item.Label)
		if item.Badge != "" {
			padding = maxLabelWidth - (lipgloss.Width(item.Label) + lipgloss.Width(" ["+item.Badge+"]"))
		}
		if padding < 0 {
			padding = 0
		}

		paddedLabel := item.Label + badgeStr + strings.Repeat(" ", padding)

		if isSelected {
			row := fmt.Sprintf("  %s %s", styleSelected.Render("›"), styleSelectedRow.Render(" "+paddedLabel+" ")+"   "+styleSelectedDesc.Render(item.Description))
			menuLines = append(menuLines, row)
		} else {
			row := fmt.Sprintf("    %s   %s", styleMuted.Render(paddedLabel), styleDescription.Render(item.Description))
			menuLines = append(menuLines, row)
		}
	}

	menuBox := styleMenuBox.Render(lipgloss.JoinVertical(lipgloss.Left, menuLines...))
	footer := styleSubtle.Render(m.config.Controls)

	content := lipgloss.JoinVertical(lipgloss.Center, header, menuBox, footer)

	if m.config.ShowLogo {
		logo := RenderLogo(m.width)
		var b strings.Builder
		b.WriteString(logo)
		b.WriteByte('\n')
		b.WriteByte('\n')

		contentLines := strings.Split(content, "\n")
		tier := ChooseTier(m.width)
		if tier == LogoCompact {
			for _, line := range contentLines {
				b.WriteString(padLine(line, compactLeftPad))
				b.WriteByte('\n')
			}
		} else {
			for _, line := range centerBlockLines(contentLines, m.width) {
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
		return b.String()
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
