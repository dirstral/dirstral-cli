package breeze

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// updateBreezeModel applies a tea message and type-asserts the next model.
func updateBreezeModel(t *testing.T, m breezeModel, msg tea.Msg) (breezeModel, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	bm, ok := next.(breezeModel)
	if !ok {
		t.Fatalf("expected breezeModel, got %T", next)
	}
	return bm, cmd
}

// keyMsg builds a rune-based key message for tests.
func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// ctrlKMsg builds a Ctrl+K key message for help-overlay tests.
func ctrlKMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlK}
}

// TestWindowSizeMsgAppliesMinimumViewportAndInputWidth validates baseline resize math.
func TestWindowSizeMsgAppliesMinimumViewportAndInputWidth(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		messages:  []string{"hello"},
	}

	m, _ = updateBreezeModel(t, m, tea.WindowSizeMsg{Width: 10, Height: 2})

	if !m.ready {
		t.Fatal("expected model to be ready after first window size update")
	}
	if got, want := m.viewport.Width, 20; got != want {
		t.Fatalf("viewport width mismatch: got %d want %d", got, want)
	}
	if got, want := m.viewport.Height, 4; got != want {
		t.Fatalf("viewport height mismatch: got %d want %d", got, want)
	}
	if got, want := m.textInput.Width, 16; got != want {
		t.Fatalf("text input width mismatch: got %d want %d", got, want)
	}

	m, _ = updateBreezeModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if got, want := m.viewport.Width, 98; got != want {
		t.Fatalf("viewport width mismatch after resize: got %d want %d", got, want)
	}
	if got, want := m.viewport.Height, 27; got != want {
		t.Fatalf("viewport height mismatch after resize: got %d want %d", got, want)
	}
	if got, want := m.textInput.Width, 84; got != want {
		t.Fatalf("text input width mismatch after resize: got %d want %d", got, want)
	}
}

// TestQuestionMarkTogglesHelpAndBlocksNormalProcessing validates '?' overlay behavior.
func TestQuestionMarkTogglesHelpAndBlocksNormalProcessing(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		viewport:  viewport.New(20, 4),
		ready:     true,
		messages:  []string{"existing message"},
	}
	m.textInput.SetValue("/quit")

	m, cmd := updateBreezeModel(t, m, keyMsg('?'))
	if cmd != nil {
		t.Fatal("expected no command when toggling help on")
	}
	if !m.showHelp {
		t.Fatal("expected help overlay to be visible after ?")
	}

	prevMsgCount := len(m.messages)
	m, cmd = updateBreezeModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected enter to be ignored while help is visible")
	}
	if !m.showHelp {
		t.Fatal("expected help overlay to remain visible after ignored key")
	}
	if got := len(m.messages); got != prevMsgCount {
		t.Fatalf("expected no new messages while help is visible: got %d want %d", got, prevMsgCount)
	}

	m, cmd = updateBreezeModel(t, m, keyMsg('?'))
	if cmd != nil {
		t.Fatal("expected no command when toggling help off")
	}
	if m.showHelp {
		t.Fatal("expected help overlay to be hidden after second ?")
	}
}

// TestCtrlKTogglesHelpAndBlocksNormalProcessing validates Ctrl+K parity with '?'.
func TestCtrlKTogglesHelpAndBlocksNormalProcessing(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		viewport:  viewport.New(20, 4),
		ready:     true,
		messages:  []string{"existing message"},
	}
	m.textInput.SetValue("/quit")

	m, cmd := updateBreezeModel(t, m, ctrlKMsg())
	if cmd != nil {
		t.Fatal("expected no command when toggling help on with ctrl+k")
	}
	if !m.showHelp {
		t.Fatal("expected help overlay to be visible after ctrl+k")
	}

	prevMsgCount := len(m.messages)
	m, cmd = updateBreezeModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected enter to be ignored while help is visible")
	}
	if !m.showHelp {
		t.Fatal("expected help overlay to remain visible after ignored key")
	}
	if got := len(m.messages); got != prevMsgCount {
		t.Fatalf("expected no new messages while help is visible: got %d want %d", got, prevMsgCount)
	}

	m, cmd = updateBreezeModel(t, m, ctrlKMsg())
	if cmd != nil {
		t.Fatal("expected no command when toggling help off with ctrl+k")
	}
	if m.showHelp {
		t.Fatal("expected help overlay to be hidden after second ctrl+k")
	}
}

// TestHelpOverlayCanBeClosedWithEitherToggleKey validates mixed-key open/close flows.
func TestHelpOverlayCanBeClosedWithEitherToggleKey(t *testing.T) {
	tests := []struct {
		name      string
		openKey   tea.KeyMsg
		closeKey  tea.KeyMsg
		closeHint string
	}{
		{
			name:      "open with question mark, close with ctrl+k",
			openKey:   keyMsg('?'),
			closeKey:  ctrlKMsg(),
			closeHint: "ctrl+k",
		},
		{
			name:      "open with ctrl+k, close with question mark",
			openKey:   ctrlKMsg(),
			closeKey:  keyMsg('?'),
			closeHint: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := breezeModel{
				textInput: textinput.New(),
				viewport:  viewport.New(20, 4),
				ready:     true,
			}

			m, _ = updateBreezeModel(t, m, tt.openKey)
			if !m.showHelp {
				t.Fatal("expected help overlay to be visible after open key")
			}

			m, cmd := updateBreezeModel(t, m, tt.closeKey)
			if cmd != nil {
				t.Fatalf("expected no command when closing help with %s", tt.closeHint)
			}
			if m.showHelp {
				t.Fatalf("expected help overlay to be hidden after %s", tt.closeHint)
			}
		})
	}
}

// TestViewIncludesHelpHintText ensures discoverability hint is always rendered.
func TestViewIncludesHelpHintText(t *testing.T) {
	m := breezeModel{
		viewport:  viewport.New(40, 10),
		textInput: textinput.New(),
		ready:     true,
	}
	m.viewport.SetContent("content")

	view := m.View()
	if !strings.Contains(view, "? help") {
		t.Fatalf("expected view to include help hint, got: %q", view)
	}
}
