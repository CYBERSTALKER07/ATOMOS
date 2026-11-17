package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DraftTypeAdjustment = "ADJUSTMENT"
	DraftTypeCreditNote = "CREDIT_NOTE"

	draftStatusPendingApproval = "PENDING_APPROVAL"
	draftStatusApproved        = "APPROVED"
	draftStatusRejected        = "REJECTED"

	draftTableAdjustments = "DraftAdjustments"
	draftTableCreditNotes = "DraftCreditNotes"
)

var (
	// ErrSelfReviewForbidden indicates a four-eyes Segregation of Duties violation.
	ErrSelfReviewForbidden = errors.New("segregation of duties violation: operator cannot review own draft")
	// ErrDraftNotFound indicates no matching draft exists in DraftAdjustments or DraftCreditNotes.
	ErrDraftNotFound = errors.New("draft_not_found")
	// ErrDraftAlreadyReviewed indicates a draft is no longer in PENDING_APPROVAL state.
	ErrDraftAlreadyReviewed = errors.New("draft_already_reviewed")
	// ErrDraftValidation indicates malformed or missing draft request parameters.
	ErrDraftValidation = errors.New("invalid_draft_request")
	// ErrFinancialDraftsUnavailable indicates required repository or Spanner dependencies are not configured.
	ErrFinancialDraftsUnavailable = errors.New("financial_drafts_unavailable")
	// ErrDraftPayloadMismatch indicates the draft payload changed relative to the pinned ApprovalRequest hash.
	ErrDraftPayloadMismatch = errors.New("payload_hash_mismatch")
	// ErrDraftApprovalForbidden indicates an approval gate rejected the reviewer/state (e.g. missing MFA or unapproved).
	ErrDraftApprovalForbidden = errors.New("draft_approval_forbidden")
)

// DraftApprovalResult reports M1.2 CEL approval policy evaluation on high-value financial drafts.
type DraftApprovalResult struct {
	Required          bool
	Approved          bool
	ApprovalRequestID string
	PolicyID          string
	PolicyVersion     int64
	PayloadHash       string
}

// DraftApprovalBridge integrates order financial drafts/credit notes with M1.2 CEL approvals.
type DraftApprovalBridge interface {
	EvaluateOrVerifyFinancialDraft(
		ctx context.Context,
		tenantID, draftID, draftType, action, actorID, actorType, actorRole, approvalID string,
		amountMinor int64,
		canonPayload map[string]any,
	) (DraftApprovalResult, error)
}

// SetDraftApprovalBridge attaches an optional M1.2 CEL Approval Engine bridge for financial drafts.
func (s *Service) SetDraftApprovalBridge(bridge DraftApprovalBridge) {
	if s != nil {
		s.draftApprovalBridge = bridge
	}
}

type draftSentinelError struct {
	primary   error
	secondary error
	cause     error
	detail    string
}

func (e draftSentinelError) Error() string {
	if e.detail != "" {
		return e.primary.Error() + ": " + e.detail
	}
	return e.primary.Error()
}

func (e draftSentinelError) Unwrap() []error {
	errs := []error{e.primary}
	if e.secondary != nil {
		errs = append(errs, e.secondary)
	}
	if e.cause != nil {
		errs = append(errs, e.cause)
	}
	return errs
}

// --- Request / Response types ---

type CreateDraftRequest struct {
	OrderID             string `json:"orderId"`
	DraftType           string `json:"draftType,omitempty"`
	ProposedAmountMinor int64  `json:"proposedAmountMinor"`
	// ProposedAmount is retained as a legacy int64 alias for callers not yet migrated to proposedAmountMinor.
	ProposedAmount int64  `json:"proposedAmount,omitempty"`
	ReasonCode     string `json:"reasonCode"`
	Notes          string `json:"notes"`
}

type ReviewDraftRequest struct {
	DraftID    string `json:"draftId"`
	DraftType  string `json:"draftType,omitempty"`
	Action     string `json:"action"` // APPROVE, REJECT
	ApprovalID string `json:"approvalId,omitempty"`
}

type DraftResponse struct {
	DraftID             string `json:"draftId,omitempty"`
	DraftType           string `json:"draftType,omitempty"`
	Status              string `json:"status"`
	ProposedAmountMinor int64  `json:"proposedAmountMinor,omitempty"`
	ApprovalRequestID   string `json:"approvalRequestId,omitempty"`
	PayloadHash         string `json:"payloadHash,omitempty"`
	Message             string `json:"message,omitempty"`
}

