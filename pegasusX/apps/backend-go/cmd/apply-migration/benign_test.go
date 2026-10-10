package main

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsBenignDDLConflictNarrow(t *testing.T) {
	t.Parallel()
	if !isBenignDDLConflict(status.Error(codes.AlreadyExists, "table exists")) {
		t.Fatal("AlreadyExists should be benign")
	}
	if isBenignDDLConflict(status.Error(codes.FailedPrecondition, "Cannot add NOT NULL column to non-empty table")) {
		t.Fatal("generic FailedPrecondition must NOT be benign")
	}
	if !isBenignDDLConflict(status.Error(codes.FailedPrecondition, "Column already exists: ClaimedBy")) {
		t.Fatal("duplicate column FailedPrecondition should be benign")
	}
	if !isBenignDDLConflict(status.Error(codes.InvalidArgument, "Table already exists: SchemaMigrations")) {
		t.Fatal("InvalidArgument already exists should be benign")
	}
	if !isBenignDDLConflict(status.Error(codes.FailedPrecondition, "Duplicate name in schema: SchemaMigrations.")) {
		t.Fatal("emulator duplicate-name FailedPrecondition should be benign")
	}
}
