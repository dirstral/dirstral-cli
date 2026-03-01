package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/alibilge/dirstral-cli/internal/mcp"
)

func TestCanonicalCodeFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "equivalent phrase in message", err: errors.New("backend failure: session not found"), want: mcp.CanonicalCodeSessionNotFound},
		{name: "unauthorized alias", err: errors.New("request failed: unauthenticated"), want: mcp.CanonicalCodeUnauthorized},
		{name: "permission variant", err: errors.New("permission denied for this route"), want: mcp.CanonicalCodePermissionDenied},
		{name: "rate limit variant", err: errors.New("rate-limit exceeded; retry later"), want: mcp.CanonicalCodeRateLimited},
		{name: "unknown", err: errors.New("something else"), want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := mcp.CanonicalCodeFromError(tc.err); got != tc.want {
				t.Fatalf("CanonicalCodeFromError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestActionableMessageForCode(t *testing.T) {
	t.Parallel()

	msg := mcp.ActionableMessageForCode(mcp.CanonicalCodeIndexNotReady)
	if !strings.Contains(msg, "index is not ready") {
		t.Fatalf("unexpected index-not-ready message: %q", msg)
	}

	msg = mcp.ActionableMessageForCode(mcp.CanonicalCodeFileNotFound)
	if !strings.Contains(msg, "file") || !strings.Contains(msg, "not found") {
		t.Fatalf("unexpected file-not-found message: %q", msg)
	}

	if unknown := mcp.ActionableMessageForCode("DOES_NOT_EXIST"); unknown != "" {
		t.Fatalf("expected empty message for unknown code, got %q", unknown)
	}
}

func TestActionableMessageFromError(t *testing.T) {
	t.Parallel()

	err := errors.New("INDEX_NOT_READY")
	msg := mcp.ActionableMessageFromError(err)
	if !strings.Contains(strings.ToLower(msg), "index") {
		t.Fatalf("unexpected actionable message: %q", msg)
	}
}
