package breeze

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/mcp"
	"github.com/alibilge/dirstral-cli/internal/ui"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mcpResponseMsg struct {
	output string
	err    error
}

type breezeModel struct {
	client    *mcp.Client
	ctx       context.Context
	viewport  viewport.Model
	textInput textinput.Model
	spinner   spinner.Model
	err       error
	messages  []string
	isLoading bool
	ready     bool
	width     int
	height    int

	// For confirmation
	confirmingTool string
	confirmArgs    map[string]any
}

func initialModel(ctx context.Context, client *mcp.Client, url string, session string) breezeModel {
	ti := textinput.New()
	ti.Placeholder = "Ask a question or type /help..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ClrBrand)

	msgs := []string{
		ui.Info("Connected to", url),
		ui.Info("Session:", session),
		ui.Dim("Type /help for commands, /quit to exit."),
	}

	return breezeModel{
		client:    client,
		ctx:       ctx,
		textInput: ti,
		spinner:   s,
		messages:  msgs,
	}
}

func (m breezeModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m breezeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.isLoading {
				return m, nil
			}

			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}
			m.textInput.SetValue("")

			if m.confirmingTool != "" {
				inputLower := strings.ToLower(input)
				if inputLower == "y" || inputLower == "yes" {
					m.messages = append(m.messages, ui.Brand.Render("Approving "+m.confirmingTool+"..."))
					m.isLoading = true
					tool := m.confirmingTool
					args := m.confirmArgs
					m.confirmingTool = ""
					m.confirmArgs = nil
					m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
					m.viewport.GotoBottom()
					return m, tea.Batch(m.runToolCmd(tool, args), m.spinner.Tick)
				} else {
					m.messages = append(m.messages, ui.Dim("Cancelled "+m.confirmingTool+"."))
					m.confirmingTool = ""
					m.confirmArgs = nil
					m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
					m.viewport.GotoBottom()
					return m, nil
				}
			}

			m.messages = append(m.messages, ui.Prompt("breeze")+input)

			if input == "/quit" || input == "/exit" {
				return m, tea.Quit
			}

			if input == "/help" {
				m.messages = append(m.messages, formatHelp())
				m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
				m.viewport.GotoBottom()
				return m, nil
			}

			m.isLoading = true
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.viewport.GotoBottom()

			return m, tea.Batch(m.processInputCmd(input), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 0
		footerHeight := 3 // input line + some margin
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case mcpResponseMsg:
		m.isLoading = false
		if msg.err != nil {
			m.messages = append(m.messages, ui.Errorf("%v", msg.err))
		} else if msg.output != "" {
			m.messages = append(m.messages, msg.output)
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()
		return m, nil
	}

	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m breezeModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	if m.confirmingTool != "" {
		b.WriteString(fmt.Sprintf("%s %s %s", ui.Yellow.Render("Run tool"), ui.Brand.Render(m.confirmingTool+"?"), ui.Dim("[y/N]: ")))
		b.WriteString(m.textInput.View())
	} else {
		if m.isLoading {
			b.WriteString(m.spinner.View() + " ")
		} else {
			b.WriteString(ui.Prompt("breeze"))
		}
		b.WriteString(m.textInput.View())
	}

	return b.String()
}

func (m *breezeModel) processInputCmd(input string) tea.Cmd {
	return func() tea.Msg {
		switch {
		case strings.HasPrefix(input, "/list"):
			prefix := strings.TrimSpace(strings.TrimPrefix(input, "/list"))
			return m.checkApprovalAndRun("dir2mcp.list_files", map[string]any{"path_prefix": prefix, "limit": 30})
		case strings.HasPrefix(input, "/search "):
			query := strings.TrimSpace(strings.TrimPrefix(input, "/search"))
			return m.checkApprovalAndRun("dir2mcp.search", map[string]any{"query": query, "k": 8})
		case strings.HasPrefix(input, "/open "):
			args := strings.Fields(strings.TrimPrefix(input, "/open"))
			if len(args) == 0 {
				return mcpResponseMsg{output: ui.Dim("usage: /open <rel_path>")}
			}
			return m.checkApprovalAndRun("dir2mcp.open_file", map[string]any{"rel_path": args[0]})
		default:
			return m.checkApprovalAndRun("dir2mcp.ask", map[string]any{"question": input, "k": 8})
		}
	}
}

func (m *breezeModel) checkApprovalAndRun(tool string, args map[string]any) tea.Msg {
	if !autoApprove[tool] {
		// Needs approval, we handle this by mutating state directly isn't safe from Cmd,
		// but since this is called from a tea.Cmd, we actually want to return a message that triggers approval.
		// Wait, instead of mutating, we can return a specific Msg to trigger the approval prompt.
		// Let's create an approvalReqMsg
		return approvalReqMsg{tool: tool, args: args}
	}
	res, err := m.client.CallTool(m.ctx, tool, args)
	if err != nil {
		return mcpResponseMsg{err: err}
	}
	output := renderResultString(tool, res)
	if res.IsError {
		output = ui.Errorf("%s returned an error\n%s", tool, output)
	}
	return mcpResponseMsg{output: output}
}

