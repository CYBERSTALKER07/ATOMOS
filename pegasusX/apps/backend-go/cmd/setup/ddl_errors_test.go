package main

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
		{name: "already exists", err: status.Error(codes.AlreadyExists, "index exists"), want: true},
		{name: "instance missing", err: status.Error(codes.NotFound, "Instance not found"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBenignDDLConflict(tt.err); got != tt.want {
				t.Fatalf("isBenignDDLConflict() = %v, want %v", got, tt.want)
			}
		})
	}
	if isBenignDDLConflict(errors.New("plain")) {
		t.Fatal("plain error must not be benign")
	}
}
