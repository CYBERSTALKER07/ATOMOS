package auth

import (
	"context"
	"testing"
)

func TestPackPayoutRail_ShippedUZ(t *testing.T) {
	p, ok := ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	rail, err := PackPayoutRail(p)
	if err != nil || rail != PayoutRailBankFile {
		t.Fatalf("rail=%s err=%v", rail, err)
	}
	if p.CheckoutReadsThis {
		t.Fatal("flag stays false")
	}
}

func TestPackPayoutRail_PlannedFailsClosed(t *testing.T) {
	p, ok := ResolveMarketPack("EU")
	if !ok || p.PayoutRail != PayoutRailSEPAFile {
		t.Fatalf("EU catalog rail: %+v", p)
	}
	if _, err := PackPayoutRail(p); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestPackPayoutRail_EmptyUnknown(t *testing.T) {
	if _, err := PackPayoutRail(MarketPack{Status: MarketPackShipped}); err != ErrPayoutRailUnknown {
		t.Fatalf("err=%v", err)
	}
	if _, err := PackPayoutRail(MarketPack{Status: MarketPackShipped, PayoutRail: "stripe-payout"}); err != ErrPayoutRailUnknown {
		t.Fatalf("unknown live name must not invent: %v", err)
	}
}

func TestPayoutRailFromContext_EnvUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	rail, err := PayoutRailFromContext(context.Background(), "")
	if err != nil || rail != PayoutRailBankFile {
		t.Fatalf("rail=%s err=%v", rail, err)
	}
}

func TestAssertPayoutLiveFromContext_UZNoLiveRail(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if err := AssertPayoutLiveFromContext(context.Background(), ""); err != ErrPayoutRailNotLive {
		t.Fatalf("err=%v", err)
	}
}

func TestAssertPayoutLiveFromContext_Planned404(t *testing.T) {
	ctx := WithClaims(context.Background(), Claims{MarketCode: "EU"})
	if err := AssertPayoutLiveFromContext(ctx, "sup-1"); err != ErrMarketPackNotShipped {
		t.Fatalf("err=%v", err)
	}
}

func TestIsLivePayoutRailImplemented_Never(t *testing.T) {
	for _, name := range []string{"bank-file", "sepa-file", "ach-file", "stripe", "globalpay-payout"} {
		if IsLivePayoutRailImplemented(name) {
			t.Fatalf("invented live rail %q", name)
		}
	}
}

func TestCanonicalPayoutRail(t *testing.T) {
	if CanonicalPayoutRail("") != "" {
		t.Fatal("empty must stay empty")
	}
	if CanonicalPayoutRail("CSV") != PayoutRailBankFile {
		t.Fatal("csv alias")
	}
	if CanonicalPayoutRail("SEPA") != PayoutRailSEPAFile {
		t.Fatal("sepa alias")
	}
}
