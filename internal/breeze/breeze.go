package breeze

import (
	"context"
	"fmt"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/mcp"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	MCPURL    string
	Transport string
	Model     string
	Verbose   bool
}

var requiredTools = []string{
	"dir2mcp.list_files",
	"dir2mcp.search",
	"dir2mcp.open_file",
	"dir2mcp.stats",
	"dir2mcp.ask",
}

var autoApprove = map[string]bool{
	"dir2mcp.search":     true,
	"dir2mcp.ask":        true,
	"dir2mcp.ask_audio":  true,
	"dir2mcp.open_file":  true,
	"dir2mcp.list_files": true,
	"dir2mcp.stats":      true,
}

func Run(ctx context.Context, opts Options) error {
	if opts.Transport != "streamable-http" {
		return fmt.Errorf("transport %q is not supported in v1; use streamable-http", opts.Transport)
	}

	client := mcp.New(opts.MCPURL, opts.Verbose)
	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("mcp initialize failed: %w", err)
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("tools/list failed: %w", err)
	}
	if err := validateTools(tools); err != nil {
		return err
	}

	p := tea.NewProgram(initialModel(ctx, client, opts.MCPURL, client.SessionID()), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func AskQuestion(ctx context.Context, client *mcp.Client, question string) (string, error) {
	res, err := client.CallTool(ctx, "dir2mcp.ask", map[string]any{"question": question, "k": 8})
	if err != nil {
		return "", err
	}
	if answer, ok := res.StructuredContent["answer"].(string); ok && strings.TrimSpace(answer) != "" {
		return answer, nil
	}
	for _, c := range res.Content {
		if c.Text != "" {
			return c.Text, nil
		}
	}
	return "", nil
}

func validateTools(tools []mcp.Tool) error {
	got := map[string]bool{}
	for _, t := range tools {
		got[t.Name] = true
	}
	missing := make([]string, 0)
	for _, req := range requiredTools {
		if !got[req] {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required tools missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
