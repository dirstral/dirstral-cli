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

func TestSettingsViewShowsUnsavedChangeList(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	m := initialModel(config.Default())
	m.width = 120
	m.height = 30

	modelIndex := findFieldIndex(t, m.fields, "model")
	m.fields[modelIndex].Value = "mistral-large-latest"
	m.recomputeDirty()

	view := m.View()
	if !strings.Contains(view, "Unsaved changes (1):") {
		t.Fatalf("expected unsaved changes header, got %q", view)
	}
	if !strings.Contains(view, "- model: mistral-small-latest -> mistral-large-latest") {
		t.Fatalf("expected unsaved model diff, got %q", view)
	}
}

func TestSettingsViewMasksSensitiveUnsavedValues(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	m := initialModel(config.Default())
	m.width = 120
	m.height = 30

	secretIndex := findFieldIndex(t, m.fields, "DIR2MCP_AUTH_TOKEN")
	m.fields[secretIndex].Value = "very-secret-token"
	m.recomputeDirty()

	view := m.View()
	if !strings.Contains(view, "- DIR2MCP_AUTH_TOKEN: (not set) -> ****") {
		t.Fatalf("expected masked unsaved secret diff, got %q", view)
	}
	if strings.Contains(view, "very-secret-token") {
		t.Fatalf("expected secret value to stay masked")
	}
}

func findFieldIndex(t *testing.T, fields []config.FieldInfo, key string) int {
	t.Helper()
	for i, f := range fields {
		if f.Key == key {
			return i
		}
	}
	t.Fatalf("missing field %q", key)
	return -1
}
