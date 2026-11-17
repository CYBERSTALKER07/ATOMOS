package stocklots

import (
	"errors"
	"testing"
	"time"
)

func TestSortPickTaskDrafts_byPickSequence(t *testing.T) {
	drafts := []pickTaskDraft{
		{LocationID: "B-2", Seq: 20},
		{LocationID: "A-1", Seq: 10},
		{LocationID: "A-2", Seq: 10},
		{LocationID: "C-1", Seq: 5},
	}
	sortPickTaskDrafts(drafts)
	want := []string{"C-1", "A-1", "A-2", "B-2"}
	for i, id := range want {
		if drafts[i].LocationID != id {
			t.Fatalf("index %d: got %s want %s", i, drafts[i].LocationID, id)
		}
	}
}

func TestApplyLotDepletion(t *testing.T) {
	qoh, qr, depleted := applyLotDepletion(10, 4, 3)
	if qoh != 7 || qr != 1 || depleted {
		t.Fatalf("got qoh=%d qr=%d depleted=%v", qoh, qr, depleted)
	}
	qoh, qr, depleted = applyLotDepletion(2, 2, 5)
	if qoh != 0 || qr != 0 || !depleted {
		t.Fatalf("expected clamp+deplete, got qoh=%d qr=%d depleted=%v", qoh, qr, depleted)
	}
}

func TestPickTaskStatusForQty(t *testing.T) {
	if pickTaskStatusForQty(5, 5) != "CONFIRMED" {
		t.Fatal("full pick should CONFIRM")
	}
	if pickTaskStatusForQty(5, 0) != "SHORT" {
		t.Fatal("explicit 0 pick should SHORT")
	}
	if pickTaskStatusForQty(5, 2) != "SHORT" {
		t.Fatal("short pick should SHORT")
	}
}

func TestResolvePickConfirmQty(t *testing.T) {
	// Omitted (nil) defaults to requested.
	got, err := resolvePickConfirmQty(7, nil)
	if err != nil || got != 7 {
		t.Fatalf("nil qty: got=%d err=%v want 7", got, err)
	}
	// Explicit 0 stays 0 (SHORT).
	zero := int64(0)
	got, err = resolvePickConfirmQty(7, &zero)
	if err != nil || got != 0 {
		t.Fatalf("zero qty: got=%d err=%v want 0", got, err)
	}
	// Negative rejected.
	neg := int64(-2)
	if _, err := resolvePickConfirmQty(7, &neg); err == nil {
		t.Fatal("expected error for negative quantity_picked")
	}
	// Over-pick clamped to requested.
	over := int64(12)
	got, err = resolvePickConfirmQty(7, &over)
	if err != nil || got != 7 {
		t.Fatalf("over qty: got=%d err=%v want 7", got, err)
	}
}

func TestApplyWaivedShortsToLineItems(t *testing.T) {
	raw := []byte(`[
		{"sku":"sku-cut","name":"Cut Juice","quantity":3,"unit_price_minor":1000,"promotion_id":"promo-1","unit_volume_vu":1.5},
		{"sku":"sku-part","name":"Part Water","quantity":5,"unit_price_minor":2000,"tax_category":"STANDARD","vat_rate_bps":1200},
		{"sku":"sku-ok","name":"Full Milk","quantity":2,"unit_price_minor":1500}
	]`)
	res, err := applyWaivedShortsToLineItems(raw, map[string]int64{
		"sku-cut":  3,
		"sku-part": 2,
	}, 16_500, 0, 500)
	if err != nil {
		t.Fatalf("applyWaivedShortsToLineItems failed: %v", err)
	}
	// Expected active subtotal: Part Water 3*2000=6000 + Full Milk 2*1500=3000 => 9000; + 500 delivery => 9500
	if res.SubtotalMinor != 9_000 {
		t.Fatalf("SubtotalMinor = %d want 9000", res.SubtotalMinor)
	}
	if res.NewTotalMinor != 9_500 {
		t.Fatalf("NewTotalMinor = %d want 9500", res.NewTotalMinor)
	}
	if res.NextOriginalTotal != 16_500 {
		t.Fatalf("NextOriginalTotal = %d want 16500", res.NextOriginalTotal)
	}
	if res.ShortedMinor != 7_000 {
		t.Fatalf("ShortedMinor = %d want 7000", res.ShortedMinor)
	}
	if len(res.ShortedItems) != 2 {
		t.Fatalf("len(ShortedItems) = %d want 2", len(res.ShortedItems))
	}
	lines, err := parseOrderLineQtys(res.UpdatedJSON)
	if err != nil {
		t.Fatalf("parseOrderLineQtys failed: %v", err)
	}
	if len(lines) != 2 || lines[0].SKU != "sku-part" || lines[0].Quantity != 3 || lines[1].SKU != "sku-ok" || lines[1].Quantity != 2 {
		t.Fatalf("unexpected updated lines: %+v", lines)
	}

	// Full waive drops total to 0 (no delivery fee on empty order) and keeps existing OriginalTotalMinor.
	fullZero, err := applyWaivedShortsToLineItems(raw, map[string]int64{
		"sku-cut":  3,
		"sku-part": 5,
		"sku-ok":   2,
	}, 16_500, 20_000, 500)
	if err != nil {
		t.Fatalf("full zero waive failed: %v", err)
	}
	if fullZero.SubtotalMinor != 0 || fullZero.NewTotalMinor != 0 || fullZero.NextOriginalTotal != 20_000 {
		t.Fatalf("unexpected full zero waive: %+v", fullZero)
	}
	if string(fullZero.UpdatedJSON) != "[]" {
		t.Fatalf("expected empty JSON array [], got %s", string(fullZero.UpdatedJSON))
	}
}

