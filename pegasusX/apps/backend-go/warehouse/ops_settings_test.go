package warehouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestSupplierRegionOwned_NilClient(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	if svc.supplierRegionOwned(context.Background(), "wh-1", "reg-1") {
		t.Fatal("nil Spanner must not accept a global Regions id")
	}
	if svc.supplierRegionExistsForSupplier(context.Background(), "sup-1", "reg-1") {
		t.Fatal("nil Spanner must reject unknown_supplier_region")
	}
}

func TestRejectUnknownSupplierRegion_NoSpanner(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	region := "global-region-1"
	rr := httptest.NewRecorder()
	if !svc.rejectUnknownSupplierRegion(rr, context.Background(), "sup-1", &region) {
		t.Fatal("expected unknown_supplier_region")
	}
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unknown_supplier_region") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestLockDeliveryFeeCurrency_PackWins(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("UZ pack")
	}
	rules := &DeliveryFeeRules{Currency: "USD", BaseFeeMinor: 0}
	if err := lockDeliveryFeeCurrency(pack, rules); err == nil {
		t.Fatal("USD on UZ must fail closed")
	} else if err != auth.ErrPackCurrencyMismatch {
		t.Fatalf("err=%v", err)
	}
	rules.Currency = ""
	if err := lockDeliveryFeeCurrency(pack, rules); err != nil {
		t.Fatalf("empty currency: %v", err)
	}
	if rules.Currency != "UZS" {
		t.Fatalf("currency=%q", rules.Currency)
	}
}
