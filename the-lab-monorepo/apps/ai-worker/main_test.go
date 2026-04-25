package main

import (
	"math"
	"testing"
	"time"
)

// ─── applyCorrection ────────────────────────────────────────────────────────

func TestApplyCorrection_NoWeight_ReturnsRaw(t *testing.T) {
	cs := &correctionStore{weights: make(map[string]map[string]correctionEntry)}
	got := cs.applyCorrection("ret-1", "wh-1", "sku-1", 10)
	if got != 10 {
		t.Errorf("got %d, want 10", got)
	}
}

func TestApplyCorrection_WithWeight(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 1.5}},
	}}
	got := cs.applyCorrection("ret-1", "wh-1", "sku-1", 10)
	if got != 15 { // round(10 * 1.5) = 15
		t.Errorf("got %d, want 15", got)
	}
}

func TestApplyCorrection_WeightLessThanOne(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 0.3}},
	}}
	got := cs.applyCorrection("ret-1", "wh-1", "sku-1", 2)
	// round(2 * 0.3) = round(0.6) = 1, but min is 1
	if got != 1 {
		t.Errorf("got %d, want 1 (min clamp)", got)
	}
}

func TestApplyCorrection_MinClamp(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 0.01}},
	}}
	got := cs.applyCorrection("ret-1", "wh-1", "sku-1", 1)
	if got != 1 {
		t.Errorf("got %d, want 1 (min clamp)", got)
	}
}

func TestApplyCorrection_UnknownRetailer(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 2.0}},
	}}
	got := cs.applyCorrection("ret-unknown", "wh-1", "sku-1", 5)
	if got != 5 {
		t.Errorf("got %d, want 5 (no weight)", got)
	}
}

func TestApplyCorrection_UnknownSku(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 2.0}},
	}}
	got := cs.applyCorrection("ret-1", "wh-1", "sku-unknown", 5)
	if got != 5 {
		t.Errorf("got %d, want 5 (no weight for sku)", got)
	}
}

func TestApplyCorrection_EmptyWarehouse(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:": {"sku-1": {Factor: 1.8}},
	}}
	got := cs.applyCorrection("ret-1", "", "sku-1", 10)
	if got != 18 {
		t.Errorf("got %d, want 18 (empty warehouse key)", got)
	}
}

func TestApplyCorrection_DifferentWarehouses(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-A": {"sku-1": {Factor: 2.0}},
		"ret-1:wh-B": {"sku-1": {Factor: 0.5}},
	}}
	gotA := cs.applyCorrection("ret-1", "wh-A", "sku-1", 10)
	gotB := cs.applyCorrection("ret-1", "wh-B", "sku-1", 10)
	if gotA != 20 {
		t.Errorf("wh-A: got %d, want 20", gotA)
	}
	if gotB != 5 {
		t.Errorf("wh-B: got %d, want 5", gotB)
	}
}

// ─── recordCorrection ───────────────────────────────────────────────────────

func TestRecordCorrection_Rejected(t *testing.T) {
	cs := &correctionStore{weights: make(map[string]map[string]correctionEntry)}
	cs.recordCorrection("ret-1", "wh-1", "sku-1", "rejected", "", "")
	entry := cs.weights["ret-1:wh-1"]["sku-1"]
	if entry.Factor != 0.5 {
		t.Errorf("rejected weight = %f, want 0.5", entry.Factor)
	}
}

func TestRecordCorrection_Amount(t *testing.T) {
	cs := &correctionStore{weights: make(map[string]map[string]correctionEntry)}
	cs.recordCorrection("ret-1", "wh-1", "sku-1", "amount", "10", "20")
	entry := cs.weights["ret-1:wh-1"]["sku-1"]
	// existing=1.0 (no prior), ratio=20/10=2.0, EMA = 1.0*0.7 + 2.0*0.3 = 1.3
	expected := 1.3
	if math.Abs(entry.Factor-expected) > 0.001 {
		t.Errorf("amount weight = %f, want %f", entry.Factor, expected)
	}
}