func (m *breezeModel) runToolCmd(tool string, args map[string]any) tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.CallTool(m.ctx, tool, args)
		if err != nil {
			return mcpResponseMsg{err: err}
		}
		output := renderResultString(tool, res)
		if res.IsError {
			output = ui.Errorf("%s returned an error\n%s", tool, output)
		}
		return mcpResponseMsg{output: output}
	}
}

type approvalReqMsg struct {
	tool string
	args map[string]any
}

func formatHelp() string {
	var b strings.Builder
	b.WriteString(ui.Brand.Render("Commands:\n"))
	b.WriteString(fmt.Sprintf("  %s  %s\n", ui.Keyword.Render("/help"), ui.Muted.Render("Show help")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", ui.Keyword.Render("/quit"), ui.Muted.Render("Exit Breeze")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", ui.Keyword.Render("/list [prefix]"), ui.Muted.Render("List indexed files")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", ui.Keyword.Render("/search <query>"), ui.Muted.Render("Search corpus")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", ui.Keyword.Render("/open <rel_path>"), ui.Muted.Render("Open file from index")))
	b.WriteString(ui.Dim("  Any other text is sent to dir2mcp.ask"))
	return b.String()
}

func renderResultString(tool string, res *mcp.ToolCallResult) string {
	switch tool {
	case "dir2mcp.list_files":
		return renderListFilesString(res.StructuredContent)
	case "dir2mcp.search":
		return renderSearchString(res.StructuredContent)
	case "dir2mcp.open_file":
		return renderOpenFileString(res.StructuredContent)
	case "dir2mcp.ask":
		return renderAskString(res.StructuredContent)
	default:
		var b strings.Builder
		for _, c := range res.Content {
			if c.Text != "" {
				b.WriteString(c.Text + "\n")
			}
		}
		return b.String()
	}
}

func renderListFilesString(sc map[string]any) string {
	files, _ := sc["files"].([]any)
	if len(files) == 0 {
		return ui.Dim("(no files)")
	}
	var b strings.Builder
	for i, f := range files {
		if i >= 20 {
			b.WriteString(ui.Dim("...\n"))
			break
		}
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", ui.Cyan.Render(asString(m["rel_path"])), ui.Dim("("+asString(m["doc_type"])+")")))
	}
	return b.String()
}

func renderSearchString(sc map[string]any) string {
	hits, _ := sc["hits"].([]any)
	if len(hits) == 0 {
		return ui.Dim("(no hits)")
	}
	var b strings.Builder
	for i, h := range hits {
		if i >= 8 {
			break
		}
		m, ok := h.(map[string]any)
		if !ok {
			continue
		}
		path := asString(m["rel_path"])
		snippet := strings.TrimSpace(asString(m["snippet"]))
		score := m["score"]
		citation := ""
		if span, ok := m["span"].(map[string]any); ok {
			citation = mcp.CitationForSpan(path, span)
		}
		b.WriteString(fmt.Sprintf("%s score=%s %s\n", ui.Brand.Render(fmt.Sprintf("%d)", i+1)), ui.Score(score), ui.Citation(citation)))
		if snippet != "" {
			b.WriteString(fmt.Sprintf("   %s\n", ui.Muted.Render(snippet)))
		}
	}
	return b.String()
}

func renderOpenFileString(sc map[string]any) string {
	path := asString(sc["rel_path"])
	content := asString(sc["content"])
	var b strings.Builder
	if span, ok := sc["span"].(map[string]any); ok {
		b.WriteString(ui.Citation(mcp.CitationForSpan(path, span)) + "\n")
	}
	b.WriteString(content)
	return b.String()
}

func renderAskString(sc map[string]any) string {
	var b strings.Builder
	answer := strings.TrimSpace(asString(sc["answer"]))
	if answer != "" {
		b.WriteString(answer + "\n")
	}
	if citations, ok := sc["citations"].([]any); ok && len(citations) > 0 {
		seen := map[string]bool{}
		ordered := []string{}
		for _, it := range citations {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			path := asString(m["rel_path"])
			span, _ := m["span"].(map[string]any)
			c := mcp.CitationForSpan(path, span)
			if !seen[c] {
				seen[c] = true
				ordered = append(ordered, c)
			}
		}
		sort.Strings(ordered)
		if len(ordered) > 0 {
			styled := make([]string, len(ordered))
			for i, c := range ordered {
				styled[i] = ui.Citation(c)
			}
			b.WriteString(fmt.Sprintf("%s %s\n", ui.Dim("Sources:"), strings.Join(styled, ui.Dim(", "))))
		}
	}
	return strings.TrimSpace(b.String())
}
