package breeze

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/mcp"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	MCPURL    string
	Transport string
	Model     string
	Verbose   bool
	JSON      bool
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
	if opts.Transport == "" {
		opts.Transport = "streamable-http"
	}
	if opts.Model == "" {
		opts.Model = "mistral-small-latest"
	}

	client := mcp.NewWithTransport(opts.MCPURL, opts.Transport, opts.Verbose)
	defer func() {
		_ = client.Close()
	}()
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
	if opts.JSON {
		return RunJSONLoopWithIO(ctx, client, opts, os.Stdin, os.Stdout)
	}

	p := tea.NewProgram(initialModel(ctx, client, opts), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func AskQuestion(ctx context.Context, client *mcp.Client, question string) (string, error) {
	res, err := client.CallTool(ctx, "dir2mcp.ask", map[string]any{"question": question, "k": AskTopKForModel("")})
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

type jsonEvent struct {
	Version string         `json:"version"`
	Type    string         `json:"type"`
	Data    map[string]any `json:"data"`
}

func runJSONLoop(ctx context.Context, client *mcp.Client, opts Options, in io.Reader, out io.Writer) error {
	enc := json.NewEncoder(out)
	write := func(kind string, data map[string]any) error {
		return enc.Encode(jsonEvent{Version: "v1", Type: kind, Data: data})
	}

	if err := write("session", map[string]any{
		"mcp_url":    opts.MCPURL,
		"transport":  opts.Transport,
		"model":      opts.Model,
		"session_id": client.SessionID(),
	}); err != nil {
		return err
	}

	s := bufio.NewScanner(in)
	for s.Scan() {
		input := strings.TrimSpace(s.Text())
		if input == "" {
			continue
		}
		parsed, err := parseForJSON(input, opts.Model)
		if err != nil {
			if err := write("error", map[string]any{"message": err.Error()}); err != nil {
				return err
			}
			continue
		}
		if parsed.Quit {
			if err := write("exit", map[string]any{"reason": "user"}); err != nil {
				return err
			}
			return nil
		}
		if parsed.Help {
			if err := write("help", map[string]any{"text": formatHelpPlain()}); err != nil {
				return err
			}
			continue
		}
		if needsApproval(parsed.Tool) {
			if err := write("approval_required", map[string]any{"tool": parsed.Tool, "approved": false}); err != nil {
				return err
			}
			continue
		}
		execRes, err := ExecuteParsed(ctx, client, parsed)
		if err != nil {
			if err := write("error", map[string]any{"tool": parsed.Tool, "message": err.Error()}); err != nil {
				return err
			}
			continue
		}
		if err := write("tool_result", map[string]any{
			"tool":               execRes.Tool,
			"args":               execRes.Args,
			"is_error":           execRes.Result != nil && execRes.Result.IsError,
			"output":             execRes.Output,
			"citations":          execRes.Citations,
			"structured_content": execRes.Result.StructuredContent,
		}); err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func RunJSONLoopWithIO(ctx context.Context, client *mcp.Client, opts Options, in io.Reader, out io.Writer) error {
	return runJSONLoop(ctx, client, opts, in, out)
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
