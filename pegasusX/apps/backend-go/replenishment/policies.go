package replenishment

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Policy holds touchless + safety-stock knobs for one supplier.
type Policy struct {
	SupplierID                string  `json:"supplier_id"`
	AutoApproveStable         bool    `json:"auto_approve_stable"`
	AutoApprovePredictivePush bool    `json:"auto_approve_predictive_push"`
	MaxDailyTransferUnits     int64   `json:"max_daily_transfer_units"`
	MinConfidenceScore        float64 `json:"min_confidence_score"`
	TargetServiceLevel        float64 `json:"target_service_level"`
	LeadTimeDays              int64   `json:"lead_time_days"`
	LeadTimeSigmaDays         float64 `json:"lead_time_sigma_days"`
}

func defaultPolicy(supplierID string) Policy {
	return Policy{
		SupplierID:                supplierID,
		AutoApproveStable:         true,
		AutoApprovePredictivePush: true,
		MaxDailyTransferUnits:     500,
		MinConfidenceScore:        0.85,
		TargetServiceLevel:        0.98,
		LeadTimeDays:              defaultLeadTimeDays,
		LeadTimeSigmaDays:         1.0,
	}
}

// LoadPolicy returns the supplier policy row or defaults when missing.
func LoadPolicy(ctx context.Context, client *spanner.Client, supplierID string) (Policy, error) {
	if client == nil || supplierID == "" {
		return Policy{}, errors.New("policy lookup unavailable")
	}
	stmt := spanner.Statement{
		SQL: `SELECT AutoApproveStable, AutoApprovePredictivePush,
		             MaxDailyTransferUnits, MinConfidenceScore,
		             COALESCE(TargetServiceLevel, 0.98),
		             COALESCE(LeadTimeDays, 2),
		             COALESCE(LeadTimeSigmaDays, 1.0)
		      FROM ReplenishmentPolicies WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return defaultPolicy(supplierID), nil
	}
	if err != nil {
		return Policy{}, err
	}
	p := defaultPolicy(supplierID)
	if err := row.Columns(
		&p.AutoApproveStable, &p.AutoApprovePredictivePush,
		&p.MaxDailyTransferUnits, &p.MinConfidenceScore,
		&p.TargetServiceLevel, &p.LeadTimeDays, &p.LeadTimeSigmaDays,
	); err != nil {
		return Policy{}, err
	}
	if p.TargetServiceLevel <= 0 {
		p.TargetServiceLevel = 0.98
	}
	if p.LeadTimeDays <= 0 {
		p.LeadTimeDays = defaultLeadTimeDays
	}
	if p.LeadTimeSigmaDays < 0 {
		p.LeadTimeSigmaDays = 1.0
	}
	return p, nil
}

// UpsertPolicy writes the full policy row.
func UpsertPolicy(ctx context.Context, client *spanner.Client, p Policy) error {
	if client == nil || p.SupplierID == "" {
		return errors.New("policy upsert unavailable")
	}
	if p.TargetServiceLevel <= 0 {
		p.TargetServiceLevel = 0.98
	}
	if p.LeadTimeDays <= 0 {
		p.LeadTimeDays = defaultLeadTimeDays
	}
	if p.LeadTimeSigmaDays < 0 {
		p.LeadTimeSigmaDays = 1.0
	}
	if p.MaxDailyTransferUnits <= 0 {
		p.MaxDailyTransferUnits = 500
	}
	if p.MinConfidenceScore <= 0 {
		p.MinConfidenceScore = 0.85
	}
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("ReplenishmentPolicies", map[string]any{
			"SupplierId":                p.SupplierID,
			"AutoApproveStable":         p.AutoApproveStable,
			"AutoApprovePredictivePush": p.AutoApprovePredictivePush,
			"MaxDailyTransferUnits":     p.MaxDailyTransferUnits,
			"MinConfidenceScore":        p.MinConfidenceScore,
			"TargetServiceLevel":        p.TargetServiceLevel,
			"LeadTimeDays":              p.LeadTimeDays,
			"LeadTimeSigmaDays":         p.LeadTimeSigmaDays,
			"UpdatedAt":                 spanner.CommitTimestamp,
		}),
	})
	return err
}

// EnsurePolicy seeds a default policy row when absent.
func EnsurePolicy(ctx context.Context, client *spanner.Client, supplierID string) error {
	if client == nil || supplierID == "" {
		return nil
	}
	p := defaultPolicy(supplierID)
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT SupplierId FROM ReplenishmentPolicies WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supplierID},
		})
		defer iter.Stop()
		_, nextErr := iter.Next()
		if nextErr == nil {
			return nil
		}
		if !errors.Is(nextErr, iterator.Done) {
			return nextErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ReplenishmentPolicies", map[string]any{
				"SupplierId":                supplierID,
				"AutoApproveStable":         p.AutoApproveStable,
				"AutoApprovePredictivePush": p.AutoApprovePredictivePush,
				"MaxDailyTransferUnits":     p.MaxDailyTransferUnits,
				"MinConfidenceScore":        p.MinConfidenceScore,
				"TargetServiceLevel":        p.TargetServiceLevel,
				"LeadTimeDays":              p.LeadTimeDays,
				"LeadTimeSigmaDays":         p.LeadTimeSigmaDays,
				"UpdatedAt":                 spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// CountAutoApprovedToday returns units auto-approved today for cap enforcement.
func CountAutoApprovedToday(ctx context.Context, client *spanner.Client, supplierID string, now time.Time) (int64, error) {
	if client == nil {
		return 0, nil
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(SuggestedQuantity), 0)
		      FROM ReplenishmentInsights
		      WHERE SupplierId = @sid AND Status = 'APPROVED'
		        AND ReasonCode IN ('PREDICTIVE_PUSH', 'MEIO_NETWORK', 'LOW_STOCK', 'HIGH_VELOCITY')
		        AND CreatedAt >= @dayStart`,
		Params: map[string]any{"sid": supplierID, "dayStart": dayStart},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	return total, row.Columns(&total)
}
