package supplier

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestNormalizePinType(t *testing.T) {
	t.Parallel()
	got, err := normalizePinType("location")
	if err != nil || got != proximity.PinTargetLocation {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := normalizePinType("STORE"); err == nil {
		t.Fatal("STORE is not a pin type")
	}
}

func TestAssertPinSameMarket(t *testing.T) {
	t.Parallel()
	if err := assertPinSameMarket("UZ", "UZ"); err != nil {
		t.Fatal(err)
	}
	if err := assertPinSameMarket("UZ", "PK"); !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
	if err := assertPinSameMarket("UZ", ""); !errors.Is(err, auth.ErrGeographyIncomplete) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizePins_CrossCountry(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	svc.lookupPinCountry = func(_ context.Context, _, _, _ string) (string, error) {
		return "PK", nil
	}
	_, err := svc.normalizePins(context.Background(), "sup-1", "UZ", []servicePinInput{{
		TargetType: "LOCATION",
		TargetID:   "loc-s",
	}})
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestEffectiveCoverageMode_Pinned(t *testing.T) {
	t.Parallel()
	mode := proximity.EffectiveCoverageMode(
		proximity.WarehouseCandidate{WarehouseID: "wh-a", CoverageCells: []string{"c"}},
		[]proximity.ServicePin{{WarehouseID: "wh-a", TargetType: proximity.PinTargetLocation, TargetID: "loc-s"}},
	)
	if mode != proximity.CoverageModePinned {
		t.Fatalf("%s", mode)
	}
}
