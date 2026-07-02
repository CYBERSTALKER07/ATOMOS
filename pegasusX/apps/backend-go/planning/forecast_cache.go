package planning

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// InvalidateForecastAggCache drops aggregated forecast read keys for a supplier.
func InvalidateForecastAggCache(ctx context.Context, c *cache.Cache, supplierID string) {
	if c == nil || supplierID == "" {
		return
	}
	for _, granularity := range []string{"macro", "regional", "micro"} {
		for _, window := range []string{"7d", "14d"} {
			c.Invalidate(ctx, ForecastAggCacheKey(supplierID, granularity, window))
		}
	}
}
