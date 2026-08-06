package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
)

func TestBillingTierWorker_LiveORDER_FINALIZEDShape(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     "ord-1",
		"supplier_id":  "sup-1",
		"retailer_id":  "ret-1",
		"amount_minor": int64(12500),
		"total": map[string]any{
			"amount":   int64(12500),
			"currency": "UZS",
		},
		"currency": "UZS",
		"status":   "COMPLETED",
	})
	if err != nil {
		t.Fatal(err)
	}

	var event orderFinalizedBillingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := billing.ResolveMeterAmountMajor(event.AmountMinor, event.Total.Amount, event.TotalMinor, event.Amount)
	if got != 125.0 {
		t.Fatalf("decoded amount = %v, want 125 (amount_minor path)", got)
	}
	if event.OrderID != "ord-1" || event.SupplierID != "sup-1" {
		t.Fatalf("ids order=%q supplier=%q", event.OrderID, event.SupplierID)
	}
}

func TestBillingTierWorker_HandleMessage_SkipsEmptyIDs(t *testing.T) {
	t.Parallel()
	w := NewBillingTierWorker(billing.NewMeterWorker(nil))
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     "",
		"supplier_id":  "sup-1",
		"amount_minor": 100,
	})
	if err := w.HandleMessage(context.Background(), payload); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestBillingTierWorker_HandleMessage_IgnoresOtherTypes(t *testing.T) {
	t.Parallel()
	w := NewBillingTierWorker(billing.NewMeterWorker(nil))
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventOrderStatusChanged,
		"order_id":     "ord-1",
		"supplier_id":  "sup-1",
		"amount_minor": 100,
	})
	if err := w.HandleMessage(context.Background(), payload); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestResolveMeterAmountMajor_Precedence(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                    string
		amountMinor, nested, tm int64
		legacy                  float64
		want                    float64
	}{
		{"amount_minor wins", 500, 900, 700, 9.0, 5.0},
		{"nested when no amount_minor", 0, 900, 700, 9.0, 9.0},
		{"total_minor", 0, 0, 700, 9.0, 7.0},
		{"legacy major", 0, 0, 0, 9.0, 9.0},
		{"zero", 0, 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := billing.ResolveMeterAmountMajor(tc.amountMinor, tc.nested, tc.tm, tc.legacy)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
