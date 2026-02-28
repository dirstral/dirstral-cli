package breeze

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alibilge/dirstral-cli/internal/mcp"
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

	fmt.Printf("Connected to %s\n", opts.MCPURL)
	fmt.Printf("Session: %s\n", client.SessionID())
	fmt.Println("Type /help for commands, /quit to exit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("breeze> ")
		if !scanner.Scan() {
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case line == "/quit" || line == "/exit":
			return nil
		case line == "/help":
			printHelp()
		case strings.HasPrefix(line, "/list"):
			prefix := strings.TrimSpace(strings.TrimPrefix(line, "/list"))
			if err := callAndRender(ctx, client, "dir2mcp.list_files", map[string]any{"path_prefix": prefix, "limit": 30}); err != nil {
				fmt.Println("error:", err)
			}
		case strings.HasPrefix(line, "/search "):
			query := strings.TrimSpace(strings.TrimPrefix(line, "/search"))
			if err := callAndRender(ctx, client, "dir2mcp.search", map[string]any{"query": query, "k": 8}); err != nil {
				fmt.Println("error:", err)
			}
		case strings.HasPrefix(line, "/open "):
			args := strings.Fields(strings.TrimPrefix(line, "/open"))
			if len(args) == 0 {
				fmt.Println("usage: /open <rel_path>")
				continue
			}
			if err := callAndRender(ctx, client, "dir2mcp.open_file", map[string]any{"rel_path": args[0]}); err != nil {
				fmt.Println("error:", err)
			}
		default:
			if err := callAndRender(ctx, client, "dir2mcp.ask", map[string]any{"question": line, "k": 8}); err != nil {
				fmt.Println("error:", err)
			}
		}
	}
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

func callAndRender(ctx context.Context, client *mcp.Client, tool string, args map[string]any) error {
	if !autoApprove[tool] {
		if !confirmApproval(tool) {
			return nil
		}
	}
	res, err := client.CallTool(ctx, tool, args)
	if err != nil {
		return err
	}
	if res.IsError {
		fmt.Println("tool error")
	}
	renderResult(tool, res)
	return nil
}

func renderResult(tool string, res *mcp.ToolCallResult) {
	switch tool {
	case "dir2mcp.list_files":
		renderListFiles(res.StructuredContent)
	case "dir2mcp.search":
		renderSearch(res.StructuredContent)
	case "dir2mcp.open_file":
		renderOpenFile(res.StructuredContent)
	case "dir2mcp.ask":
		renderAsk(res.StructuredContent)
	default:
		for _, c := range res.Content {
			if c.Text != "" {
				fmt.Println(c.Text)
			}
		}
	}
}

func renderListFiles(sc map[string]any) {
	files, _ := sc["files"].([]any)
	if len(files) == 0 {
		fmt.Println("(no files)")
		return
	}
	for i, f := range files {
		if i >= 20 {
			fmt.Println("...")
			break
		}
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		fmt.Printf("- %s (%s)\n", asString(m["rel_path"]), asString(m["doc_type"]))
	}
}

func renderSearch(sc map[string]any) {
	hits, _ := sc["hits"].([]any)
	if len(hits) == 0 {
		fmt.Println("(no hits)")
		return
	}
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
		fmt.Printf("%d) score=%v %s\n", i+1, score, citation)
		if snippet != "" {
			fmt.Printf("   %s\n", snippet)
		}
	}
}

func renderOpenFile(sc map[string]any) {
	path := asString(sc["rel_path"])
	content := asString(sc["content"])
	if span, ok := sc["span"].(map[string]any); ok {
		fmt.Println(mcp.CitationForSpan(path, span))
	}
	fmt.Println(content)
}

func renderAsk(sc map[string]any) {
	answer := strings.TrimSpace(asString(sc["answer"]))
	if answer != "" {
		fmt.Println(answer)
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
			fmt.Printf("Sources: %s\n", strings.Join(ordered, ", "))
		}
	}
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

func confirmApproval(tool string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Run tool %s? [y/N]: ", tool)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help               Show help")
	fmt.Println("  /quit               Exit Breeze")
	fmt.Println("  /list [prefix]      List indexed files")
	fmt.Println("  /search <query>     Search corpus")
	fmt.Println("  /open <rel_path>    Open file from index")
	fmt.Println("  Any other text is sent to dir2mcp.ask")
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
