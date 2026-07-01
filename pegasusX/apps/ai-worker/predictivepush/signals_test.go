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

func TestExternalWeatherSummer(t *testing.T) {
	summer := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	out := externalWeatherSignals("sup-1", summer)
	if len(out) != 1 || out[0].Source != "weather_forecast_stub" {
		t.Fatalf("unexpected weather signals: %+v", out)
	}
}

func TestExternalPOSMonthStart(t *testing.T) {
	out := externalPOSSignals("sup-1", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if len(out) != 1 || out[0].Source != "pos_calendar_stub" {
		t.Fatalf("unexpected pos signals: %+v", out)
	}
}

func testDay() time.Time {
	return time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
}
