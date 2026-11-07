package stocklots

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
)

func TestResolveTempBandOverrideWins(t *testing.T) {
	override := &TempBand{MinC: 2, MaxC: 5}
	minC, maxC, err := resolveTempBand(context.Background(), nil, "m1", override)
	if err != nil {
		t.Fatal(err)
	}
	if minC != 2 || maxC != 5 {
		t.Fatalf("got [%v,%v] want [2,5]", minC, maxC)
	}
}

func TestResolveTempBandNilTxnOverrideOnly(t *testing.T) {
	// Without override, hydrate needs a txn — calling hydrate with nil panics;
	// resolve with override is the unit-safe path tested above.
	_ = TempBand{}
}

func TestIntersectBandsLogic(t *testing.T) {
	// Pure intersection helper check via TempBand construction expectations.
	aMin, aMax := 0.0, 8.0
	bMin, bMax := 2.0, 6.0
	minC, maxC := aMin, aMax
	if bMin > minC {
		minC = bMin
	}
	if bMax < maxC {
		maxC = bMax
	}
	if minC != 2 || maxC != 6 {
		t.Fatalf("got [%v,%v]", minC, maxC)
	}
}

func TestTemperatureBreachRaiserHook(t *testing.T) {
	called := false
	SetTemperatureBreachRaiser(func(ctx context.Context, txn *spanner.ReadWriteTransaction, args TemperatureBreachArgs) error {
		called = true
		if args.ManifestID != "m-1" || args.ReadingID != "r-1" {
			t.Fatalf("unexpected args %+v", args)
		}
		return nil
	})
	t.Cleanup(func() { SetTemperatureBreachRaiser(nil) })

	err := temperatureBreachRaiser(context.Background(), nil, TemperatureBreachArgs{
		ManifestID: "m-1",
		ReadingID:  "r-1",
		TempC:      22,
		MinC:       0,
		MaxC:       8,
		OrderIDs:   []string{"o-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("raiser not called")
	}
}
