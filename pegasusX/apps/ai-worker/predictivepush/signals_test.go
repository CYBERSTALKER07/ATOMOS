package predictivepush

import (
	"testing"
	"time"
)

func TestCompositeSignalProviderCollectEmpty(t *testing.T) {
	p := &CompositeSignalProvider{}
	out, err := p.Collect(nil, "sup-1", testDay())
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected non-nil slice")
	}
}

func TestCompositeSignalProviderCollectSeasonalityFriday(t *testing.T) {
	p := &CompositeSignalProvider{}
	friday := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	out, err := p.Collect(nil, "sup-1", friday)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sig := range out {
		if sig.Source == "seasonality_stub" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected seasonality_stub on Friday")
	}
}

func TestExternalWeatherNoStub(t *testing.T) {
	summer := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	out := externalWeatherSignals("sup-1", summer)
	if len(out) != 0 {
		t.Fatalf("expected empty weather signals, got %+v", out)
	}
}

func TestExternalPOSNoStub(t *testing.T) {
	out := externalPOSSignals("sup-1", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if len(out) != 0 {
		t.Fatalf("expected empty POS signals, got %+v", out)
	}
}

func testDay() time.Time {
	return time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
}
