package order

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CanLeaveOnCredit(order *Order, profile *credit.Profile, score *credit.RetailerCreditScore, cfg TimeoutConfig, featureEnabled bool) error {
	if profile == nil || profile.Status != "ACTIVE" {
		return status.Errorf(codes.FailedPrecondition, "credit profile not active")
	}
	// Note: prompt said AvailableMinor < order.GrossMinor, but order usually has TotalMinor.
	// Will use TotalMinor as used in the existing DecideShopClosedTimeout.
	if profile.AvailableCreditMinor < order.TotalMinor {
		return status.Errorf(codes.FailedPrecondition, "insufficient credit")
	}
	if order.TotalMinor > cfg.MaxAutoCreditMinor {
		return status.Errorf(codes.FailedPrecondition, "order total exceeds max auto credit")
	}

	if featureEnabled {
		var riskTier int
		if score != nil {
			riskTier = riskTierLevel(score.RiskTier)
		} else {
			riskTier = 2 // treat as neutral
		}
		if riskTier > cfg.MaxRiskTierForAutoCredit {
			return status.Errorf(codes.FailedPrecondition,
				"retailer risk tier %d exceeds max allowed %d", riskTier, cfg.MaxRiskTierForAutoCredit)
		}
	}

	return nil
}

func getRetailerCreditScore(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID string) (*credit.RetailerCreditScore, error) {
	row, err := txn.ReadRow(ctx, "RetailerCreditScores", spanner.Key{retailerID}, []string{
		"Score", "RiskTier",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var rcs credit.RetailerCreditScore
	var riskTier spanner.NullString
	if err := row.Columns(&rcs.Score, &riskTier); err != nil {
		return nil, err
	}
	rcs.RetailerID = retailerID
	if riskTier.Valid {
		rcs.RiskTier = credit.RiskTier(riskTier.StringVal)
	}
	return &rcs, nil
}

