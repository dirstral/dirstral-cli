package mcp

import (
	"errors"
	"strings"
	"testing"
)

func TestCanonicalCodeFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "canonical code token in rpc message",
			err:  &jsonRPCError{Code: -32001, Message: "SESSION_NOT_FOUND"},
			want: CanonicalCodeSessionNotFound,
		},
		{
			name: "equivalent phrase in message",
			err:  errors.New("backend failure: session not found"),
			want: CanonicalCodeSessionNotFound,
		},
		{
			name: "unauthorized alias",
			err:  errors.New("request failed: unauthenticated"),
			want: CanonicalCodeUnauthorized,
		},
		{
			name: "unknown",
			err:  errors.New("something else"),
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := CanonicalCodeFromError(tc.err); got != tc.want {
				t.Fatalf("CanonicalCodeFromError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestActionableMessageForCode(t *testing.T) {
	t.Parallel()

	msg := ActionableMessageForCode(CanonicalCodeIndexNotReady)
	if !strings.Contains(msg, "index is not ready") {
		t.Fatalf("unexpected index-not-ready message: %q", msg)
	}

	msg = ActionableMessageForCode(CanonicalCodeFileNotFound)
	if !strings.Contains(msg, "file") || !strings.Contains(msg, "not found") {
		t.Fatalf("unexpected file-not-found message: %q", msg)
	}

	if unknown := ActionableMessageForCode("DOES_NOT_EXIST"); unknown != "" {
		t.Fatalf("expected empty message for unknown code, got %q", unknown)
	}
}

func TestActionableMessageFromError(t *testing.T) {
	t.Parallel()

	err := &jsonRPCError{Code: -32001, Message: "INDEX_NOT_READY"}
	msg := ActionableMessageFromError(err)
	if !strings.Contains(msg, "index") {
		t.Fatalf("unexpected actionable message: %q", msg)
	}
}
