package settings

import (
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsViewUsesMenuLikeChrome(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	m := initialModel(config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	view := updated.(model).View()

	if !strings.Contains(view, "Settings") {
		t.Fatalf("expected settings title in view")
	}
	if !strings.Contains(view, "Edit config and API defaults for Dirstral.") {
		t.Fatalf("expected menu-style intro in view")
	}
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Fatalf("expected bordered panel chrome in view")
	}
	if !strings.Contains(view, "? help") {
		t.Fatalf("expected controls footer to include help hint")
	}
}

func TestSettingsHelpOverlayToggle(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	m := initialModel(config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	m = updated.(model)

	if strings.Contains(m.View(), "Settings Keymap") {
		t.Fatalf("expected help overlay hidden by default")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	withHelp := updated.(model)
	if !strings.Contains(withHelp.View(), "Settings Keymap") {
		t.Fatalf("expected help overlay visible after ?")
	}

	updated, cmd := withHelp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatalf("expected q to close help overlay without quitting")
	}
	withoutHelp := updated.(model)
	if strings.Contains(withoutHelp.View(), "Settings Keymap") {
		t.Fatalf("expected help overlay hidden after q")
	}
}

func TestSettingsControlsReflectEditingState(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	m := initialModel(config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = updated.(model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	editing := updated.(model)
	if !strings.Contains(editing.View(), "enter confirm") {
		t.Fatalf("expected editing controls hint")
	}
}
