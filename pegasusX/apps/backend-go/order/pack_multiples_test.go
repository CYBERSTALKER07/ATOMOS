package order

import (
	"testing"
)

func TestSnapToPackMultiple_FailsWithoutImplementation(t *testing.T) {
	// Pack size 6 (case of 6)
	pack := int64(6)

	// Mode NEAREST: 7 -> 6, 10 -> 12, 6 -> 6
	if got := SnapToPackMultiple(7, pack, SnapModeNearest); got != 6 {
		t.Errorf("SnapToPackMultiple(7, 6, NEAREST) = %d; want 6", got)
	}
	if got := SnapToPackMultiple(10, pack, SnapModeNearest); got != 12 {
		t.Errorf("SnapToPackMultiple(10, 6, NEAREST) = %d; want 12", got)
	}
	if got := SnapToPackMultiple(6, pack, SnapModeNearest); got != 6 {
		t.Errorf("SnapToPackMultiple(6, 6, NEAREST) = %d; want 6", got)
	}

	// Mode CEIL: 7 -> 12
	if got := SnapToPackMultiple(7, pack, SnapModeCeil); got != 12 {
		t.Errorf("SnapToPackMultiple(7, 6, CEIL) = %d; want 12", got)
	}

	// Mode FLOOR: 7 -> 6
	if got := SnapToPackMultiple(7, pack, SnapModeFloor); got != 6 {
		t.Errorf("SnapToPackMultiple(7, 6, FLOOR) = %d; want 6", got)
	}

	// Single / EACH (pack <= 1): no snapping
	if got := SnapToPackMultiple(7, 1, SnapModeNearest); got != 7 {
		t.Errorf("SnapToPackMultiple(7, 1, NEAREST) = %d; want 7", got)
	}
	if got := SnapToPackMultiple(7, 0, SnapModeNearest); got != 7 {
		t.Errorf("SnapToPackMultiple(7, 0, NEAREST) = %d; want 7", got)
	}

	// Non-positive qty -> 0
	if got := SnapToPackMultiple(-5, pack, SnapModeNearest); got != 0 {
		t.Errorf("SnapToPackMultiple(-5, 6, NEAREST) = %d; want 0", got)
	}
	if got := SnapToPackMultiple(0, pack, SnapModeNearest); got != 0 {
		t.Errorf("SnapToPackMultiple(0, 6, NEAREST) = %d; want 0", got)
	}
}

func TestEvaluatePackMultipleRules_BatchMultiSKU(t *testing.T) {
	rules := map[string]productPackRule{
		"sku-case-6":   {SaleUnit: "CASE", UnitsPerPack: 6},
		"sku-block-12": {SaleUnit: "block", UnitsPerPack: 12},
		"sku-each":     {SaleUnit: "EACH", UnitsPerPack: 10},
		"sku-case-1":   {SaleUnit: "CASE", UnitsPerPack: 1},
	}

	validItems := []LineItem{
		{SKU: "sku-case-6", Quantity: 18},
		{SKU: "sku-block-12", Quantity: 24},
		{SKU: "sku-each", Quantity: 7},
		{SKU: "sku-case-1", Quantity: 3},
		{SKU: "sku-unknown", Quantity: 5},
	}
	if errs := evaluatePackMultipleRules(validItems, rules); errs != nil {
		t.Fatalf("expected nil errors for valid pack multiples, got %v", errs)
	}

	invalidItems := []LineItem{
		{SKU: "sku-case-6", Quantity: 7},
		{SKU: "sku-block-12", Quantity: 15},
		{SKU: "sku-each", Quantity: 7},
	}
	errs := evaluatePackMultipleRules(invalidItems, rules)
	if len(errs) != 2 {
		t.Fatalf("expected 2 validation errors, got %v", errs)
	}
	if errs["sku-case-6"] != "quantity must be a multiple of 6 (CASE)" {
		t.Errorf("unexpected error for sku-case-6: %q", errs["sku-case-6"])
	}
	if errs["sku-block-12"] != "quantity must be a multiple of 12 (BLOCK)" {
		t.Errorf("unexpected error for sku-block-12: %q", errs["sku-block-12"])
	}
}
