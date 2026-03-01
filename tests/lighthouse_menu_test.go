package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/app"
)

func TestLighthouseMenuItemsOrder(t *testing.T) {
	want := []string{"Start Server", "Server Status", "Stop Server", "Back"}
	if got := app.LighthouseMenuItems(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lighthouse options: got %v want %v", got, want)
	}
}

func TestLighthouseMenuControlsAreKeyboardFirst(t *testing.T) {
	cfg := app.LighthouseMenuConfig()
	if !strings.Contains(cfg.Controls, "j/k") {
		t.Fatalf("expected j/k controls, got %q", cfg.Controls)
	}
	if !strings.Contains(cfg.Controls, "esc/q") {
		t.Fatalf("expected esc/q controls, got %q", cfg.Controls)
	}
}
