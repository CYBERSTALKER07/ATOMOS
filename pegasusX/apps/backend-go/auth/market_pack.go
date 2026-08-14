package auth

import (
	"os"
	"strings"
)

// DefaultMarketCode is the only shipped pack until GS-M/GS-C add more.
const DefaultMarketCode = "UZ"

// DefaultHomeCell is the first production cell id (logical, not a GCP project).
const DefaultHomeCell = "cell-uz"

// MarketPackStatus is honesty for catalog rows.
const (
	MarketPackShipped = "shipped" // adapters exist in-tree (may still be unkeyed)
	MarketPackPlanned = "planned" // declared; resolve for checkout must fail-closed
)

// MarketPack is the versioned market contract. Checkout does not apply it until
// CheckoutReadsThis is true (GS-M). Session/JWT may still advertise the pack.
type MarketPack struct {
	Code                  string   `json:"code"`
	Name                  string   `json:"name"`
	Status                string   `json:"status"`
	HomeCell              string   `json:"home_cell"`
	Timezone              string   `json:"timezone"`
	CurrencyCode          string   `json:"currency_code"`
	CurrencyDecimalPlaces int64    `json:"currency_decimal_places"`
	Locales               []string `json:"locales"`
	FiscalAdapter         string   `json:"fiscal_adapter"`
	PSPAdapters           []string `json:"psp_adapters"`
	SMSAdapter            string   `json:"sms_adapter,omitempty"`
	MapsAdapter           string   `json:"maps_adapter"`
	GridSystem            string   `json:"grid_system"`
	DistanceUnit          string   `json:"distance_unit"`
	BreachRadiusMeters    float64  `json:"breach_radius_meters"`
	ShopClosedGraceMin    int64    `json:"shop_closed_grace_minutes"`
	CashCustodyAlertHours int64    `json:"cash_custody_alert_hours"`
	CheckoutReadsThis     bool     `json:"checkout_reads_this"`
	OpsReadsThis          bool     `json:"ops_reads_this"`
}

// DefaultMarketCodeFromEnv is DEFAULT_MARKET_CODE or UZ.
func DefaultMarketCodeFromEnv() string {
	c := NormalizeMarketCode(os.Getenv("DEFAULT_MARKET_CODE"))
	if c == "" {
		return DefaultMarketCode
	}
	return c
}

// DefaultHomeCellFromEnv is HOME_CELL or cell-uz.
func DefaultHomeCellFromEnv() string {
	c := strings.ToLower(strings.TrimSpace(os.Getenv("HOME_CELL")))
	if c == "" {
		return DefaultHomeCell
	}
	return c
}

// NormalizeMarketCode uppercases ISO-like codes (UZ, EU, US, KZ).
func NormalizeMarketCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// ListMarketPacks returns the catalog (shipped + planned). No secrets.
func ListMarketPacks() []MarketPack {
	return []MarketPack{
		uzMarketPack(),
		plannedPack("EU", "European Union (PEPPOL / multi-VAT)", "cell-eu", "Europe/Berlin", "EUR", []string{"en", "de", "fr"}, "PEPPOL", []string{"STRIPE", "CASH"}),
		plannedPack("US", "United States", "cell-us", "America/New_York", "USD", []string{"en"}, "COMMERCIAL", []string{"STRIPE", "CASH"}),
		plannedPack("KZ", "Kazakhstan", "cell-kz", "Asia/Almaty", "KZT", []string{"ru", "kk", "en"}, "PLANNED", []string{"CASH"}),
	}
}

// ResolveMarketPack returns a shipped or planned pack. Unknown → false.
func ResolveMarketPack(code string) (MarketPack, bool) {
	code = NormalizeMarketCode(code)
	if code == "" {
		code = DefaultMarketCodeFromEnv()
	}
	for _, p := range ListMarketPacks() {
		if p.Code == code {
			return p, true
		}
	}
	return MarketPack{}, false
}

// ResolveShippedMarketPack fails closed for planned/unknown codes.
func ResolveShippedMarketPack(code string) (MarketPack, bool) {
	p, ok := ResolveMarketPack(code)
	if !ok || p.Status != MarketPackShipped {
		return MarketPack{}, false
	}
	return p, true
}

func uzMarketPack() MarketPack {
	return MarketPack{
		Code:                  DefaultMarketCode,
		Name:                  "Uzbekistan",
		Status:                MarketPackShipped,
		HomeCell:              DefaultHomeCell,
		Timezone:              "Asia/Tashkent",
		CurrencyCode:          "UZS",
		CurrencyDecimalPlaces: 2,
		Locales:               []string{"uz", "ru", "en"},
		FiscalAdapter:         "MY_SOLIQ",
		PSPAdapters:           []string{"GLOBAL_PAY", "CASH"},
		SMSAdapter:            "PLAYMOBILE",
		MapsAdapter:           "GOOGLE_ROUTES",
		GridSystem:            "H3",
		DistanceUnit:          "km",
		BreachRadiusMeters:    150,
		ShopClosedGraceMin:    10,
		CashCustodyAlertHours: 24,
		// GS-M flips this when checkout/fiscal/proximity read the pack.
		CheckoutReadsThis: false,
		OpsReadsThis:      true,
	}
}

func plannedPack(code, name, cell, tz, ccy string, locales []string, fiscal string, psp []string) MarketPack {
	return MarketPack{
		Code:                  code,
		Name:                  name,
		Status:                MarketPackPlanned,
		HomeCell:              cell,
		Timezone:              tz,
		CurrencyCode:          ccy,
		CurrencyDecimalPlaces: 2,
		Locales:               locales,
		FiscalAdapter:         fiscal,
		PSPAdapters:           psp,
		MapsAdapter:           "GOOGLE_ROUTES",
		GridSystem:            "H3",
		DistanceUnit:          "km",
		CheckoutReadsThis:     false,
		OpsReadsThis:          false,
	}
}

// EffectiveMarketCode prefers claim, then env default.
func EffectiveMarketCode(c Claims) string {
	if code := NormalizeMarketCode(c.MarketCode); code != "" {
		return code
	}
	return DefaultMarketCodeFromEnv()
}

// EffectiveHomeCell prefers claim, then pack home, then env.
func EffectiveHomeCell(c Claims) string {
	if cell := strings.ToLower(strings.TrimSpace(c.HomeCell)); cell != "" {
		return cell
	}
	if p, ok := ResolveMarketPack(EffectiveMarketCode(c)); ok && p.HomeCell != "" {
		return p.HomeCell
	}
	return DefaultHomeCellFromEnv()
}