func TestRecordCorrection_Amount_WithExisting(t *testing.T) {
	cs := &correctionStore{weights: map[string]map[string]correctionEntry{
		"ret-1:wh-1": {"sku-1": {Factor: 1.3}},
	}}
	cs.recordCorrection("ret-1", "wh-1", "sku-1", "amount", "10", "15")
	entry := cs.weights["ret-1:wh-1"]["sku-1"]
	// existing=1.3, ratio=15/10=1.5, EMA = 1.3*0.7 + 1.5*0.3 = 0.91 + 0.45 = 1.36
	expected := 1.36
	if math.Abs(entry.Factor-expected) > 0.001 {
		t.Errorf("weight = %f, want %f", entry.Factor, expected)
	}
}

func TestRecordCorrection_Amount_ZeroOld_NoChange(t *testing.T) {
	cs := &correctionStore{weights: make(map[string]map[string]correctionEntry)}
	cs.recordCorrection("ret-1", "wh-1", "sku-1", "amount", "0", "20")
	if _, ok := cs.weights["ret-1:wh-1"]; ok {
		if _, ok := cs.weights["ret-1:wh-1"]["sku-1"]; ok {
			t.Error("zero old amount should not create a weight")
		}
	}
}

func TestRecordCorrection_UnknownField_NoOp(t *testing.T) {
	cs := &correctionStore{weights: make(map[string]map[string]correctionEntry)}
	cs.recordCorrection("ret-1", "wh-1", "sku-1", "unknown_field", "1", "2")
	if retailers, ok := cs.weights["ret-1:wh-1"]; ok {
		if _, ok := retailers["sku-1"]; ok {
			t.Error("unknown field should not create a weight")
		}
	}
}

// ─── deduplicator ───────────────────────────────────────────────────────────

func TestDeduplicator_FirstCall_NotSkipped(t *testing.T) {
	d := &deduplicator{last: make(map[string]time.Time)}
	if d.shouldSkip("ret-1", "wh-1") {
		t.Error("first call should not be skipped")
	}
}

func TestDeduplicator_SecondCall_Skipped(t *testing.T) {
	d := &deduplicator{last: make(map[string]time.Time)}
	d.shouldSkip("ret-1", "wh-1") // first call registers timestamp
	if !d.shouldSkip("ret-1", "wh-1") {
		t.Error("second call within 1h should be skipped")
	}
}

func TestDeduplicator_DifferentRetailers_Independent(t *testing.T) {
	d := &deduplicator{last: make(map[string]time.Time)}
	d.shouldSkip("ret-1", "wh-1")
	if d.shouldSkip("ret-2", "wh-1") {
		t.Error("different retailer should not be skipped")
	}
}

func TestDeduplicator_ExpiredEntry(t *testing.T) {
	d := &deduplicator{last: map[string]time.Time{
		"ret-1:wh-1": time.Now().Add(-2 * time.Hour), // 2 hours ago — past the 1h window
	}}
	if d.shouldSkip("ret-1", "wh-1") {
		t.Error("expired entry should allow a new prediction")
	}
}

func TestDeduplicator_DifferentWarehouses_Independent(t *testing.T) {
	d := &deduplicator{last: make(map[string]time.Time)}
	d.shouldSkip("ret-1", "wh-A")
	if d.shouldSkip("ret-1", "wh-B") {
		t.Error("different warehouse should not be skipped")
	}
}

func TestDeduplicator_EmptyWarehouse(t *testing.T) {
	d := &deduplicator{last: make(map[string]time.Time)}
	if d.shouldSkip("ret-1", "") {
		t.Error("first call with empty warehouse should not be skipped")
	}
	if !d.shouldSkip("ret-1", "") {
		t.Error("second call with empty warehouse within 1h should be skipped")
	}
}

