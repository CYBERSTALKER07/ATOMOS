package gl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Approval errors returned by ManualApprovalBridge.
var (
	ErrApprovalRequired        = errors.New("gl: manual journal approval required")
	ErrApprovalPending         = errors.New("gl: approval request is not approved")
	ErrApprovalNotFound        = errors.New("gl: approval request not found")
	ErrApprovalPayloadMismatch = errors.New("gl: approval payload hash mismatch")
)

// ManualApprovalResult reports CEL approval evaluation or verification for manual journals.
type ManualApprovalResult struct {
	Required          bool   `json:"required"`
	ApprovalRequestID string `json:"approval_request_id,omitempty"`
	PolicyID          string `json:"policy_id,omitempty"`
	PolicyVersion     int64  `json:"policy_version,omitempty"`
	PayloadHash       string `json:"payload_hash,omitempty"`
	Status            string `json:"status,omitempty"`
}

// ManualApprovalBridge evaluates CEL policies for high-value manual entries
// and verifies approved requests with payload hash integrity.
type ManualApprovalBridge interface {
	EvaluateOrVerifyManualJournal(
		ctx context.Context,
		entityID, actorID, actorType, actorRole, approvalID string,
		amountMinor int64,
		in JournalEntryInput,
		canonicalPayload map[string]any,
	) (ManualApprovalResult, error)
}

// LegalEntityOwnershipChecker validates whether a LegalEntityId belongs to a tenant.
type LegalEntityOwnershipChecker interface {
	IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error)
}

// Service orchestrates GL journal postings, trial balances, reversals, and backfills.
type Service struct {
	repo                   Repository
	approvalBridge         ManualApprovalBridge
	mutationContextHook    func(ctx context.Context, tenantID, action, objectType, objectID, approvalID string)
	entityOwnershipChecker LegalEntityOwnershipChecker
}

// SetApprovalBridge wires the M1.2 CEL Approval Engine bridge for manual journal gating.
func (s *Service) SetApprovalBridge(b ManualApprovalBridge) {
	if s != nil {
		s.approvalBridge = b
	}
}

// SetLegalEntityOwnershipChecker attaches an optional legal-entity ownership verifier.
func (s *Service) SetLegalEntityOwnershipChecker(c LegalEntityOwnershipChecker) {
	if s != nil {
		s.entityOwnershipChecker = c
	}
}

// SetMutationContextHook attaches an optional audit.SetMutationContext callback without importing audit.
func (s *Service) SetMutationContextHook(fn func(ctx context.Context, tenantID, action, objectType, objectID, approvalID string)) {
	if s != nil {
		s.mutationContextHook = fn
	}
}

func (s *Service) setMutationContext(ctx context.Context, tenantID, action, objectType, objectID, approvalID string) {
	if s != nil && s.mutationContextHook != nil {
		s.mutationContextHook(ctx, tenantID, action, objectType, objectID, approvalID)
	}
}

// NewService creates a Spanner-backed GL Service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{}
	}
	return &Service{repo: NewSpannerRepository(client)}
}

// NewServiceWithRepository creates a GL Service using a custom Repository (e.g., for testing).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo}
}

