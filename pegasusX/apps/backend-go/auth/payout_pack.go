package auth

import (
	"context"
	"errors"
	"strings"
)

const (
	PayoutRailBankFile = "bank-file"
	PayoutRailSEPAFile = "sepa-file"
	PayoutRailACHFile  = "ach-file"
)

var (
	ErrPayoutRailUnknown = errors.New("payout_rail_unknown")
	ErrPayoutRailNotLive = errors.New("no_live_rail")
)

// CanonicalPayoutRail maps aliases onto pack rail names. Empty stays empty
// (do not invent bank-file).
func CanonicalPayoutRail(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bank-file", "bank_file", "csv":
		return PayoutRailBankFile
	case "sepa-file", "sepa_file", "sepa":
		return PayoutRailSEPAFile
	case "ach-file", "ach_file", "ach":
		return PayoutRailACHFile
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

// IsKnownFilePayoutRail is true for catalog file rails (none move money).
func IsKnownFilePayoutRail(name string) bool {
	switch CanonicalPayoutRail(name) {
	case PayoutRailBankFile, PayoutRailSEPAFile, PayoutRailACHFile:
		return true
	default:
		return false
	}
}

// IsLivePayoutRailImplemented is always false until a real executor is wired.
// GS-M6 does not invent Stripe / SEPA / GP payout.
func IsLivePayoutRailImplemented(name string) bool {
	return false
}

// PackPayoutRail returns the shipped pack file-rail name (GS-M6).
func PackPayoutRail(pack MarketPack) (string, error) {
	if pack.Status != MarketPackShipped {
		return "", ErrMarketPackNotShipped
	}
	rail := CanonicalPayoutRail(pack.PayoutRail)
	if rail == "" || !IsKnownFilePayoutRail(rail) {
		return "", ErrPayoutRailUnknown
	}
	return rail, nil
}

// PayoutRailFromContext resolves payout_rail: claims → supplier profile → env shipped pack.
func PayoutRailFromContext(ctx context.Context, supplierID string) (string, error) {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return "", err
	}
	return PackPayoutRail(pack)
}

// AssertPackPayoutLive fails closed for live dispatch.
// Planned/unknown pack → 404 sentinels. Unknown or file rail → no_live_rail.
func AssertPackPayoutLive(pack MarketPack) error {
	if pack.Status != MarketPackShipped {
		if pack.Code == "" {
			return ErrMarketPackUnknown
		}
		return ErrMarketPackNotShipped
	}
	rail := CanonicalPayoutRail(pack.PayoutRail)
	if rail == "" || !IsKnownFilePayoutRail(rail) {
		return ErrPayoutRailNotLive
	}
	if !IsLivePayoutRailImplemented(rail) {
		return ErrPayoutRailNotLive
	}
	return nil
}

// AssertPayoutLiveFromContext is the live-dispatch gate (GS-M6).
func AssertPayoutLiveFromContext(ctx context.Context, supplierID string) error {
	pack, err := FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return err
	}
	return AssertPackPayoutLive(pack)
}
