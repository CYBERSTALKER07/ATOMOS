package telemetryaudit

import (
	"context"
	"errors"
	"testing"

	"backend-go/telemetry"

	goKafka "github.com/segmentio/kafka-go"
)

func TestJournal_CloseIsIdempotentAndEmitAfterCloseReturnsClosed(t *testing.T) {
	journal := &Journal{
		writer: &goKafka.Writer{},
		queue:  make(chan telemetry.AuditEvent, 1),
		done:   make(chan struct{}),
	}

	go func() {
		for range journal.queue {
		}
		close(journal.done)
	}()

	if err := journal.Close(); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}
	if err := journal.Close(); err != nil {
		t.Fatalf("Close(second) error = %v", err)
	}

	err := journal.Emit(context.Background(), telemetry.AuditEvent{TraceID: "trace-1", DriverID: "DRV-1"})
	if !errors.Is(err, errJournalClosed) {
		t.Fatalf("Emit(after close) error = %v, want %v", err, errJournalClosed)
	}
}
