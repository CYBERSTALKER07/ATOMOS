package predictivepush

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
)

type Allocator struct {
	spannerClient *spanner.Client
	locator       *Locator
}

func NewAllocator(client *spanner.Client, locator *Locator) *Allocator {
	return &Allocator{
		spannerClient: client,
		locator:       locator,
	}
}

// Allocate proactively generates ReplenishmentInsights for the predicted demand events.
func (a *Allocator) Allocate(ctx context.Context, events []*DemandEvent) error {
	if a == nil || a.spannerClient == nil {
		return nil
	}
	now := time.Now().UTC()
	insights := make([]planning.ReplenishmentInsightWriteInput, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		warehouse, err := a.locator.FindNearestWarehouse(ctx, event.RetailerId, event.SupplierId)
		if err != nil {
			return err
		}
		if warehouse == nil {
			continue
		}

		spread := int64(math.Max(1, math.Round(float64(event.Quantity)*0.1)))
		low := event.Quantity - spread
		if low < 0 {
			low = 0
		}
		high := event.Quantity + spread
		confPct := int64(math.Round(event.Confidence * 100))
		if confPct <= 0 {
			confPct = 65
		}
		demandBreakdown, _ := json.Marshal(map[string]interface{}{
			"retailerId":      event.RetailerId,
			"targetDate":      event.TargetDate.Format("2006-01-02"),
			"confidence":      event.Confidence,
			"confidence_pct":  confPct,
			"patternDays":     event.PatternDays,
			"predictedQty":    event.Quantity,
			"low_units":       low,
			"high_units":      high,
			"baseline_source": "moving_average",
			"label":           "standard",
		})

		insights = append(insights, planning.ReplenishmentInsightWriteInput{
			InsightID:       uuid.New().String(),
			WarehouseID:     warehouse.WarehouseId,
			ProductID:       event.ProductId,
			SupplierID:      event.SupplierId,
			SuggestedQty:    event.Quantity,
			UrgencyLevel:    "PROACTIVE",
			ReasonCode:      "PREDICTIVE_PUSH",
			Status:          "PENDING",
			TargetFactoryID: warehouse.PrimaryFactoryId,
			DemandBreakdown: string(demandBreakdown),
		})
	}

	if err := planning.WriteReplenishmentInsightsWithOutbox(ctx, a.spannerClient, now, insights); err != nil {
		return err
	}
	return a.writeDemandBaselines(ctx, events, now)
}

func (a *Allocator) writeDemandBaselines(ctx context.Context, events []*DemandEvent, now time.Time) error {
	if a == nil || a.spannerClient == nil || len(events) == 0 {
		return nil
	}
	// §8.1: Croston/HW nightly job owns DemandForecastBaseline when enabled.
	if forecastAlgoEnabled() {
		return nil
	}
	day := now.Truncate(24 * time.Hour)
	for _, event := range events {
		if event == nil || event.SupplierId == "" || event.ProductId == "" {
			continue
		}
		warehouse, err := a.locator.FindNearestWarehouse(ctx, event.RetailerId, event.SupplierId)
		if err != nil || warehouse == nil {
			continue
		}
		if err := planning.WriteBaselineWithOutbox(ctx, a.spannerClient, now, planning.BaselineWriteInput{
			SupplierID:     event.SupplierId,
			WarehouseID:    warehouse.WarehouseId,
			ProductID:      event.ProductId,
			ForecastDate:   day,
			BaselineQty:    event.Quantity,
			Confidence:     event.Confidence,
			Source:         "predictive_push",
			BaselineSource: "moving_average",
		}); err != nil {
			return err
		}
	}
	return nil
}

func forecastAlgoEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FORECAST_ALGO_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