// ─── resolveStartDateForSku ─────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func TestResolveStartDate_VariantOverride(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
		VariantOverrides:   []OverrideEntry{{SkuID: "sku-1", AnalyticsStartDate: strPtr("2024-06-01")}},
		ProductOverrides:   []OverrideEntry{{ProductID: "prod-1", AnalyticsStartDate: strPtr("2024-03-01")}},
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-06-01" {
		t.Errorf("got %q, want variant override 2024-06-01", got)
	}
}

func TestResolveStartDate_ProductOverride(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
		ProductOverrides:   []OverrideEntry{{ProductID: "prod-1", AnalyticsStartDate: strPtr("2024-03-01")}},
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-03-01" {
		t.Errorf("got %q, want product override 2024-03-01", got)
	}
}

func TestResolveStartDate_CategoryOverride(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
		CategoryOverrides:  []OverrideEntry{{CategoryID: "cat-1", AnalyticsStartDate: strPtr("2024-04-01")}},
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-04-01" {
		t.Errorf("got %q, want category override 2024-04-01", got)
	}
}

func TestResolveStartDate_SupplierOverride(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
		SupplierOverrides:  []OverrideEntry{{SupplierID: "sup-1", AnalyticsStartDate: strPtr("2024-05-01")}},
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-05-01" {
		t.Errorf("got %q, want supplier override 2024-05-01", got)
	}
}

func TestResolveStartDate_GlobalFallback(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-01-01" {
		t.Errorf("got %q, want global 2024-01-01", got)
	}
}

