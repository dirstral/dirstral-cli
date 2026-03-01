package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestCallToolRecoversSessionOnceOnSessionNotFound(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	initializeCalls := 0
	notifyCalls := 0
	toolCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		switch req.Method {
		case "initialize":
			initializeCalls++
			session := "session-1"
			if initializeCalls > 1 {
				session = "session-2"
			}
			w.Header().Set("MCP-Session-Id", session)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  map[string]any{},
			})
		case "notifications/initialized":
			notifyCalls++
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			toolCalls++
			sessionHeader := r.Header.Get("MCP-Session-Id")
			if toolCalls == 1 {
				if sessionHeader != "session-1" {
					t.Fatalf("expected first tool call with session-1, got %q", sessionHeader)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"error": map[string]any{
						"code":    -32001,
						"message": "SESSION_NOT_FOUND",
					},
				})
				return
			}
			if sessionHeader != "session-2" {
				t.Fatalf("expected retried tool call with session-2, got %q", sessionHeader)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"content": []any{map[string]any{"type": "text", "text": "ok"}},
				},
			})
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
	}))
	defer server.Close()

	client := NewWithTransport(server.URL, "streamable-http", false)
	ctx := context.Background()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initial initialize failed: %v", err)
	}

	res, err := client.CallTool(ctx, "dir2mcp.search", map[string]any{"query": "hello"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if len(res.Content) != 1 || res.Content[0].Text != "ok" {
		t.Fatalf("unexpected tool result: %#v", res.Content)
	}

	mu.Lock()
	defer mu.Unlock()
	if initializeCalls != 2 {
		t.Fatalf("expected 2 initialize calls, got %d", initializeCalls)
	}
	if notifyCalls != 2 {
		t.Fatalf("expected 2 notifications/initialized calls, got %d", notifyCalls)
	}
	if toolCalls != 2 {
		t.Fatalf("expected 2 tools/call attempts, got %d", toolCalls)
	}
}

func TestCallToolSessionRecoveryIsBoundedToSingleRetry(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	initializeCalls := 0
	toolCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		switch req.Method {
		case "initialize":
			initializeCalls++
			w.Header().Set("MCP-Session-Id", "session")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			toolCalls++
			if toolCalls > 2 {
				t.Fatalf("unexpected extra retry, tools/call=%d", toolCalls)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error": map[string]any{
					"code":    -32001,
					"message": "session not found",
				},
			})
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
	}))
	defer server.Close()

	client := NewWithTransport(server.URL, "streamable-http", false)
	ctx := context.Background()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initial initialize failed: %v", err)
	}

	_, err := client.CallTool(ctx, "dir2mcp.search", map[string]any{"query": "hello"})
	if err == nil {
		t.Fatal("expected error after bounded retry")
	}

	mu.Lock()
	defer mu.Unlock()
	if initializeCalls != 2 {
		t.Fatalf("expected 2 initialize calls (initial + recovery), got %d", initializeCalls)
	}
	if toolCalls != 2 {
		t.Fatalf("expected exactly 2 tools/call attempts, got %d", toolCalls)
	}
}
