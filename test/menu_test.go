package test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/app"
)

func TestStartMenuItemsOrder(t *testing.T) {
	want := []string{"Breeze", "Tempest", "Lighthouse", "Quit"}
	if got := app.StartMenuItems(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected startup menu options: got %v want %v", got, want)
	}
}

func TestChooseTierByWidth(t *testing.T) {
	if tier := app.ChooseTier(50); tier != app.LogoCompact {
		t.Fatalf("expected compact tier for narrow terminal, got %d", tier)
	}
	if tier := app.ChooseTier(80); tier != app.LogoMedium {
		t.Fatalf("expected medium tier for 80-col terminal, got %d", tier)
	}
	if tier := app.ChooseTier(220); tier != app.LogoFull {
		t.Fatalf("expected full tier for wide terminal, got %d", tier)
	}
}

func TestRenderLogoThreeTiers(t *testing.T) {
	// Compact: plain text
	if got := app.RenderLogo(50); !strings.Contains(got, app.CompactLogoText) {
		t.Fatalf("expected compact logo text for narrow width")
	}
	// Medium: block letters with folder+star
	med := app.RenderLogo(80)
	if !strings.Contains(med, "██████╗") {
		t.Fatalf("expected block letters in medium logo")
	}
	if !strings.Contains(med, "✦") {
		t.Fatalf("expected star glyph in medium logo")
	}
	// Full: large folder glyph
	full := app.RenderLogo(220)
	if !strings.Contains(full, "██████╗") {
		t.Fatalf("expected block letters in full logo")
	}
	if !strings.Contains(full, "▄████████▄") {
		t.Fatalf("expected folder tab glyph in full logo")
	}
	if !strings.Contains(full, "▀██████████████▀") {
		t.Fatalf("expected folder bottom edge in full logo")
	}
}

func TestTerminalWidthParsingAndFallback(t *testing.T) {
	t.Setenv("COLUMNS", " 121 ")
	if got := app.TerminalWidth(); got != 121 {
		t.Fatalf("unexpected terminal width: %d", got)
	}

	t.Setenv("COLUMNS", "not-a-number")
	if got := app.TerminalWidth(); got != app.DefaultTerminalWidth {
		t.Fatalf("unexpected fallback width: %d", got)
	}
}

func TestNormalizeLeftSpacingRemovesSharedIndent(t *testing.T) {
	got := app.NormalizeLeftSpacing([]string{"    alpha", "      beta", "    gamma"})
	want := []string{"alpha", "  beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalization: got %v want %v", got, want)
	}
}
