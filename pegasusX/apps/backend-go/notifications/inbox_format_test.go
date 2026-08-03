package notifications

import (
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestFormatFromEvent_OrderCreated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":     events.EventOrderCreated,
		"order_id": "ord-99",
	})
	got := FormatFromEvent(events.EventOrderCreated, payload)
	if got.Title != "New Order Received" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/orders/ord-99" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestShouldPersistInboxEvent_TelemetryFalse(t *testing.T) {
	if ShouldPersistInboxEvent(events.EventDriverLocationUpdated) {
		t.Fatal("telemetry should not persist")
	}
}

func TestFormatFromEvent_ShopClosed(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":      events.EventShopClosed,
		"order_id":  "ord-sc-1",
		"driver_id": "drv-1",
	})
	got := FormatFromEvent(events.EventShopClosed, payload)
	if got.Title != "Shop Closed Reported" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/exceptions/shop-closed" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_DriverCreated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":         events.EventDriverCreated,
		"driver_id":    "drv-42",
		"home_node_id": "wh-1",
	})
	got := FormatFromEvent(events.EventDriverCreated, payload)
	if got.Title != "Driver Added" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/org-fleet" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_ManifestSealed(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventManifestSealed,
		"manifest_id": "mf-1",
	})
	got := FormatFromEvent(events.EventManifestSealed, payload)
	if got.Title != "Manifest Sealed" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/manifests/mf-1" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_ManifestOrderException(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventManifestOrderException,
		"manifest_id": "mf-2",
		"order_id":    "ord-1",
		"reason":      "capacity overflow",
	})
	got := FormatFromEvent(events.EventManifestOrderException, payload)
	if got.Title != "Manifest Exception" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/manifest-exceptions" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_RetailerPriceOverride(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventRetailerPriceOverride,
		"retailer_id": "ret-1",
		"product_id":  "SSMR-SKU-1",
		"price_minor": 42000,
		"action":      "CREATED",
	})
	got := FormatFromEvent(events.EventRetailerPriceOverride, payload)
	if got.Title != "Custom pricing applied" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/catalog" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_CashReconciliationCreated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":               "cash_reconciliation.created",
		"reconciliation_id":  "cr-1",
		"driver_id":          "drv-1",
		"difference_minor":   500,
	})
	got := FormatFromEvent("cash_reconciliation.created", payload)
	if got.DeepLink != "/treasury/cash-reconciliations" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
	if got.Priority != "high" {
		t.Fatalf("priority=%q", got.Priority)
	}
}

func TestFormatFromEvent_CreditNoteIssued(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":           "credit_note.issued",
		"credit_note_id": "cn-9",
		"order_id":       "ord-9",
	})
	got := FormatFromEvent("credit_note.issued", payload)
	if got.DeepLink != "/finance/credit-notes" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_CreditScoreUpdated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":        "credit.score.updated",
		"retailer_id": "ret-1",
		"score":       720,
		"risk_tier":   "LOW",
	})
	got := FormatFromEvent("credit.score.updated", payload)
	if got.Title != "Credit score updated" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/credit/collections" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestFormatFromEvent_ReorderSuggestionUpdated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":          "reorder.suggestion.updated",
		"retailer_id":   "ret-2",
		"sku":           "SKU-1",
		"suggested_qty": 12,
	})
	got := FormatFromEvent("reorder.suggestion.updated", payload)
	if got.Title != "Reorder suggestion" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/replenishment/suggestions" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}
