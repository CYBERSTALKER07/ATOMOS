package order

import (
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CanLeaveOnCredit allows CREDIT_LEAVE from status + available credit only.
// Credit risk scoring / RiskTier is intentionally not used (Phase A removal).
func CanLeaveOnCredit(order *Order, profile *credit.Profile, cfg TimeoutConfig) error {
	if profile == nil || profile.Status != credit.StatusActive {
		return status.Errorf(codes.FailedPrecondition, "credit profile not active")
	}
	if profile.Available() < order.TotalMinor {
		return status.Errorf(codes.FailedPrecondition, "insufficient credit")
	}
	if cfg.MaxAutoCreditMinor > 0 && order.TotalMinor > cfg.MaxAutoCreditMinor {
		return status.Errorf(codes.FailedPrecondition, "order total exceeds max auto credit")
	}
	return nil
}
