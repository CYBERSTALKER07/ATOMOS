package credit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Service evaluates and mutates retailer credit state.
type Service struct {
	repo                Repository
	policy              PolicyGate
	now                 func() time.Time
	newID               func() string
	scoreMetrics        ScoreMetricsProvider   // optional AR/payment feed for G3 scoring
	deskMetadataFn      DeskMetadataLookup     // optional override for desk metadata in unit tests
	scoreRepo           ScoreHistoryRepository // V2 CreditScoreHistory + consent + governed limits
	mlScorer            MLScorer               // config-driven CREDIT_ML_SCORER_URL adapter
	limitApprovalBridge LimitApprovalBridge    // optional M1.2 CEL approval + SoD bridge
	policySvc           *PolicyService         // optional PolicyService reference for consent/terms lookups
	mutationContextHook func(ctx context.Context, tenantID, action, objectType, objectID, reasonCode, approvalID string)
	mutMu               sync.Mutex
}

// NewService builds a credit service.
func NewService(repo Repository) *Service {
	svc := &Service{
		repo:     repo,
		mlScorer: NewHTTPMLScorerClientFromEnv(),
		now:      func() time.Time { return time.Now().UTC() },
		newID:    func() string { return "crp_" + strings.ReplaceAll(uuid.NewString(), "-", "") },
	}
	if sr, ok := repo.(*SpannerRepository); ok && sr != nil && sr.client != nil {
		svc.scoreRepo = NewSpannerScoreHistoryRepo(sr.client)
		svc.scoreMetrics = &ARScoreMetrics{Client: sr.client}
	}
	return svc
}

// SetScoreHistoryRepo wires the V2 score history & governed limit repository (tests / custom wiring).
func (s *Service) SetScoreHistoryRepo(r ScoreHistoryRepository) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.scoreRepo = r
	s.mutMu.Unlock()
}

// SetMLScorer overrides the ML credit scorer client (tests / custom wiring).
func (s *Service) SetMLScorer(m MLScorer) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.mlScorer = m
	s.mutMu.Unlock()
}

// SetLimitApprovalBridge wires M1.2 CEL approval gating for POST /v1/credit/limits/{id}.
func (s *Service) SetLimitApprovalBridge(b LimitApprovalBridge) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.limitApprovalBridge = b
	s.mutMu.Unlock()
}

// SetPolicyService wires PolicyService for reading payment terms / ConsentVerified when needed.
func (s *Service) SetPolicyService(ps *PolicyService) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.policySvc = ps
	s.mutMu.Unlock()
}

// SetMutationContextHook attaches an optional audit.SetMutationContext callback without importing audit.
func (s *Service) SetMutationContextHook(fn func(ctx context.Context, tenantID, action, objectType, objectID, reasonCode, approvalID string)) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	s.mutationContextHook = fn
	s.mutMu.Unlock()
}

func (s *Service) setMutationContext(ctx context.Context, tenantID, action, objectType, objectID, reasonCode, approvalID string) {
	if s == nil {
		return
	}
	s.mutMu.Lock()
	fn := s.mutationContextHook
	s.mutMu.Unlock()
	if fn != nil {
		fn(ctx, tenantID, action, objectType, objectID, reasonCode, approvalID)
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
		RiskTier:         profile.RiskTier,
		RiskScore:        profile.RiskScore,
		PrepayOnlyLocked: profile.PrepayOnlyLocked,
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

	if profile.PrepayOnlyLocked {
		// Prepay lock blocks CREDIT orders while Cash/Card paths remain open.
		result.Reason = "prepay_only_locked"
		result.Shortfall = amountMinor
		return result, nil
	}

	if profile.DelinquencyCount >= MaxDelinquencyTolerance() {
		result.Reason = "delinquent_account_hold"
		result.Shortfall = amountMinor
		return result, nil
	}

	if UnlimitedPaymentsEnabled() {
		result.Allowed = true
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
	emitClear := func(txn outbox.TxnBuffer, clearedMinor int64) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: s.now().Format(time.RFC3339Nano)},
			ProfileID:      profileID(retailerID, supplierID),
			RetailerID:     retailerID,
			SupplierID:     supplierID,
			CurrentBalance: -clearedMinor,
			Reason:         fmt.Sprintf("credit_payment:%s", orderID),
		})
	}
	if clearer, ok := s.repo.(OrderBalanceClearer); ok && strings.TrimSpace(orderID) != "" {
		return clearer.ClearOrderBalance(ctx, retailerID, supplierID, orderID, amountMinor, emitClear)
	}
	return s.repo.AdjustBalance(ctx, retailerID, supplierID, -amountMinor, func(txn outbox.TxnBuffer) error {
		return emitClear(txn, amountMinor)
	})
}

