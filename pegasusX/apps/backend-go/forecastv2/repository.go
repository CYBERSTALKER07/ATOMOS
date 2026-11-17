package forecastv2

import (
	"context"
	"time"
)

// Repository persists M1.5 forecast runs, quantile values, explanations, and reads historical demand.
type Repository interface {
	SaveRunAndValues(ctx context.Context, run ForecastRun, values []ForecastValue, explanations []ForecastExplanation) error
	GetRun(ctx context.Context, entityID, runID string) (ForecastRun, bool, error)
	GetLatestRun(ctx context.Context, entityID string) (ForecastRun, bool, error)
	ListForecastValues(ctx context.Context, q ForecastQuery) ([]ForecastValue, error)
	ListExplanations(ctx context.Context, entityID, runID, locationID, sku string, limit int) ([]ForecastExplanation, error)
	GetExplanation(ctx context.Context, entityID, explanationID string) (ForecastExplanation, bool, error)
	SaveExplanation(ctx context.Context, exp ForecastExplanation) error
	UpdateExplanation(ctx context.Context, exp ForecastExplanation) error
	LoadHistoricalSeries(ctx context.Context, entityID, locationID, sku string, asOf time.Time, lookbackDays int) ([]SeriesHistoryInput, []SeasonalWindow, error)
	LoadActualMilli(ctx context.Context, entityID, locationID, sku string, windowStart, windowEnd time.Time) (int64, bool, error)
	IsRetailerLinked(ctx context.Context, supplierID, retailerID string) (bool, error)
}