// DraftSoDCheckInput carries financial draft maker-checker context for iam.Service.CheckFourEyes.
type DraftSoDCheckInput struct {
	TenantID    string
	ActorID     string
	ActorType   string
	Role        string
	Claims      *auth.Claims
	Action      string
	MakerAction string
	ObjectType  string
	ObjectID    string
	MakerID     string
	RoutePath   string
	IsApproval  bool
}

// --- Shared Validation / Scoping / SoD Helpers ---

func normalizeDraftType(raw, defaultType string) (string, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	if trimmed == "" {
		return defaultType, nil
	}
	switch trimmed {
	case DraftTypeAdjustment, "DRAFT_ADJUSTMENT":
		return DraftTypeAdjustment, nil
	case DraftTypeCreditNote, "CREDITNOTE", "DRAFT_CREDIT_NOTE":
		return DraftTypeCreditNote, nil
	default:
		return "", draftSentinelError{
			primary: ErrDraftValidation,
			detail:  fmt.Sprintf("unsupported draftType %q", raw),
		}
	}
}

func resolveProposedAmountMinor(req CreateDraftRequest, draftType string) (int64, error) {
	if req.ProposedAmountMinor != 0 && req.ProposedAmount != 0 && req.ProposedAmountMinor != req.ProposedAmount {
		return 0, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "conflicting proposedAmountMinor and proposedAmount",
		}
	}
	amountMinor := req.ProposedAmountMinor
	if amountMinor == 0 {
		amountMinor = req.ProposedAmount
	}
	if amountMinor == 0 {
		return 0, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "proposedAmountMinor must be non-zero",
		}
	}
	if draftType == DraftTypeCreditNote && amountMinor <= 0 {
		return 0, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "credit note proposedAmountMinor must be positive",
		}
	}
	return amountMinor, nil
}

func resolveDraftAdminSupplierID(ctx context.Context, claims auth.Claims) (string, error) {
	claimSupplier := strings.TrimSpace(claims.SupplierID)
	tenant, hasTenant := auth.TenantFromContext(ctx)
	tenantSupplier := strings.TrimSpace(tenant.SupplierID)

	if hasTenant && claimSupplier != "" && tenantSupplier != claimSupplier {
		return "", draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "supplier tenant mismatch",
		}
	}
	resolved := claimSupplier
	if resolved == "" {
		resolved = tenantSupplier
	}
	if resolved == "" {
		return "", draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "supplier scope required",
		}
	}
	return resolved, nil
}

func validateDraftCreateScope(ctx context.Context, claims auth.Claims, ord Order) error {
	switch claims.Role {
	case auth.RoleAdmin:
		callerSupplier, err := resolveDraftAdminSupplierID(ctx, claims)
		if err != nil {
			return err
		}
		if callerSupplier != strings.TrimSpace(ord.SupplierID) {
			return draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "order belongs to another supplier",
			}
		}
		return nil

	case auth.RoleRetailer:
		retailerOrg := strings.TrimSpace(auth.ResolveRetailerOrgID(claims))
		if retailerOrg == "" || retailerOrg != strings.TrimSpace(ord.RetailerID) {
			return draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "order belongs to another retailer",
			}
		}
		claimSupplier := strings.TrimSpace(claims.SupplierID)
		tenant, hasTenant := auth.TenantFromContext(ctx)
		tenantSupplier := strings.TrimSpace(tenant.SupplierID)
		if hasTenant && claimSupplier != "" && tenantSupplier != claimSupplier {
			return draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "supplier tenant mismatch",
			}
		}
		if claimSupplier != "" && claimSupplier != strings.TrimSpace(ord.SupplierID) {
			return draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "order belongs to another supplier",
			}
		}
		if tenantSupplier != "" && tenantSupplier != strings.TrimSpace(ord.SupplierID) {
			return draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "order belongs to another supplier",
			}
		}
		return nil

	default:
		return draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "role not permitted to create draft",
		}
	}
}

