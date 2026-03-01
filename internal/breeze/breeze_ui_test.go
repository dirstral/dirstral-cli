package breeze

import (
	"errors"
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

func TestMCPErrorIncludesActionableHint(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		viewport:  viewport.New(40, 8),
		ready:     true,
		messages:  []string{"connected"},
	}

	m, _ = updateBreezeModel(t, m, mcpResponseMsg{err: errors.New("SESSION_NOT_FOUND")})
	joined := strings.Join(m.messages, "\n")
	if !strings.Contains(joined, "Hint:") {
		t.Fatalf("expected actionable hint in error output, got %q", joined)
	}
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
	if got, want := m.viewport.Width, 8; got != want {
		t.Fatalf("viewport width mismatch: got %d want %d", got, want)
	}
	if got, want := m.viewport.Height, 1; got != want {
		t.Fatalf("viewport height mismatch: got %d want %d", got, want)
	}
	if got, want := m.textInput.Width, 1; got != want {
		t.Fatalf("text input width mismatch: got %d want %d", got, want)
	}

	m, _ = updateBreezeModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if got, want := m.viewport.Width, 98; got != want {
		t.Fatalf("viewport width mismatch after resize: got %d want %d", got, want)
	}
	if got, want := m.viewport.Height, 28; got != want {
		t.Fatalf("viewport height mismatch after resize: got %d want %d", got, want)
	}
	if got, want := m.textInput.Width, 84; got != want {
		t.Fatalf("text input width mismatch after resize: got %d want %d", got, want)
	}
}

func TestWindowSizeMsgRepeatedTinyResizesStayBounded(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		messages:  []string{"hello"},
	}

	sizes := []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 9, Height: 3},
		{Width: 5, Height: 1},
		{Width: 100, Height: 30},
		{Width: 7, Height: 2},
	}

	for _, sz := range sizes {
		m, _ = updateBreezeModel(t, m, sz)
		if m.viewport.Width < 1 {
			t.Fatalf("viewport width should be >=1 after %+v, got %d", sz, m.viewport.Width)
		}
		if m.viewport.Height < 1 {
			t.Fatalf("viewport height should be >=1 after %+v, got %d", sz, m.viewport.Height)
		}
		if m.textInput.Width < 1 {
			t.Fatalf("text input width should be >=1 after %+v, got %d", sz, m.textInput.Width)
		}
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

func TestHelpToggleRecomputesViewportForSmallTerminal(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		messages:  []string{"hello"},
	}

	m, _ = updateBreezeModel(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	if got, want := m.viewport.Height, 8; got != want {
		t.Fatalf("unexpected viewport height before help: got %d want %d", got, want)
	}

	m, _ = updateBreezeModel(t, m, keyMsg('?'))
	if !m.showHelp {
		t.Fatal("expected help overlay enabled")
	}
	if m.viewport.Height >= 8 {
		t.Fatalf("expected viewport to shrink when help is shown, got %d", m.viewport.Height)
	}

	m, _ = updateBreezeModel(t, m, tea.WindowSizeMsg{Width: 22, Height: 5})
	if m.viewport.Height < 1 {
		t.Fatalf("expected viewport height >= 1 for tiny terminal, got %d", m.viewport.Height)
	}

	view := m.View()
	if !strings.Contains(view, "? help") {
		t.Fatalf("expected help/status hint to stay visible in tiny terminal, got: %q", view)
	}
	if !strings.Contains(view, "Help:") {
		t.Fatalf("expected compact help text in tiny terminal, got: %q", view)
	}
}

func TestClearCommandResetsMessagesToBanner(t *testing.T) {
	m := breezeModel{
		textInput: textinput.New(),
		viewport:  viewport.New(40, 8),
		ready:     true,
		modelName: "mistral-small-latest",
		banner:    []string{"connected"},
		messages:  []string{"connected", "old message"},
	}

	msg := m.processInputCmd("/clear")()
	resp, ok := msg.(mcpResponseMsg)
	if !ok {
		t.Fatalf("expected mcpResponseMsg, got %T", msg)
	}
	if !resp.clear {
		t.Fatal("expected /clear command to produce clear response")
	}

	m, _ = updateBreezeModel(t, m, resp)
	if got, want := len(m.messages), 1; got != want {
		t.Fatalf("message count after clear: got %d want %d", got, want)
	}
	if got := m.messages[0]; got != "connected" {
		t.Fatalf("expected banner message to remain, got %q", got)
	}
}
