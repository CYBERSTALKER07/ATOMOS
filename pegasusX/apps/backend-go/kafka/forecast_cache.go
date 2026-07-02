package kafka

import (
	"context"
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func invalidateForecastAggCache(ctx context.Context, c *cache.Cache, supplierID string) {
	if c == nil || supplierID == "" {
		return
	}
	for _, granularity := range []string{"macro", "regional", "micro"} {
		for _, window := range []string{"7d", "14d"} {
			key := fmt.Sprintf("planning:forecast:agg:%s:%s:%s", supplierID, granularity, window)
			c.Invalidate(ctx, key)
		}
	}
}