func validateDraftReviewState(callerSupplierID, reviewerID, existingSupplierID, existingOperatorID, existingStatus string) error {
	if strings.TrimSpace(reviewerID) == "" {
		return draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "reviewer subject required",
		}
	}
	if strings.TrimSpace(callerSupplierID) == "" {
		return draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "supplier scope required",
		}
	}
	if strings.TrimSpace(existingSupplierID) == "" || strings.TrimSpace(existingSupplierID) != strings.TrimSpace(callerSupplierID) {
		return draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "draft belongs to another supplier",
		}
	}

	operatorID := strings.TrimSpace(existingOperatorID)
	if operatorID == "" {
		return draftSentinelError{
			primary:   ErrSelfReviewForbidden,
			secondary: ErrOrderForbidden,
			detail:    "draft operator_id is empty",
		}
	}
	if operatorID == strings.TrimSpace(reviewerID) {
		return draftSentinelError{
			primary:   ErrSelfReviewForbidden,
			secondary: ErrOrderForbidden,
			detail:    "operator cannot approve or reject their own draft",
		}
	}

	if existingStatus != draftStatusPendingApproval {
		return draftSentinelError{
			primary: ErrDraftAlreadyReviewed,
			detail:  fmt.Sprintf("draft already %s; cannot review", existingStatus),
		}
	}
	return nil
}

func isSpannerNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, iterator.Done) || spanner.ErrCode(err) == codes.NotFound || status.Code(err) == codes.NotFound {
		return true
	}
	return false
}

// --- Domain Logic ---

// CreateFinancialDraft persists a pending adjustment or credit-note draft with int64 minor units.
func (s *Service) CreateFinancialDraft(ctx context.Context, claims auth.Claims, req CreateDraftRequest) (DraftResponse, error) {
	return s.createFinancialDraftWithDefaultType(ctx, claims, req, DraftTypeAdjustment, false)
}

// CreateDraftCreditNote persists a pending DraftCreditNotes row with int64 minor units.
func (s *Service) CreateDraftCreditNote(ctx context.Context, claims auth.Claims, req CreateDraftRequest) (DraftResponse, error) {
	return s.createFinancialDraftWithDefaultType(ctx, claims, req, DraftTypeCreditNote, true)
}

func (s *Service) createFinancialDraftWithDefaultType(ctx context.Context, claims auth.Claims, req CreateDraftRequest, defaultType string, enforceDefaultType bool) (DraftResponse, error) {
	if auth.IsReadOnlyAuditor(claims) {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "AUDITOR role is strictly read-only",
		}
	}

	operatorID := strings.TrimSpace(claims.Subject)
	if operatorID == "" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "operator subject required",
		}
	}

	draftType, err := normalizeDraftType(req.DraftType, defaultType)
	if err != nil {
		return DraftResponse{}, err
	}
	if enforceDefaultType && draftType != defaultType {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  fmt.Sprintf("draftType must be %s", defaultType),
		}
	}
	if claims.Role == auth.RoleAdmin && (claims.SupplierRole != "" || len(claims.SupplierPerms) > 0) {
		if draftType == DraftTypeCreditNote && !auth.HasSupplierPerm(claims, auth.PermCreditNotePropose) {
			return DraftResponse{}, draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "missing_permission: credit_note.propose",
			}
		}
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "order_id required",
		}
	}
	reasonCode := strings.TrimSpace(req.ReasonCode)
	if reasonCode == "" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "reason_code required",
		}
	}
	amountMinor, err := resolveProposedAmountMinor(req, draftType)
	if err != nil {
		return DraftResponse{}, err
	}

	if s == nil || s.repo == nil || s.spannerClient == nil {
		return DraftResponse{}, ErrFinancialDraftsUnavailable
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return DraftResponse{}, err
	}
	if !ok {
		return DraftResponse{}, ErrOrderNotFound
	}

	if err := validateDraftCreateScope(ctx, claims, current); err != nil {
		return DraftResponse{}, err
	}

	// Mutability Boundary: pre-dispatch orders should be edited directly.
	if current.Status == StatusPending || current.Status == StatusLoaded || current.Status == StatusScheduled {
		return DraftResponse{}, errors.New("order has not crossed the mutability boundary; edit it directly")
	}

	draftID := uuid.NewString()
	var mut *spanner.Mutation
	if draftType == DraftTypeCreditNote {
		if strings.TrimSpace(current.RetailerID) == "" {
			return DraftResponse{}, draftSentinelError{
				primary: ErrDraftValidation,
				detail:  "order retailer_id required for credit note draft",
			}
		}
		mut = spanner.Insert(draftTableCreditNotes,
			[]string{"DraftId", "SupplierId", "RetailerId", "OrderId", "ProposedAmountMinor", "ReasonCode", "OperatorId", "Status", "CreatedAt"},
			[]any{draftID, current.SupplierID, current.RetailerID, current.OrderID, amountMinor, reasonCode, operatorID, draftStatusPendingApproval, spanner.CommitTimestamp},
		)
	} else {
		mut = spanner.Insert(draftTableAdjustments,
			[]string{"DraftId", "SupplierId", "OrderId", "ProposedAmountMinor", "ReasonCode", "OperatorId", "Status", "CreatedAt"},
			[]any{draftID, current.SupplierID, current.OrderID, amountMinor, reasonCode, operatorID, draftStatusPendingApproval, spanner.CommitTimestamp},
		)
	}

	_, err = s.spannerClient.Apply(ctx, []*spanner.Mutation{mut})
	if err != nil {
		return DraftResponse{}, fmt.Errorf("create draft: %w", err)
	}
	return DraftResponse{
		DraftID:             draftID,
		DraftType:           draftType,
		Status:              draftStatusPendingApproval,
		ProposedAmountMinor: amountMinor,
	}, nil
}

