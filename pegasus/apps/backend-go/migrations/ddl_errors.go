package migrations

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsBenignDDLConflict reports whether a Spanner DDL error is safe to ignore
// during idempotent schema convergence. Instance/database missing must fail.
func IsBenignDDLConflict(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.AlreadyExists, codes.FailedPrecondition:
		return true
	case codes.InvalidArgument:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "already has a constraint")
	default:
		return false
	}
}

// IsInfrastructureNotFound reports missing Spanner instance/database/admin resources.
func IsInfrastructureNotFound(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	if st.Code() != codes.NotFound {
		return false
	}
	msg := strings.ToLower(st.Message())
	return strings.Contains(msg, "instance not found") ||
		strings.Contains(msg, "database not found") ||
		strings.Contains(msg, "instance does not exist")
}
