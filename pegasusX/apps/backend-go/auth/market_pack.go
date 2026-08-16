package auth

import (
	"os"
	"strings"
	"sync"
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

// MarketPack is the versioned market contract. GS-M1 checkout reads currency +
// PSP. GS-M2 fiscal fail-closes on the shipped pack adapter. Catalog
// CheckoutReadsThis stays false while SSMR cell default is PEGASUS/FAKE and
// the UZ pack tax adapter is MY_SOLIQ.
type MarketPack struct {
	Code                   string   `json:"code"`
	Name                   string   `json:"name"`
	Status                 string   `json:"status"`
	HomeCell               string   `json:"home_cell"`
	Timezone               string   `json:"timezone"`
	CurrencyCode           string   `json:"currency_code"`
	CurrencyDecimalPlaces  int64    `json:"currency_decimal_places"`
	Locales                []string `json:"locales"`
	FiscalAdapter          string   `json:"fiscal_adapter"`
	PSPAdapters            []string `json:"psp_adapters"`
	SMSAdapter             string   `json:"sms_adapter,omitempty"`
	MapsAdapter            string   `json:"maps_adapter"`
	GridSystem             string   `json:"grid_system"`
	DistanceUnit           string   `json:"distance_unit"`
	BreachRadiusMeters     float64  `json:"breach_radius_meters"`
	ShopClosedGraceMin     int64    `json:"shop_closed_grace_minutes"`
	CashCustodyAlertHours  int64    `json:"cash_custody_alert_hours"`
	WeatherScope           string   `json:"weather_scope,omitempty"`
	FactorySLADefaultHours int64    `json:"factory_sla_default_hours,omitempty"`
	LaborMaxShiftHours     int64    `json:"labor_max_shift_hours,omitempty"`
	PayoutRail             string   `json:"payout_rail,omitempty"`
	CheckoutReadsThis      bool     `json:"checkout_reads_this"`
	OpsReadsThis           bool     `json:"ops_reads_this"`
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
		plannedPack("EU", "European Union (PEPPOL / multi-VAT)", "cell-eu", "Europe/Berlin", "EUR", []string{"en", "de", "fr"}, "PEPPOL", []string{"STRIPE", "CASH"}, "sepa-file"),
		plannedPack("US", "United States", "cell-us", "America/New_York", "USD", []string{"en"}, "COMMERCIAL", []string{"STRIPE", "CASH"}, "ach-file"),
		plannedPack("KZ", "Kazakhstan", "cell-kz", "Asia/Almaty", "KZT", []string{"ru", "kk", "en"}, "PLANNED", []string{"CASH"}, "bank-file"),
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
		Code:                   DefaultMarketCode,
		Name:                   "Uzbekistan",
		Status:                 MarketPackShipped,
		HomeCell:               DefaultHomeCell,
		Timezone:               "Asia/Tashkent",
		CurrencyCode:           "UZS",
		CurrencyDecimalPlaces:  2,
		Locales:                []string{"uz", "ru", "en"},
		FiscalAdapter:          "MY_SOLIQ",
		PSPAdapters:            []string{"GLOBAL_PAY", "CASH"},
		SMSAdapter:             "PLAYMOBILE",
		MapsAdapter:            "GOOGLE_ROUTES",
		GridSystem:             "H3",
		DistanceUnit:           "km",
		BreachRadiusMeters:     150,
		ShopClosedGraceMin:     10,
		CashCustodyAlertHours:  24,
		WeatherScope:           "city:Tashkent",
		FactorySLADefaultHours: 48,
		LaborMaxShiftHours:     12,
		PayoutRail:             "bank-file",
		// Stays false: M1/M2 read the pack, but SSMR cell default is still PEGASUS/FAKE.
		CheckoutReadsThis: false,
		OpsReadsThis:      true,
	}
}

func plannedPack(code, name, cell, tz, ccy string, locales []string, fiscal string, psp []string, payoutRail string) MarketPack {
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
		PayoutRail:            payoutRail,
		MapsAdapter:           "GOOGLE_ROUTES",
		GridSystem:            "H3",
		DistanceUnit:          "km",
		CheckoutReadsThis:     false,
		OpsReadsThis:          false,
	}
}

