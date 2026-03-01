package mcp

import (
	"errors"
	"strings"
)

const (
	CanonicalCodeUnauthorized     = "UNAUTHORIZED"
	CanonicalCodeSessionNotFound  = "SESSION_NOT_FOUND"
	CanonicalCodeIndexNotReady    = "INDEX_NOT_READY"
	CanonicalCodeFileNotFound     = "FILE_NOT_FOUND"
	CanonicalCodePermissionDenied = "PERMISSION_DENIED"
	CanonicalCodeRateLimited      = "RATE_LIMITED"
)

// CanonicalCodeFromError extracts a backend canonical code when present.
func CanonicalCodeFromError(err error) string {
	if err == nil {
		return ""
	}

	var rpcErr *jsonRPCError
	if errors.As(err, &rpcErr) {
		if code := canonicalCodeFromText(rpcErr.Message); code != "" {
			return code
		}
	}

	return canonicalCodeFromText(err.Error())
}

// ActionableMessageForCode maps a canonical backend code to user guidance.
func ActionableMessageForCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case CanonicalCodeUnauthorized:
		return "Authentication failed (UNAUTHORIZED). Set DIR2MCP_AUTH_TOKEN or refresh your credentials, then retry."
	case CanonicalCodeSessionNotFound:
		return "The MCP session was not found. Reconnect to the server and retry your command."
	case CanonicalCodeIndexNotReady:
		return "The index is not ready yet. Wait for indexing to finish, then retry."
	case CanonicalCodeFileNotFound:
		return "The requested file was not found. Verify the path or use list_files/search first."
	case CanonicalCodePermissionDenied:
		return "Permission denied for this operation. Check server auth/scope and retry."
	case CanonicalCodeRateLimited:
		return "Request rate limit reached. Wait briefly and retry."
	default:
		return ""
	}
}

// ActionableMessageFromError derives canonical code and returns user guidance.
func ActionableMessageFromError(err error) string {
	return ActionableMessageForCode(CanonicalCodeFromError(err))
}

func canonicalCodeFromText(text string) string {
	upper := strings.ToUpper(text)

	if strings.Contains(upper, CanonicalCodeUnauthorized) || strings.Contains(upper, "UNAUTHENTICATED") {
		return CanonicalCodeUnauthorized
	}
	if strings.Contains(upper, CanonicalCodeSessionNotFound) || strings.Contains(upper, "SESSION NOT FOUND") {
		return CanonicalCodeSessionNotFound
	}
	if strings.Contains(upper, CanonicalCodeIndexNotReady) || strings.Contains(upper, "INDEX NOT READY") {
		return CanonicalCodeIndexNotReady
	}
	if strings.Contains(upper, CanonicalCodeFileNotFound) || strings.Contains(upper, "FILE NOT FOUND") {
		return CanonicalCodeFileNotFound
	}
	if containsAny(upper,
		CanonicalCodePermissionDenied,
		"PERMISSION DENIED",
		"PERMISSION",
		"DENIED",
		"FORBIDDEN",
	) {
		return CanonicalCodePermissionDenied
	}
	if containsAny(upper,
		CanonicalCodeRateLimited,
		"RATE LIMIT",
		"RATE-LIMIT",
		"RATE_LIMIT",
		"RATE LIMIT EXCEEDED",
		"LIMIT EXCEEDED",
		"QUOTA",
		"THROTTLE",
		"THROTTLED",
		"TOO MANY REQUESTS",
	) {
		return CanonicalCodeRateLimited
	}

	return ""
}

func containsAny(value string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}
	return false
}
