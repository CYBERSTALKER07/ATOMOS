package main

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isBenignDDLConflict reports whether a Spanner DDL error is safe to ignore
// during idempotent schema convergence (column/index already exists, or already dropped).
// Infrastructure errors (instance/database missing, permission denied) must fail.
func isBenignDDLConflict(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.AlreadyExists:
		return true
	case codes.NotFound:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	case codes.FailedPrecondition:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate name in schema") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	case codes.InvalidArgument:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "already has a constraint") ||
			strings.Contains(msg, "index not found") ||
			strings.Contains(msg, "table not found")
	default:
		return false
	}
}
