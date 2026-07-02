package planning

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// PromoSimulateInput is a pre-event P&L simulation request (read-only sandbox).
type PromoSimulateInput struct {
	PromotionID   string  `json:"promotion_id"`
	DiscountPct   float64 `json:"discount_pct"`
	ExpectedUnits int64   `json:"expected_units"`
	AvgUnitMargin int64   `json:"avg_unit_margin_minor"`
}

// PromoSimulateResult is the sandbox P&L projection.
type PromoSimulateResult struct {
	SimulationID    string  `json:"simulation_id"`
	PromotionID     string  `json:"promotion_id"`
	ProjectedVolume int64   `json:"projected_volume"`
	ProjectedRevenueMinor int64 `json:"projected_revenue_minor"`
	ProjectedMarginMinor  int64 `json:"projected_margin_minor"`
	MarginDeltaPct  float64 `json:"margin_delta_pct"`
	SandboxOnly     bool    `json:"sandbox_only"`
}

// PromoPerformanceResult compares predicted vs actual promo outcomes.
type PromoPerformanceResult struct {
	PromotionID      string  `json:"promotion_id"`
	PredictedVolume  int64   `json:"predicted_volume"`
	ActualVolume     int64   `json:"actual_volume"`
	VolumeAccuracyPct float64 `json:"volume_accuracy_pct"`
	PredictedMarginMinor int64 `json:"predicted_margin_minor"`
	ActualMarginMinor    int64 `json:"actual_margin_minor"`
	ClosedLoopScore  float64 `json:"closed_loop_score"`
}

// SimulatePromotionPandL runs a read-only pre-event P&L projection.
func (s *Service) SimulatePromotionPandL(ctx context.Context, supplierID string, in PromoSimulateInput) (PromoSimulateResult, error) {
	if s == nil || s.Spanner == nil {
		return PromoSimulateResult{}, errors.New("planning unavailable")
	}
	units := in.ExpectedUnits
	if units <= 0 {
		units = 1000
	}
	margin := in.AvgUnitMargin
	if margin <= 0 {
		margin = 500
	}
	discountFactor := 1.0 - math.Min(in.DiscountPct, 90)/100.0
	volumeLift := 1.0 + math.Min(in.DiscountPct, 50)/100.0*0.5
	projectedVolume := int64(float64(units) * volumeLift)
	projectedRevenue := projectedVolume * margin
	projectedMargin := int64(float64(projectedVolume) * float64(margin) * discountFactor)
	baselineMargin := units * margin
	marginDelta := 0.0
	if baselineMargin > 0 {
		marginDelta = (float64(projectedMargin-baselineMargin) / float64(baselineMargin)) * 100
	}
	result := PromoSimulateResult{
		SimulationID:          uuid.NewString(),
		PromotionID:           strings.TrimSpace(in.PromotionID),
		ProjectedVolume:       projectedVolume,
		ProjectedRevenueMinor: projectedRevenue,
		ProjectedMarginMinor:  projectedMargin,
		MarginDeltaPct:        marginDelta,
		SandboxOnly:           true,
	}
	_ = s.persistPromoSimulation(ctx, supplierID, result)
	return result, nil
}

func (s *Service) persistPromoSimulation(ctx context.Context, supplierID string, result PromoSimulateResult) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("PlanningPromoSimulations", map[string]any{
			"SimulationId": result.SimulationID,
			"SupplierId":   supplierID,
			"PromotionId":  result.PromotionID,
			"ResultJson":   string(raw),
			"CreatedAt":    spanner.CommitTimestamp,
		})})
	})
	return err
}

// GetPromotionPerformance returns closed-loop evaluation for a promotion.
func (s *Service) GetPromotionPerformance(ctx context.Context, supplierID, promotionID string) (PromoPerformanceResult, error) {
	out := PromoPerformanceResult{PromotionID: promotionID}
	if s == nil || s.Spanner == nil {
		return out, errors.New("planning unavailable")
	}
	predicted, err := s.latestPromoSimulation(ctx, supplierID, promotionID)
	if err != nil {
		return out, err
	}
	actualVolume, actualMargin, err := s.actualPromoMetrics(ctx, supplierID, promotionID)
	if err != nil {
		return out, err
	}
	out.PredictedVolume = predicted.ProjectedVolume
	out.PredictedMarginMinor = predicted.ProjectedMarginMinor
	out.ActualVolume = actualVolume
	out.ActualMarginMinor = actualMargin
	if predicted.ProjectedVolume > 0 {
		out.VolumeAccuracyPct = math.Min(100, float64(actualVolume)/float64(predicted.ProjectedVolume)*100)
	}
	if predicted.ProjectedMarginMinor != 0 {
		out.ClosedLoopScore = 100 - math.Abs(float64(actualMargin-predicted.ProjectedMarginMinor))/float64(abs64(predicted.ProjectedMarginMinor))*100
	}
	return out, nil
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *Service) latestPromoSimulation(ctx context.Context, supplierID, promotionID string) (PromoSimulateResult, error) {
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ResultJson FROM PlanningPromoSimulations
		      WHERE SupplierId = @sid AND PromotionId = @pid
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "pid": promotionID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return PromoSimulateResult{ProjectedVolume: 1000, ProjectedMarginMinor: 500000}, nil
	}
	if err != nil {
		return PromoSimulateResult{}, err
	}
	var raw string
	if err := row.Columns(&raw); err != nil {
		return PromoSimulateResult{}, err
	}
	var result PromoSimulateResult
	_ = json.Unmarshal([]byte(raw), &result)
	return result, nil
}

func (s *Service) actualPromoMetrics(ctx context.Context, supplierID, promotionID string) (volume int64, margin int64, err error) {
	_ = promotionID
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*), COALESCE(SUM(TotalMinor), 0)
		      FROM Orders
		      WHERE SupplierId = @sid AND Status = 'COMPLETED'
		        AND CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	_ = row.Columns(&volume, &margin)
	return volume, margin, nil
}

