package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/host"
)

func TestComputeMCPURLDeterministic(t *testing.T) {
	url, deterministic := host.ComputeMCPURL("0.0.0.0:8087", "mcp")
	if !deterministic {
		t.Fatalf("expected deterministic url")
	}
	if url != "http://127.0.0.1:8087/mcp" {
		t.Fatalf("unexpected url: %s", url)
	}

	url, deterministic = host.ComputeMCPURL("127.0.0.1:0", "/mcp")
	if deterministic {
		t.Fatalf("expected non-deterministic for ephemeral port, got %s", url)
	}
}

func TestStatusReturnsNilWhenMCPReady(t *testing.T) {
	setTestConfigDir(t)

	server := newMockMCPServer(t, true)
	defer server.Close()

	s := host.State{PID: os.Getpid(), StartedAt: "now", MCPURL: server.URL}
	if err := host.SaveState(s); err != nil {
		t.Fatalf("save state: %v", err)
	}
	t.Cleanup(func() {
		_ = host.ClearState()
	})

	if err := host.Status(); err != nil {
		t.Fatalf("expected ready status, got error: %v", err)
	}
}

func TestStatusReturnsErrorWhenMCPUnready(t *testing.T) {
	setTestConfigDir(t)

	server := newMockMCPServer(t, false)
	defer server.Close()

	s := host.State{PID: os.Getpid(), StartedAt: "now", MCPURL: server.URL}
	if err := host.SaveState(s); err != nil {
		t.Fatalf("save state: %v", err)
	}
	t.Cleanup(func() {
		_ = host.ClearState()
	})

	err := host.Status()
	if err == nil {
		t.Fatalf("expected non-nil status error when MCP is unready")
	}
	if !strings.Contains(err.Error(), "lighthouse not ready") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newMockMCPServer(t *testing.T, ready bool) *httptest.Server {
	t.Helper()

	const sessionID = "sess-test"

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		method, _ := req["method"].(string)
		id, hasID := req["id"]

		switch method {
		case "initialize":
			w.Header().Set("MCP-Session-Id", sessionID)
			w.Header().Set("Content-Type", "application/json")
			if !hasID {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"capabilities": map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			if r.Header.Get("MCP-Session-Id") != sessionID {
				http.Error(w, "unknown session", http.StatusNotFound)
				return
			}
			if !ready {
				http.Error(w, "warming up", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{},
				},
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}
