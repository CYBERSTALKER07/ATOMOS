package replenishment

import (
	"net/url"
	"reflect"
	"testing"
)

func TestMergeDemandVelocities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		base       float64
		stUnits    float64
		days       int
		wantDemand float64
		wantSrc    []string
	}{
		{
			name:       "wholesale only",
			base:       10,
			stUnits:    0,
			days:       7,
			wantDemand: 10,
			wantSrc:    []string{SourceWholesaleHistory},
		},
		{
			name:       "POS only",
			base:       0,
			stUnits:    14,
			days:       7,
			wantDemand: 2,
			wantSrc:    []string{SourceStorePOS},
		},
		{
			name:       "POS hotter than base",
			base:       1,
			stUnits:    21,
			days:       7,
			wantDemand: 3,
			wantSrc:    []string{SourceWholesaleHistory, SourceStorePOS},
		},
		{
			name:       "base hotter than POS",
			base:       5,
			stUnits:    7,
			days:       7,
			wantDemand: 5,
			wantSrc:    []string{SourceWholesaleHistory, SourceStorePOS},
		},
		{
			name:       "equal velocities both sources",
			base:       2,
			stUnits:    14,
			days:       7,
			wantDemand: 2,
			wantSrc:    []string{SourceWholesaleHistory, SourceStorePOS},
		},
		{
			name:       "empty both",
			base:       0,
			stUnits:    0,
			days:       7,
			wantDemand: 0,
			wantSrc:    []string{SourceWholesaleHistory},
		},
		{
			name:       "default days when zero",
			base:       0,
			stUnits:    7,
			days:       0,
			wantDemand: 1,
			wantSrc:    []string{SourceStorePOS},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, src := MergeDemandVelocities(tt.base, tt.stUnits, tt.days)
			if got != tt.wantDemand {
				t.Fatalf("demand=%v want %v", got, tt.wantDemand)
			}
			if !reflect.DeepEqual(src, tt.wantSrc) {
				t.Fatalf("sources=%v want %v", src, tt.wantSrc)
			}
		})
	}
}

func TestStripSellThroughFactor(t *testing.T) {
	t.Parallel()
	base, f := StripSellThroughFactor(12, 3, map[string]float64{"SELL_THROUGH": 5, "OTHER": 1})
	if f != 5 {
		t.Fatalf("f=%v", f)
	}
	if base != 7 {
		t.Fatalf("base=%v want 7", base)
	}
	// negative clamp to BaseVelocity
	base2, _ := StripSellThroughFactor(2, 4, map[string]float64{"SELL_THROUGH": 10})
	if base2 != 4 {
		t.Fatalf("base2=%v want 4", base2)
	}
	// no factor
	base3, f3 := StripSellThroughFactor(9, 1, nil)
	if base3 != 9 || f3 != 0 {
		t.Fatalf("base3=%v f3=%v", base3, f3)
	}
}

func TestEncodeDecodeSourcesJSON(t *testing.T) {
	t.Parallel()
	raw := EncodeSourcesJSON([]string{SourceStorePOS, SourceWholesaleHistory})
	got := DecodeSourcesJSON(raw)
	if !reflect.DeepEqual(got, []string{SourceStorePOS, SourceWholesaleHistory}) {
		t.Fatalf("got=%v", got)
	}
}

func TestParseFactorsJSON(t *testing.T) {
	t.Parallel()
	m := ParseFactorsJSON(`{"SELL_THROUGH":5.5,"PROMO":1}`)
	if m["SELL_THROUGH"] != 5.5 || m["PROMO"] != 1 {
		t.Fatalf("%+v", m)
	}
}

func TestParseDemandSourcesQuery(t *testing.T) {
	t.Parallel()
	got, err := ParseDemandSourcesQuery(url.Values{"source": []string{"store_pos"}})
	if err != nil || len(got) != 1 || got[0] != SourceStorePOS {
		t.Fatalf("got=%v err=%v", got, err)
	}
	_, err = ParseDemandSourcesQuery(url.Values{"source": []string{"AI"}})
	if err == nil {
		t.Fatal("want invalid_source")
	}
	got, err = ParseDemandSourcesQuery(url.Values{"sources": []string{"STORE_POS,WHOLESALE_HISTORY"}})
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestSourcesMatchAny(t *testing.T) {
	t.Parallel()
	if !SourcesMatchAny([]string{SourceStorePOS}, []string{SourceStorePOS}) {
		t.Fatal("expected match")
	}
	if SourcesMatchAny([]string{SourceWholesaleHistory}, []string{SourceStorePOS}) {
		t.Fatal("expected no match")
	}
	if !SourcesMatchAny(nil, nil) {
		t.Fatal("empty filter matches")
	}
	if !SourcesMatchAny(nil, []string{SourceWholesaleHistory}) {
		t.Fatal("empty row defaults wholesale")
	}
}
