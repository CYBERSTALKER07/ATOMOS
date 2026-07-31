package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Service evaluates and mutates retailer credit state.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() string
}

// NewService builds a credit service.
func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   func() time.Time { return time.Now().UTC() },
		newID: func() string { return fmt.Sprintf("crp_%d", time.Now().UnixNano()) },
	}
}

// SetNow overrides the clock (tests only).
func (s *Service) SetNow(now func() time.Time) {
	s.now = now
}

// SetNewID overrides id generation (tests only).
func (s *Service) SetNewID(fn func() string) {
	s.newID = fn
}

// CheckOrder returns a credit check result for an order amount. A missing
// profile is treated as zero limit (blocked) unless the requested amount is zero.
func (s *Service) CheckOrder(ctx context.Context, retailerID, supplierID string, amountMinor int64) (CheckResult, error) {
	profile, found, err := s.repo.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return CheckResult{}, fmt.Errorf("credit check failed: %w", err)
	}
	if !found {
		if amountMinor == 0 {
			return CheckResult{Allowed: true, CreditLimitMinor: 0, CurrentBalance: 0, RequestedAmount: 0}, nil
		}
		return CheckResult{
			Allowed:          false,
			CreditLimitMinor: 0,
			CurrentBalance:   0,
			RequestedAmount:  amountMinor,
			Shortfall:        amountMinor,
			Reason:           "no_credit_profile",
		}, nil
	}

	result := CheckResult{
		Allowed:          false,
		CreditLimitMinor: profile.CreditLimitMinor,
		CurrentBalance:   profile.CurrentBalanceMinor,
		RequestedAmount:  amountMinor,
	}

	switch profile.Status {
	case StatusBlacklisted:
		result.Reason = "profile_blacklisted"
		result.Shortfall = amountMinor
		return result, nil
	case StatusFrozen:
		result.Reason = "profile_frozen"
		result.Shortfall = amountMinor
		return result, nil
	}

	if profile.RiskTier == RiskTierBlock {
		result.Reason = "risk_tier_block"
		result.Shortfall = amountMinor
		return result, nil
	}

	if profile.CreditLimitMinor <= 0 {
		result.Reason = "no_credit_limit"
		result.Shortfall = amountMinor
		return result, nil
	}

	projected := profile.CurrentBalanceMinor + amountMinor
	if projected > profile.CreditLimitMinor {
		result.Shortfall = projected - profile.CreditLimitMinor
		result.Reason = "credit_limit_breached"
		return result, nil
	}

	result.Allowed = true
	return result, nil
}

// MarkBalance increases the current balance when a credit delivery is recorded.
func (s *Service) MarkBalance(ctx context.Context, retailerID, supplierID string, amountMinor int64, orderID string) error {
	if amountMinor <= 0 {
		return nil
	}
	return s.repo.AdjustBalance(ctx, retailerID, supplierID, amountMinor, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: s.now().Format(time.RFC3339Nano)},
			ProfileID:      profileID(retailerID, supplierID),
			RetailerID:     retailerID,
			SupplierID:     supplierID,
			CurrentBalance: amountMinor,
			RiskTier:       string(RiskTierMedium),
			Reason:         fmt.Sprintf("credit_delivery:%s", orderID),
		})
	})
}

// ClearBalance decreases the current balance when a credit delivery is paid.
func (s *Service) ClearBalance(ctx context.Context, retailerID, supplierID string, amountMinor int64, orderID string) error {
	if amountMinor <= 0 {
		return nil
	}
	return s.repo.AdjustBalance(ctx, retailerID, supplierID, -amountMinor, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: s.now().Format(time.RFC3339Nano)},
			ProfileID:      profileID(retailerID, supplierID),
			RetailerID:     retailerID,
			SupplierID:     supplierID,
			CurrentBalance: -amountMinor,
			RiskTier:       string(RiskTierMedium),
			Reason:         fmt.Sprintf("credit_payment:%s", orderID),
		})
	})
}

// GetProfile loads a retailer credit profile for a supplier.
func (s *Service) GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	if s == nil || s.repo == nil {
		return Profile{}, false, fmt.Errorf("credit service unavailable")
	}
	return s.repo.GetProfile(ctx, retailerID, supplierID)
}

// UpsertProfile creates or updates a credit profile.
func (s *Service) UpsertProfile(ctx context.Context, p Profile, actorID, reason string) error {
	now := s.now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.LastEvaluatedAt.IsZero() {
		p.LastEvaluatedAt = now
	}
	if p.RiskTier == "" {
		p.RiskTier = RiskTierMedium
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	p.AvailableCreditMinor = p.CreditLimitMinor - p.CurrentBalanceMinor
	if p.AvailableCreditMinor < 0 {
		p.AvailableCreditMinor = 0
	}

	return s.repo.UpsertProfile(ctx, p, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, p.RetailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: now.Format(time.RFC3339Nano)},
			ProfileID:        profileID(p.RetailerID, p.SupplierID),
			RetailerID:       p.RetailerID,
			SupplierID:       p.SupplierID,
			CreditLimitMinor: p.CreditLimitMinor,
			CurrentBalance:   p.CurrentBalanceMinor,
			RiskTier:         string(p.RiskTier),
			Delinquent:       p.DelinquencyCount > 0,
			Reason:           reason,
		})
	})
}

// EvaluateRisk recomputes risk tier from delinquency count and balance.
func (s *Service) EvaluateRisk(delinquencyCount, balanceMinor, limitMinor int64) RiskTier {
	return deriveRiskTier(delinquencyCount, balanceMinor, limitMinor)
}

// ListSupplierProfiles returns supplier-scoped credit profiles for the collections desk.
func (s *Service) ListSupplierProfiles(ctx context.Context, supplierID, status string, limit int) ([]Profile, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("credit service unavailable")
	}
	list, err := s.repo.ListBySupplier(ctx, supplierID, status, limit)
	if err != nil {
		return nil, err
	}
	// Collections-first ordering: open balance / frozen / high risk before idle active lines.
	sortProfilesForCollections(list)
	return list, nil
}

func sortProfilesForCollections(list []Profile) {
	// Stable priority: BLACKLISTED/FROZEN first, then balance desc, then updated desc.
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if profilePriority(list[j]) < profilePriority(list[i]) {
				list[i], list[j] = list[j], list[i]
			} else if profilePriority(list[j]) == profilePriority(list[i]) &&
				list[j].CurrentBalanceMinor > list[i].CurrentBalanceMinor {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

func profilePriority(p Profile) int {
	switch p.Status {
	case StatusBlacklisted:
		return 0
	case StatusFrozen:
		return 1
	case StatusClosed:
		return 4
	default:
		if p.CurrentBalanceMinor > 0 {
			return 2
		}
		return 3
	}
}

func profileID(retailerID, supplierID string) string {
	return fmt.Sprintf("%s:%s", retailerID, supplierID)
}
