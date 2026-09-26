package credit

import (
	"errors"
	"cloud.google.com/go/spanner"

	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Service evaluates and mutates retailer credit state.
type Service struct {
	repo         Repository
	policy       PolicyGate
	now          func() time.Time
	newID        func() string
	scoreMetrics ScoreMetricsProvider // optional AR/payment feed for G3 scoring
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

// CheckOrder returns a credit check result for an order amount on the credit path.
// Freeze / inactive / missing profile reject CREDIT only — cash/card callers must not use this gate.
// Available = limit - balance - reserved (CAS headroom).
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
		ReservedMinor:    profile.ReservedMinor,
		RequestedAmount:  amountMinor,
	}

	switch profile.Status {
	case StatusBlacklisted:
		result.Reason = "profile_blacklisted"
		result.Shortfall = amountMinor
		return result, nil
	case StatusFrozen:
		// Freeze = credit-path only; relationship stays ON and visible.
		result.Reason = "profile_frozen"
		result.Shortfall = amountMinor
		return result, nil
	case StatusInactive, StatusClosed:
		result.Reason = "credit_not_enabled"
		result.Shortfall = amountMinor
		return result, nil
	}

	if profile.CreditLimitMinor <= 0 {
		result.Reason = "no_credit_limit"
		result.Shortfall = amountMinor
		return result, nil
	}

	avail := profile.Available()
	if amountMinor > avail {
		result.Shortfall = amountMinor - avail
		result.Reason = "credit_limit_breached"
		return result, nil
	}

	result.Allowed = true
	return result, nil
}

// PolicyGate optionally enforces program + relationship enablement (CREDIT_POLICY_V2).
type PolicyGate interface {
	CreditPathAllowed(ctx context.Context, retailerID, supplierID string) (allowed bool, reason string, termsDays int64, err error)
	ResolveDueAt(ctx context.Context, retailerID, supplierID string, creditLeaveAt time.Time) (dueAt time.Time, termsDays int64, err error)
}

// SetPolicyGate wires the irreversible enable / terms gate.
func (s *Service) SetPolicyGate(g PolicyGate) {
	s.policy = g
}

// CheckCreditPath combines profile CheckOrder with policy program/relationship enablement.
func (s *Service) CheckCreditPath(ctx context.Context, retailerID, supplierID string, amountMinor int64) (CheckResult, error) {
	if s.policy != nil {
		ok, reason, termsDays, err := s.policy.CreditPathAllowed(ctx, retailerID, supplierID)
		if err != nil {
			return CheckResult{}, err
		}
		if !ok {
			return CheckResult{
				Allowed:         false,
				RequestedAmount: amountMinor,
				Shortfall:       amountMinor,
				Reason:          reason,
				TermsDays:       termsDays,
			}, nil
		}
	}
	result, err := s.CheckOrder(ctx, retailerID, supplierID, amountMinor)
	if err != nil {
		return result, err
	}
	if result.Allowed && s.policy != nil {
		due, terms, derr := s.policy.ResolveDueAt(ctx, retailerID, supplierID, s.now())
		if derr == nil {
			result.DueAt = due.UTC().Format(time.RFC3339)
			result.TermsDays = terms
		}
	}
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

// BumpDelinquency increments DelinquencyCount by 1 for collections (first OVERDUE).
// Idempotent only at the call-site (dunning worker bumps once per step transition).
func (s *Service) BumpDelinquency(ctx context.Context, retailerID, supplierID string) error {
	p, found, err := s.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	p.DelinquencyCount++
	p.LastEvaluatedAt = s.now()
	return s.UpsertProfile(ctx, p, "system:dunning", "delinquency_bump")
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
	if p.Status == "" {
		p.Status = StatusInactive
	}
	p.RiskTier = "" // scoring product removed
	p.AvailableCreditMinor = p.Available()

	return s.repo.UpsertProfile(ctx, p, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, p.RetailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: now.Format(time.RFC3339Nano)},
			ProfileID:        profileID(p.RetailerID, p.SupplierID),
			RetailerID:       p.RetailerID,
			SupplierID:       p.SupplierID,
			CreditLimitMinor: p.CreditLimitMinor,
			CurrentBalance:   p.CurrentBalanceMinor,
			Delinquent:       p.DelinquencyCount > 0,
			Reason:           reason,
		})
	})
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
	// G3.B: refresh risk scores for desk ordering (in-memory; persist best-effort).
	if ScoringEnabled() {
		for i := range list {
			sc, err := s.EvaluateProfileScore(ctx, list[i])
			if err != nil {
				continue
			}
			ApplyScoreToProfile(&list[i], sc)
		}
	}
	// Collections-first ordering: open balance / frozen / high risk before idle active lines.
	sortProfilesForCollections(list)
	return list, nil
}

// GetScoresForRetailers returns computed risk scores (G3.B). Empty map when scoring disabled.
func (s *Service) GetScoresForRetailers(ctx context.Context, supplierID string, retailerIDs []string) (map[string]RetailerCreditScore, error) {
	out := make(map[string]RetailerCreditScore, len(retailerIDs))
	if s == nil || s.repo == nil || !ScoringEnabled() {
		return out, nil
	}
	for _, rid := range retailerIDs {
		rid = strings.TrimSpace(rid)
		if rid == "" {
			continue
		}
		p, found, err := s.repo.GetProfile(ctx, rid, supplierID)
		if err != nil || !found {
			continue
		}
		sc, err := s.EvaluateProfileScore(ctx, p)
		if err != nil {
			continue
		}
		out[rid] = sc
	}
	return out, nil
}

func sortProfilesForCollections(list []Profile) {
	// Priority: BLACKLISTED/FROZEN, then lower risk score (worse), then balance desc.
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			pi, pj := profilePriority(list[i]), profilePriority(list[j])
			if pj < pi {
				list[i], list[j] = list[j], list[i]
				continue
			}
			if pj == pi {
				// Worse (lower) score first for collections attention.
				if list[j].RiskScore > 0 && list[i].RiskScore > 0 && list[j].RiskScore < list[i].RiskScore {
					list[i], list[j] = list[j], list[i]
					continue
				}
				if list[j].CurrentBalanceMinor > list[i].CurrentBalanceMinor {
					list[i], list[j] = list[j], list[i]
				}
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
		if p.CurrentBalanceMinor > 0 || p.DelinquencyCount > 0 {
			return 2
		}
		return 3
	}
}

func profileID(retailerID, supplierID string) string {
	return fmt.Sprintf("%s:%s", retailerID, supplierID)
}

func (s *Service) ReserveOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	res := OrderReservation{
		OrderID:     orderID,
		RetailerID:  retailerID,
		SupplierID:  supplierID,
		AmountMinor: amountMinor,
		Status:      ReservationReserved,
	}
	if r, ok := s.repo.(*SpannerRepository); ok {
		return r.ReserveOrderInTxn(ctx, txn, res)
	}
	return errors.New("not supported")
}