// Repo returns the configured Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// PostEntry validates and appends one balanced journal entry.
func (s *Service) PostEntry(ctx context.Context, input JournalEntryInput) (JournalEntryResult, error) {
	if s == nil || s.repo == nil {
		return JournalEntryResult{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.PostEntry(ctx, input)
}

// PostJournal validates and appends a multi-entity-aware journal entry via PostInput.
func (s *Service) PostJournal(ctx context.Context, input PostInput) (JournalEntryResult, error) {
	if s == nil || s.repo == nil {
		return JournalEntryResult{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.PostEntry(ctx, input.ToJournalEntryInput())
}

// PostJournalInTxn validates and buffers a PostInput inside an open Spanner ReadWriteTransaction.
func (s *Service) PostJournalInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, input PostInput) (JournalEntryResult, error) {
	return BufferJournalEntryInTxn(ctx, txn, outboxBuf, input.ToJournalEntryInput())
}

// PostEvent evaluates PostingRules for eventType and appends the balanced journal entry.
func (s *Service) PostEvent(ctx context.Context, eventType string, ev RuleEvalContext) (JournalEntryResult, error) {
	if s == nil || s.repo == nil {
		return JournalEntryResult{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.PostEvent(ctx, eventType, ev)
}

// ReverseEntry appends a balanced inverse entry referencing entryID.
func (s *Service) ReverseEntry(ctx context.Context, entityID, entryID, idempotencyKey, reason, actor string) (JournalEntryResult, error) {
	if s == nil || s.repo == nil {
		return JournalEntryResult{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.ReverseEntry(ctx, entityID, entryID, idempotencyKey, reason, actor)
}

// GetEntry returns one JournalEntries record with its ordered postings.
func (s *Service) GetEntry(ctx context.Context, entityID, entryID string) (JournalEntryRecord, error) {
	if s == nil || s.repo == nil {
		return JournalEntryRecord{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.GetEntry(ctx, entityID, entryID)
}

// TrialBalance returns the period-scoped Trial Balance for entityID.
func (s *Service) TrialBalance(ctx context.Context, entityID, periodID string) (TrialBalanceReport, error) {
	if s == nil || s.repo == nil {
		return TrialBalanceReport{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.GetTrialBalance(ctx, entityID, periodID)
}

// TrialBalanceForEntity returns the period-scoped Trial Balance for tenantID, optionally filtered by legalEntityID.
func (s *Service) TrialBalanceForEntity(ctx context.Context, tenantID, periodID, legalEntityID string) (TrialBalanceReport, error) {
	if s == nil || s.repo == nil {
		return TrialBalanceReport{}, fmt.Errorf("gl service unavailable")
	}
	if strings.TrimSpace(legalEntityID) == "" || strings.TrimSpace(legalEntityID) == strings.TrimSpace(tenantID) {
		return s.repo.GetTrialBalance(ctx, tenantID, periodID)
	}
	if me, ok := s.repo.(MultiEntityJournalRepository); ok {
		return me.GetTrialBalanceFiltered(ctx, tenantID, periodID, legalEntityID)
	}
	return TrialBalanceReport{}, fmt.Errorf("gl repository does not support legal entity filtering")
}

// ListJournals returns journal entries matching filter.
func (s *Service) ListJournals(ctx context.Context, filter JournalFilter) ([]JournalEntryRecord, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("gl service unavailable")
	}
	if me, ok := s.repo.(MultiEntityJournalRepository); ok {
		return me.ListJournals(ctx, filter)
	}
	return nil, fmt.Errorf("gl repository does not support journal listing")
}

// IsLegalEntityOwned reports whether entityID belongs to tenantID.
func (s *Service) IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	if tenantID == "" || entityID == "" {
		return false, nil
	}
	if tenantID == entityID {
		return true, nil
	}
	if s.entityOwnershipChecker != nil {
		return s.entityOwnershipChecker.IsLegalEntityOwned(ctx, tenantID, entityID)
	}
	if me, ok := s.repo.(MultiEntityJournalRepository); ok {
		return me.IsLegalEntityOwned(ctx, tenantID, entityID)
	}
	return false, nil
}

// AccountLedger returns postings for accountCode scoped to entityID and optional periodID.
func (s *Service) AccountLedger(ctx context.Context, entityID, accountCode, periodID string, limit int) ([]AccountLedgerRow, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("gl service unavailable")
	}
	return s.repo.GetAccountLedger(ctx, entityID, accountCode, periodID, limit)
}

// ListPostingRules lists active posting rules for countryPack and optional eventType.
func (s *Service) ListPostingRules(ctx context.Context, countryPack, eventType string) ([]PostingRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("gl service unavailable")
	}
	return s.repo.ListPostingRules(ctx, countryPack, eventType)
}

// SeedDefaults idempotently seeds default GlAccounts for entityID and PostingRules for UZ & KZ.
func (s *Service) SeedDefaults(ctx context.Context, entityID, countryPack, currency string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("gl service unavailable")
	}
	if strings.TrimSpace(entityID) != "" {
		if _, err := s.repo.SeedDefaultAccounts(ctx, entityID, countryPack, currency); err != nil {
			return err
		}
	}
	if _, err := s.repo.SeedPostingRules(ctx, "UZ"); err != nil {
		return err
	}
	if _, err := s.repo.SeedPostingRules(ctx, "KZ"); err != nil {
		return err
	}
	return nil
}

// BackfillFromPaymentLedgerEntries backfills PaymentLedgerEntries rows for entityID into GL.
func (s *Service) BackfillFromPaymentLedgerEntries(ctx context.Context, entityID string, opts BackfillOptions) (BackfillReport, error) {
	if s == nil || s.repo == nil {
		return BackfillReport{}, fmt.Errorf("gl service unavailable")
	}
	return s.repo.BackfillFromPaymentLedgerEntries(ctx, entityID, opts)
}
