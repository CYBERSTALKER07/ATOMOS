package order

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestToCustomerStage(t *testing.T) {
	tests := []struct {
		status   Status
		expected CustomerStage
	}{
		{StatusScheduled, CustomerStagePlaced},
		{StatusAutoAccepted, CustomerStagePlaced},
		{StatusBackordered, CustomerStagePlaced},
		{StatusPending, CustomerStageProcessing},
		{StatusDelayed, CustomerStageProcessing},
		{StatusLoaded, CustomerStageProcessing},
		{StatusInTransit, CustomerStageOutForDelivery},
		{StatusArrived, CustomerStageOutForDelivery},
		{StatusAwaitingPayment, CustomerStageOutForDelivery},
		{StatusPendingCashCollection, CustomerStageOutForDelivery},
		{StatusDeliveredOnCredit, CustomerStageDelivered},
		{StatusFiscalizing, CustomerStageDelivered},
		{StatusCompleted, CustomerStageDelivered},
		{StatusCancelled, CustomerStageCancelled},
		{StatusShopClosedPending, CustomerStageExceptionActionRequired},
		{StatusCancelRequested, CustomerStageExceptionActionRequired},
		{StatusFiscalFailed, CustomerStageExceptionActionRequired},
		{StatusReconciliationRequired, CustomerStageExceptionActionRequired},
		// Legacy aliases still emitted by parts of the backend.
		{Status("ARRIVING"), CustomerStageOutForDelivery},
		{Status("DISPATCHED"), CustomerStageProcessing},
		{Status("ARRIVED_SHOP_CLOSED"), CustomerStageExceptionActionRequired},
	}

	for _, tt := range tests {
		got := ToCustomerStage(tt.status)
		if got != tt.expected {
			t.Errorf("ToCustomerStage(%s) = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestCustomerStageProgressPercent(t *testing.T) {
	if pct := CustomerStageProgressPercent(CustomerStageDelivered); pct != 100 {
		t.Errorf("expected 100 for delivered, got %d", pct)
	}
	if pct := CustomerStageProgressPercent(CustomerStagePlaced); pct != 20 {
		t.Errorf("expected 20 for placed, got %d", pct)
	}
}

// Adding a status to the canonical graph without mapping it in the shared
// fixture must fail CI, otherwise clients silently drift.
func TestEveryCanonicalStatusIsInStageFixture(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "..", "..", "contracts", "order-customer-stage.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f struct {
		Mapping map[string]string `json:"mapping"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	statuses := append(AllStatuses(),
		StatusPending, StatusLoaded, StatusInTransit, StatusArrived, StatusShopClosedPending,
		StatusAwaitingPayment, StatusPendingCashCollection, StatusDeliveredOnCredit,
		StatusFiscalizing, StatusFiscalFailed, StatusCompleted, StatusCancelled,
		StatusCancelRequested, StatusReconciliationRequired, StatusDelayed,
		StatusBackordered, StatusScheduled, StatusAutoAccepted,
	)
	for _, s := range statuses {
		if _, ok := f.Mapping[string(s)]; !ok {
			t.Errorf("canonical status %s missing from contracts/order-customer-stage.json", s)
		}
	}
}
