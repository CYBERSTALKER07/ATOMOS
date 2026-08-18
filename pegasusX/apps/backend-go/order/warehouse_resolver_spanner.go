package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

// SpannerWarehouseResolver resolves the nearest covering warehouse for a supplier.
type SpannerWarehouseResolver struct {
	client *spanner.Client
}

// NewSpannerWarehouseResolver builds a Spanner-backed warehouse resolver.
func NewSpannerWarehouseResolver(client *spanner.Client) *SpannerWarehouseResolver {
	return &SpannerWarehouseResolver{client: client}
}

// ResolveNearestWarehouseID returns the closest warehouse that covers the retailer.
// Matching is proximity.ResolveServingWarehouse (GS-L2/L3), including service pins.
func (r *SpannerWarehouseResolver) ResolveNearestWarehouseID(
	ctx context.Context,
	supplierID string,
	store proximity.StorePoint,
) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("spanner warehouse resolver: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return "", fmt.Errorf("supplier_id required")
	}
	if store.Lat == 0 && store.Lng == 0 {
		return "", nil
	}

	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return "", err
	}
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return "", err
	}
	if err := auth.AssertSameMarket(packCountry, store.CountryCode); err != nil {
		return "", err
	}

	cov := proximity.CoverageStore{Client: r.client}
	warehouses, err := cov.ListWarehouses(ctx, supplierID)
	if err != nil {
		return "", err
	}
	pins, err := cov.ListPins(ctx, supplierID)
	if err != nil {
		return "", err
	}
	if store.H3Cell == "" {
		store.H3Cell = proximity.MatchingH3Cell(store.Lat, store.Lng)
	}
	return proximity.ResolveServingWarehouse(packCountry, store, warehouses, pins)
}

func (s *Service) lookupRetailerCountry(ctx context.Context, retailerID string) string {
	if s == nil || s.spannerClient == nil {
		return ""
	}
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return ""
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"CountryCode"})
	if err != nil {
		return ""
	}
	var code spanner.NullString
	if err := row.Column(0, &code); err != nil || !code.Valid {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(code.StringVal))
}
