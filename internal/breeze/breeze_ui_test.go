package breeze

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func updateBreezeModel(t *testing.T, m breezeModel, msg tea.Msg) (breezeModel, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	bm, ok := next.(breezeModel)
	if !ok {
		t.Fatalf("expected breezeModel, got %T", next)
	}
	return bm, cmd
}

func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

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