type draftTableTarget struct {
	tableName string
	draftType string
}

// ReviewFinancialDraft approves or rejects a draft inside a guarded RW transaction,
// enforcing four-eyes Segregation of Duties (reviewer != operator) and supplier scoping.
// When DraftType is omitted, both DraftAdjustments and DraftCreditNotes are checked so
// credit-note drafts cannot bypass Segregation of Duties via the generic review endpoint.
func (s *Service) ReviewFinancialDraft(ctx context.Context, claims auth.Claims, req ReviewDraftRequest) (DraftResponse, error) {
	return s.reviewFinancialDraftWithDefaultType(ctx, claims, req, "", false)
}

// ReviewDraftCreditNote approves or rejects a DraftCreditNotes row inside a guarded RW transaction.
func (s *Service) ReviewDraftCreditNote(ctx context.Context, claims auth.Claims, req ReviewDraftRequest) (DraftResponse, error) {
	return s.reviewFinancialDraftWithDefaultType(ctx, claims, req, DraftTypeCreditNote, true)
}

func (s *Service) reviewFinancialDraftWithDefaultType(ctx context.Context, claims auth.Claims, req ReviewDraftRequest, defaultType string, enforceDefaultType bool) (DraftResponse, error) {
	if claims.Role != auth.RoleAdmin {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "unauthorized: finance/admin only",
		}
	}

	reviewerID := strings.TrimSpace(claims.Subject)
	if reviewerID == "" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "reviewer subject required",
		}
	}

	callerSupplierID, err := resolveDraftAdminSupplierID(ctx, claims)
	if err != nil {
		return DraftResponse{}, err
	}

	draftType, err := normalizeDraftType(req.DraftType, defaultType)
	if err != nil {
		return DraftResponse{}, err
	}
	if enforceDefaultType && draftType != defaultType {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  fmt.Sprintf("draftType must be %s", defaultType),
		}
	}

	requiredPerm := auth.PermDraftApprove
	if draftType == DraftTypeCreditNote {
		requiredPerm = auth.PermCreditNoteApprove
	}
	routePath := "/v1/finance/drafts/review"
	if draftType == DraftTypeCreditNote {
		routePath = "/v1/finance/credit-note-drafts/review"
	}

	if s != nil && s.draftSoDCheck != nil && (auth.IsReadOnlyAuditor(claims) || auth.IsAgentActor(claims)) {
		cp := claims
		_ = s.draftSoDCheck(ctx, DraftSoDCheckInput{
			TenantID:   callerSupplierID,
			ActorID:    reviewerID,
			ActorType:  auth.EffectiveActorType(claims),
			Role:       auth.EffectiveSupplierRole(claims),
			Claims:     &cp,
			Action:     requiredPerm,
			ObjectType: "FinancialDraft",
			ObjectID:   strings.TrimSpace(req.DraftID),
			RoutePath:  routePath,
			IsApproval: true,
		})
	}
	if auth.IsReadOnlyAuditor(claims) {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "AUDITOR role is strictly read-only",
		}
	}
	if auth.IsAgentActor(claims) {
		return DraftResponse{}, draftSentinelError{
			primary: ErrOrderForbidden,
			detail:  "AGENT actor cannot approve or reject drafts",
		}
	}
	switch draftType {
	case DraftTypeCreditNote:
		if !auth.HasSupplierPerm(claims, auth.PermCreditNoteApprove) {
			return DraftResponse{}, draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "missing_permission: credit_note.approve",
			}
		}
	case DraftTypeAdjustment:
		if !auth.HasSupplierPerm(claims, auth.PermDraftApprove) {
			return DraftResponse{}, draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "missing_permission: draft.approve",
			}
		}
	default:
		if !auth.HasAnySupplierPerm(claims, auth.PermDraftApprove, auth.PermCreditNoteApprove) {
			return DraftResponse{}, draftSentinelError{
				primary: ErrOrderForbidden,
				detail:  "missing_permission: draft.approve",
			}
		}
	}

	draftID := strings.TrimSpace(req.DraftID)
	if draftID == "" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "draft_id required",
		}
	}

	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action != "APPROVE" && action != "REJECT" {
		return DraftResponse{}, draftSentinelError{
			primary: ErrDraftValidation,
			detail:  "action must be APPROVE or REJECT",
		}
	}

	if s == nil || s.spannerClient == nil {
		return DraftResponse{}, ErrFinancialDraftsUnavailable
	}

	nextStatus := draftStatusApproved
	if action == "REJECT" {
		nextStatus = draftStatusRejected
	}

	var targets []draftTableTarget
	switch draftType {
	case DraftTypeAdjustment:
		targets = []draftTableTarget{{tableName: draftTableAdjustments, draftType: DraftTypeAdjustment}}
	case DraftTypeCreditNote:
		targets = []draftTableTarget{{tableName: draftTableCreditNotes, draftType: DraftTypeCreditNote}}
	default:
		targets = []draftTableTarget{
			{tableName: draftTableAdjustments, draftType: DraftTypeAdjustment},
			{tableName: draftTableCreditNotes, draftType: DraftTypeCreditNote},
		}
	}

	var resolvedDraftType string
	var resolvedAmountMinor int64
	var approvalRequestID, payloadHash string
	var observedSupplierID, observedOperatorID, observedDraftType string
	_, _, _, _ = observedSupplierID, observedOperatorID, observedDraftType, s.now

	if (s.draftApprovalBridge != nil && action == "APPROVE") || s.draftSoDCheck != nil {
		preRo := s.spannerClient.ReadOnlyTransaction()
		var preSupplierID, preOperatorID, preStatus, preDraftType string
		var preAmountMinor int64
		var preFound bool
		for _, target := range targets {
			row, readErr := preRo.ReadRow(ctx, target.tableName,
				spanner.Key{draftID},
				[]string{"DraftId", "SupplierId", "OperatorId", "Status", "ProposedAmountMinor"},
			)
			if readErr != nil {
				if isSpannerNotFound(readErr) {
					continue
				}
				preRo.Close()
				return DraftResponse{}, fmt.Errorf("read draft for approval check: %w", readErr)
			}
			var tmpID string
			if err := row.Columns(&tmpID, &preSupplierID, &preOperatorID, &preStatus, &preAmountMinor); err != nil {
				preRo.Close()
				return DraftResponse{}, fmt.Errorf("scan draft for approval check: %w", err)
			}
			preDraftType = target.draftType
			preFound = true
			break
		}
		preRo.Close()
		if preFound {
			if preDraftType == DraftTypeCreditNote && !auth.HasSupplierPerm(claims, auth.PermCreditNoteApprove) {
				return DraftResponse{}, draftSentinelError{
					primary: ErrOrderForbidden,
					detail:  "missing_permission: credit_note.approve",
				}
			}
			if preDraftType == DraftTypeAdjustment && !auth.HasSupplierPerm(claims, auth.PermDraftApprove) {
				return DraftResponse{}, draftSentinelError{
					primary: ErrOrderForbidden,
					detail:  "missing_permission: draft.approve",
				}
			}
			if err := validateDraftReviewState(callerSupplierID, reviewerID, preSupplierID, preOperatorID, preStatus); err != nil {
				if errors.Is(err, ErrSelfReviewForbidden) && s.draftSoDCheck != nil {
					cp := claims
					checkPerm, makerPerm := auth.PermDraftApprove, auth.PermSupplierOrderCancel
					if preDraftType == DraftTypeCreditNote {
						checkPerm, makerPerm = auth.PermCreditNoteApprove, auth.PermCreditNotePropose
					}
					_ = s.draftSoDCheck(ctx, DraftSoDCheckInput{
						TenantID: firstNonEmptyString(preSupplierID, callerSupplierID), ActorID: reviewerID,
						ActorType: auth.EffectiveActorType(claims), Role: auth.EffectiveSupplierRole(claims),
						Claims: &cp, Action: checkPerm, MakerAction: makerPerm,
						ObjectType: preDraftType, ObjectID: draftID,
						MakerID: strings.TrimSpace(preOperatorID), RoutePath: routePath, IsApproval: true,
					})
				}
				return DraftResponse{}, err
			}
			// Enforce SoD policy BEFORE opening or committing any Spanner write transaction.
			if s.draftSoDCheck != nil {
				cp := claims
				checkPerm, makerPerm := auth.PermDraftApprove, auth.PermSupplierOrderCancel
				if preDraftType == DraftTypeCreditNote {
					checkPerm, makerPerm = auth.PermCreditNoteApprove, auth.PermCreditNotePropose
				}
				if sodErr := s.draftSoDCheck(ctx, DraftSoDCheckInput{
					TenantID: firstNonEmptyString(preSupplierID, callerSupplierID), ActorID: reviewerID,
					ActorType: auth.EffectiveActorType(claims), Role: auth.EffectiveSupplierRole(claims),
					Claims: &cp, Action: checkPerm, MakerAction: makerPerm,
					ObjectType: preDraftType, ObjectID: draftID,
					MakerID: strings.TrimSpace(preOperatorID), RoutePath: routePath, IsApproval: true,
				}); sodErr != nil {
					return DraftResponse{}, draftSentinelError{
						primary:   ErrSelfReviewForbidden,
						secondary: ErrOrderForbidden,
						cause:     sodErr,
						detail:    sodErr.Error(),
					}
				}
			}
			if s.draftApprovalBridge != nil && action == "APPROVE" {
				canonPayload := map[string]any{
					"draft_id":              draftID,
					"draft_type":            preDraftType,
					"supplier_id":           preSupplierID,
					"proposed_amount_minor": preAmountMinor,
					"action":                action,
				}
				appRes, bErr := s.draftApprovalBridge.EvaluateOrVerifyFinancialDraft(
					ctx, preSupplierID, draftID, preDraftType, action, reviewerID,
					auth.EffectiveActorType(claims), auth.EffectiveSupplierRole(claims),
					strings.TrimSpace(req.ApprovalID), preAmountMinor, canonPayload,
				)
				if bErr != nil {
					return DraftResponse{}, bErr
				}
				approvalRequestID = appRes.ApprovalRequestID
				payloadHash = appRes.PayloadHash
				if appRes.Required && !appRes.Approved {
					return DraftResponse{
						DraftID:             draftID,
						DraftType:           preDraftType,
						Status:              draftStatusPendingApproval,
						ProposedAmountMinor: preAmountMinor,
						ApprovalRequestID:   appRes.ApprovalRequestID,
						PayloadHash:         appRes.PayloadHash,
						Message:             "approval_required",
					}, nil
				}
			}
		}
	}

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, target := range targets {
			row, readErr := txn.ReadRow(ctx, target.tableName,
				spanner.Key{draftID},
				[]string{"DraftId", "SupplierId", "OperatorId", "Status", "ProposedAmountMinor"},
			)
			if readErr != nil {
				if isSpannerNotFound(readErr) {
					continue
				}
				return fmt.Errorf("read draft: %w", readErr)
			}

			var existingDraftID, existingSupplierID, existingOperatorID, existingStatus string
			var proposedAmountMinor int64
			if err := row.Columns(&existingDraftID, &existingSupplierID, &existingOperatorID, &existingStatus, &proposedAmountMinor); err != nil {
				return fmt.Errorf("scan draft: %w", err)
			}
			observedSupplierID = existingSupplierID
			observedOperatorID = existingOperatorID
			observedDraftType = target.draftType

			if target.draftType == DraftTypeCreditNote && !auth.HasSupplierPerm(claims, auth.PermCreditNoteApprove) {
				return draftSentinelError{
					primary: ErrOrderForbidden,
					detail:  "missing_permission: credit_note.approve",
				}
			}
			if target.draftType == DraftTypeAdjustment && !auth.HasSupplierPerm(claims, auth.PermDraftApprove) {
				return draftSentinelError{
					primary: ErrOrderForbidden,
					detail:  "missing_permission: draft.approve",
				}
			}

			if err := validateDraftReviewState(callerSupplierID, reviewerID, existingSupplierID, existingOperatorID, existingStatus); err != nil {
				return err
			}

			resolvedDraftType = target.draftType
			resolvedAmountMinor = proposedAmountMinor

			mut := spanner.Update(target.tableName,
				[]string{"DraftId", "Status", "ReviewedAt", "ReviewedBy"},
				[]any{draftID, nextStatus, spanner.CommitTimestamp, reviewerID},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		}
		return ErrDraftNotFound
	})
	if errors.Is(err, ErrSelfReviewForbidden) && s.draftSoDCheck != nil {
		cp := claims
		checkPerm := auth.PermDraftApprove
		makerPerm := auth.PermSupplierOrderCancel
		if observedDraftType == DraftTypeCreditNote {
			checkPerm = auth.PermCreditNoteApprove
			makerPerm = auth.PermCreditNotePropose
		}
		_ = s.draftSoDCheck(ctx, DraftSoDCheckInput{
			TenantID:    firstNonEmptyString(observedSupplierID, callerSupplierID),
			ActorID:     reviewerID,
			ActorType:   auth.EffectiveActorType(claims),
			Role:        auth.EffectiveSupplierRole(claims),
			Claims:      &cp,
			Action:      checkPerm,
			MakerAction: makerPerm,
			ObjectType:  firstNonEmptyString(observedDraftType, "FinancialDraft"),
			ObjectID:    draftID,
			MakerID:     strings.TrimSpace(observedOperatorID),
			RoutePath:   routePath,
			IsApproval:  true,
		})
	}
	if err != nil {
		return DraftResponse{}, err
	}
	return DraftResponse{
		DraftID:             draftID,
		DraftType:           resolvedDraftType,
		Status:              nextStatus,
		ProposedAmountMinor: resolvedAmountMinor,
		ApprovalRequestID:   approvalRequestID,
		PayloadHash:         payloadHash,
	}, nil
}

