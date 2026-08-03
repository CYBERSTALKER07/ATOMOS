package demand

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// DemandResult is the final number the rest of the system should use
type DemandResult struct {
	Value      float64            `json:"value"`
	Source     string             `json:"source"`
	Base       float64            `json:"base"`
	Multiplier float64            `json:"multiplier"`
	Factors    map[string]float64 `json:"factors"`
	ComputedAt time.Time          `json:"computedAt"`
}

// VelocityRecord holds fallback velocity data
type VelocityRecord struct {
	UnitsPerDay float64
}

// adjRepo handles fetching DemandAdjustments
type adjRepo struct {
	spanner *spanner.Client
}

func (r *adjRepo) Get(ctx context.Context, retailerID, sku string, date time.Time) (*DemandAdjustment, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT BaseVelocity, Adjustment, AdjustedDemand, FactorsJson, ComputedAt
			FROM DemandAdjustments
			WHERE RetailerId = @RetailerId AND Sku = @Sku AND Date = @Date
		`,
		Params: map[string]interface{}{
			"RetailerId": retailerID,
			"Sku":        sku,
			"Date":       civil.DateOf(date),
		},
	}
	iter := r.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // not found
	}
	if err != nil {
		return nil, err
	}

	var base, adj, adjusted float64
	var factorsJson spanner.NullString
	var computedAt time.Time
	if err := row.Columns(&base, &adj, &adjusted, &factorsJson, &computedAt); err != nil {
		return nil, err
	}

	var factors map[string]float64
	if factorsJson.Valid {
		_ = json.Unmarshal([]byte(factorsJson.StringVal), &factors)
	}

	return &DemandAdjustment{
		RetailerId:     retailerID,
		Sku:            sku,
		Date:           date,
		BaseVelocity:   base,
		Adjustment:     adj,
		AdjustedDemand: adjusted,
		Factors:        factors,
		ComputedAt:     computedAt,
	}, nil
}

// velocityRepo handles fetching raw velocity fallback data
type velocityRepo struct {
	spanner *spanner.Client
}

func (r *velocityRepo) Get(ctx context.Context, retailerID, sku string) (*VelocityRecord, error) {
	now := time.Now().UTC()
	twentyEightDaysAgo := now.Add(-28 * 24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `
			SELECT SUM(l.DeliveredQty) / 28.0 as UnitsPerDay
			FROM Orders o
			JOIN OrderLines l ON o.OrderId = l.OrderId
			WHERE o.Status = 'DELIVERED' 
			  AND o.CreatedAt >= @Start 
			  AND o.RetailerId = @RetailerId
			  AND l.Sku = @Sku
			HAVING SUM(l.DeliveredQty) > 0
		`,
		Params: map[string]interface{}{
			"Start":      twentyEightDaysAgo,
			"RetailerId": retailerID,
			"Sku":        sku,
		},
	}
	iter := r.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var vel float64
	if err := row.Columns(&vel); err != nil {
		return nil, err
	}

	return &VelocityRecord{UnitsPerDay: vel}, nil
}

func (r *velocityRepo) GetCategoryAverage(ctx context.Context, sku string) (float64, error) {
	now := time.Now().UTC()
	twentyEightDaysAgo := now.Add(-28 * 24 * time.Hour)

	// Since we don't have category info easily joined here, we average across all retailers for this SKU.
	stmt := spanner.Statement{
		SQL: `
			SELECT SUM(l.DeliveredQty) / (28.0 * COUNT(DISTINCT o.RetailerId)) as AvgVelocity
			FROM Orders o
			JOIN OrderLines l ON o.OrderId = l.OrderId
			WHERE o.Status = 'DELIVERED'
			  AND o.CreatedAt >= @Start
			  AND l.Sku = @Sku
			HAVING SUM(l.DeliveredQty) > 0
		`,
		Params: map[string]interface{}{
			"Start": twentyEightDaysAgo,
			"Sku":   sku,
		},
	}
	iter := r.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return 1.0, nil // absolute fallback
	}
	if err != nil {
		return 0, err
	}

	var cat float64
	if err := row.Columns(&cat); err != nil {
		return 0, err
	}
	return cat, nil
}

// EmpathyEngine computes final expected demand for inventory logic.
type EmpathyEngine struct {
	adjRepo      *adjRepo
	velocityRepo *velocityRepo
}

// NewEmpathyEngine initializes the engine with internal repositories.
func NewEmpathyEngine(client *spanner.Client) *EmpathyEngine {
	return &EmpathyEngine{
		adjRepo:      &adjRepo{spanner: client},
		velocityRepo: &velocityRepo{spanner: client},
	}
}

// GetExpectedDemand executes the fallback chain to yield deterministic demand.
func (e *EmpathyEngine) GetExpectedDemand(
	ctx context.Context,
	retailerID, sku string,
	date time.Time,
) (DemandResult, error) {

	// 1. Try adjusted demand first
	adj, err := e.adjRepo.Get(ctx, retailerID, sku, date)
	if err == nil && adj != nil {
		return DemandResult{
			Value:      adj.AdjustedDemand,
			Source:     "ADJUSTED",
			Base:       adj.BaseVelocity,
			Multiplier: adj.Adjustment,
			Factors:    adj.Factors,
			ComputedAt: adj.ComputedAt,
		}, nil
	}

	// 2. Fallback to pure velocity
	vel, err := e.velocityRepo.Get(ctx, retailerID, sku)
	if err == nil && vel != nil {
		return DemandResult{
			Value:  vel.UnitsPerDay,
			Source: "VELOCITY",
			Base:   vel.UnitsPerDay,
		}, nil
	}

	// 3. Final fallback – category or global average
	cat, err := e.velocityRepo.GetCategoryAverage(ctx, sku)
	if err != nil {
		return DemandResult{}, err
	}
	return DemandResult{
		Value:  cat,
		Source: "CATEGORY",
		Base:   cat,
	}, nil
}
