package app

import (
	"testing"

	"github.com/alibilge/dirstral-cli/internal/config"
)

func TestBuildTempestOptionsPrefersFlags(t *testing.T) {
	cfg := config.Config{}
	cfg.MCP.URL = "http://default-mcp"
	cfg.ElevenLabs.Voice = "DefaultVoice"
	cfg.ElevenLabs.BaseURL = "https://default-elevenlabs"

	got := buildTempestOptions(cfg, "http://flag-mcp", "FlagVoice", "Mic A", true, true, "https://flag-elevenlabs")

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

func TestBuildTempestOptionsFallsBackToConfig(t *testing.T) {
	cfg := config.Config{}
	cfg.MCP.URL = "http://default-mcp"
	cfg.ElevenLabs.Voice = "DefaultVoice"
	cfg.ElevenLabs.BaseURL = "https://default-elevenlabs"

	got := buildTempestOptions(cfg, "", "", "", false, false, "")

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