func TestResolveStartDate_NoOverrides_Empty(t *testing.T) {
	s := &AutoOrderSettings{}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolveStartDate_EmptyOverrideSkipped(t *testing.T) {
	s := &AutoOrderSettings{
		AnalyticsStartDate: strPtr("2024-01-01"),
		VariantOverrides:   []OverrideEntry{{SkuID: "sku-1", AnalyticsStartDate: strPtr("")}},
	}
	got := resolveStartDateForSku(s, "sku-1", "prod-1", "cat-1", "sup-1")
	if got != "2024-01-01" {
		t.Errorf("got %q — empty override should fall through to global", got)
	}
}

// ─── calculateMedianHours ───────────────────────────────────────────────────

func TestCalculateMedianHours_Empty(t *testing.T) {
	if got := calculateMedianHours(nil); got != 0 {
		t.Errorf("got %f, want 0", got)
	}
}

func TestCalculateMedianHours_Single(t *testing.T) {
	if got := calculateMedianHours([]float64{5.0}); got != 5.0 {
		t.Errorf("got %f, want 5.0", got)
	}
}

func TestCalculateMedianHours_OddCount(t *testing.T) {
	got := calculateMedianHours([]float64{3.0, 1.0, 2.0})
	if got != 2.0 {
		t.Errorf("got %f, want 2.0 (median of [1,2,3])", got)
	}
}

func TestCalculateMedianHours_EvenCount(t *testing.T) {
	got := calculateMedianHours([]float64{4.0, 1.0, 3.0, 2.0})
	// sorted: [1,2,3,4] → (2+3)/2 = 2.5
	if got != 2.5 {
		t.Errorf("got %f, want 2.5", got)
	}
}

func TestCalculateMedianHours_Unsorted(t *testing.T) {
	got := calculateMedianHours([]float64{100, 1, 50})
	if got != 50 {
		t.Errorf("got %f, want 50 (should sort before computing)", got)
	}
}

// ─── calculateMedianInt64 ───────────────────────────────────────────────────

func TestCalculateMedianInt64_Empty(t *testing.T) {
	if got := calculateMedianInt64(nil); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCalculateMedianInt64_Single(t *testing.T) {
	if got := calculateMedianInt64([]int64{42}); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestCalculateMedianInt64_OddCount(t *testing.T) {
	got := calculateMedianInt64([]int64{5, 1, 3})
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestCalculateMedianInt64_EvenCount(t *testing.T) {
	got := calculateMedianInt64([]int64{4, 1, 3, 2})
	// sorted: [1,2,3,4] → (2+3)/2 = 2 (integer division)
	if got != 2 {
		t.Errorf("got %d, want 2 (integer median)", got)
	}
}

// ─── applyPackagingConstraint ───────────────────────────────────────────────

func TestApplyPackaging_BasicCeil(t *testing.T) {
	// raw=7, moq=6, step=6 → ceil(7/6)*6 = 12
	got := applyPackagingConstraint(7, 6, 6)
	if got != 12 {
		t.Errorf("got %d, want 12", got)
	}
}

func TestApplyPackaging_ExactMultiple(t *testing.T) {
	got := applyPackagingConstraint(12, 6, 6)
	if got != 12 {
		t.Errorf("got %d, want 12", got)
	}
}

func TestApplyPackaging_BelowMOQ(t *testing.T) {
	// raw=1, moq=10, step=5 → constrained=5 (ceil(1/5)*5), then moq enforcement → ceil(10/5)*5 = 10
	got := applyPackagingConstraint(1, 10, 5)
	if got != 10 {
		t.Errorf("got %d, want 10 (moq enforcement)", got)
	}
}

func TestApplyPackaging_ZeroRaw(t *testing.T) {
	got := applyPackagingConstraint(0, 6, 6)
	if got != 6 {
		t.Errorf("got %d, want 6 (moq when raw<=0)", got)
	}
}

func TestApplyPackaging_NegativeRaw(t *testing.T) {
	got := applyPackagingConstraint(-5, 6, 3)
	if got != 6 {
		t.Errorf("got %d, want 6 (moq when raw<=0)", got)
	}
}

func TestApplyPackaging_ZeroStep(t *testing.T) {
	got := applyPackagingConstraint(7, 5, 0)
	// step defaults to 1 → ceil(7/1)*1=7, which is >= moq(5)
	if got != 7 {
		t.Errorf("got %d, want 7 (step defaults to 1)", got)
	}
}

func TestApplyPackaging_ZeroMOQ(t *testing.T) {
	got := applyPackagingConstraint(7, 0, 3)
	// moq defaults to stepSize(3) → constrained = ceil(7/3)*3 = 9, 9 >= 3 ✓
	if got != 9 {
		t.Errorf("got %d, want 9", got)
	}
}

func TestApplyPackaging_LargeValues(t *testing.T) {
	got := applyPackagingConstraint(999, 100, 24)
	// ceil(999/24)*24 = ceil(41.625)*24 = 42*24 = 1008, >= moq(100) ✓
	if got != 1008 {
		t.Errorf("got %d, want 1008", got)
	}
}

func TestApplyPackaging_StepSizeOne(t *testing.T) {
	got := applyPackagingConstraint(7, 5, 1)
	if got != 7 {
		t.Errorf("got %d, want 7 (step=1 is identity ceiling)", got)
	}
}

// ─── Event Struct Fields ────────────────────────────────────────────────────

func TestOrderCompletedEvent_Fields(t *testing.T) {
	e := OrderCompletedEvent{OrderID: "ord-1", RetailerID: "ret-1", WarehouseId: "wh-1", Timestamp: "2024-01-01T00:00:00Z"}
	if e.OrderID != "ord-1" || e.RetailerID != "ret-1" || e.WarehouseId != "wh-1" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestHistoryItem_Fields(t *testing.T) {
	h := HistoryItem{
		SkuID: "sku-1", ProductID: "prod-1", CategoryID: "cat-1",
		SupplierID: "sup-1", WarehouseId: "wh-1", Quantity: 10, UnitPrice: 5000,
		OrderDate: "2024-01-01", MinimumOrderQty: 6, StepSize: 6,
	}
	if h.Quantity != 10 || h.StepSize != 6 || h.WarehouseId != "wh-1" {
		t.Errorf("unexpected: %+v", h)
	}
}

func TestAutoOrderSettings_GlobalFallback(t *testing.T) {
	s := AutoOrderSettings{
		GlobalEnabled:      true,
		AnalyticsStartDate: strPtr("2024-01-01"),
	}
	if !s.GlobalEnabled {
		t.Error("GlobalEnabled should be true")
	}
}