// MarketProfile is a durable pack assignment. GS-A2 persists this on Suppliers;
// A1 only reads it when a lookup is registered.
type MarketProfile struct {
	MarketCode string
	HomeCell   string
}

type marketProfileLookupFunc func(supplierID string) (MarketProfile, bool)

var (
	marketProfileMu     sync.RWMutex
	marketProfileLookup marketProfileLookupFunc
)

// Session source honesty (GS-A2). Empty profile ≠ silent “user chose UZ”.
const (
	MarketSourceClaim   = "claim"
	MarketSourceProfile = "profile"
	MarketSourceEnv     = "env"
)

// MarketAssignment is the resolved pack plus which layer supplied it.
type MarketAssignment struct {
	MarketCode string
	HomeCell   string
	Source     string
}

// SetMarketProfileLookup registers an optional profile reader (claim → profile → env).
// Pass nil to clear. GS-A2 wires Suppliers.MarketCode / HomeCell here.
func SetMarketProfileLookup(fn marketProfileLookupFunc) {
	marketProfileMu.Lock()
	defer marketProfileMu.Unlock()
	marketProfileLookup = fn
}

func lookupMarketProfile(supplierID string) (MarketProfile, bool) {
	marketProfileMu.RLock()
	fn := marketProfileLookup
	marketProfileMu.RUnlock()
	if fn == nil {
		return MarketProfile{}, false
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return MarketProfile{}, false
	}
	p, ok := fn(sid)
	if !ok {
		return MarketProfile{}, false
	}
	p.MarketCode = NormalizeMarketCode(p.MarketCode)
	p.HomeCell = strings.ToLower(strings.TrimSpace(p.HomeCell))
	if p.MarketCode == "" {
		return MarketProfile{}, false
	}
	return p, true
}

func resolveHomeCell(code, cell string) string {
	if c := strings.ToLower(strings.TrimSpace(cell)); c != "" {
		return c
	}
	if p, ok := ResolveMarketPack(code); ok && strings.TrimSpace(p.HomeCell) != "" {
		return strings.ToLower(strings.TrimSpace(p.HomeCell))
	}
	return DefaultHomeCellFromEnv()
}

// ResolveMarketAssignment reports pack + source: claim | profile | env.
// An env-stamped JWT (A1) must not look like a chosen pack when the row is empty.
func ResolveMarketAssignment(c Claims) MarketAssignment {
	claimCode := NormalizeMarketCode(c.MarketCode)
	claimCell := strings.ToLower(strings.TrimSpace(c.HomeCell))
	envCode := DefaultMarketCodeFromEnv()
	profile, hasProfile := lookupMarketProfile(c.SupplierID)

	if hasProfile {
		if claimCode != "" && claimCode != profile.MarketCode && claimCode != envCode {
			return MarketAssignment{
				MarketCode: claimCode,
				HomeCell:   resolveHomeCell(claimCode, claimCell),
				Source:     MarketSourceClaim,
			}
		}
		return MarketAssignment{
			MarketCode: profile.MarketCode,
			HomeCell:   resolveHomeCell(profile.MarketCode, profile.HomeCell),
			Source:     MarketSourceProfile,
		}
	}
	if claimCode != "" && claimCode != envCode {
		return MarketAssignment{
			MarketCode: claimCode,
			HomeCell:   resolveHomeCell(claimCode, claimCell),
			Source:     MarketSourceClaim,
		}
	}
	return MarketAssignment{
		MarketCode: envCode,
		HomeCell:   resolveHomeCell(envCode, ""),
		Source:     MarketSourceEnv,
	}
}

// StampMarketClaims fills MarketCode and HomeCell from ResolveMarketAssignment.
// Issue calls this so every login / refresh / WS ticket carries the pack.
func StampMarketClaims(c Claims) Claims {
	a := ResolveMarketAssignment(c)
	c.MarketCode = a.MarketCode
	c.HomeCell = a.HomeCell
	return c
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
