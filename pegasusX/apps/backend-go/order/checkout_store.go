package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

// resolveCheckoutStore is the W11 pin: JWT active location → org row → body refine.
// Body lat/lng cannot change country. Empty country is left empty so test stubs
// still resolve; the engine fail-closes on the live Spanner path.
func (s *Service) resolveCheckoutStore(ctx context.Context, retailerID string, bodyLat, bodyLng float64) (proximity.StorePoint, error) {
	retailerID = strings.TrimSpace(retailerID)
	base := proximity.StorePoint{
		RetailerID:  retailerID,
		CountryCode: s.lookupRetailerCountry(ctx, retailerID),
	}
	activeLocationID := ""
	if claims, ok := auth.FromContext(ctx); ok {
		activeLocationID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if s != nil && s.spannerClient != nil {
		loaded, err := (&proximity.CoverageStore{Client: s.spannerClient}).LoadStore(ctx, retailerID, activeLocationID)
		if err == nil {
			base = loaded
		}
	}
	out := proximity.MergeStorePin(base, proximity.StorePoint{}, bodyLat, bodyLng)
	if out.Lat == 0 && out.Lng == 0 {
		return out, errors.New("lat/lng required")
	}
	return out, nil
}

func mapWarehouseResolveError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, auth.ErrGeographyIncomplete) || errors.Is(err, auth.ErrCrossMarketDeferred) {
		return err
	}
	if errors.Is(err, proximity.ErrZoneMiss) {
		return fmt.Errorf("%w: no_eligible_warehouse", ErrZoneMiss)
	}
	return fmt.Errorf("%w: resolve nearest warehouse: %v", ErrServiceabilityUnavailable, err)
}
