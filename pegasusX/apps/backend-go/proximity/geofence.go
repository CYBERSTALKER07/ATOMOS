package proximity

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// WithinDeliveryApproach reports whether distKm is inside the shipped pack
// breach_radius_meters (GS-M3). Uses the env shipped pack. Fail-closed if the
// pack is planned or radius is unset — never 500 m.
func WithinDeliveryApproach(distKm float64) bool {
	return WithinDeliveryApproachForSupplier(context.Background(), "", distKm)
}

// WithinDeliveryApproachForSupplier uses claims → supplier profile → env pack.
func WithinDeliveryApproachForSupplier(ctx context.Context, supplierID string, distKm float64) bool {
	radius, err := auth.BreachRadiusFromContext(ctx, supplierID)
	if err != nil || radius <= 0 {
		return false
	}
	return distKm*1000 < radius
}
