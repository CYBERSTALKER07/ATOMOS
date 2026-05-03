package supplier

import (
	"testing"
	"time"
)

func TestReconcileReturnOutcome(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		wantStatus     string
		wantResolution string
		wantRestock    bool
	}{
		{
			name:           "restock action maps to returned to stock",
			action:         "RESTOCK",
			wantStatus:     "RETURNED_TO_STOCK",
			wantResolution: "RETURN_TO_STOCK",
			wantRestock:    true,
		},
		{
			name:           "write off damaged maps to write off",
			action:         "WRITE_OFF_DAMAGED",
			wantStatus:     "WRITE_OFF",
			wantResolution: "WRITE_OFF",
			wantRestock:    false,
		},
		{
			name:           "unknown action is rejected",
			action:         "UNKNOWN",
			wantStatus:     "",
			wantResolution: "",
			wantRestock:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotResolution, gotRestock := reconcileReturnOutcome(tt.action)
			if gotStatus != tt.wantStatus {
				t.Fatalf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotResolution != tt.wantResolution {
				t.Fatalf("resolution = %q, want %q", gotResolution, tt.wantResolution)
			}
			if gotRestock != tt.wantRestock {
				t.Fatalf("restock = %v, want %v", gotRestock, tt.wantRestock)
			}
		})
	}
}

func TestNewReturnRestockAuditEntry_DefaultsAdjustedByAndComputesNewQty(t *testing.T) {
	entry := newReturnRestockAuditEntry("sku-1", "supplier-1", "", 12, 5)

	if entry.AdjustedBy != "supplier-1" {
		t.Fatalf("adjustedBy = %q, want supplier fallback", entry.AdjustedBy)
	}
	if entry.NewQty() != 17 {
		t.Fatalf("newQty = %d, want 17", entry.NewQty())
	}
	if entry.Delta != 5 {
		t.Fatalf("delta = %d, want 5", entry.Delta)
	}
}

func TestBuildReturnResolvedEvent_UsesCanonicalFields(t *testing.T) {
	ts := time.UnixMilli(1710000000000).UTC()
	event := buildReturnResolvedEvent("line-1", "order-1", "sku-1", 4, "RETURN_TO_STOCK", "supplier-1", "restocked", ts)

	if event["type"] != "RETURN_RESOLVED" {
		t.Fatalf("type = %v, want RETURN_RESOLVED", event["type"])
	}
	if event["line_item_id"] != "line-1" {
		t.Fatalf("line_item_id = %v, want line-1", event["line_item_id"])
	}
	if event["resolution"] != "RETURN_TO_STOCK" {
		t.Fatalf("resolution = %v, want RETURN_TO_STOCK", event["resolution"])
	}
	if event["timestamp"] != ts.UnixMilli() {
		t.Fatalf("timestamp = %v, want %d", event["timestamp"], ts.UnixMilli())
	}
}
