package multientity

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// HandleListEntities serves GET /v1/org/entities?status=.
func (s *Service) HandleListEntities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"), r.URL.Query().Get("entity"))
	if !ok {
		return
	}
	items, err := s.ListEntities(r.Context(), tenantID, r.URL.Query().Get("status"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []LegalEntity{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"count":     len(items),
		"entities":  items,
	})
}

// HandleCreateEntity serves POST /v1/org/entities.
func (s *Service) HandleCreateEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgent(w, r, true) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req CreateEntityInput
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	tenantID, ok := resolveTenantScope(w, r, req.TenantID, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	req.TenantID = tenantID
	claims, _ := auth.FromContext(r.Context())
	req.ActorID = auth.ActorFromContext(r.Context())
	req.ActorType = auth.EffectiveActorType(claims)
	req.Role = auth.EffectiveSupplierRole(claims)
	req.TraceID = r.Header.Get("X-Trace-Id")

	ent, err := s.CreateEntity(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"entity": ent,
	})
}

// HandlePatchEntity serves PATCH /v1/org/entities/{entityID}.
func (s *Service) HandlePatchEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgent(w, r, true) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	entityID := strings.TrimSpace(chi.URLParam(r, "entityID"))
	if entityID == "" {
		writeJSONError(w, http.StatusBadRequest, "entity_id_required", "entityID is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req PatchEntityInput
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	tenantID, ok := resolveTenantScope(w, r, req.TenantID, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	req.TenantID = tenantID
	req.EntityID = entityID
	claims, _ := auth.FromContext(r.Context())
	req.ActorID = auth.ActorFromContext(r.Context())
	req.ActorType = auth.EffectiveActorType(claims)
	req.Role = auth.EffectiveSupplierRole(claims)
	req.TraceID = r.Header.Get("X-Trace-Id")

	ent, err := s.PatchEntity(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entity": ent,
	})
}

// HandleListTransfers serves GET /v1/finance/intercompany-transfers?period=&entity=&limit=.
func (s *Service) HandleListTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"), "")
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	items, err := s.ListTransfers(r.Context(), tenantID, r.URL.Query().Get("period"), r.URL.Query().Get("entity"), limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if items == nil {
		items = []IntercompanyTransfer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"count":     len(items),
		"transfers": items,
	})
}

// HandleCreateTransfer serves POST /v1/finance/intercompany-transfers.
func (s *Service) HandleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgent(w, r, true) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req CreateTransferInput
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	tenantID, ok := resolveTenantScope(w, r, req.TenantID, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	idemKey := coalesceNonEmpty(
		req.IdempotencyKey,
		r.Header.Get("Idempotency-Key"),
		r.Header.Get("X-Idempotency-Key"),
	)
	if idemKey == "" {
		writeJSONError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header or idempotency_key is required")
		return
	}

	claims, _ := auth.FromContext(r.Context())
	req.TenantID = tenantID
	req.IdempotencyKey = idemKey
	req.CreatedBy = auth.ActorFromContext(r.Context())
	req.ActorType = auth.EffectiveActorType(claims)
	req.Role = auth.EffectiveSupplierRole(claims)
	req.TraceID = r.Header.Get("X-Trace-Id")

	res, err := s.CreateIntercompanyTransfer(r.Context(), req)
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

// HandleListFxRevaluations serves GET /v1/finance/fx-revaluation?period=&entity=&limit=.
func (s *Service) HandleListFxRevaluations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"), "")
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	runs, err := s.ListFxRevaluationRuns(r.Context(), tenantID, r.URL.Query().Get("period"), r.URL.Query().Get("entity"), limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if runs == nil {
		runs = []FxRevaluationRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"count":     len(runs),
		"runs":      runs,
	})
}

// HandleFxRevaluation serves POST /v1/finance/fx-revaluation.
func (s *Service) HandleFxRevaluation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !rejectAuditorOrAgent(w, r, true) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	var req FxRevalInput
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
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

	tenantID, ok := resolveTenantScope(w, r, req.TenantID, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	idemKey := coalesceNonEmpty(
		req.IdempotencyKey,
		r.Header.Get("Idempotency-Key"),
		r.Header.Get("X-Idempotency-Key"),
	)
	if idemKey == "" {
		writeJSONError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header or idempotency_key is required")
		return
	}

	claims, _ := auth.FromContext(r.Context())
	req.TenantID = tenantID
	req.IdempotencyKey = idemKey
	req.CreatedBy = auth.ActorFromContext(r.Context())
	req.ActorType = auth.EffectiveActorType(claims)
	req.Role = auth.EffectiveSupplierRole(claims)
	req.TraceID = r.Header.Get("X-Trace-Id")

	res, err := s.RunFxRevaluation(r.Context(), req)
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

// HandleConsolidation serves GET /v1/finance/consolidation?period=&currency=&entities=.
func (s *Service) HandleConsolidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", "multientity service unavailable")
		return
	}
	q := r.URL.Query()
	tenantID, ok := resolveTenantScope(w, r, q.Get("tenant_id"), "")
	if !ok {
		return
	}
	var entityIDs []string
	if raw := strings.TrimSpace(q.Get("entities")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(p); s != "" {
				entityIDs = append(entityIDs, s)
			}
		}
	}
	report, err := s.Consolidate(r.Context(), tenantID, q.Get("period"), q.Get("currency"), entityIDs)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func resolveTenantScope(w http.ResponseWriter, r *http.Request, requestedTenant, fallbackTenant string) (string, bool) {
	req := coalesceNonEmpty(requestedTenant, fallbackTenant)
	ctx := r.Context()

	var callerTenant string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerTenant = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerTenant = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := coalesceNonEmpty(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_required", "tenant_id is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "tenant_scope_mismatch", "requested tenant does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func rejectAuditorOrAgent(w http.ResponseWriter, r *http.Request, blockAgent bool) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "authenticated caller claims are required")
		return false
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if blockAgent && auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_mutation_forbidden", "AGENT actor cannot mutate multi-entity or FX records")
		return false
	}
	return true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEntityNotFound), errors.Is(err, ErrTransferNotFound):
		writeJSONError(w, http.StatusNotFound, "entity_not_found", err.Error())
	case errors.Is(err, ErrEntityExists):
		writeJSONError(w, http.StatusConflict, "entity_already_exists", err.Error())
	case errors.Is(err, ErrPeriodClosed), errors.Is(err, gl.ErrPeriodClosed):
		writeJSONError(w, http.StatusConflict, "period_closed", err.Error())
	case errors.Is(err, ErrFxRateMissing):
		writeJSONError(w, http.StatusUnprocessableEntity, "fx_rate_missing", err.Error())
	case errors.Is(err, ErrSameEntityTransfer):
		writeJSONError(w, http.StatusUnprocessableEntity, "same_entity_transfer", err.Error())
	case errors.Is(err, ErrEntityInactive):
		writeJSONError(w, http.StatusUnprocessableEntity, "entity_inactive", err.Error())
	case errors.Is(err, ErrInvalidPeriod):
		writeJSONError(w, http.StatusBadRequest, "invalid_period", err.Error())
	case errors.Is(err, ErrInvalidCurrency):
		writeJSONError(w, http.StatusBadRequest, "invalid_currency", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "multientity_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "multientity_internal_error", err.Error())
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
