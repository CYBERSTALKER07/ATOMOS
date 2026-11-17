package driver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestSpannerRepository_Apply_NilClient(t *testing.T) {
	repo := NewSpannerRepository(nil)
	err := repo.Apply(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil spanner client, got nil")
	}
	if !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("expected nil client error, got %v", err)
	}
}

func TestSpannerRepository_ApplyAvailability_NilClient(t *testing.T) {
	repo := NewSpannerRepository(nil)
	upd := AvailabilityUpdate{
		DriverID:  "drv-1",
		OnShift:   true,
		UpdatedAt: time.Now().UTC(),
	}
	err := repo.ApplyAvailability(context.Background(), upd, nil)
	if err == nil {
		t.Fatal("expected error for nil spanner client, got nil")
	}
	if !strings.Contains(err.Error(), "nil client") {
		t.Fatalf("expected nil client error, got %v", err)
	}
}

func TestInMemoryRepository_Apply_MutateError(t *testing.T) {
	repo := NewInMemoryRepository()
	expectedErr := errors.New("mutate failed")
	err := repo.Apply(context.Background(), func() error {
		return expectedErr
	}, nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestInMemoryRepository_Apply_EmitSuccess(t *testing.T) {
	repo := NewInMemoryRepository()
	emitted := false
	err := repo.Apply(context.Background(), func() error {
		return nil
	}, func(b outbox.TxnBuffer) error {
		emitted = true
		return b.BufferOutbox(context.Background(), outbox.Event{
			EventID:       "evt-1",
			AggregateType: "driver",
			AggregateID:   "drv-1",
			TopicName:     "driver-events",
			Payload:       []byte(`{"status":"active"}`),
		})
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !emitted {
		t.Fatal("expected emit function to be called")
	}
}

func TestInMemoryRepository_ApplyAvailability(t *testing.T) {
	repo := NewInMemoryRepository()
	upd := AvailabilityUpdate{
		DriverID:  "drv-1",
		OnShift:   false,
		Reason:    "LUNCH",
		Note:      "Taking 30m break",
		UpdatedAt: time.Now().UTC(),
	}
	emitted := false
	writer, ok := repo.(AvailabilityWriter)
	if !ok {
		t.Fatal("expected inMemoryRepository to implement AvailabilityWriter")
	}
	err := writer.ApplyAvailability(context.Background(), upd, func(b outbox.TxnBuffer) error {
		emitted = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !emitted {
		t.Fatal("expected emit function to be called in ApplyAvailability")
	}
}

func TestSpannerTxnBuffer_BufferOutboxAndAudit(t *testing.T) {
	buf := &spannerTxnBuffer{}
	ctx := context.Background()

	evt := outbox.Event{
		EventID:       "evt-100",
		AggregateType: "driver",
		AggregateID:   "drv-1",
		TopicName:     "pegasusx-main",
		Payload:       []byte(`{"driver_id":"drv-1"}`),
		CreatedAt:     time.Now().UTC(),
	}
	if err := buf.BufferOutbox(ctx, evt); err != nil {
		t.Fatalf("BufferOutbox failed: %v", err)
	}

	audit := outbox.AuditEntry{
		AuditID:       "aud-100",
		SupplierID:    "sup-1",
		ActorID:       "drv-1",
		ActorRole:     "DRIVER",
		Action:        "AVAILABILITY_CHANGED",
		AggregateType: "Driver",
		AggregateID:   "drv-1",
		DetailsJSON:   []byte(`{"on_shift":true}`),
		CreatedAt:     time.Now().UTC(),
	}
	if err := buf.BufferAudit(ctx, audit); err != nil {
		t.Fatalf("BufferAudit failed: %v", err)
	}

	if len(buf.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(buf.events))
	}
	if len(buf.audits) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(buf.audits))
	}
	if buf.events[0].EventID != "evt-100" {
		t.Fatalf("expected evt-100, got %s", buf.events[0].EventID)
	}
	if buf.audits[0].AuditID != "aud-100" {
		t.Fatalf("expected aud-100, got %s", buf.audits[0].AuditID)
	}
}

func TestOutboxMutations(t *testing.T) {
	eventsList := []outbox.Event{
		{
			EventID:       "evt-1",
			AggregateType: "driver",
			AggregateID:   "drv-1",
			TopicName:     "test",
			Payload:       []byte(`{}`),
		},
		{
			EventID:       "evt-2",
			AggregateType: "driver",
			AggregateID:   "drv-2",
			TopicName:     "test",
			Payload:       []byte(`{}`),
			CreatedAt:     time.Now().UTC(),
		},
	}
	muts := outboxMutations(eventsList)
	if len(muts) != 2 {
		t.Fatalf("expected 2 mutations, got %d", len(muts))
	}
	for i, m := range muts {
		if m == nil {
			t.Fatalf("mutation %d is nil", i)
		}
	}
}
