package tempest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/breeze"
	"github.com/alibilge/dirstral-cli/internal/mcp"
	"github.com/alibilge/dirstral-cli/internal/ui"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	stateIdle appState = iota
	stateRecording
	stateTranscribing
	stateThinking
	stateSpeaking
)

type tempestModel struct {
	opts     Options
	client   *mcp.Client
	ctx      context.Context
	viewport viewport.Model
	spinner  spinner.Model
	state    appState
	messages []string
	ready    bool
	width    int
	height   int
	showHelp bool
}

type recordDoneMsg struct {
	path string
	err  error
}

type transcribeDoneMsg struct {
	text string
	err  error
}

type thinkDoneMsg struct {
	answer string
	err    error
}

type speakDoneMsg struct {
	err error
}

func initialModel(ctx context.Context, client *mcp.Client, opts Options) tempestModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ClrBrand)

	msgs := []string{
		ui.Info("Connected to", opts.MCPURL),
		ui.Dim("Tempest mode: Press Enter to record (6s), Esc/Ctrl+C to quit."),
	}

	return tempestModel{
		opts:     opts,
		client:   client,
		ctx:      ctx,
		spinner:  s,
		messages: msgs,
		state:    stateIdle,
	}
}

func (m tempestModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m tempestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "?" || msg.String() == "ctrl+k" {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			switch msg.String() {
			case "esc", "q", "?", "ctrl+k":
				m.showHelp = false
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.state == stateIdle {
				m.state = stateRecording
				m.scrollToBottom()
				return m, tea.Batch(m.recordCmd(), m.spinner.Tick)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 0
		footerHeight := 2 // status line + margin
		vpWidth := maxInt(msg.Width-2, 20)
		vpHeight := maxInt(msg.Height-footerHeight-headerHeight, 4)

		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = vpHeight
		}

	case recordDoneMsg:
		if msg.err != nil {
			m.messages = append(m.messages, ui.Errorf("recording: %v", msg.err))
			m.state = stateIdle
			m.scrollToBottom()
			return m, nil
		}
		m.state = stateTranscribing
		m.scrollToBottom()
		return m, m.transcribeCmd(msg.path)

	case transcribeDoneMsg:
		if msg.err != nil {
			m.messages = append(m.messages, ui.Errorf("transcription: %v", msg.err))
			m.state = stateIdle
			m.scrollToBottom()
			return m, nil
		}
		m.messages = append(m.messages, ui.Dim("you said: ")+msg.text)
		m.state = stateThinking
		m.scrollToBottom()
		return m, m.thinkCmd(msg.text)

	case thinkDoneMsg:
		if msg.err != nil {
			m.messages = append(m.messages, ui.Errorf("tool call: %v", msg.err))
			m.state = stateIdle
			m.scrollToBottom()
			return m, nil
		}
		m.messages = append(m.messages, ui.Brand.Render("assistant: ")+msg.answer)

		if m.opts.Mute {
			m.state = stateIdle
			m.scrollToBottom()
			return m, nil
		}

		m.state = stateSpeaking
		m.scrollToBottom()
		return m, m.speakCmd(msg.answer)

	case speakDoneMsg:
		if msg.err != nil {
			m.messages = append(m.messages, ui.Errorf("playback: %v", msg.err))
		}
		m.state = stateIdle
		m.scrollToBottom()
		return m, nil
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(vpCmd, spCmd)
}

func (m *tempestModel) scrollToBottom() {
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

func (m tempestModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	if m.showHelp {
		helpText := strings.Join([]string{
			ui.Brand.Render("Tempest Keymap"),
			ui.Muted.Render("enter  start recording"),
			ui.Muted.Render("esc    quit"),
			ui.Muted.Render("?      toggle help"),
		}, "\n")
		help := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ui.ClrSubtle).Padding(0, 1).MaxWidth(maxInt(m.width-2, 24)).Render(helpText)
		b.WriteString(help)
		b.WriteString("\n\n")
	}

	status := ""
	switch m.state {
	case stateIdle:
		status = ui.Subtle.Render("[ Idle: Press Enter to record ]")
	case stateRecording:
		status = ui.Red.Render(m.spinner.View() + " [🔴 Recording audio...]")
	case stateTranscribing:
		status = ui.Yellow.Render(m.spinner.View() + " [⏳ Transcribing...]")
	case stateThinking:
		status = ui.Cyan.Render(m.spinner.View() + " [🧠 Asking dir2mcp...]")
	case stateSpeaking:
		status = ui.Green.Render(m.spinner.View() + " [🔊 Speaking...]")
	}

	b.WriteString(lipgloss.NewStyle().MaxWidth(maxInt(m.width-2, 20)).Render(status))
	b.WriteString("\n")
	b.WriteString(ui.Dim("? help"))
	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *tempestModel) recordCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := recordAudio(m.ctx, m.opts.Device)
		return recordDoneMsg{path: path, err: err}
	}
}

func (m *tempestModel) transcribeCmd(path string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			_ = os.Remove(path)
		}()
		text, err := transcribeElevenLabs(m.ctx, m.opts.BaseURL, path)
		return transcribeDoneMsg{text: text, err: err}
	}
}

func (m *tempestModel) thinkCmd(question string) tea.Cmd {
	return func() tea.Msg {
		parsed := breeze.ParseInput(question, m.opts.Model)
		if breeze.RequiresApproval(parsed.Tool) {
			return thinkDoneMsg{err: mcpErrApprovalRequired(parsed.Tool)}
		}
		execRes, err := breeze.ExecuteParsed(m.ctx, m.client, parsed)
		if err != nil {
			return thinkDoneMsg{err: err}
		}
		return thinkDoneMsg{answer: strings.TrimSpace(execRes.Output)}
	}
}

func mcpErrApprovalRequired(tool string) error {
	return fmt.Errorf("tempest requires manual confirmation for tool %s; use breeze for interactive approval", tool)
}

func (m *tempestModel) speakCmd(text string) tea.Cmd {
	return func() tea.Msg {
		voiceID := m.opts.Voice
		if voiceID == "" {
			voiceID = "Rachel"
		}
		path, err := synthesizeElevenLabs(m.ctx, m.opts.BaseURL, voiceID, text)
		if err != nil {
			return speakDoneMsg{err: err}
		}
		defer func() {
			_ = os.Remove(path)
		}()
		err = playAudio(m.ctx, path)
		return speakDoneMsg{err: err}
	}
}
