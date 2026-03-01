package test

import (
	"context"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/tempest"
)

func TestTempestRunFailsFastWithoutAudioPrereqs(t *testing.T) {
	t.Setenv("ELEVENLABS_API_KEY", "")

	err := tempest.Run(context.Background(), tempest.Options{Mute: true})
	if err == nil {
		t.Fatal("expected tempest run to fail without local audio prerequisites")
	}

	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "tempest requires") {
		t.Fatalf("unexpected tempest preflight error: %v", err)
	}
	// ELEVENLABS_API_KEY is explicitly unset, so the error must always mention it.
	if !strings.Contains(msg, "elevenlabs_api_key") {
		t.Fatalf("preflight error should mention missing ELEVENLABS_API_KEY: %v", err)
	}
}
