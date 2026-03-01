package test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/app"
	"github.com/alibilge/dirstral-cli/internal/config"
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
