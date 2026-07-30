package analytics

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

type mockApplier struct {
	mutations []*spanner.Mutation
}

func (m *mockApplier) Apply(ctx context.Context, ms []*spanner.Mutation, opts ...spanner.ApplyOption) (time.Time, error) {
	m.mutations = ms
	return time.Now(), nil
}

func TestRouteAnalyticsWorker_ComputeAndSave(t *testing.T) {
	mock := &mockApplier{}
	worker := NewRouteAnalyticsWorker(mock)

	err := worker.ComputeAndSave(context.Background(), "route-1")
	if err != nil {
		t.Fatalf("ComputeAndSave failed: %v", err)
	}

	if len(mock.mutations) != 1 {
		t.Fatalf("Expected 1 mutation, got %d", len(mock.mutations))
	}
}