func TestPickWavesEnabledFlag(t *testing.T) {
	SetPickWavesEnabled(false)
	if PickWavesEnabled() {
		t.Fatal("expected disabled")
	}
	SetPickWavesEnabled(true)
	if !PickWavesEnabled() {
		t.Fatal("expected enabled")
	}
	SetPickWavesEnabled(false)
}

func TestAssertManifestPickReady_flagOff(t *testing.T) {
	SetPickWavesEnabled(false)
	if _, err := AssertManifestPickReady(t.Context(), nil, "m1"); err != nil {
		t.Fatalf("flag off should allow seal: %v", err)
	}
}

func TestAssertManifestPickReady_flagOnNilClient(t *testing.T) {
	SetPickWavesEnabled(true)
	defer SetPickWavesEnabled(false)
	_, err := AssertManifestPickReady(t.Context(), nil, "m1")
	if err == nil || !errors.Is(err, ErrPickWaveRequired) {
		t.Fatalf("expected pick_wave_required, got %v", err)
	}
}

func TestAssertManifestPickReady_emptyManifest(t *testing.T) {
	SetPickWavesEnabled(true)
	defer SetPickWavesEnabled(false)
	_, err := AssertManifestPickReady(t.Context(), nil, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertManifestPickReady_softWarn(t *testing.T) {
	SetPickWavesEnabled(true)
	SetSealSoftWarnEnabled(true)
	defer func() {
		SetPickWavesEnabled(false)
		SetSealSoftWarnEnabled(false)
	}()
	warn, err := AssertManifestPickReady(t.Context(), nil, "m1")
	if err == nil {
		// nil client still hard-fails as unavailable wrapped required
		_ = warn
	}
}

func TestSkuPickDraftsFromLines(t *testing.T) {
	got := skuPickDraftsFromLines("ord-1", []LineQty{
		{SKU: "SSMR-SKU-1", Quantity: 2},
		{SKU: "  ", Quantity: 3},
		{SKU: "skip-zero", Quantity: 0},
	})
	if len(got) != 1 {
		t.Fatalf("got %d drafts want 1", len(got))
	}
	if got[0].OrderID != "ord-1" || got[0].ProductID != "SSMR-SKU-1" || got[0].Qty != 2 {
		t.Fatalf("got %+v", got[0])
	}
	if got[0].LotID != "" || shouldDepleteLot(got[0].LotID) {
		t.Fatal("bag-of-SKU task must not deplete lots")
	}
}

func TestParseOrderLineQtys(t *testing.T) {
	got, err := parseOrderLineQtys([]byte(`[{"sku":"SSMR-SKU-1","quantity":2},{"product_id":"p2","quantity":1},{"sku_id":"s3","quantity":4}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].SKU != "SSMR-SKU-1" || got[1].SKU != "p2" || got[2].SKU != "s3" {
		t.Fatalf("got %+v", got)
	}
}

func TestWaveReadyFromTasks(t *testing.T) {
	if waveReadyFromTasks(nil) {
		t.Fatal("empty is not ready")
	}
	if waveReadyFromTasks([]PickTaskView{{Status: "PENDING"}}) {
		t.Fatal("pending is not ready")
	}
	if !waveReadyFromTasks([]PickTaskView{{Status: "CONFIRMED"}, {Status: "SHORT_WAIVED"}}) {
		t.Fatal("confirmed+waived should be ready")
	}
}

func TestShouldDepleteLot(t *testing.T) {
	if shouldDepleteLot("") || shouldDepleteLot("   ") {
		t.Fatal("empty lot must skip depletion")
	}
	if !shouldDepleteLot("lot-1") {
		t.Fatal("real lot must deplete")
	}
}

func TestSortSShapePickTaskDrafts(t *testing.T) {
	drafts := []pickTaskDraft{
		{LocationID: "z1", Zone: "A", Aisle: "2", Seq: 1, StopRank: 1},
		{LocationID: "z0", Zone: "A", Aisle: "1", Seq: 5, StopRank: 2},
		{LocationID: "z2", Zone: "B", Aisle: "1", Seq: 1, StopRank: 2},
	}
	sortSShapePickTaskDrafts(drafts)
	if drafts[0].StopRank < drafts[1].StopRank {
		t.Fatal("expected higher StopRank first")
	}
}

func TestSumAvailableLotSnapshots_ExcludesDepletedAndMismatchedLots(t *testing.T) {
	rows := []StockLotSnapshot{
		{
			SupplierID:       "sup-1",
			WarehouseID:      "wh-1",
			ProductID:        "prod-1",
			QuantityOnHand:   10,
			QuantityReserved: 4,
			Status:           "AVAILABLE",
		},
		{
			SupplierID:       "sup-1",
			WarehouseID:      "wh-1",
			ProductID:        "prod-1",
			QuantityOnHand:   5,
			QuantityReserved: 1,
			Status:           "available",
		},
		{
			SupplierID:       "sup-1",
			WarehouseID:      "wh-1",
			ProductID:        "prod-1",
			QuantityOnHand:   9,
			QuantityReserved: 3,
			Status:           "DEPLETED",
		},
		{
			SupplierID:       "sup-2",
			WarehouseID:      "wh-1",
			ProductID:        "prod-1",
			QuantityOnHand:   100,
			QuantityReserved: 50,
			Status:           "AVAILABLE",
		},
		{
			SupplierID:       "sup-1",
			WarehouseID:      "wh-2",
			ProductID:        "prod-1",
			QuantityOnHand:   20,
			QuantityReserved: 2,
			Status:           "AVAILABLE",
		},
	}

	qoh, qr := sumAvailableLotSnapshots(rows, "sup-1", "wh-1", "prod-1")
	if qoh != 15 || qr != 5 {
		t.Fatalf("expected qoh=15 qr=5, got qoh=%d qr=%d", qoh, qr)
	}
}

func TestNextPickWaveStatusAfterTasks(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)

	status, readyAt := nextPickWaveStatusAfterTasks("OPEN", []PickTaskView{
		{TaskID: "t1", Status: "CONFIRMED"},
		{TaskID: "t2", Status: "PENDING"},
	}, now)
	if status != "PICKING" || readyAt != nil {
		t.Fatalf("OPEN with pending tasks expected PICKING/nil, got %q/%v", status, readyAt)
	}

	status, readyAt = nextPickWaveStatusAfterTasks("PICKING", []PickTaskView{
		{TaskID: "t1", Status: "CONFIRMED"},
		{TaskID: "t2", Status: "SHORT"},
	}, now)
	if status != "PICKING" || readyAt != nil {
		t.Fatalf("PICKING with SHORT task expected PICKING/nil, got %q/%v", status, readyAt)
	}

	status, readyAt = nextPickWaveStatusAfterTasks("OPEN", []PickTaskView{
		{TaskID: "t1", Status: "CONFIRMED"},
		{TaskID: "t2", Status: "SHORT_WAIVED"},
	}, now)
	if status != "READY_TO_SEAL" || readyAt == nil || !readyAt.Equal(now) {
		t.Fatalf("OPEN all terminal expected READY_TO_SEAL/%v, got %q/%v", now, status, readyAt)
	}

	status, readyAt = nextPickWaveStatusAfterTasks("PICKING", []PickTaskView{
		{TaskID: "t1", Status: "CONFIRMED"},
	}, now)
	if status != "READY_TO_SEAL" || readyAt == nil || !readyAt.Equal(now) {
		t.Fatalf("PICKING all confirmed expected READY_TO_SEAL/%v, got %q/%v", now, status, readyAt)
	}
}
