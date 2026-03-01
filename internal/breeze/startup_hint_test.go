package breeze

import (
	"strings"
	"testing"
)

func TestStartupStatsHintFromContentWhenIndexingRunning(t *testing.T) {
	hint := startupStatsHintFromContent(map[string]any{
		"indexing": map[string]any{"running": true},
	})
	if !strings.Contains(strings.ToLower(hint), "indexing") {
		t.Fatalf("expected indexing hint, got %q", hint)
	}
}

func TestStartupStatsHintFromContentWhenIndexingIdle(t *testing.T) {
	hint := startupStatsHintFromContent(map[string]any{
		"indexing": map[string]any{"running": false},
	})
	if hint != "" {
		t.Fatalf("expected empty hint when indexing is not running, got %q", hint)
	}
}

func TestConnectedBannerIncludesStartupHint(t *testing.T) {
	banner := connectedBanner("http://127.0.0.1:7777/mcp", "streamable-http", "session-1", "mistral-small-latest", "Indexing is still running; results may be partial.")
	joined := strings.Join(banner, "\n")
	if !strings.Contains(joined, "Warning:") {
		t.Fatalf("expected warning line in banner, got %q", joined)
	}
	if !strings.Contains(joined, "results may be partial") {
		t.Fatalf("expected partial-results hint in banner, got %q", joined)
	}
}
