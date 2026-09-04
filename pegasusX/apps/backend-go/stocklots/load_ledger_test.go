package stocklots

import (
	"context"
	"errors"
	"testing"
)

func TestAssertLoadLedgerReady_flagOff(t *testing.T) {
	SetLoadLedgerEnabled(false)
	ResetLoadLedgerMemory()
	if err := AssertLoadLedgerReady(t.Context(), nil, "m1"); err != nil {
		t.Fatalf("flag off: %v", err)
	}
}

func TestAssertLoadLedgerReady_incomplete(t *testing.T) {
	SetLoadLedgerEnabled(true)
	defer SetLoadLedgerEnabled(false)
	ResetLoadLedgerMemory()
	SeedLoadLedgerMemory("m1", []LoadLineSeed{
		{OrderID: "o1", LineItemID: "l1", SkuID: "sku1", RequiredQty: 2},
	})
	err := AssertLoadLedgerReady(t.Context(), nil, "m1")
	if err == nil || !errors.Is(err, ErrLoadLedgerIncomplete) {
		t.Fatalf("expected incomplete, got %v", err)
	}
	if _, err := ScanLoadLineMemory("m1", "o1", "sku1", 2); err != nil {
		t.Fatal(err)
	}
	if err := AssertLoadLedgerReady(t.Context(), nil, "m1"); err != nil {
		t.Fatalf("after full scan: %v", err)
	}
}

func TestAssertLoadLedgerReady_variance(t *testing.T) {
	SetLoadLedgerEnabled(true)
	defer SetLoadLedgerEnabled(false)
	ResetLoadLedgerMemory()
	SeedLoadLedgerMemory("m2", []LoadLineSeed{
		{OrderID: "o1", LineItemID: "l1", SkuID: "sku1", RequiredQty: 5},
	})
	if _, err := ApproveLoadVarianceMemory("m2", "o1", "l1"); err != nil {
		t.Fatal(err)
	}
	if err := AssertLoadLedgerReady(t.Context(), nil, "m2"); err != nil {
		t.Fatalf("variance approved should allow seal: %v", err)
	}
}

func TestEffectivePickWaves_tenantOverride(t *testing.T) {
	SetPickWavesEnabled(false)
	SetFlagEvaluator(&stubFlags{on: map[string]bool{"WMS_PICK_WAVES_ENABLED|WAREHOUSE|wh1": true}})
	defer SetFlagEvaluator(nil)
	if !EffectivePickWaves(t.Context(), "wh1", "") {
		t.Fatal("expected tenant override")
	}
	if EffectivePickWaves(t.Context(), "wh2", "") {
		t.Fatal("other warehouse should be off")
	}
}

type stubFlags struct {
	on map[string]bool
}

func (s *stubFlags) Evaluate(_ context.Context, flagKey, tenantType, tenantID string) (bool, string, error) {
	k := flagKey + "|" + tenantType + "|" + tenantID
	return s.on[k], "tenant_override", nil
}
