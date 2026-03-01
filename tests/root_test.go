package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/app"
	"github.com/alibilge/dirstral-cli/internal/config"
	"github.com/alibilge/dirstral-cli/internal/host"
)

func TestBuildTempestOptionsPrefersFlags(t *testing.T) {
	cfg := config.Config{}
	cfg.MCP.URL = "http://default-mcp"
	cfg.ElevenLabs.Voice = "DefaultVoice"
	cfg.ElevenLabs.BaseURL = "https://default-elevenlabs"

	got := app.BuildTempestOptions(cfg, "http://flag-mcp", "FlagVoice", "Mic A", true, true, "https://flag-elevenlabs")

	if got.MCPURL != "http://flag-mcp" {
		t.Fatalf("unexpected mcp url: %s", got.MCPURL)
	}
	if got.Voice != "FlagVoice" {
		t.Fatalf("unexpected voice: %s", got.Voice)
	}
	if got.Device != "Mic A" {
		t.Fatalf("unexpected device: %s", got.Device)
	}
	if !got.Mute {
		t.Fatalf("expected mute true")
	}
	if !got.Verbose {
		t.Fatalf("expected verbose true")
	}
	if got.BaseURL != "https://flag-elevenlabs" {
		t.Fatalf("unexpected base url: %s", got.BaseURL)
	}
}

func TestBuildModeFeedbackSuccess(t *testing.T) {
	fb := app.BuildModeFeedback("Breeze", nil)
	if fb.IsError {
		t.Fatalf("expected success feedback")
	}
	if !strings.Contains(fb.Message, "closed") {
		t.Fatalf("expected closed message, got %q", fb.Message)
	}
	if !strings.Contains(fb.Recovery, "Quit") {
		t.Fatalf("expected quit recovery hint, got %q", fb.Recovery)
	}
}

func TestBuildModeFeedbackCancellation(t *testing.T) {
	fb := app.BuildModeFeedback("Tempest", context.Canceled)
	if fb.IsError {
		t.Fatalf("expected canceled feedback to be non-error")
	}
	if !strings.Contains(strings.ToLower(fb.Message), "canceled") {
		t.Fatalf("expected canceled message, got %q", fb.Message)
	}
}

func TestBuildModeFeedbackFailure(t *testing.T) {
	fb := app.BuildModeFeedback("Tempest", errors.New("missing key"))
	if !fb.IsError {
		t.Fatalf("expected failure feedback")
	}
	if !strings.Contains(strings.ToLower(fb.Message), "failed") {
		t.Fatalf("expected failed message, got %q", fb.Message)
	}
	if !strings.Contains(strings.ToLower(fb.Recovery), "retry") {
		t.Fatalf("expected retry hint, got %q", fb.Recovery)
	}
}

func TestBuildTempestOptionsFallsBackToConfig(t *testing.T) {
	cfg := config.Config{}
	cfg.MCP.URL = "http://default-mcp"
	cfg.ElevenLabs.Voice = "DefaultVoice"
	cfg.ElevenLabs.BaseURL = "https://default-elevenlabs"

	got := app.BuildTempestOptions(cfg, "", "", "", false, false, "")

	if got.MCPURL != cfg.MCP.URL {
		t.Fatalf("expected config mcp url, got %s", got.MCPURL)
	}
	if got.Voice != cfg.ElevenLabs.Voice {
		t.Fatalf("expected config voice, got %s", got.Voice)
	}
	if got.BaseURL != cfg.ElevenLabs.BaseURL {
		t.Fatalf("expected config base url, got %s", got.BaseURL)
	}
}

func TestResolveMCPURLPrefersExplicitOverride(t *testing.T) {
	got := app.ResolveMCPURL("http://default-mcp", "http://override-mcp", true)
	if got != "http://override-mcp" {
		t.Fatalf("expected explicit override to win, got %q", got)
	}
}

func TestResolveMCPURLUsesActiveHostStateWhenNoOverride(t *testing.T) {
	setTestConfigDir(t)

	server := newProbeReadyServer(t)
	t.Cleanup(server.Close)

	state := host.State{PID: os.Getpid(), StartedAt: "now", MCPURL: server.URL + "/mcp"}
	if err := host.SaveState(state); err != nil {
		t.Fatalf("save host state: %v", err)
	}
	t.Cleanup(func() {
		_ = host.ClearState()
	})

	got := app.ResolveMCPURL("http://default-mcp", "", false)
	if got != state.MCPURL {
		t.Fatalf("expected active host endpoint %q, got %q", state.MCPURL, got)
	}
}

func TestResolveMCPURLFallsBackToDefaultWithoutActiveHost(t *testing.T) {
	setTestConfigDir(t)
	_ = host.ClearState()

	got := app.ResolveMCPURL("http://default-mcp", "", false)
	if got != "http://default-mcp" {
		t.Fatalf("expected default endpoint, got %q", got)
	}
}

func newProbeReadyServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		method, _ := req["method"].(string)
		switch method {
		case "initialize":
			w.Header().Set("MCP-Session-Id", "test-session")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": map[string]any{"ok": true}})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": map[string]any{"tools": []map[string]any{{"name": "dir2mcp.list_files"}}}})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}
