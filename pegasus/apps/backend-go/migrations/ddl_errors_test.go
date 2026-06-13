package migrations

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsBenignDDLConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "already exists", err: status.Error(codes.AlreadyExists, "table exists"), want: true},
		{name: "instance missing", err: status.Error(codes.NotFound, "Instance not found"), want: false},
		{name: "database missing", err: status.Error(codes.NotFound, "Database does not exist"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBenignDDLConflict(tt.err); got != tt.want {
				t.Fatalf("IsBenignDDLConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInfrastructureNotFound(t *testing.T) {
	err := status.Error(codes.NotFound, "Instance not found: projects/x/instances/y")
	if !IsInfrastructureNotFound(err) {
		t.Fatalf("expected infrastructure not found")
	}
	if IsInfrastructureNotFound(errors.New("plain")) {
		t.Fatalf("plain error must not match")
	}
}
