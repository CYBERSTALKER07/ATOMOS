package synthesis

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildSupplierRecommendation_Reorder(t *testing.T) {
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	signal := OrderSignal{
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		TotalMinor: 100_000,
		Currency:   "UZS",
		LineItems: []Line{
			{ProductID: "p1", Quantity: 10, UnitPrice: 1000},
			{ProductID: "p2", Quantity: 5, UnitPrice: 2000},
		},
	}
	rec := BuildSupplierRecommendation(signal, nil, now)
	if rec.Action != "SUGGEST_REORDER" {
		t.Fatalf("action = %q", rec.Action)
	}
	if rec.Score < 0.4 || rec.Score > 1 {
		t.Fatalf("score out of range: %v", rec.Score)
	}
	if rec.SupplierID != "sup-1" || rec.AggregateType != "ORDER" {
		t.Fatalf("ids: %+v", rec)
	}
	if len(rec.PredictionID) == 0 || len(rec.AggregateID) != 36 {
		t.Fatalf("ids not uuid-shaped: pred=%q agg=%q", rec.PredictionID, rec.AggregateID)
	}
}

func TestBuildSupplierRecommendation_HighDemand(t *testing.T) {
	now := time.Now().UTC()
	signal := OrderSignal{
		OrderID:    "ord-big",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		TotalMinor: 900_000,
		LineItems:  []Line{{ProductID: "p1", Quantity: 80, UnitPrice: 10000}},
	}
	history := []OrderSignal{{OrderID: "o0"}, {OrderID: "o1"}, {OrderID: "o2"}}
	rec := BuildSupplierRecommendation(signal, history, now)
	if rec.Action != "FLAG_HIGH_DEMAND" {
		t.Fatalf("action = %q want FLAG_HIGH_DEMAND", rec.Action)
	}
	if rec.Score < 0.55 {
		t.Fatalf("expected higher score, got %v", rec.Score)
	}
}

func TestParseOrderSignal_FlatAndNested(t *testing.T) {
	flat, _ := json.Marshal(OrderSignal{
		OrderID: "ord-flat", SupplierID: "s1", RetailerID: "r1", TotalMinor: 10,
	})
	sig, err := ParseOrderSignal(flat)
	if err != nil || sig.OrderID != "ord-flat" {
		t.Fatalf("flat: %+v err=%v", sig, err)
	}

	nested, _ := json.Marshal(map[string]any{
		"type": "ORDER_CREATED",
		"payload": map[string]any{
			"order_id":    "ord-nested",
			"supplier_id": "s2",
			"retailer_id": "r2",
		},
	})
	sig, err = ParseOrderSignal(nested)
	if err != nil || sig.OrderID != "ord-nested" || sig.SupplierID != "s2" {
		t.Fatalf("nested: %+v err=%v", sig, err)
	}
}

func TestParseOrderSignal_SkipsAIPreorderLoopInEnginePath(t *testing.T) {
	// Engine.HandleOrderEvent returns nil without write when source is AI_PREORDER.
	// Pure check that source field parses.
	raw, _ := json.Marshal(OrderSignal{
		OrderID: "ord-ai", SupplierID: "s1", RetailerID: "r1", OrderSource: "AI_PREORDER",
	})
	sig, err := ParseOrderSignal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sig.OrderSource != "AI_PREORDER" {
		t.Fatalf("source=%q", sig.OrderSource)
	}
}
