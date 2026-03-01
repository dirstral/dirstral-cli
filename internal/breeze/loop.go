package breeze

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/mcp"
	"github.com/alibilge/dirstral-cli/internal/ui"
)

type ParsedInput struct {
	Quit bool
	Help bool
	Tool string
	Args map[string]any
}

type ToolExecution struct {
	Tool       string
	Args       map[string]any
	Result     *mcp.ToolCallResult
	Output     string
	Citations  []string
	NeedsHuman bool
}

func ParseInput(input, model string) ParsedInput {
	trimmed := strings.TrimSpace(input)
	switch {
	case trimmed == "/quit" || trimmed == "/exit":
		return ParsedInput{Quit: true}
	case trimmed == "/help":
		return ParsedInput{Help: true}
	case strings.HasPrefix(trimmed, "/list"):
		prefix := strings.TrimSpace(strings.TrimPrefix(trimmed, "/list"))
		return ParsedInput{Tool: "dir2mcp.list_files", Args: map[string]any{"path_prefix": prefix, "limit": 30}}
	case strings.HasPrefix(trimmed, "/search "):
		query := strings.TrimSpace(strings.TrimPrefix(trimmed, "/search"))
		return ParsedInput{Tool: "dir2mcp.search", Args: map[string]any{"query": query, "k": 8}}
	case strings.HasPrefix(trimmed, "/open "):
		args := strings.Fields(strings.TrimPrefix(trimmed, "/open"))
		if len(args) == 0 {
			return ParsedInput{Help: true}
		}
		return ParsedInput{Tool: "dir2mcp.open_file", Args: map[string]any{"rel_path": args[0]}}
	default:
		return ParsedInput{Tool: "dir2mcp.ask", Args: map[string]any{"question": trimmed, "k": AskTopKForModel(model)}}
	}
}

func AskTopKForModel(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "large"), strings.Contains(m, "medium"):
		return 12
	case strings.Contains(m, "small"), strings.Contains(m, "mini"):
		return 6
	default:
		return 8
	}
}

func ExecuteParsed(ctx context.Context, client *mcp.Client, parsed ParsedInput) (*ToolExecution, error) {
	if parsed.Tool == "" {
		return &ToolExecution{}, nil
	}
	res, err := client.CallTool(ctx, parsed.Tool, parsed.Args)
	if err != nil {
		return nil, err
	}
	output := renderResultString(parsed.Tool, res)
	if res.IsError {
		output = ui.Errorf("%s returned an error\n%s", parsed.Tool, output)
	}
	return &ToolExecution{
		Tool:      parsed.Tool,
		Args:      parsed.Args,
		Result:    res,
		Output:    output,
		Citations: citationsFor(parsed.Tool, res.StructuredContent),
	}, nil
}

func needsApproval(tool string) bool {
	return !autoApprove[tool]
}

func RequiresApproval(tool string) bool {
	return needsApproval(tool)
}

func citationsFor(tool string, sc map[string]any) []string {
	seen := map[string]bool{}
	list := []string{}
	add := func(c string) {
		if c == "" || seen[c] {
			return
		}
		seen[c] = true
		list = append(list, c)
	}

	switch tool {
	case "dir2mcp.search":
		hits, _ := sc["hits"].([]any)
		for _, h := range hits {
			m, ok := h.(map[string]any)
			if !ok {
				continue
			}
			path := asString(m["rel_path"])
			span, _ := m["span"].(map[string]any)
			add(mcp.CitationForSpan(path, span))
		}
	case "dir2mcp.open_file":
		path := asString(sc["rel_path"])
		span, _ := sc["span"].(map[string]any)
		add(mcp.CitationForSpan(path, span))
	case "dir2mcp.ask":
		citations, _ := sc["citations"].([]any)
		for _, it := range citations {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			path := asString(m["rel_path"])
			span, _ := m["span"].(map[string]any)
			add(mcp.CitationForSpan(path, span))
		}
	}
	sort.Strings(list)
	return list
}

func connectedBanner(url, transport, session, model string) []string {
	msgs := []string{
		ui.Info("Connected to", url),
		ui.Info("Transport:", transport),
		ui.Info("Model:", model),
	}
	if strings.TrimSpace(session) != "" {
		msgs = append(msgs, ui.Info("Session:", session))
	}
	msgs = append(msgs, ui.Dim("Type /help for commands, /quit to exit."))
	return msgs
}

func parseForJSON(input, model string) (ParsedInput, error) {
	if strings.HasPrefix(strings.TrimSpace(input), "/open") {
		args := strings.Fields(strings.TrimPrefix(strings.TrimSpace(input), "/open"))
		if len(args) == 0 {
			return ParsedInput{}, fmt.Errorf("usage: /open <rel_path>")
		}
	}
	return ParseInput(input, model), nil
}
