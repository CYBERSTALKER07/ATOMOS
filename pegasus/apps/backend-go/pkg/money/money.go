package money

import (
	"fmt"
	"math"
	"strings"
)

// Config describes the currency properties for formatting and conversion.
type Config struct {
	Code          string // ISO 4217 currency code, e.g. "UZS", "USD"
	DecimalPlaces int    // 0 for UZS, 2 for USD/EUR, 3 for KWD
	Symbol        string // Optional display symbol, e.g. "сўм", "$"
}

// CommonConfigs holds well-known currency configurations.
var CommonConfigs = map[string]Config{
	"UZS": {Code: "UZS", DecimalPlaces: 0, Symbol: "сўм"},
	"USD": {Code: "USD", DecimalPlaces: 2, Symbol: "$"},
	"EUR": {Code: "EUR", DecimalPlaces: 2, Symbol: "€"},
	"GBP": {Code: "GBP", DecimalPlaces: 2, Symbol: "£"},
	"RUB": {Code: "RUB", DecimalPlaces: 2, Symbol: "₽"},
	"KZT": {Code: "KZT", DecimalPlaces: 2, Symbol: "₸"},
	"TRY": {Code: "TRY", DecimalPlaces: 2, Symbol: "₺"},
	"KWD": {Code: "KWD", DecimalPlaces: 3, Symbol: "د.ك"},
}

// MinorUnitsMultiplier returns 10^decimalPlaces. For UZS (0 decimals) → 1.
// For USD (2 decimals) → 100. For KWD (3 decimals) → 1000.
func MinorUnitsMultiplier(decimalPlaces int) int64 {
	return int64(math.Pow10(decimalPlaces))
}

// ToMinorUnits converts a major-unit float to minor-unit int64.
// e.g. ToMinorUnits(12.50, 2) → 1250  (USD cents)
// e.g. ToMinorUnits(50000, 0)  → 50000 (UZS, no decimals)
func ToMinorUnits(majorAmount float64, decimalPlaces int) int64 {
	return int64(math.Round(majorAmount * float64(MinorUnitsMultiplier(decimalPlaces))))
}

// FromMinorUnits converts minor-unit int64 back to major-unit float.
// e.g. FromMinorUnits(1250, 2) → 12.50
func FromMinorUnits(minorAmount int64, decimalPlaces int) float64 {
	return float64(minorAmount) / float64(MinorUnitsMultiplier(decimalPlaces))
}

// Format renders an amount (in minor units) as a human-readable string.
// e.g. Format(1250, Config{Code:"USD", DecimalPlaces:2, Symbol:"$"}) → "$12.50"
// e.g. Format(50000, Config{Code:"UZS", DecimalPlaces:0, Symbol:"сўм"}) → "50 000 сўм"
func Format(minorAmount int64, cfg Config) string {
	major := FromMinorUnits(minorAmount, cfg.DecimalPlaces)

	var formatted string
	if cfg.DecimalPlaces == 0 {
		formatted = formatWithThousandsSep(int64(major))
	} else {
		intPart := int64(major)
		fracPart := int64(math.Round((major - float64(intPart)) * float64(MinorUnitsMultiplier(cfg.DecimalPlaces))))
		if fracPart < 0 {
			fracPart = -fracPart
		}
		formatted = fmt.Sprintf("%s.%0*d", formatWithThousandsSep(intPart), cfg.DecimalPlaces, fracPart)
	}

	if cfg.Symbol != "" {
		// Prefix symbol for $, €, £ — suffix for others
		switch cfg.Symbol {
		case "$", "€", "£":
			return cfg.Symbol + formatted
		default:
			return formatted + " " + cfg.Symbol
		}
	}
	return formatted + " " + cfg.Code
}

// FormatCode renders an amount with just the currency code appended.
// e.g. FormatCode(50000, "UZS", 0) → "50 000 UZS"
func FormatCode(minorAmount int64, code string, decimalPlaces int) string {
	cfg := Config{Code: code, DecimalPlaces: decimalPlaces}
	if known, ok := CommonConfigs[code]; ok {
		cfg = known
		cfg.Symbol = "" // Force code-only display
	}
	major := FromMinorUnits(minorAmount, cfg.DecimalPlaces)

	var formatted string
	if decimalPlaces == 0 {
		formatted = formatWithThousandsSep(int64(major))
	} else {
		intPart := int64(major)
		fracPart := int64(math.Round((major - float64(intPart)) * float64(MinorUnitsMultiplier(decimalPlaces))))
		if fracPart < 0 {
			fracPart = -fracPart
		}
		formatted = fmt.Sprintf("%s.%0*d", formatWithThousandsSep(intPart), decimalPlaces, fracPart)
	}
	return formatted + " " + code
}

// ConfigFromCountry creates a Config from country config parameters.
func ConfigFromCountry(currencyCode string, decimalPlaces int) Config {
	if known, ok := CommonConfigs[currencyCode]; ok {
		known.DecimalPlaces = decimalPlaces
		return known
	}
	return Config{Code: currencyCode, DecimalPlaces: decimalPlaces}
}

// formatWithThousandsSep formats an integer with space thousand-separators.
func formatWithThousandsSep(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)

	result := strings.Join(parts, " ")
	if negative {
		return "-" + result
	}
	return result
}