// GetProfile loads a retailer credit profile for a supplier.
func (s *Service) GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	if s == nil || s.repo == nil {
		return Profile{}, false, fmt.Errorf("credit service unavailable")
	}
	return s.repo.GetProfile(ctx, retailerID, supplierID)
}

const defaultMaxDelinquencyTolerance = int64(2)

// UnlimitedPaymentsEnabled reports whether payment and credit limits are removed across the ecosystem.
func UnlimitedPaymentsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("UNLIMITED_PAYMENTS")))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	v2 := strings.ToLower(strings.TrimSpace(os.Getenv("AML_LIMITS_DISABLED")))
	return v2 == "1" || v2 == "true" || v2 == "yes"
}

// MaxDelinquencyTolerance returns the delinquency count threshold that triggers auto-freeze and blocks credit.
func MaxDelinquencyTolerance() int64 {
	raw := strings.TrimSpace(os.Getenv("CREDIT_MAX_DELINQUENCY_TOLERANCE"))
	if raw == "" {
		return defaultMaxDelinquencyTolerance
	}
	var n int64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return defaultMaxDelinquencyTolerance
		}
		n = n*10 + int64(c-'0')
	}
	if n <= 0 {
		return defaultMaxDelinquencyTolerance
	}
	return n
}

func (s *Service) emitProfileChanged(ctx context.Context, txn outbox.TxnBuffer, p Profile, reason string) error {
	ts := p.UpdatedAt
	if ts.IsZero() {
		ts = s.now()
	}
	return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, p.RetailerID, events.TopicMain, events.CreditProfileEvent{
		BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: ts.Format(time.RFC3339Nano)},
		ProfileID:        profileID(p.RetailerID, p.SupplierID),
		RetailerID:       p.RetailerID,
		SupplierID:       p.SupplierID,
		CreditLimitMinor: p.CreditLimitMinor,
		CurrentBalance:   p.CurrentBalanceMinor,
		Delinquent:       p.DelinquencyCount > 0,
		Reason:           reason,
	})
}

func (s *Service) emitHoldEvent(ctx context.Context, txn outbox.TxnBuffer, p Profile, eventType, action, reason, actorID string) error {
	if eventType == "" {
		return nil
	}
	ts := p.UpdatedAt
	if ts.IsZero() {
		ts = s.now()
	}
	tsStr := ts.UTC().Format(time.RFC3339Nano)
	return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, p.RetailerID, events.TopicMain, events.CreditHoldEvent{
		BaseEvent: events.BaseEvent{
			Type:      eventType,
			Version:   1,
			Timestamp: tsStr,
		},
		RetailerID:       p.RetailerID,
		SupplierID:       p.SupplierID,
		Action:           action,
		Status:           string(p.Status),
		PrepayOnlyLocked: p.PrepayOnlyLocked,
		Score:            p.RiskScore,
		RiskTier:         string(p.RiskTier),
		Reason:           reason,
		ActorID:          actorID,
		OccurredAt:       tsStr,
	})
}

func (s *Service) mutateProfileAtomic(
	ctx context.Context,
	retailerID, supplierID string,
	reason string,
	mutate func(existing *Profile) (Profile, error),
) (Profile, error) {
	return s.mutateProfileAtomicWithEvents(ctx, retailerID, supplierID, mutate, func(txn outbox.TxnBuffer, p Profile) error {
		return s.emitProfileChanged(ctx, txn, p, reason)
	})
}

func (s *Service) mutateProfileAtomicWithEvents(
	ctx context.Context,
	retailerID, supplierID string,
	mutate func(existing *Profile) (Profile, error),
	emit func(outbox.TxnBuffer, Profile) error,
) (Profile, error) {
	if atomicRepo, ok := s.repo.(AtomicProfileMutator); ok {
		return atomicRepo.MutateProfileAtomic(ctx, retailerID, supplierID, mutate, emit)
	}
	s.mutMu.Lock()
	defer s.mutMu.Unlock()
	existing, found, err := s.repo.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return Profile{}, err
	}
	var existingPtr *Profile
	if found {
		existingPtr = &existing
	}
	next, err := mutate(existingPtr)
	if err != nil {
		if errors.Is(err, errProfileNoop) && existingPtr != nil {
			return *existingPtr, nil
		}
		return Profile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, next, func(txn outbox.TxnBuffer) error {
		if emit != nil {
			return emit(txn, next)
		}
		return nil
	}); err != nil {
		return Profile{}, err
	}
	return next, nil
}

