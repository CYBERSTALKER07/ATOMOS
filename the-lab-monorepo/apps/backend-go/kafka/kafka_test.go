package kafka

import (
	"strings"
	"testing"
)

// ─── GenerateTxnId ──────────────────────────────────────────────────────────

func TestGenerateTxnId_Format(t *testing.T) {
	id := GenerateTxnId("ORD-1", "CREDIT_LAB", 5000)
	if !strings.HasPrefix(id, "TXN-") {
		t.Errorf("expected TXN- prefix, got %q", id)
	}
	// TXN- + 16 hex chars (sha256[:8] = 8 bytes = 16 hex)
	if len(id) != 4+16 {
		t.Errorf("length = %d, want 20", len(id))
	}
}

func TestGenerateTxnId_Deterministic(t *testing.T) {
	a := GenerateTxnId("ORD-1", "CREDIT_LAB", 5000)
	b := GenerateTxnId("ORD-1", "CREDIT_LAB", 5000)
	if a != b {
		t.Error("same inputs should produce same TxnId")
	}
}

func TestGenerateTxnId_DifferentInputs(t *testing.T) {
	a := GenerateTxnId("ORD-1", "CREDIT_LAB", 5000)
	b := GenerateTxnId("ORD-1", "CREDIT_SUPPLIER", 95000)
	if a == b {
		t.Error("different inputs should produce different TxnIds")
	}
}

func TestGenerateTxnId_DifferentOrders(t *testing.T) {
	a := GenerateTxnId("ORD-1", "CREDIT_LAB", 5000)
	b := GenerateTxnId("ORD-2", "CREDIT_LAB", 5000)
	if a == b {
		t.Error("different orders should produce different TxnIds")
	}
}

// ─── Event Struct Serialization ─────────────────────────────────────────────

func TestLogisticsEvent_Fields(t *testing.T) {
	e := LogisticsEvent{
		EventName:  "ORDER_COMPLETED",
		OrderId:    "ord-1",
		RetailerId: "ret-1",
		Amount:  100000,
	}
	if e.EventName != "ORDER_COMPLETED" || e.Amount != 100000 {
		t.Errorf("unexpected: %+v", e)
	}
}

// ─── Ledger Split Math ──────────────────────────────────────────────────────

func TestLedgerSplitMath_5PercentCommission(t *testing.T) {
	tests := []struct {
		name            string
		amount       int64
		expectLab       int64
		expectSupplier  int64
	}{
		{"100000", 100000, 5000, 95000},
		{"1", 1, 0, 1},       // integer division: 1*5/100 = 0
		{"99", 99, 4, 95},     // 99*5/100 = 4
		{"0", 0, 0, 0},
		{"500000", 500000, 25000, 475000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commissionRate := int64(5)
			labCommission := (tt.amount * commissionRate) / 100
			supplierPayout := tt.amount - labCommission

			if labCommission != tt.expectLab {
				t.Errorf("lab = %d, want %d", labCommission, tt.expectLab)
			}
			if supplierPayout != tt.expectSupplier {
				t.Errorf("supplier = %d, want %d", supplierPayout, tt.expectSupplier)
			}
			if labCommission+supplierPayout != tt.amount {
				t.Errorf("split doesn't sum: %d + %d != %d", labCommission, supplierPayout, tt.amount)
			}
		})
	}
}
