package planning

import "strings"

// Production v1 forecast sources (math-only; no ML inference in hot path).
const (
	BaselineSourceMovingAverage    = "moving_average"
	BaselineSourceSeasonalTemplate = "seasonal_template"
	BaselineSourceMixed            = "mixed"
	BaselineSourceInventoryHint    = "inventory_hint" // AI recommendations / reorder hints — not ML inference
	BaselineSourceCroston          = "croston"
	BaselineSourceHoltWinters      = "holt_winters"
	BaselineSourceSES              = "ses"
)

// NormalizeBaselineSource maps legacy/internal labels to the production v1 math contract.
// Never returns "ml" — training inference is deferred (PX-PROD-ML-*).
func NormalizeBaselineSource(parts ...string) string {
	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case "demand_forecast_baseline", "moving_average", "predictive_push":
			return BaselineSourceMovingAverage
		case "seasonal_template", "seasonality_stub":
			return BaselineSourceSeasonalTemplate
		case "mixed":
			return BaselineSourceMixed
		case "ai_recommendations", "inventory_hint":
			return BaselineSourceInventoryHint
		case "croston":
			return BaselineSourceCroston
		case "holt_winters":
			return BaselineSourceHoltWinters
		case "ses":
			return BaselineSourceSES
		case "ml":
			return BaselineSourceMovingAverage
		}
	}
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			if s == "ml" {
				return BaselineSourceMovingAverage
			}
			return s
		}
	}
	return BaselineSourceMovingAverage
}
