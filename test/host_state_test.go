package test

import (
	"testing"

	"github.com/alibilge/dirstral-cli/internal/host"
)

func TestHostStateRoundTrip(t *testing.T) {
	s := host.State{PID: 12345, StartedAt: "now", MCPURL: "http://127.0.0.1:8087/mcp"}
	if err := host.SaveState(s); err != nil {
		t.Fatalf("save state: %v", err)
	}
	defer func() {
		if err := host.ClearState(); err != nil {
			t.Errorf("clear state: %v", err)
		}
	}()

	loaded, err := host.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if loaded.PID != s.PID || loaded.MCPURL != s.MCPURL {
		t.Fatalf("unexpected loaded state: %#v", loaded)
	}
}
