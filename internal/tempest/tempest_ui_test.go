package tempest

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

func TestTempestHelpToggleAndEnterBlockedWhenHelpOpen(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got := updated.(tempestModel)
	if !got.showHelp {
		t.Fatalf("expected help to be shown after '?' toggle")
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
		t.Fatalf("expected help to be hidden after second '?' toggle")
	}
}

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
