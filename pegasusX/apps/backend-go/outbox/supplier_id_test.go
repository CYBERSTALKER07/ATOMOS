package outbox

import (
	"testing"
	"time"
)

func TestSupplierIDFromPayload(t *testing.T) {
	if got := SupplierIDFromPayload([]byte(`{"supplier_id":"sup-a"}`)); got != "sup-a" {
		t.Fatalf("got=%q", got)
	}
	if got := SupplierIDFromPayload([]byte(`{"SupplierId":"sup-b"}`)); got != "sup-b" {
		t.Fatalf("got=%q", got)
	}
	if got := SupplierIDFromPayload([]byte(`{"x":1}`)); got != "" {
		t.Fatalf("empty got=%q", got)
	}
}

func TestResolveSupplierIDPlatformSentinel(t *testing.T) {
	if got := ResolveSupplierID("", []byte(`{"x":1}`)); got != PlatformSupplierID {
		t.Fatalf("want %q got %q", PlatformSupplierID, got)
	}
	if got := ResolveSupplierID("  ", []byte(`{}`)); got != PlatformSupplierID {
		t.Fatalf("want %q got %q", PlatformSupplierID, got)
	}
	if got := ResolveSupplierID("sup-x", []byte(`{}`)); got != "sup-x" {
		t.Fatalf("got=%q", got)
	}
	if got := ResolveSupplierID("", []byte(`{"supplier_id":"sup-y"}`)); got != "sup-y" {
		t.Fatalf("got=%q", got)
	}
}

func TestClampEventID(t *testing.T) {
	if got := ClampEventID("short"); got != "short" {
		t.Fatalf("short=%q", got)
	}
	long := "return-scan-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee-1"
	if len(long) <= EventIDMax {
		t.Fatalf("fixture too short: %d", len(long))
	}
	got := ClampEventID(long)
	if len(got) > EventIDMax || got == "" {
		t.Fatalf("clamped=%q len=%d", got, len(got))
	}
	if ClampEventID(long) != got {
		t.Fatal("clamp must be deterministic")
	}
	if row := EventRowMap(Event{EventID: long, AggregateType: "o", AggregateID: "a", TopicName: "t", Payload: []byte(`{}`)}); row["EventId"] != got {
		t.Fatalf("EventRowMap EventId=%v", row["EventId"])
	}
}

func TestEventRowMapIncludesSupplierId(t *testing.T) {
	row := EventRowMap(Event{
		EventID:       "e1",
		AggregateType: "order",
		AggregateID:   "o1",
		TopicName:     "main",
		Payload:       []byte(`{}`),
		CreatedAt:     time.Unix(1, 0).UTC(),
		SupplierID:    "sup-z",
	})
	if row["SupplierId"] != "sup-z" {
		t.Fatalf("row=%v", row)
	}
}

func TestEventRowMapPlatformWhenEmpty(t *testing.T) {
	row := EventRowMap(Event{
		EventID:       "e2",
		AggregateType: "system",
		AggregateID:   "s1",
		TopicName:     "main",
		Payload:       []byte(`{"ok":true}`),
		CreatedAt:     time.Unix(1, 0).UTC(),
	})
	if row["SupplierId"] != PlatformSupplierID {
		t.Fatalf("want platform sentinel, row=%v", row)
	}
}
