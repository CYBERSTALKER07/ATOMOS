package payment

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestAvailablePSPs_UZOmitsForeignRails(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	got := AvailablePSPs(pack)
	codes := map[string]PSPListing{}
	for _, listing := range got {
		codes[listing.Code] = listing
	}
	if _, ok := codes["STRIPE"]; ok {
		t.Fatal("UZ catalog must not list STRIPE")
	}
	if _, ok := codes["ADYEN"]; ok {
		t.Fatal("UZ catalog must not list ADYEN")
	}
	if listing, ok := codes["GLOBAL_PAY"]; !ok || listing.Status != PSPStatusLive || !listing.Selectable {
		t.Fatalf("GLOBAL_PAY=%+v", listing)
	}
	if listing, ok := codes["CASH"]; !ok || listing.Status != PSPStatusLive {
		t.Fatalf("CASH=%+v", listing)
	}
	if listing, ok := codes["PAYME"]; !ok || listing.Status != PSPStatusUnkeyed || !listing.Selectable {
		t.Fatalf("PAYME=%+v", listing)
	}
	if listing, ok := codes["CLICK"]; !ok || listing.Status != PSPStatusUnkeyed || !listing.Selectable {
		t.Fatalf("CLICK=%+v", listing)
	}
}

func TestLivePackGateways_UZIsCashAndGlobalPay(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	got := LivePackGateways(pack)
	if !containsGateway(got, "CASH") || !containsGateway(got, "GLOBAL_PAY") {
		t.Fatalf("live defaults=%#v", got)
	}
	if containsGateway(got, "PAYME") || containsGateway(got, "CLICK") || containsGateway(got, "ADYEN") {
		t.Fatalf("unkeyed/foreign must not be empty-config default: %#v", got)
	}
}

func TestAvailablePSPs_CAPlannedListsStripeAdyenCash(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveMarketPack("CA")
	if !ok || pack.Status != auth.MarketPackPlanned {
		t.Fatalf("CA pack=%+v ok=%v", pack, ok)
	}
	got := AvailablePSPs(pack)
	codes := map[string]PSPListing{}
	for _, listing := range got {
		codes[listing.Code] = listing
	}
	if _, ok := codes["CASH"]; !ok {
		t.Fatal("CA must list CASH")
	}
	if listing, ok := codes["STRIPE"]; !ok || listing.Selectable {
		t.Fatalf("STRIPE=%+v", listing)
	}
	if listing, ok := codes["ADYEN"]; !ok || listing.Selectable {
		t.Fatalf("ADYEN=%+v", listing)
	}
	if _, ok := codes["GLOBAL_PAY"]; ok {
		t.Fatal("CA must not list GLOBAL_PAY")
	}
}

func containsGateway(gateways []string, want string) bool {
	for _, gateway := range gateways {
		if gateway == want {
			return true
		}
	}
	return false
}
