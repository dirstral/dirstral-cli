package tempest

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestTempestUpdateWindowSizeIgnoresRepeatedTinyResize verifies tiny resize events do not collapse layout.
func TestTempestUpdateWindowSizeIgnoresRepeatedTinyResize(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	got := updated.(tempestModel)

	if !got.ready {
		t.Fatalf("expected model to be ready after first window size")
	}
	expectedWidth := 78
	expectedHeight := 22
	if got.viewport.Width != expectedWidth || got.viewport.Height != expectedHeight {
		t.Fatalf("expected initial viewport %dx%d, got %dx%d", expectedWidth, expectedHeight, got.viewport.Width, got.viewport.Height)
	}

	updated, _ = got.Update(tea.WindowSizeMsg{Width: 1, Height: 1})
	got = updated.(tempestModel)
	if got.viewport.Width != expectedWidth || got.viewport.Height != expectedHeight {
		t.Fatalf("expected viewport to ignore tiny resize, got %dx%d", got.viewport.Width, got.viewport.Height)
	}

	updated, _ = got.Update(tea.WindowSizeMsg{Width: 2, Height: 2})
	got = updated.(tempestModel)
	if got.viewport.Width != expectedWidth || got.viewport.Height != expectedHeight {
		t.Fatalf("expected viewport to stay stable on repeated tiny resize, got %dx%d", got.viewport.Width, got.viewport.Height)
	}
}

// TestTempestHelpToggleParityWithCtrlKAndEnterBlockedWhenOpen verifies toggle-key parity and input blocking.
// This intentionally runs before any WindowSizeMsg to cover pre-initialized model behavior from initialModel.
func TestTempestHelpToggleParityWithCtrlKAndEnterBlockedWhenOpen(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	got := updated.(tempestModel)
	if !got.showHelp {
		t.Fatalf("expected help to be shown after ctrl+k toggle")
	}

	updated, cmd := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(tempestModel)
	if cmd != nil {
		t.Fatalf("expected no command when pressing enter with help open")
	}
	if got.state != stateIdle {
		t.Fatalf("expected state to stay idle with help open, got %v", got.state)
	}
	if !got.showHelp {
		t.Fatalf("expected help to remain open after enter press")
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got = updated.(tempestModel)
	if got.showHelp {
		t.Fatalf("expected help to be hidden after '?' toggle")
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got = updated.(tempestModel)
	if !got.showHelp {
		t.Fatalf("expected help to be shown after '?' toggle")
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	got = updated.(tempestModel)
	if got.showHelp {
		t.Fatalf("expected help to be hidden after ctrl+k toggle")
	}
}

// TestTempestViewIncludesHelpHintText ensures the help discoverability hint is visible.
func TestTempestViewIncludesHelpHintText(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	got := updated.(tempestModel)

	view := got.View()
	if !strings.Contains(view, "? help") {
		t.Fatalf("expected view to include help hint text, got: %q", view)
	}
}

func TestTempestViewNarrowWidthKeepsHelpAndStatusRenderable(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 22, Height: 8})
	got := updated.(tempestModel)

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got = updated.(tempestModel)

	view := got.View()
	if !strings.Contains(view, "Tempest Keymap") {
		t.Fatalf("expected narrow view to include help content")
	}
	if !strings.Contains(view, "Idle") {
		t.Fatalf("expected narrow view to include status content")
	}

	maxWidth := got.renderWidth()
	for _, line := range strings.Split(view, "\n") {
		if lipgloss.Width(line) > maxWidth {
			t.Fatalf("line exceeds render width %d: %q", maxWidth, line)
		}
	}
}