// PatchSupplierProfile atomically updates credit limit (preserving existing when limitMinor == nil) and/or status.
func (s *Service) PatchSupplierProfile(
	ctx context.Context,
	supplierID, retailerID string,
	limitMinor *int64,
	status Status,
	actorID, reason string,
) (Profile, error) {
	return s.patchSupplierProfile(ctx, supplierID, retailerID, limitMinor, status, nil, actorID, reason)
}

// SetPrepayOnlyLocked atomically toggles PrepayOnlyLocked on a retailer's credit profile.
func (s *Service) SetPrepayOnlyLocked(
	ctx context.Context,
	supplierID, retailerID string,
	locked bool,
	actorID, reason string,
) (Profile, error) {
	if strings.TrimSpace(reason) == "" {
		if locked {
			reason = "prepay_only_locked"
		} else {
			reason = "prepay_only_unlocked"
		}
	}
	return s.patchSupplierProfile(ctx, supplierID, retailerID, nil, "", &locked, actorID, reason)
}

func (s *Service) patchSupplierProfile(
	ctx context.Context,
	supplierID, retailerID string,
	limitMinor *int64,
	status Status,
	prepayOnlyLocked *bool,
	actorID, reason string,
) (Profile, error) {
	if s == nil || s.repo == nil {
		return Profile{}, fmt.Errorf("credit service unavailable")
	}
	now := s.now()
	var holdEventType, holdAction string
	return s.mutateProfileAtomicWithEvents(ctx, retailerID, supplierID, func(existing *Profile) (Profile, error) {
		holdEventType, holdAction = "", ""
		p := Profile{
			RetailerID: retailerID,
			SupplierID: supplierID,
			Status:     StatusActive,
			CreatedAt:  now,
		}
		var prevStatus Status
		var prevLocked bool
		if existing != nil {
			p = *existing
			prevStatus = existing.Status
			prevLocked = existing.PrepayOnlyLocked
		}
		if limitMinor != nil {
			p.CreditLimitMinor = *limitMinor
		}
		if status != "" {
			p.Status = status
		}
		if prepayOnlyLocked != nil {
			p.PrepayOnlyLocked = *prepayOnlyLocked
		}
		if p.Status == StatusFrozen && prevStatus != StatusFrozen {
			holdEventType = events.EventCreditHoldApplied
			holdAction = "FREEZE"
		} else if prevStatus == StatusFrozen && p.Status == StatusActive {
			holdEventType = events.EventCreditHoldReleased
			holdAction = "UNFREEZE"
		} else if p.PrepayOnlyLocked && !prevLocked {
			holdEventType = events.EventCreditHoldApplied
			holdAction = "PREPAY_LOCK"
		} else if !p.PrepayOnlyLocked && prevLocked {
			holdEventType = events.EventCreditHoldReleased
			holdAction = "PREPAY_UNLOCK"
		}
		p.RiskTier = ""
		p.UpdatedAt = now
		if p.LastEvaluatedAt.IsZero() {
			p.LastEvaluatedAt = now
		}
		p.AvailableCreditMinor = p.Available()
		return p, nil
	}, func(txn outbox.TxnBuffer, p Profile) error {
		if err := s.emitProfileChanged(ctx, txn, p, reason); err != nil {
			return err
		}
		return s.emitHoldEvent(ctx, txn, p, holdEventType, holdAction, reason, actorID)
	})
}

// BumpDelinquency increments DelinquencyCount by 1 for collections (first OVERDUE).
// When delinquency reaches or exceeds tolerance, it automatically sets the profile to FROZEN.
func (s *Service) BumpDelinquency(ctx context.Context, retailerID, supplierID string) error {
	now := s.now()
	tolerance := MaxDelinquencyTolerance()
	var autoFrozen bool
	_, err := s.mutateProfileAtomicWithEvents(ctx, retailerID, supplierID, func(existing *Profile) (Profile, error) {
		autoFrozen = false
		if existing == nil {
			return Profile{}, errProfileNoop
		}
		p := *existing
		p.DelinquencyCount++
		p.LastEvaluatedAt = now
		p.UpdatedAt = now
		if p.DelinquencyCount >= tolerance && p.Status == StatusActive {
			p.Status = StatusFrozen
			autoFrozen = true
		}
		return p, nil
	}, func(txn outbox.TxnBuffer, p Profile) error {
		if err := s.emitProfileChanged(ctx, txn, p, "delinquency_bump"); err != nil {
			return err
		}
		if autoFrozen {
			return s.emitHoldEvent(ctx, txn, p, events.EventCreditHoldApplied, "DELINQUENCY_FREEZE", "delinquency_bump", "system")
		}
		return nil
	})
	return err
}

