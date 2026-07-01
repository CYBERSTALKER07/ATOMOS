package planning

import "fmt"

const (
	scenarioCacheTTL   = 15 * 60 // seconds
	seasonalCacheTTL   = 24 * 60 * 60
	forecastAggCacheTTL = 15 * 60
)

// ScenarioCacheKey returns the Redis key for a scenario sandbox result.
func ScenarioCacheKey(supplierID, cacheKey string) string {
	return fmt.Sprintf("planning:scenario:%s:%s", supplierID, cacheKey)
}

// SeasonalCacheKey returns the Redis key for an active seasonal template.
func SeasonalCacheKey(supplierID, templateID string) string {
	return fmt.Sprintf("planning:seasonal:%s:%s", supplierID, templateID)
}

// ForecastAggCacheKey returns the Redis key for aggregated forecast reads.
func ForecastAggCacheKey(supplierID, granularity, window string) string {
	return fmt.Sprintf("planning:forecast:agg:%s:%s:%s", supplierID, granularity, window)
}

// ForecastAggInvalidationPrefix returns keys to invalidate after baseline write.
func ForecastAggInvalidationPrefixes(supplierID string) []string {
	return []string{
		fmt.Sprintf("planning:forecast:agg:%s:", supplierID),
	}
}
