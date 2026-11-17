package demand

import (
	"context"
	"encoding/json"
	"strings"
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
	iter := r.spanner.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
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
			SELECT LineItemsJson
			FROM Orders
			WHERE Status = 'DELIVERED' 
			  AND CreatedAt >= @Start 
			  AND RetailerId = @RetailerId
		`,
		Params: map[string]interface{}{
			"Start":      twentyEightDaysAgo,
			"RetailerId": retailerID,
		},
	}
	iter := r.spanner.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	targetSKU := strings.TrimSpace(sku)
	var sumDelivered float64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var raw []byte
		if err := row.ColumnByName("LineItemsJson", &raw); err != nil {
			var rawStr string
			if errStr := row.ColumnByName("LineItemsJson", &rawStr); errStr == nil {
				raw = []byte(rawStr)
			} else {
				return nil, err
			}
		}
		if len(raw) == 0 {
			continue
		}
		var lines []struct {
			SKU          string  `json:"sku"`
			ProductID    string  `json:"product_id"`
			DeliveredQty float64 `json:"delivered_qty"`
			Quantity     float64 `json:"quantity"`
			Qty          float64 `json:"qty"`
		}
		if err := json.Unmarshal(raw, &lines); err != nil {
			continue
		}
		for _, l := range lines {
			lineSKU := strings.TrimSpace(l.SKU)
			if lineSKU == "" {
				lineSKU = strings.TrimSpace(l.ProductID)
			}
			if lineSKU != targetSKU {
				continue
			}
			qty := l.DeliveredQty
			if qty == 0 {
				qty = l.Quantity
			}
			if qty == 0 {
				qty = l.Qty
			}
			if qty > 0 {
				sumDelivered += qty
			}
		}
	}

	if sumDelivered <= 0 {
		return nil, nil
	}

	return &VelocityRecord{UnitsPerDay: sumDelivered / 28.0}, nil
}

func (r *velocityRepo) GetCategoryAverage(ctx context.Context, sku string) (float64, error) {
	now := time.Now().UTC()
	twentyEightDaysAgo := now.Add(-28 * 24 * time.Hour)

	// Since we don't have category info easily joined here, we average across all retailers for this SKU.
	stmt := spanner.Statement{
		SQL: `
			SELECT RetailerId, LineItemsJson
			FROM Orders
			WHERE Status = 'DELIVERED'
			  AND CreatedAt >= @Start
		`,
		Params: map[string]interface{}{
			"Start": twentyEightDaysAgo,
		},
	}
	iter := r.spanner.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	targetSKU := strings.TrimSpace(sku)
	var sumDelivered float64
	distinctRetailers := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		var retailerID string
		if err := row.ColumnByName("RetailerId", &retailerID); err != nil {
			return 0, err
		}
		var raw []byte
		if err := row.ColumnByName("LineItemsJson", &raw); err != nil {
			var rawStr string
			if errStr := row.ColumnByName("LineItemsJson", &rawStr); errStr == nil {
				raw = []byte(rawStr)
			} else {
				return 0, err
			}
		}
		if len(raw) == 0 {
			continue
		}
		var lines []struct {
			SKU          string  `json:"sku"`
			ProductID    string  `json:"product_id"`
			DeliveredQty float64 `json:"delivered_qty"`
			Quantity     float64 `json:"quantity"`
			Qty          float64 `json:"qty"`
		}
		if err := json.Unmarshal(raw, &lines); err != nil {
			continue
		}
		for _, l := range lines {
			lineSKU := strings.TrimSpace(l.SKU)
			if lineSKU == "" {
				lineSKU = strings.TrimSpace(l.ProductID)
			}
			if lineSKU != targetSKU {
				continue
			}
			qty := l.DeliveredQty
			if qty == 0 {
				qty = l.Quantity
			}
			if qty == 0 {
				qty = l.Qty
			}
			if qty > 0 {
				sumDelivered += qty
				distinctRetailers[retailerID] = struct{}{}
			}
		}
	}

	if sumDelivered <= 0 || len(distinctRetailers) == 0 {
		return 1.0, nil // absolute fallback
	}

	return sumDelivered / (28.0 * float64(len(distinctRetailers))), nil
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
