package order

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestCardGatewaysOnly_PackFilterNoInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	got := cardGatewaysOnly(context.Background(), []string{"CASH", "GLOBAL_PAY", "STRIPE", "ADYEN"})
	if len(got) != 1 || got[0] != "GLOBAL_PAY" {
		t.Fatalf("got=%#v", got)
	}
	empty := cardGatewaysOnly(context.Background(), []string{"CASH", "STRIPE"})
	if len(empty) != 0 {
		t.Fatalf("must not invent GLOBAL_PAY when pack-filtered list is empty: %#v", empty)
	}
}

func TestCardGatewaysOnly_UnkeyedUZRailsStay(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "UZ"})
	got := cardGatewaysOnly(ctx, []string{"PAYME", "CLICK", "ADYEN"})
	if len(got) != 2 || got[0] != "PAYME" || got[1] != "CLICK" {
		t.Fatalf("got=%#v", got)
	}
}
