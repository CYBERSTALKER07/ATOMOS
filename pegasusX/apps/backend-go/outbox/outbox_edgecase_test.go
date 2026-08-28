package outbox

import (
	"context"
	"strings"
	"testing"
)

func TestEmitJSON_NilTxn_ReturnsError(t *testing.T) {
	t.Parallel()
	err := EmitJSON(context.Background(), nil, "Order", "o1", "orders", map[string]string{"id": "1"})
	if err == nil {
		t.Fatal("expected error for nil txn")
	}
}

func TestEmitJSON_EmptyAggregateType_ReturnsError(t *testing.T) {
	t.Parallel()
	buf := &mockTxnBuffer{}
	err := EmitJSON(context.Background(), buf, "", "o1", "orders", map[string]string{"id": "1"})
	if err == nil {
		t.Fatal("expected error for empty aggregateType")
	}
}

func TestEmitJSON_EmptyAggregateID_ReturnsError(t *testing.T) {
	t.Parallel()
	buf := &mockTxnBuffer{}
	err := EmitJSON(context.Background(), buf, "Order", "", "orders", map[string]string{"id": "1"})
	if err == nil {
		t.Fatal("expected error for empty aggregateID")
	}
}

func TestEmitJSON_EmptyTopic_ReturnsError(t *testing.T) {
	t.Parallel()
	buf := &mockTxnBuffer{}
	err := EmitJSON(context.Background(), buf, "Order", "o1", "", map[string]string{"id": "1"})
	if err == nil {
		t.Fatal("expected error for empty topic")
	}
}

func TestEmitJSON_TraceIDNotDuplicated(t *testing.T) {
	t.Parallel()
	buf := &mockTxnBuffer{}
	ctx := WithTraceID(context.Background(), "trace-xyz")
	err := EmitJSON(ctx, buf, "Order", "o1", "orders.v1", map[string]string{"trace_id": "original"})
	if err != nil {
		t.Fatal(err)
	}
	payload := string(buf.bufferedEvents[0].Payload)
	if !strings.Contains(payload, "original") {
		t.Fatalf("existing trace_id was overwritten: %s", payload)
	}
}

func TestResolveSupplierID_Explicit(t *testing.T) {
	t.Parallel()
	got := ResolveSupplierID("sup-explicit", []byte(`{}`))
	if got != "sup-explicit" {
		t.Fatalf("got=%s want sup-explicit", got)
	}
}

func TestResolveSupplierID_FromPayload(t *testing.T) {
	t.Parallel()
	got := ResolveSupplierID("", []byte(`{"supplier_id":"sup-payload"}`))
	if got != "sup-payload" {
		t.Fatalf("got=%s want sup-payload", got)
	}
}

func TestResolveSupplierID_PlatformFallback(t *testing.T) {
	t.Parallel()
	got := ResolveSupplierID("", []byte(`{}`))
	if got != PlatformSupplierID {
		t.Fatalf("got=%s want %s", got, PlatformSupplierID)
	}
}

func TestResolveSupplierID_WhitespaceExplicit(t *testing.T) {
	t.Parallel()
	got := ResolveSupplierID("  ", []byte(`{"supplier_id":"sup-from-json"}`))
	if got != "sup-from-json" {
		t.Fatalf("got=%s want sup-from-json (whitespace explicit should fallthrough)", got)
	}
}

func TestClampEventID_EmptyGeneratesUUID(t *testing.T) {
	t.Parallel()
	id := ClampEventID("")
	if id == "" {
		t.Fatal("empty id should generate UUID")
	}
	if len(id) > EventIDMax {
		t.Fatalf("len=%d exceeds %d", len(id), EventIDMax)
	}
}

func TestClampEventID_ShortPassthrough(t *testing.T) {
	t.Parallel()
	id := ClampEventID("abc-123")
	if id != "abc-123" {
		t.Fatalf("got=%s want abc-123", id)
	}
}

func TestClampEventID_ExactMax(t *testing.T) {
	t.Parallel()
	exact := strings.Repeat("x", EventIDMax)
	id := ClampEventID(exact)
	if id != exact {
		t.Fatal("exact-max id was modified")
	}
}

func TestClampEventID_OversizeDeterministic(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 100)
	id := ClampEventID(long)
	if len(id) > EventIDMax {
		t.Fatalf("len=%d exceeds %d", len(id), EventIDMax)
	}
	if id != ClampEventID(long) {
		t.Fatal("oversize clamp not deterministic")
	}
}

func TestWithTraceID_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := WithTraceID(context.Background(), "  my-trace  ")
	got := TraceIDFromContext(ctx)
	if got != "my-trace" {
		t.Fatalf("got=%q want my-trace", got)
	}
}

func TestWithTraceID_Empty(t *testing.T) {
	t.Parallel()
	ctx := WithTraceID(context.Background(), "  ")
	got := TraceIDFromContext(ctx)
	if got != "" {
		t.Fatalf("got=%q want empty for whitespace-only", got)
	}
}

func TestTraceIDFromContext_Nil(t *testing.T) {
	t.Parallel()
	got := TraceIDFromContext(nil)
	if got != "" {
		t.Fatalf("got=%q want empty for nil context", got)
	}
}

func TestEventRowMap_SupplierIdFromPayload(t *testing.T) {
	t.Parallel()
	e := Event{
		EventID:       "ev-1",
		AggregateType: "Order",
		AggregateID:   "ord-1",
		TopicName:     "orders.v1",
		Payload:       []byte(`{"supplier_id":"sup-99"}`),
		SupplierID:    "",
	}
	row := EventRowMap(e)
	if row["SupplierId"] != "sup-99" {
		t.Fatalf("SupplierId=%v want sup-99", row["SupplierId"])
	}
}

func TestSupplierIDFromPayload_Variants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{"snake_case", `{"supplier_id":"s1"}`, "s1"},
		{"PascalCase", `{"SupplierId":"s2"}`, "s2"},
		{"camelCase", `{"supplierId":"s3"}`, "s3"},
		{"missing", `{"order_id":"o1"}`, ""},
		{"invalid_json", `not json`, ""},
		{"empty_object", `{}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SupplierIDFromPayload([]byte(tt.payload))
			if got != tt.want {
				t.Fatalf("got=%s want=%s", got, tt.want)
			}
		})
	}
}