// FreezeProfile transitions a profile to FROZEN with reason.
func (s *Service) FreezeProfile(ctx context.Context, retailerID, supplierID, reason string) error {
	now := s.now()
	_, err := s.mutateProfileAtomicWithEvents(ctx, retailerID, supplierID, func(existing *Profile) (Profile, error) {
		if existing == nil {
			return Profile{}, ErrProfileNotFound
		}
		if existing.Status == StatusFrozen {
			return Profile{}, errProfileNoop
		}
		p := *existing
		p.Status = StatusFrozen
		p.LastEvaluatedAt = now
		p.UpdatedAt = now
		return p, nil
	}, func(txn outbox.TxnBuffer, p Profile) error {
		if err := s.emitProfileChanged(ctx, txn, p, reason); err != nil {
			return err
		}
		return s.emitHoldEvent(ctx, txn, p, events.EventCreditHoldApplied, "FREEZE", reason, "system")
	})
	return err
}

// UnfreezeProfile transitions a profile back to ACTIVE from FROZEN.
func (s *Service) UnfreezeProfile(ctx context.Context, retailerID, supplierID, reason string) error {
	now := s.now()
	_, err := s.mutateProfileAtomicWithEvents(ctx, retailerID, supplierID, func(existing *Profile) (Profile, error) {
		if existing == nil {
			return Profile{}, ErrProfileNotFound
		}
		if existing.Status != StatusFrozen {
			return Profile{}, errProfileNoop
		}
		p := *existing
		p.Status = StatusActive
		p.LastEvaluatedAt = now
		p.UpdatedAt = now
		return p, nil
	}, func(txn outbox.TxnBuffer, p Profile) error {
		if err := s.emitProfileChanged(ctx, txn, p, reason); err != nil {
			return err
		}
		return s.emitHoldEvent(ctx, txn, p, events.EventCreditHoldReleased, "UNFREEZE", reason, "system")
	})
	return err
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
		if err := s.emitProfileChanged(ctx, txn, p, reason); err != nil {
			return err
		}
		if p.Status == StatusFrozen && reason == "hold" {
			return s.emitHoldEvent(ctx, txn, p, events.EventCreditHoldApplied, "HOLD", reason, actorID)
		}
		if p.Status == StatusActive && reason == "unhold" {
			return s.emitHoldEvent(ctx, txn, p, events.EventCreditHoldReleased, "UNHOLD", reason, actorID)
		}
		return nil
	})
}

// ListSupplierProfiles returns supplier-scoped credit profiles for the collections desk.
func (s *Service) ListSupplierProfiles(ctx context.Context, supplierID, status string, limit int) ([]Profile, error) {
	list, _, err := s.ListSupplierProfilesPage(ctx, supplierID, status, limit, "")
	return list, err
}

// ListSupplierProfilesPage returns keyset-paginated supplier-scoped credit profiles for the collections desk.
func (s *Service) ListSupplierProfilesPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Profile, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", fmt.Errorf("credit service unavailable")
	}
	var list []Profile
	var nextCursor string
	var err error
	if pager, ok := s.repo.(ProfilePager); ok {
		list, nextCursor, err = pager.ListBySupplierPage(ctx, supplierID, status, limit, cursor)
	} else {
		if strings.TrimSpace(cursor) != "" {
			return nil, "", ErrInvalidCursor
		}
		list, err = s.repo.ListBySupplier(ctx, supplierID, status, limit)
	}
	if err != nil {
		return nil, "", err
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
	return list, nextCursor, nil
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
	// Priority: BLACKLISTED/FROZEN, then lower risk score (worse, when both > 0), then balance desc.
	slices.SortStableFunc(list, func(a, b Profile) int {
		if c := cmp.Compare(profilePriority(a), profilePriority(b)); c != 0 {
			return c
		}
		if a.RiskScore > 0 && b.RiskScore > 0 {
			if c := cmp.Compare(a.RiskScore, b.RiskScore); c != 0 {
				return c
			}
		}
		return cmp.Compare(b.CurrentBalanceMinor, a.CurrentBalanceMinor)
	})
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
