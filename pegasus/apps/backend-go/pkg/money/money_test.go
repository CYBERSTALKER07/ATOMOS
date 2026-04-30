package money

import (
	"testing"
)

func TestMinorUnitsMultiplier(t *testing.T) {
	tests := []struct {
		decimals int
		want     int64
	}{
		{0, 1},
		{2, 100},
		{3, 1000},
	}
	for _, tt := range tests {
		got := MinorUnitsMultiplier(tt.decimals)
		if got != tt.want {
			t.Errorf("MinorUnitsMultiplier(%d) = %d, want %d", tt.decimals, got, tt.want)
		}
	}
}

func TestToMinorUnits(t *testing.T) {
	tests := []struct {
		major    float64
		decimals int
		want     int64
	}{
		{50000, 0, 50000},  //: no decimals
		{12.50, 2, 1250},   // USD: 2 decimals
		{1.234, 3, 1234},   // KWD: 3 decimals
		{0, 2, 0},          // zero
		{-25.50, 2, -2550}, // negative
	}
	for _, tt := range tests {
		got := ToMinorUnits(tt.major, tt.decimals)
		if got != tt.want {
			t.Errorf("ToMinorUnits(%f, %d) = %d, want %d", tt.major, tt.decimals, got, tt.want)
		}
	}
}

func TestFromMinorUnits(t *testing.T) {
	tests := []struct {
		minor    int64
		decimals int
		want     float64
	}{
		{50000, 0, 50000},
		{1250, 2, 12.50},
		{1234, 3, 1.234},
	}
	for _, tt := range tests {
		got := FromMinorUnits(tt.minor, tt.decimals)
		if got != tt.want {
			t.Errorf("FromMinorUnits(%d, %d) = %f, want %f", tt.minor, tt.decimals, got, tt.want)
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		amount int64
		cfg    Config
		want   string
	}{
		{50000, Config{Code: "UZS", DecimalPlaces: 0, Symbol: "сўм"}, "50 000 сўм"},
		{1250, Config{Code: "USD", DecimalPlaces: 2, Symbol: "$"}, "$12.50"},
		{100, Config{Code: "EUR", DecimalPlaces: 2, Symbol: "€"}, "€1.00"},
		{0, Config{Code: "UZS", DecimalPlaces: 0, Symbol: "сўм"}, "0 сўм"},
		{1234500, Config{Code: "UZS", DecimalPlaces: 0, Symbol: ""}, "1 234 500 UZS"},
	}
	for _, tt := range tests {
		got := Format(tt.amount, tt.cfg)
		if got != tt.want {
			t.Errorf("Format(%d, %+v) = %q, want %q", tt.amount, tt.cfg, got, tt.want)
		}
	}
}

func TestFormatCode(t *testing.T) {
	got := FormatCode(50000, "UZS", 0)
	if got != "50 000 UZS" {
		t.Errorf("FormatCode(50000,, 0) = %q, want %q", got, "50 000 UZS")
	}
}

func TestConfigFromCountry(t *testing.T) {
	cfg := ConfigFromCountry("UZS", 0)
	if cfg.Code != "UZS" || cfg.Symbol != "сўм" {
		t.Errorf("ConfigFromCountry(UZS, 0) = %+v, want Code=UZS Symbol=сўм", cfg)
	}

	cfg = ConfigFromCountry("XYZ", 2)
	if cfg.Code != "XYZ" || cfg.Symbol != "" {
		t.Errorf("ConfigFromCountry(XYZ, 2) = %+v, want Code=XYZ Symbol=empty", cfg)
	}
}
