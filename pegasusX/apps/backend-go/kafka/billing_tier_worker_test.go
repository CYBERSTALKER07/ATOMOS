package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
)

func TestBillingFX_ConvertUSDToUZS(t *testing.T) {
	t.Parallel()
	repo := fxrates.NewMemoryRepository()
	at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_ = repo.Upsert(context.Background(), fxrates.ScaledRate("USD", "UZS", 12_750_000_000, "TEST", at))
	fx := fxrates.NewService(repo)

	// Resolve conversion the same way as HandleMessage for a USD event.
	minor := billing.ResolveMeterAmountMinor(200, 0, 0, 0)
	converted, err := fx.ConvertMinor(context.Background(), "USD", "UZS", minor, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got := billing.MinorToMajor(converted)
	if got != 255.0 {
		t.Fatalf("got %v want 255", got)
	}
}

func TestBillingFX_MissingRateSkips(t *testing.T) {
	t.Parallel()
	fx := fxrates.NewService(fxrates.NewMemoryRepository())
	w := NewBillingTierWorker(billing.NewMeterWorker(nil)).WithFx(fx, "UZS")
	w.Now = func() time.Time { return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) }
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     "ord-usd-1",
		"supplier_id":  "sup-1",
		"amount_minor": int64(200),
		"total": map[string]any{
			"amount":   int64(200),
			"currency": "EUR",
		},
	})
	// MeterWorker with nil client returns error on Process — but missing rate should return nil before meter.
	if err := w.HandleMessage(context.Background(), payload); err != nil {
		t.Fatalf("expected skip nil, got %v", err)
	}
}

func TestBillingFX_SameCurrencyMeters(t *testing.T) {
	t.Parallel()
	w := NewBillingTierWorker(billing.NewMeterWorker(nil)).WithFx(nil, "UZS")
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     "ord-uzs-1",
		"supplier_id":  "sup-1",
		"amount_minor": int64(12500),
		"total": map[string]any{
			"amount":   int64(12500),
			"currency": "UZS",
		},
	})
	// nil MeterWorker client → ProcessOrderFinalized returns error for positive amount.
	err := w.HandleMessage(context.Background(), payload)
	if err == nil {
		t.Fatal("expected meter error with nil client (proves path reached ProcessOrderFinalized)")
	}
}

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

func TestNewBillingTierWorker_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	w := NewBillingTierWorker(billing.NewMeterWorker(nil))
	if w.OperatingCurrency != "UZS" {
		t.Fatalf("operating=%q want UZS from pack", w.OperatingCurrency)
	}
}

func TestNewBillingTierWorker_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	w := NewBillingTierWorker(billing.NewMeterWorker(nil))
	if w.OperatingCurrency != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", w.OperatingCurrency)
	}
}

func TestHandleMessage_PlannedPackSkipsWithoutMetering(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	w := NewBillingTierWorker(billing.NewMeterWorker(nil))
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     "ord-eu-1",
		"supplier_id":  "sup-1",
		"amount_minor": int64(12500),
	})
	if err := w.HandleMessage(context.Background(), payload); err != nil {
		t.Fatalf("planned pack should skip, got %v", err)
	}
}
