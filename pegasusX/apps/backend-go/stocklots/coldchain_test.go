package stocklots

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/spanner"
)

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

	fn := currentTemperatureBreachRaiser()
	if fn == nil {
		t.Fatal("expected registered raiser")
	}
	err := fn(context.Background(), nil, TemperatureBreachArgs{
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

func TestResolveTempBandFromClasses_Defaults(t *testing.T) {
	chilled := resolveTempBandFromClasses([]string{"CHILLED", "GENERAL"}, nil)
	if chilled.MinC != 0 || chilled.MaxC != 8 {
		t.Fatalf("expected chilled band [0,8], got %+v", chilled)
	}
	frozen := resolveTempBandFromClasses([]string{"CHILLED", "FROZEN"}, nil)
	if frozen.MinC != -22 || frozen.MaxC != -15 {
		t.Fatalf("expected frozen band [-22,-15], got %+v", frozen)
	}
	custom := &TempBand{MinC: 2, MaxC: 6}
	gotOverride := resolveTempBandFromClasses([]string{"FROZEN"}, custom)
	if gotOverride != *custom {
		t.Fatalf("expected override %+v, got %+v", *custom, gotOverride)
	}
}

func TestAssertColdChainBaselineReady_FailsClosedWhenEnabledWithoutReading(t *testing.T) {
	orig := ColdChainEnabled()
	SetColdChainEnabled(true)
	t.Cleanup(func() { SetColdChainEnabled(orig) })

	err := AssertColdChainBaselineReady(t.Context(), nil, "man-missing-1")
	if !errors.Is(err, ErrColdChainBaselineRequired) {
		t.Fatalf("expected ErrColdChainBaselineRequired when client nil & no reading, got %v", err)
	}

	if _, err := RecordTemperatureReadingMemory("man-cold-ok-1", "sens-1", 4.2, nil); err != nil {
		t.Fatalf("record memory reading: %v", err)
	}
	t.Cleanup(func() {
		coldChainMemMu.Lock()
		delete(coldChainMemReadings, "man-cold-ok-1")
		coldChainMemMu.Unlock()
	})
	if err := AssertColdChainBaselineReady(t.Context(), nil, "man-cold-ok-1"); err != nil {
		t.Fatalf("expected baseline ready with memory reading, got %v", err)
	}
}
