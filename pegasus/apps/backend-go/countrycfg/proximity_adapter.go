package countrycfg

import (
	"context"
)

// ProximityConfigAdapter wraps Service to implement proximity.ConfigProvider.
type ProximityConfigAdapter struct {
	Svc *Service
}

// GetBreachRadius returns the effective breach radius for a supplier.
// Uses supplier override → country config → default 100m.
func (a *ProximityConfigAdapter) GetBreachRadius(ctx context.Context, supplierID, countryCode string) float64 {
	if countryCode == "" {
		countryCode = a.resolveSupplierCountry(ctx, supplierID)
	}
	cfg := a.Svc.GetEffectiveConfig(ctx, supplierID, countryCode)
	if cfg.BreachRadiusMeters > 0 {
		return cfg.BreachRadiusMeters
	}
	return 100.0
}

func (a *ProximityConfigAdapter) resolveSupplierCountry(ctx context.Context, supplierID string) string {
	if a == nil || a.Svc == nil {
		return "UZ"
	}
	return a.Svc.ResolveSupplierCountryCode(ctx, supplierID)
}
