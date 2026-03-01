package tempest

import (
	"context"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/breeze"
	"github.com/alibilge/dirstral-cli/internal/mcp"
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

func TestTempestApprovalRequestMovesToConfirmingState(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com", Mute: true})
	m.parseInputFn = func(input, model string) breeze.ParsedInput {
		return breeze.ParsedInput{Tool: "dir2mcp.dangerous", Args: map[string]any{"value": "x"}}
	}
	m.executeParsed = func(ctx context.Context, client *mcp.Client, parsed breeze.ParsedInput) (*breeze.ToolExecution, error) {
		return &breeze.ToolExecution{Output: "unexpected"}, nil
	}

	updated, cmd := m.Update(transcribeDoneMsg{text: "run custom tool"})
	got := updated.(tempestModel)
	if got.state != stateThinking {
		t.Fatalf("expected thinking state before processing, got %v", got.state)
	}
	if cmd == nil {
		t.Fatalf("expected think command after transcript")
	}

	approval := cmd().(approvalReqMsg)
	updated, _ = got.Update(approval)
	got = updated.(tempestModel)

	if got.state != stateConfirming {
		t.Fatalf("expected confirming state, got %v", got.state)
	}
	if !got.hasPendingTool {
		t.Fatalf("expected pending approval tool")
	}
	if got.pendingParsed.Tool != "dir2mcp.dangerous" {
		t.Fatalf("expected pending tool dir2mcp.dangerous, got %q", got.pendingParsed.Tool)
	}
	if !strings.Contains(strings.Join(got.messages, "\n"), "Approval required for") {
		t.Fatalf("expected approval required message in transcript")
	}
}

func TestTempestApprovalYesRunsPendingTool(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com", Mute: true})
	m.parseInputFn = func(input, model string) breeze.ParsedInput {
		return breeze.ParsedInput{Tool: "dir2mcp.dangerous", Args: map[string]any{"value": "x"}}
	}

	callCount := 0
	m.executeParsed = func(ctx context.Context, client *mcp.Client, parsed breeze.ParsedInput) (*breeze.ToolExecution, error) {
		callCount++
		if parsed.Tool != "dir2mcp.dangerous" {
			t.Fatalf("expected approved tool to execute, got %q", parsed.Tool)
		}
		return &breeze.ToolExecution{Output: "approved output"}, nil
	}

	updated, thinkCmd := m.Update(transcribeDoneMsg{text: "run custom tool"})
	got := updated.(tempestModel)
	approval := thinkCmd().(approvalReqMsg)
	updated, _ = got.Update(approval)
	got = updated.(tempestModel)

	if callCount != 0 {
		t.Fatalf("expected no execution before user approval")
	}

	updated, runCmd := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = updated.(tempestModel)
	if runCmd == nil {
		t.Fatalf("expected command to run after approval")
	}
	if callCount != 0 {
		t.Fatalf("expected deferred execution until run command is processed")
	}

	for _, msg := range runCmdMessages(runCmd) {
		updated, _ = got.Update(msg)
		got = updated.(tempestModel)
	}
	if callCount != 1 {
		t.Fatalf("expected one execution after approval, got %d", callCount)
	}
	if got.state != stateIdle {
		t.Fatalf("expected idle state after approved execution in mute mode, got %v", got.state)
	}
	if !strings.Contains(strings.Join(got.messages, "\n"), "approved output") {
		t.Fatalf("expected approved output in transcript")
	}
}

func TestTempestApprovalNoCancelsPendingTool(t *testing.T) {
	m := initialModel(context.Background(), nil, Options{MCPURL: "http://example.com", Mute: true})
	m.parseInputFn = func(input, model string) breeze.ParsedInput {
		return breeze.ParsedInput{Tool: "dir2mcp.dangerous", Args: map[string]any{"value": "x"}}
	}

	callCount := 0
	m.executeParsed = func(ctx context.Context, client *mcp.Client, parsed breeze.ParsedInput) (*breeze.ToolExecution, error) {
		callCount++
		return &breeze.ToolExecution{Output: "should-not-run"}, nil
	}

	updated, thinkCmd := m.Update(transcribeDoneMsg{text: "run custom tool"})
	got := updated.(tempestModel)
	approval := thinkCmd().(approvalReqMsg)
	updated, _ = got.Update(approval)
	got = updated.(tempestModel)

	updated, cancelCmd := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got = updated.(tempestModel)

	if cancelCmd != nil {
		t.Fatalf("expected no command when cancelling approval")
	}
	if got.state != stateIdle {
		t.Fatalf("expected idle state after cancellation, got %v", got.state)
	}
	if got.hasPendingTool {
		t.Fatalf("expected pending tool to be cleared on cancellation")
	}
	if callCount != 0 {
		t.Fatalf("expected no execution after cancellation")
	}
	if !strings.Contains(strings.Join(got.messages, "\n"), "Cancelled dir2mcp.dangerous") {
		t.Fatalf("expected cancellation message in transcript")
	}
}

func runCmdMessages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	out := make([]tea.Msg, 0, len(batch))
	for _, sub := range batch {
		out = append(out, runCmdMessages(sub)...)
	}
	return out
}
