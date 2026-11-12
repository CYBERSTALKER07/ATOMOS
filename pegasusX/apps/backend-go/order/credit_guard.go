package order

import (
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CanLeaveOnCredit allows CREDIT_LEAVE from status + available credit only.
// Enforces PegasusX B2B Operating Model Invariants:
// 1) Frozen or perishable goods are never left outside unattended.
// 2) Closed-shop credit leave abuse: 3 leaves in 14 days revokes the leave-behind right.
// 3) Available credit headroom must cover the order total.
// Credit risk scoring / RiskTier is intentionally not used (Phase A removal).
func CanLeaveOnCredit(order *Order, profile *credit.Profile, cfg TimeoutConfig) error {
	if order == nil {
		return status.Errorf(codes.InvalidArgument, "order is nil")
	}

	// Invariant: Perishable/Frozen goods cannot sit outside unattended
	for _, item := range order.LineItems {
		if item.IsPerishable || strings.EqualFold(item.HandlingClass, "FROZEN") || strings.EqualFold(item.HandlingClass, "CHILLED") {
			return status.Errorf(codes.FailedPrecondition, "cannot leave perishable or frozen goods unattended")
		}
	}

	// Invariant: 3 closed-shop credit leaves in 14 days revokes leave-behind right (abuse policy)
	maxLeaves := cfg.MaxClosedShopLeaves14d
	if maxLeaves <= 0 {
		maxLeaves = 3
	}
	if order.RecentClosedShopLeavesCount >= maxLeaves {
		return status.Errorf(codes.FailedPrecondition, "leave-behind revoked: 3 or more closed-shop credit leaves in 14 days (abuse policy)")
	}

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