// --- HTTP Handlers ---

func (s *Service) HandleCreateFinancialDraft(w http.ResponseWriter, r *http.Request) {
	s.handleCreateDraft(w, r, s.CreateFinancialDraft, "create draft failed")
}

func (s *Service) HandleCreateDraftCreditNote(w http.ResponseWriter, r *http.Request) {
	s.handleCreateDraft(w, r, s.CreateDraftCreditNote, "create credit note draft failed")
}

func (s *Service) handleCreateDraft(
	w http.ResponseWriter,
	r *http.Request,
	fn func(context.Context, auth.Claims, CreateDraftRequest) (DraftResponse, error),
	op string,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := fn(r.Context(), claims, req)
	if err != nil {
		s.writeFinancialDraftError(w, op, req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Service) HandleReviewFinancialDraft(w http.ResponseWriter, r *http.Request) {
	s.handleReviewDraft(w, r, s.ReviewFinancialDraft, "review draft failed")
}

func (s *Service) HandleReviewDraftCreditNote(w http.ResponseWriter, r *http.Request) {
	s.handleReviewDraft(w, r, s.ReviewDraftCreditNote, "review credit note draft failed")
}

func (s *Service) handleReviewDraft(
	w http.ResponseWriter,
	r *http.Request,
	fn func(context.Context, auth.Claims, ReviewDraftRequest) (DraftResponse, error),
	op string,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ReviewDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if strings.TrimSpace(req.ApprovalID) == "" {
		req.ApprovalID = strings.TrimSpace(r.Header.Get("X-Approval-Id"))
	}

	resp, err := fn(r.Context(), claims, req)
	if err != nil {
		s.writeFinancialDraftError(w, op, req.DraftID, err)
		return
	}
	if resp.Status == draftStatusPendingApproval && resp.ApprovalRequestID != "" {
		writeJSON(w, http.StatusAccepted, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) writeFinancialDraftError(w http.ResponseWriter, op, id string, err error) {
	switch {
	case errors.Is(err, ErrDraftValidation):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrSelfReviewForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "sod_violation"})
	case errors.Is(err, ErrDraftApprovalForbidden), errors.Is(err, ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "approval_forbidden"})
	case errors.Is(err, ErrDraftPayloadMismatch):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "payload_hash_mismatch"})
	case errors.Is(err, ErrDraftNotFound), errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrDraftAlreadyReviewed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrFinancialDraftsUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		s.writeOrderMutationError(w, op, id, err)
	}
}

func firstNonEmptyString(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
