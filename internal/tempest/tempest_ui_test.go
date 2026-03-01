package tempest

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTempestUpdateWindowSizeAppliesSafeMinimumViewport(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com"})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 1, Height: 1})
	got := updated.(tempestModel)

	if !got.ready {
		t.Fatalf("expected model to be ready after first window size")
	}
	if got.viewport.Width != 20 {
		t.Fatalf("expected viewport width minimum 20, got %d", got.viewport.Width)
	}
	if got.viewport.Height != 4 {
		t.Fatalf("expected viewport height minimum 4, got %d", got.viewport.Height)
	}

	updated, _ = got.Update(tea.WindowSizeMsg{Width: 2, Height: 2})
	got = updated.(tempestModel)
	if got.viewport.Width != 20 || got.viewport.Height != 4 {
		t.Fatalf("expected viewport to keep minimum dimensions 20x4, got %dx%d", got.viewport.Width, got.viewport.Height)
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
