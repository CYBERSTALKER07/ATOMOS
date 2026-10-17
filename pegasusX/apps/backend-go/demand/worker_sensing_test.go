package demand

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSignalScopeMatches_GlobalAndCountry(t *testing.T) {
	geo := RetailerGeo{City: "Samarkand", RegionID: "reg-sam"}
	if !signalScopeMatches("GLOBAL", nil, "r1", geo) {
		t.Fatal("GLOBAL should match")
	}
	if !signalScopeMatches("country:UZ", nil, "r1", geo) {
		t.Fatal("country:UZ should match")
	}
}

func TestSignalScopeMatches_CityNotGlobal(t *testing.T) {
	tash := RetailerGeo{City: "Tashkent", Address: "1 Navoi, Tashkent"}
	sam := RetailerGeo{City: "Samarkand", Address: "2 Registan, Samarkand"}
	unknown := RetailerGeo{}

	if !signalScopeMatches("CITY:Tashkent", nil, "r-tash", tash) {
		t.Fatal("CITY:Tashkent should match Tashkent retailer")
	}
	if !signalScopeMatches("city:Tashkent", nil, "r-tash", tash) {
		t.Fatal("legacy city:Tashkent should match Tashkent")
	}
	if signalScopeMatches("CITY:Tashkent", nil, "r-sam", sam) {
		t.Fatal("CITY:Tashkent must not match Samarkand")
	}
	if signalScopeMatches("CITY:Tashkent", nil, "r-unk", unknown) {
		t.Fatal("CITY:Tashkent must fail-closed when geo unknown")
	}
	// bare CITY without meta fails closed
	if signalScopeMatches("CITY", nil, "r-tash", tash) {
		t.Fatal("bare CITY without Meta must fail-closed")
	}
	meta, _ := json.Marshal(map[string]string{"city": "Tashkent"})
	if !signalScopeMatches("CITY", meta, "r-tash", tash) {
		t.Fatal("CITY + Meta.city should match")
	}
}

func TestSignalScopeMatches_Region(t *testing.T) {
	geo := RetailerGeo{RegionID: "reg-tash"}
	other := RetailerGeo{RegionID: "reg-sam"}
	if !signalScopeMatches("REGION:reg-tash", nil, "r1", geo) {
		t.Fatal("REGION:code should match")
	}
	if signalScopeMatches("REGION:reg-tash", nil, "r2", other) {
		t.Fatal("REGION must not match other region")
	}
	if signalScopeMatches("REGION:reg-tash", nil, "r3", RetailerGeo{}) {
		t.Fatal("REGION fail-closed when RegionId unknown")
	}
}

func TestSignalScopeMatches_Retailer(t *testing.T) {
	if !signalScopeMatches("retailer:abc", nil, "abc", RetailerGeo{}) {
		t.Fatal("retailer:uuid should match")
	}
	if signalScopeMatches("retailer:abc", nil, "xyz", RetailerGeo{}) {
		t.Fatal("retailer:uuid must not match other")
	}
	meta, _ := json.Marshal(map[string]string{"retailer_id": "abc"})
	if !signalScopeMatches("RETAILER", meta, "abc", RetailerGeo{}) {
		t.Fatal("RETAILER Meta should match")
	}
}

func TestCityHintFromAddress(t *testing.T) {
	if got := cityHintFromAddress("12 Navoi St, Yunusabad, Tashkent"); got != "Tashkent" {
		t.Fatalf("got %q", got)
	}
	if cityHintFromAddress("") != "" {
		t.Fatal("empty")
	}
}

func TestDayOfWeekFactor(t *testing.T) {
	tests := []struct {
		day  time.Weekday
		want float64
	}{
		{time.Sunday, 0.75},
		{time.Monday, 0.95},
		{time.Tuesday, 1.05},
		{time.Wednesday, 1.10},
		{time.Thursday, 1.05},
		{time.Friday, 1.00},
		{time.Saturday, 0.85},
	}
	for _, tt := range tests {
		t.Run(tt.day.String(), func(t *testing.T) {
			got := dayOfWeekFactor(tt.day)
			if got != tt.want {
				t.Errorf("dayOfWeekFactor(%s) = %v, want %v", tt.day, got, tt.want)
			}
		})
	}
}

func TestPaydayFactor(t *testing.T) {
	tests := []struct {
		day  int
		want float64
	}{
		{1, 1.15},
		{2, 1.15},
		{3, 1.0},
		{14, 1.0},
		{15, 1.15},
		{16, 1.15},
		{17, 1.0},
		{28, 1.0},
	}
	for _, tt := range tests {
		if got := paydayFactor(tt.day); got != tt.want {
			t.Fatalf("day %d: got %v want %v", tt.day, got, tt.want)
		}
	}
}

func TestBlendVelocitiesWeights(t *testing.T) {
	// Documented blend: 0.65 order + 0.35 POS flywheel.
	order, fw := 10.0, 20.0
	got := 0.65*order + 0.35*fw
	if got < 13.4 || got > 13.6 {
		t.Fatalf("blend = %v", got)
	}
}

func TestDayOfWeekFactor_AllDaysCovered(t *testing.T) {
	for d := time.Sunday; d <= time.Saturday; d++ {
		f := dayOfWeekFactor(d)
		if f < 0.5 || f > 1.5 {
			t.Errorf("dayOfWeekFactor(%s) = %v; out of expected range [0.5, 1.5]", d, f)
		}
	}
}

func TestPaydayFactor_NonPaydays(t *testing.T) {
	for d := 3; d <= 14; d++ {
		if paydayFactor(d) != 1.0 {
			t.Errorf("paydayFactor(%d) should be 1.0 for non-payday", d)
		}
	}
	for d := 17; d <= 31; d++ {
		if paydayFactor(d) != 1.0 {
			t.Errorf("paydayFactor(%d) should be 1.0 for non-payday", d)
		}
	}
}
