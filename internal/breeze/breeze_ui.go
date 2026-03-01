package breeze

import (
	"context"
	"fmt"
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
	quit   bool
	clear  bool
}

type breezeModel struct {
	client    *mcp.Client
	ctx       context.Context
	modelName string
	viewport  viewport.Model
	textInput textinput.Model
	spinner   spinner.Model
	messages  []string
	banner    []string
	isLoading bool
	ready     bool
	width     int
	height    int
	showHelp  bool

	// For confirmation
	confirmingTool string
	confirmArgs    map[string]any
}

func initialModel(ctx context.Context, client *mcp.Client, opts Options) breezeModel {
	ti := textinput.New()
	ti.Placeholder = "Ask a question or type /help..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ClrBrand)

	msgs := connectedBanner(opts.MCPURL, opts.Transport, client.SessionID(), opts.Model, opts.StartupHint)

	return breezeModel{
		client:    client,
		ctx:       ctx,
		modelName: opts.Model,
		textInput: ti,
		spinner:   s,
		messages:  msgs,
		banner:    append([]string(nil), msgs...),
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
		if msg.String() == "?" || msg.String() == "ctrl+k" {
			m.showHelp = !m.showHelp
			m.applyWindowSize(m.width, m.height)
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

			m.isLoading = true
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.viewport.GotoBottom()

			return m, tea.Batch(m.processInputCmd(input), m.spinner.Tick)
		}

	case tea.WindowSizeMsg:
		m.applyWindowSize(msg.Width, msg.Height)

	case mcpResponseMsg:
		m.isLoading = false
		if msg.quit {
			return m, tea.Quit
		}
		if msg.clear {
			m.messages = append([]string(nil), m.banner...)
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.viewport.GotoBottom()
			return m, nil
		}
		if msg.err != nil {
			errLine := ui.Errorf("%v", msg.err)
			if hint := mcp.ActionableMessageFromError(msg.err); hint != "" {
				errLine += "\n" + ui.Dim("Hint: "+hint)
			}
			m.messages = append(m.messages, errLine)
		} else if msg.output != "" {
			m.messages = append(m.messages, msg.output)
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()
		return m, nil

	case approvalReqMsg:
		m.isLoading = false
		m.confirmingTool = msg.tool
		m.confirmArgs = msg.args
		m.messages = append(m.messages, ui.Yellow.Render("Approval required for ")+ui.Brand.Render(msg.tool))
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
	b.WriteString("\n")
	if m.showHelp {
		help := m.renderHelpBlock(m.width, m.height)
		b.WriteString(help)
		b.WriteString("\n")
	}

	if m.confirmingTool != "" {
		prompt := fmt.Sprintf("%s %s %s", ui.Yellow.Render("Run tool"), ui.Brand.Render(m.confirmingTool+"?"), ui.Dim("[y/N]: "))
		b.WriteString(lipgloss.NewStyle().MaxWidth(maxInt(m.width-2, 20)).Render(prompt))
		b.WriteString(m.textInput.View())
	} else {
		if m.isLoading {
			b.WriteString(m.spinner.View() + " ")
		} else {
			b.WriteString(ui.Prompt("breeze"))
		}
		b.WriteString(m.textInput.View())
	}
	b.WriteString("\n")
	b.WriteString(ui.Dim("? help"))

	return b.String()
}

func (m *breezeModel) applyWindowSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	m.width = width
	m.height = height

	vpWidth := maxInt(width-2, 1)
	m.textInput.Width = maxInt(width-16, 1)

	reservedHeight := 2 // input row + status row
	if m.showHelp {
		helpBlock := m.renderHelpBlock(width, height)
		reservedHeight += lipgloss.Height(helpBlock) + 1
	}
	vpHeight := maxInt(height-reservedHeight, 1)

	if !m.ready {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.ready = true
		return
	}

	m.viewport.Width = vpWidth
	m.viewport.Height = vpHeight
}

func (m breezeModel) renderHelpBlock(width, height int) string {
	helpText := formatHelp()
	if width < 56 || height < 14 {
		helpText = "Help: /help, /quit, /clear, /list [prefix], /search <query>, /open <rel_path>. Press ? to close."
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ClrSubtle).
		Padding(0, 1).
		MaxWidth(maxInt(width-2, 1)).
		Render(helpText)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *breezeModel) processInputCmd(input string) tea.Cmd {
	return func() tea.Msg {
		plan, err := PlanTurn(input, m.modelName)
		if err != nil {
			return mcpResponseMsg{err: err}
		}
		if plan.Quit {
			return mcpResponseMsg{quit: true}
		}
		if plan.Help {
			return mcpResponseMsg{output: formatHelp()}
		}
		if plan.Clear {
			return mcpResponseMsg{clear: true}
		}
		if len(plan.Steps) == 0 {
			return mcpResponseMsg{}
		}
		return m.checkApprovalAndRunPlan(plan)
	}
}

func (m *breezeModel) checkApprovalAndRunPlan(plan TurnPlan) tea.Msg {
	for _, step := range plan.Steps {
		if needsApproval(step.Tool) {
			return approvalReqMsg{tool: step.Tool, args: step.Args}
		}
	}
	execRes, err := ExecutePlan(m.ctx, m.client, plan)
	if err != nil {
		return mcpResponseMsg{err: err}
	}
	return mcpResponseMsg{output: execRes.Output}
}

func (m *breezeModel) runToolCmd(tool string, args map[string]any) tea.Cmd {
	return func() tea.Msg {
		execRes, err := ExecuteParsed(m.ctx, m.client, ParsedInput{Tool: tool, Args: args})
		if err != nil {
			return mcpResponseMsg{err: err}
		}
		return mcpResponseMsg{output: execRes.Output}
	}
}

type approvalReqMsg struct {
	tool string
	args map[string]any
}

func formatHelp() string {
	var b strings.Builder
	b.WriteString(ui.Brand.Render("Commands:\n"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/help"), ui.Muted.Render("Show help"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/quit"), ui.Muted.Render("Exit Breeze"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/clear"), ui.Muted.Render("Clear chat history"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/list [prefix]"), ui.Muted.Render("List indexed files"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/search <query>"), ui.Muted.Render("Search corpus"))
	fmt.Fprintf(&b, "  %s  %s\n", ui.Keyword.Render("/open <rel_path>"), ui.Muted.Render("Open file from index"))
	b.WriteString(ui.Dim("  Any other text is sent to dir2mcp.ask"))
	return b.String()
}

func formatHelpPlain() string {
	return strings.Join([]string{
		"/help - Show help",
		"/quit - Exit Breeze",
		"/clear - Clear chat history",
		"/list [prefix] - List indexed files",
		"/search <query> - Search corpus",
		"/open <rel_path> - Open file from index",
		"Any other text is sent to dir2mcp.ask",
	}, "\n")
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
	total, _ := sc["total"].(float64)
	if len(files) == 0 {
		return ui.Dim("No files found.")
	}
	var b strings.Builder
	if total > 0 {
		b.WriteString(ui.Dim(fmt.Sprintf("Showing %d of %d files:\n", len(files), int(total))))
	}
	for i, f := range files {
		if i >= 30 {
			remaining := len(files) - 30
			b.WriteString(ui.Dim(fmt.Sprintf("  ... +%d more\n", remaining)))
			break
		}
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		path := asString(m["rel_path"])
		docType := asString(m["doc_type"])
		status := asString(m["status"])

		icon := "  "
		switch docType {
		case "code":
			icon = ui.Cyan.Render("  ")
		case "md", "text":
			icon = ui.Muted.Render("  ")
		case "pdf":
			icon = ui.Yellow.Render("  ")
		case "audio":
			icon = ui.Green.Render("  ")
		case "image":
			icon = ui.Brand.Render("  ")
		}

		line := fmt.Sprintf("%s%s", icon, ui.Cyan.Render(path))
		if docType != "" {
			line += " " + ui.Dim("("+docType+")")
		}
		switch status {
		case "error":
			line += " " + ui.Red.Render("[error]")
		case "skipped":
			line += " " + ui.Yellow.Render("[skipped]")
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func renderSearchString(sc map[string]any) string {
	hits, _ := sc["hits"].([]any)
	if len(hits) == 0 {
		return ui.Dim("No results found. Try a different query or broader search terms.")
	}
	var b strings.Builder
	for i, h := range hits {
		if i >= 8 {
			remaining := len(hits) - 8
			b.WriteString(ui.Dim(fmt.Sprintf("  ... +%d more results\n", remaining)))
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
		if i > 0 {
			b.WriteString(ui.Dim("  ───\n"))
		}
		fmt.Fprintf(&b, "  %s  %s  %s\n", ui.Brand.Render(fmt.Sprintf("#%d", i+1)), ui.Citation(citation), ui.Dim("score=")+ui.Score(score))
		if snippet != "" {
			// Indent and wrap long snippets
			lines := strings.Split(snippet, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					fmt.Fprintf(&b, "     %s\n", ui.Muted.Render(trimmed))
				}
			}
		}
	}
	return b.String()
}

func renderOpenFileString(sc map[string]any) string {
	path := asString(sc["rel_path"])
	content := asString(sc["content"])
	docType := asString(sc["doc_type"])
	truncated, _ := sc["truncated"].(bool)

	var b strings.Builder

	// Header with file info
	header := ui.Citation(path)
	if docType != "" {
		header += " " + ui.Dim("("+docType+")")
	}
	if span, ok := sc["span"].(map[string]any); ok {
		header = ui.Citation(mcp.CitationForSpan(path, span))
		if docType != "" {
			header += " " + ui.Dim("("+docType+")")
		}
	}
	b.WriteString(header + "\n")
	b.WriteString(ui.Dim("─────────────────────────────────────────") + "\n")

	// Content with line numbers for code
	if docType == "code" && content != "" {
		lines := strings.Split(content, "\n")
		startLine := 1
		if span, ok := sc["span"].(map[string]any); ok {
			if sl, ok := span["start_line"].(float64); ok {
				startLine = int(sl)
			}
		}
		for i, line := range lines {
			lineNum := ui.Dim(fmt.Sprintf("%4d ", startLine+i))
			b.WriteString(lineNum + line + "\n")
		}
	} else {
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
	}

	b.WriteString(ui.Dim("─────────────────────────────────────────"))
	if truncated {
		b.WriteString("\n" + ui.Yellow.Render("⚠ Output truncated (file exceeds max_chars limit)"))
	}
	return b.String()
}

func renderAskString(sc map[string]any) string {
	var b strings.Builder
	answer := strings.TrimSpace(asString(sc["answer"]))
	if answer != "" {
		b.WriteString(answer + "\n")
	} else {
		// Fallback for stub/empty answers
		b.WriteString(ui.Dim("(no answer generated — indexing may still be in progress)") + "\n")
	}
	if ordered := citationsFor("dir2mcp.ask", sc); len(ordered) > 0 {
		b.WriteString("\n" + ui.Dim("───── Sources ─────") + "\n")
		for i, c := range ordered {
			fmt.Fprintf(&b, "  %s %s\n", ui.Dim(fmt.Sprintf("[%d]", i+1)), ui.Citation(c))
		}
	}
	return strings.TrimSpace(b.String())
}
