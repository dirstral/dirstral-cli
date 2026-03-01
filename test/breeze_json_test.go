package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/breeze"
	"github.com/alibilge/dirstral-cli/internal/mcp"
)

func TestAskTopKForModel(t *testing.T) {
	if got := breeze.AskTopKForModel("mistral-large-latest"); got != 12 {
		t.Fatalf("unexpected top-k for large model: %d", got)
	}
	if got := breeze.AskTopKForModel("mistral-small-latest"); got != 6 {
		t.Fatalf("unexpected top-k for small model: %d", got)
	}
	if got := breeze.AskTopKForModel("custom-model"); got != 8 {
		t.Fatalf("unexpected top-k default: %d", got)
	}
}

func TestParseInputUsesModelProfileForAsk(t *testing.T) {
	parsed := breeze.ParseInput("how does this work?", "mistral-small-latest")
	if parsed.Tool != "dir2mcp.ask" {
		t.Fatalf("expected dir2mcp.ask, got %q", parsed.Tool)
	}
	if got := parsed.Args["k"]; got != 6 {
		t.Fatalf("expected k=6 for small model, got %v", got)
	}
}

func TestBreezeJSONOutputStructureAndModelPropagation(t *testing.T) {
	var askK any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		method, _ := req["method"].(string)
		switch method {
		case "initialize":
			w.Header().Set("MCP-Session-Id", "sess-test")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			params, _ := req["params"].(map[string]any)
			args, _ := params["arguments"].(map[string]any)
			askK = args["k"]
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]any{
					"isError": false,
					"content": []map[string]any{{"type": "text", "text": "answer"}},
					"structuredContent": map[string]any{
						"answer": "answer",
						"citations": []map[string]any{{
							"rel_path": "src/main.go",
							"span":     map[string]any{"kind": "lines", "start_line": 3, "end_line": 9},
						}},
					},
				},
			})
		default:
			t.Fatalf("unexpected method: %s", method)
		}
	}))
	defer server.Close()

	client := mcp.New(server.URL, false)
	if err := client.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	in := bytes.NewBufferString("what changed\n/quit\n")
	out := &bytes.Buffer{}
	opts := breeze.Options{MCPURL: server.URL, Transport: "streamable-http", Model: "mistral-large-latest", JSON: true}
	if err := breeze.RunJSONLoopWithIO(context.Background(), client, opts, in, out); err != nil {
		t.Fatalf("json loop: %v", err)
	}

	if askK != float64(12) {
		t.Fatalf("expected ask k=12 from model profile, got %v", askK)
	}

	s := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	events := make([]map[string]any, 0)
	for s.Scan() {
		var evt map[string]any
		if err := json.Unmarshal(s.Bytes(), &evt); err != nil {
			t.Fatalf("invalid json line: %v", err)
		}
		events = append(events, evt)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
	if events[0]["version"] != "v1" || events[0]["type"] != "session" {
		t.Fatalf("unexpected first event: %#v", events[0])
	}

	var toolEvent map[string]any
	for _, evt := range events {
		if evt["type"] == "tool_result" {
			toolEvent = evt
			break
		}
	}
	if toolEvent == nil {
		t.Fatalf("missing tool_result event")
	}
	data, _ := toolEvent["data"].(map[string]any)
	if data["tool"] != "dir2mcp.ask" {
		t.Fatalf("unexpected tool_result tool: %v", data["tool"])
	}
	citations, _ := data["citations"].([]any)
	if len(citations) == 0 {
		t.Fatalf("expected citations in tool_result")
	}
}
