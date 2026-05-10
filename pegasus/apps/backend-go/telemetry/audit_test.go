package telemetry

import "testing"

func TestNormalizePayloadTimestamp(t *testing.T) {
	t.Run("seconds_to_millis", func(t *testing.T) {
		got := NormalizePayloadTimestamp(1_735_000_000)
		if got != 1_735_000_000_000 {
			t.Fatalf("NormalizePayloadTimestamp(seconds) = %d", got)
		}
	})

	t.Run("millis_unchanged", func(t *testing.T) {
		want := int64(1_735_000_000_123)
		got := NormalizePayloadTimestamp(want)
		if got != want {
			t.Fatalf("NormalizePayloadTimestamp(millis) = %d, want %d", got, want)
		}
	})
}

func TestBuildAuditEvent_DerivesStableTraceID(t *testing.T) {
	payload := GPSPayload{
		DriverID:  "DRV-TASH-001",
		Latitude:  41.311081,
		Longitude: 69.240562,
		Timestamp: 1_735_000_000,
	}

	first, normalizedOne, err := BuildAuditEvent(payload, "SUP-TASH-001")
	if err != nil {
		t.Fatalf("BuildAuditEvent(first) error = %v", err)
	}
	second, normalizedTwo, err := BuildAuditEvent(payload, "SUP-TASH-001")
	if err != nil {
		t.Fatalf("BuildAuditEvent(second) error = %v", err)
	}

	if first.TraceID == "" {
		t.Fatal("expected derived trace id")
	}
	if first.TraceID != second.TraceID {
		t.Fatalf("TraceID mismatch: %q vs %q", first.TraceID, second.TraceID)
	}
	if normalizedOne.Timestamp != 1_735_000_000_000 {
		t.Fatalf("normalized timestamp = %d", normalizedOne.Timestamp)
	}
	if normalizedTwo.TraceID != first.TraceID {
		t.Fatalf("normalized trace = %q, want %q", normalizedTwo.TraceID, first.TraceID)
	}
}
