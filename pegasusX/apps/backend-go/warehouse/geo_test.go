package warehouse

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/uber/h3-go/v4"
)

func TestStampWarehouseEntity_DefaultsPackCountryAndRes7(t *testing.T) {
	t.Parallel()
	lat, lng := 41.3111, 69.2797
	w := Warehouse{Lat: &lat, Lng: &lng}
	if err := stampWarehouseEntity(context.Background(), &w); err != nil {
		t.Fatal(err)
	}
	if w.CountryCode != "UZ" {
		t.Fatalf("country=%q", w.CountryCode)
	}
	if w.H3Cell == nil || *w.H3Cell == "" {
		t.Fatal("h3 empty")
	}
	cell := h3.Cell(h3.IndexFromString(*w.H3Cell))
	if !cell.IsValid() || cell.Resolution() != 7 {
		t.Fatalf("h3=%q res=%d", *w.H3Cell, cell.Resolution())
	}
}

func TestStampWarehouseEntity_RejectsForeignCountry(t *testing.T) {
	t.Parallel()
	lat, lng := 41.3111, 69.2797
	w := Warehouse{Lat: &lat, Lng: &lng, CountryCode: "US"}
	if err := stampWarehouseEntity(context.Background(), &w); !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleSupplyRequestAccepted_RequiresWarehouseID(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	err := svc.HandleSupplyRequestAccepted(context.Background(), []byte(`{"request_id":"sr-1"}`))
	if !errors.Is(err, errWarehouseIDRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateDeliveryFeeRules_UsesPackCurrency(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	rules := DeliveryFeeRules{}
	if err := validateDeliveryFeeRules(&rules); err != nil {
		t.Fatal(err)
	}
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	want, err := auth.PackCurrency(pack)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Currency != want {
		t.Fatalf("currency=%q want %q", rules.Currency, want)
	}
}

func TestNewService_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	pack, _ := auth.ResolveShippedMarketPack("UZ")
	want, _ := auth.PackCurrency(pack)
	if svc.currency != want {
		t.Fatalf("currency=%q want %q", svc.currency, want)
	}
}
