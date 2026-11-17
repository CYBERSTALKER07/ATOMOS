package gl

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

var periodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// ManualJournalRequest is the JSON request body for POST /v1/finance/journal/manual.
type ManualJournalRequest struct {
	EntityID       string         `json:"entity_id,omitempty"`
	LegalEntityID  string         `json:"legal_entity_id,omitempty"`
	PeriodID       string         `json:"period_id,omitempty"`
	SourceType     string         `json:"source_type,omitempty"`
	SourceID       string         `json:"source_id"`
	OrderID        string         `json:"order_id,omitempty"`
	Memo           string         `json:"memo"`
	Currency       string         `json:"currency"`
	CountryPack    string         `json:"country_pack,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	OccurredAt     string         `json:"occurred_at,omitempty"`
	ApprovalID     string         `json:"approval_id,omitempty"`
	Postings       []PostingInput `json:"postings"`
}

// ReverseJournalRequest is the optional JSON body for POST /v1/finance/journal/{id}/reverse.
type ReverseJournalRequest struct {
	EntityID       string `json:"entity_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// BackfillRequest is the optional JSON body for POST /v1/finance/gl/backfill.
type BackfillRequest struct {
	EntityID string `json:"entity_id,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// HandleTrialBalance serves GET /v1/finance/gl/trial-balance?entity=&period=
// and GET /v1/finance/trial-balance?entity=&period=.
func (s *Service) HandleTrialBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	q := r.URL.Query()
	tenantID, legalEntityID, ok := s.resolveGLScope(w, r, q.Get("tenant_id"), q.Get("entity"), q.Get("legal_entity"))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(q.Get("period"))
	if periodID != "" && !periodPattern.MatchString(periodID) {
		writeJSONError(w, http.StatusBadRequest, "invalid_period", "period must be formatted YYYY-MM")
		return
	}
	if periodID == "" {
		periodID = ComputePeriodID(r.Context(), tenantID, "", time.Now().UTC())
	}
	report, err := s.TrialBalanceForEntity(r.Context(), tenantID, periodID, legalEntityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// HandleListJournals serves GET /v1/finance/gl?entity=&legal_entity=&period=&source_type=&account_code=&limit=.
func (s *Service) HandleListJournals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	q := r.URL.Query()
	tenantID, legalEntityID, ok := s.resolveGLScope(w, r, q.Get("tenant_id"), q.Get("entity"), q.Get("legal_entity"))
	if !ok {
		return
	}
	periodID := strings.TrimSpace(q.Get("period"))
	if periodID != "" && !periodPattern.MatchString(periodID) {
		writeJSONError(w, http.StatusBadRequest, "invalid_period", "period must be formatted YYYY-MM")
		return
	}
	limit := 50
	if rawLimit := strings.TrimSpace(q.Get("limit")); rawLimit != "" {
		n, err := strconv.Atoi(rawLimit)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	items, err := s.ListJournals(r.Context(), JournalFilter{
		TenantID:      tenantID,
		LegalEntityID: legalEntityID,
		PeriodID:      periodID,
		SourceType:    q.Get("source_type"),
		AccountCode:   q.Get("account_code"),
		Limit:         limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []JournalEntryRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":       tenantID,
		"entity_id":       coalesceNonEmpty(legalEntityID, tenantID),
		"legal_entity_id": legalEntityID,
		"period_id":       periodID,
		"count":           len(items),
		"items":           items,
	})
}

// HandleAccountLedger serves GET /v1/finance/gl/accounts/{code}/ledger?entity=&period=&limit=.
func (s *Service) HandleAccountLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	entityID, ok := resolveGLEntityScope(w, r, r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		writeJSONError(w, http.StatusBadRequest, "account_code_required", "account code is required")
		return
	}
	periodID := strings.TrimSpace(r.URL.Query().Get("period"))
	if periodID != "" && !periodPattern.MatchString(periodID) {
		writeJSONError(w, http.StatusBadRequest, "invalid_period", "period must be formatted YYYY-MM")
		return
	}
	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		n, err := strconv.Atoi(rawLimit)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	rows, err := s.AccountLedger(r.Context(), entityID, code, periodID, limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entity_id":    entityID,
		"account_code": strings.ToUpper(code),
		"period_id":    periodID,
		"limit":        limit,
		"count":        len(rows),
		"items":        rows,
	})
}

// HandleGetJournalEntry serves GET /v1/finance/journal/{entryId}?entity=.
func (s *Service) HandleGetJournalEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	entityID, ok := resolveGLEntityScope(w, r, r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	entryID := strings.TrimSpace(chi.URLParam(r, "entryId"))
	if entryID == "" {
		entryID = strings.TrimSpace(chi.URLParam(r, "id"))
	}
	if entryID == "" {
		writeJSONError(w, http.StatusBadRequest, "entry_id_required", "entryId is required")
		return
	}
	rec, err := s.GetEntry(r.Context(), entityID, entryID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// HandleManualJournal serves POST /v1/finance/journal/manual.
func (s *Service) HandleManualJournal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !requireGLMutationPerm(w, r, auth.PermJournalPost) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req ManualJournalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	requestedEntity := coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity"))
	requestedLegal := coalesceNonEmpty(req.LegalEntityID, r.URL.Query().Get("legal_entity"))
	entityID, legalEntityID, ok := s.resolveGLScope(w, r, "", requestedEntity, requestedLegal)
	if !ok {
		return
	}

	if req.PeriodID != "" && !periodPattern.MatchString(strings.TrimSpace(req.PeriodID)) {
		writeJSONError(w, http.StatusBadRequest, "invalid_period", "period_id must be formatted YYYY-MM")
		return
	}

	var occurredAt time.Time
	if strings.TrimSpace(req.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(req.OccurredAt))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_occurred_at", "occurred_at must be RFC3339")
			return
		}
		occurredAt = parsed.UTC()
	}

	idemKey := coalesceNonEmpty(
		req.IdempotencyKey,
		r.Header.Get("Idempotency-Key"),
		r.Header.Get("X-Idempotency-Key"),
	)
	if idemKey == "" {
		writeJSONError(w, http.StatusBadRequest, "idempotency_key_required", "idempotency_key or Idempotency-Key header is required")
		return
	}

	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = "MANUAL"
	}
	sourceID := coalesceNonEmpty(req.SourceID, idemKey)

	in := JournalEntryInput{
		EntityID:       entityID,
		LegalEntityID:  legalEntityID,
		PeriodID:       req.PeriodID,
		SourceType:     sourceType,
		SourceID:       sourceID,
		OrderID:        req.OrderID,
		Memo:           req.Memo,
		Currency:       req.Currency,
		CountryPack:    req.CountryPack,
		IdempotencyKey: idemKey,
		PostedBy:       auth.ActorFromContext(r.Context()),
		OccurredAt:     occurredAt,
		Postings:       req.Postings,
	}

	if s.approvalBridge != nil {
		normCopy := in
		normCopy.Postings = append([]PostingInput(nil), in.Postings...)
		totalDebit, _, valErr := ValidateJournalInput(r.Context(), &normCopy)
		if valErr != nil {
			writeDomainError(w, valErr)
			return
		}
		claims, _ := auth.FromContext(r.Context())
		canonPayload := map[string]any{
			"entity_id":    normCopy.EntityID,
			"period_id":    normCopy.PeriodID,
			"source_type":  normCopy.SourceType,
			"source_id":    normCopy.SourceID,
			"order_id":     normCopy.OrderID,
			"memo":         normCopy.Memo,
			"currency":     normCopy.Currency,
			"country_pack": normCopy.CountryPack,
			"amount_minor": totalDebit,
			"postings":     normCopy.Postings,
		}
		if normCopy.LegalEntityID != "" {
			canonPayload["legal_entity_id"] = normCopy.LegalEntityID
		}
		if strings.TrimSpace(req.OccurredAt) != "" {
			canonPayload["occurred_at"] = normCopy.OccurredAt.UTC().Format(time.RFC3339Nano)
		}
		appRes, appErr := s.approvalBridge.EvaluateOrVerifyManualJournal(
			r.Context(),
			normCopy.EntityID,
			auth.ActorFromContext(r.Context()),
			auth.EffectiveActorType(claims),
			auth.EffectiveSupplierRole(claims),
			strings.TrimSpace(req.ApprovalID),
			totalDebit,
			normCopy,
			canonPayload,
		)
		if appErr != nil {
			writeDomainError(w, appErr)
			return
		}
		if appRes.Required && strings.TrimSpace(req.ApprovalID) == "" {
			s.setMutationContext(r.Context(), entityID, "gl.journal.manual.approval_requested", "ApprovalRequest", appRes.ApprovalRequestID, appRes.ApprovalRequestID)
			writeJSON(w, http.StatusAccepted, map[string]any{
				"status":              "PENDING_APPROVAL",
				"code":                "approval_required",
				"approval_request_id": appRes.ApprovalRequestID,
				"policy_id":           appRes.PolicyID,
				"policy_version":      appRes.PolicyVersion,
				"payload_hash":        appRes.PayloadHash,
			})
			return
		}
		if strings.TrimSpace(req.ApprovalID) != "" {
			s.setMutationContext(r.Context(), entityID, "", "", "", strings.TrimSpace(req.ApprovalID))
		}
	}

	res, err := s.PostEntry(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status := http.StatusCreated
	if res.AlreadyPosted {
		status = http.StatusOK
	}
	writeJSON(w, status, res)
}

// HandleReverseJournal serves POST /v1/finance/journal/{id}/reverse.
func (s *Service) HandleReverseJournal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !requireGLMutationPerm(w, r, auth.PermJournalReverse) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	entryID := strings.TrimSpace(chi.URLParam(r, "id"))
	if entryID == "" {
		entryID = strings.TrimSpace(chi.URLParam(r, "entryId"))
	}
	if entryID == "" {
		writeJSONError(w, http.StatusBadRequest, "entry_id_required", "journal entry id is required")
		return
	}

	var req ReverseJournalRequest
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
			return
		}
		defer r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}

	requestedEntity := coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity"))
	entityID, ok := resolveGLEntityScope(w, r, requestedEntity)
	if !ok {
		return
	}

	idemKey := coalesceNonEmpty(
		req.IdempotencyKey,
		r.Header.Get("Idempotency-Key"),
		r.Header.Get("X-Idempotency-Key"),
		"reverse:"+entryID,
	)

	res, err := s.ReverseEntry(r.Context(), entityID, entryID, idemKey, req.Reason, auth.ActorFromContext(r.Context()))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status := http.StatusCreated
	if res.AlreadyPosted {
		status = http.StatusOK
	}
	writeJSON(w, status, res)
}

// HandleBackfill serves POST /v1/finance/gl/backfill.
func (s *Service) HandleBackfill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !requireGLMutationPerm(w, r, auth.PermJournalPost) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gl_unavailable", "GL repository unavailable")
		return
	}
	var req BackfillRequest
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
			return
		}
		defer r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}

	requestedEntity := coalesceNonEmpty(req.EntityID, r.URL.Query().Get("entity"))
	entityID, ok := resolveGLEntityScope(w, r, requestedEntity)
	if !ok {
		return
	}

	report, err := s.BackfillFromPaymentLedgerEntries(r.Context(), entityID, BackfillOptions{
		Limit:    req.Limit,
		PostedBy: auth.ActorFromContext(r.Context()),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// resolveGLScope enforces fail-closed tenant isolation while allowing an authenticated tenant
// to scope GL reads/writes to any owned LegalEntityId registered in LegalEntities.
func (s *Service) resolveGLScope(w http.ResponseWriter, r *http.Request, requestedTenant, requestedEntity, requestedLegalEntity string) (tenantID, legalEntityID string, ok bool) {
	reqTenant := strings.TrimSpace(requestedTenant)
	reqEntity := strings.TrimSpace(requestedEntity)
	reqLegal := strings.TrimSpace(requestedLegalEntity)
	ctx := r.Context()

	var callerTenant string
	if t, tOk := auth.TenantFromContext(ctx); tOk && strings.TrimSpace(t.SupplierID) != "" {
		callerTenant = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, sOk := auth.ResolveSupplierID(ctx); sOk {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		targetTenant := coalesceNonEmpty(reqTenant, reqEntity, callerTenant)
		if targetTenant == "" {
			writeJSONError(w, http.StatusBadRequest, "entity_required", "entity query/body parameter is required for platform admin")
			return "", "", false
		}
		legal := reqLegal
		if reqTenant != "" && reqEntity != "" && reqEntity != reqTenant && legal == "" {
			legal = reqEntity
		}
		if legal == targetTenant {
			legal = ""
		}
		return targetTenant, legal, true
	}

	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", "", false
	}
	if reqTenant != "" && reqTenant != callerTenant {
		writeJSONError(w, http.StatusForbidden, "entity_scope_mismatch", "requested tenant does not match authenticated supplier tenant")
		return "", "", false
	}

	checkOwned := func(candidate string) bool {
		owned, err := s.IsLegalEntityOwned(ctx, callerTenant, candidate)
		if err != nil {
			writeDomainError(w, err)
			return false
		}
		if !owned {
			writeJSONError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
			return false
		}
		return true
	}

	if reqEntity != "" && reqEntity != callerTenant {
		if !checkOwned(reqEntity) {
			return "", "", false
		}
		legalEntityID = reqEntity
	}
	if reqLegal != "" && reqLegal != callerTenant {
		if legalEntityID != "" && legalEntityID != reqLegal {
			writeJSONError(w, http.StatusForbidden, "entity_scope_mismatch", "conflicting legal_entity and entity scope")
			return "", "", false
		}
		if !checkOwned(reqLegal) {
			return "", "", false
		}
		legalEntityID = reqLegal
	}
	return callerTenant, legalEntityID, true
}

// resolveGLEntityScope enforces fail-closed TenantContext / SupplierId isolation.
// Platform admins may inspect/operate on an explicitly supplied entity; all other
// callers are strictly bound to their authenticated tenant SupplierId.
func resolveGLEntityScope(w http.ResponseWriter, r *http.Request, requestedEntity string) (string, bool) {
	requested := strings.TrimSpace(requestedEntity)
	ctx := r.Context()

	var callerEntity string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerEntity = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerEntity = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := coalesceNonEmpty(requested, callerEntity)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "entity_required", "entity query/body parameter is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerEntity == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if requested != "" && requested != callerEntity {
		writeJSONError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
		return "", false
	}
	return callerEntity, true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPeriodClosed):
		writeJSONError(w, http.StatusConflict, "period_closed", err.Error())
	case errors.Is(err, ErrApprovalPayloadMismatch):
		writeJSONError(w, http.StatusConflict, "payload_hash_mismatch", err.Error())
	case errors.Is(err, ErrApprovalPending):
		writeJSONError(w, http.StatusForbidden, "approval_not_approved", err.Error())
	case errors.Is(err, ErrApprovalNotFound):
		writeJSONError(w, http.StatusNotFound, "approval_not_found", err.Error())
	case errors.Is(err, ErrEntryNotFound):
		writeJSONError(w, http.StatusNotFound, "journal_entry_not_found", err.Error())
	case errors.Is(err, ErrAlreadyReversed):
		writeJSONError(w, http.StatusConflict, "journal_entry_already_reversed", err.Error())
	case errors.Is(err, ErrUnbalancedEntry),
		errors.Is(err, ErrInsufficientPostings),
		errors.Is(err, ErrInvalidPostingAmount),
		errors.Is(err, ErrInvalidPostingDirection),
		errors.Is(err, ErrCurrencyMismatch),
		errors.Is(err, ErrUnknownAccount),
		errors.Is(err, ErrInactiveAccount),
		errors.Is(err, ErrInvalidJournalInput):
		writeJSONError(w, http.StatusUnprocessableEntity, "invalid_journal_entry", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "gl_internal_error", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error":   message,
		"code":    code,
		"message": message,
	})
}

func requireGLMutationPerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authenticated caller claims are required")
		return false
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) && auth.IsAgentForbiddenPerm(perm) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot execute restricted journal mutation")
		return false
	}
	if !auth.HasSupplierPerm(claims, perm) {
		writeJSONError(w, http.StatusForbidden, "missing_permission", "missing_permission: "+perm)
		return false
	}
	return true
}
