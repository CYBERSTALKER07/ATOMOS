package proximity

import (
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// MatchingResolution is H3 res 7 — checkout, coverage cells, and node persist.
const MatchingResolution = 7

// NodeGeography is the stamped country + matching cell for a warehouse/factory/store.
type NodeGeography struct {
	CountryCode string
	H3Cell      string
}

// MatchingH3Cell returns the res-7 cell for coordinates, or empty if H3 fails.
func MatchingH3Cell(lat, lng float64) string {
	cell, err := PanicSafeLatLngToCell(lat, lng, MatchingResolution)
	if err != nil || !cell.IsValid() {
		return ""
	}
	return cell.String()
}

// ResolveNodeCountry defaults empty requested country to the shipped pack.
// A non-empty country that disagrees with the pack is cross_market_deferred.
func ResolveNodeCountry(pack auth.MarketPack, requestedCountry string) (string, error) {
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return "", err
	}
	req := auth.NormalizeCountryCode(requestedCountry)
	if req == "" {
		return packCountry, nil
	}
	if req != packCountry {
		return "", auth.ErrCrossMarketDeferred
	}
	return req, nil
}

// StampNodeGeography inherits pack country when omitted, rejects mismatch,
// and always derives H3 res 7. Used by every warehouse/factory/store writer.
func StampNodeGeography(pack auth.MarketPack, lat, lng float64, requestedCountry string) (NodeGeography, error) {
	country, err := ResolveNodeCountry(pack, requestedCountry)
	if err != nil {
		return NodeGeography{}, err
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return NodeGeography{}, auth.ErrGeographyIncomplete
	}
	cell := MatchingH3Cell(lat, lng)
	if cell == "" {
		return NodeGeography{}, auth.ErrGeographyIncomplete
	}
	return NodeGeography{CountryCode: country, H3Cell: cell}, nil
}
