package esf

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
)

// HandleList serves GET /v1/fiscal/esf?entity_id=&direction=&status=&period_id=&order_id=&limit=.
func (s *Service) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	q := r.URL.Query()
	entityID, ok := resolveEntityScope(w, r, firstNonEmptyStr(q.Get("entity_id"), q.Get("supplier_id")))
	if !ok {
		return
	}
	limit := 200
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}

	docs, err := s.List(r.Context(), ListFilter{
		EntityID:  entityID,
		Direction: strings.TrimSpace(q.Get("direction")),
		Status:    strings.TrimSpace(q.Get("status")),
		PeriodID:  strings.TrimSpace(q.Get("period_id")),
		OrderID:   strings.TrimSpace(q.Get("order_id")),
		Limit:     limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entity_id": entityID,
		"documents": docs,
		"count":     len(docs),
	})
}

// HandleCreateOutbound serves POST /v1/fiscal/esf.
func (s *Service) HandleCreateOutbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}

	var req CreateOutboundRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	entityID, ok := resolveEntityScope(w, r, req.EntityID)
	if !ok {
		return
	}
	req.EntityID = entityID
	req.ActorID = auth.ActorFromContext(r.Context())

	doc, err := s.CreateOutbound(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUpstreamSubmission) && doc.EsfID != "" {
			afterJSON, _ := json.Marshal(doc)
			audit.SetMutationContext(r.Context(), audit.MutationContext{
				TenantID:   entityID,
				Action:     "fiscal.esf.submit_failed",
				ObjectType: "EsfDocument",
				ObjectID:   doc.EsfID,
				AfterJSON:  string(afterJSON),
			})
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":    err.Error(),
				"code":     "esf_upstream_failed",
				"message":  err.Error(),
				"document": doc,
			})
			return
		}
		writeDomainError(w, err)
		return
	}

	afterJSON, _ := json.Marshal(doc)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "fiscal.esf.create_outbound",
		ObjectType: "EsfDocument",
		ObjectID:   doc.EsfID,
		AfterJSON:  string(afterJSON),
	})
	writeJSON(w, http.StatusCreated, doc)
}

// HandleGet serves GET /v1/fiscal/esf/{id}.
func (s *Service) HandleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	esfID := strings.TrimSpace(chi.URLParam(r, "id"))
	if esfID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_esf_id", "esf id is required")
		return
	}
	entityID, ok := resolveEntityScope(w, r, firstNonEmptyStr(r.URL.Query().Get("entity_id"), r.URL.Query().Get("supplier_id")))
	if !ok {
		return
	}

	detail, err := s.Get(r.Context(), entityID, esfID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// HandleResend serves POST /v1/fiscal/esf/{id}/resend.
func (s *Service) HandleResend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	esfID := strings.TrimSpace(chi.URLParam(r, "id"))
	if esfID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_esf_id", "esf id is required")
		return
	}

	var body struct {
		EntityID string `json:"entity_id"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	entityID, ok := resolveEntityScope(w, r, firstNonEmptyStr(body.EntityID, r.URL.Query().Get("entity_id")))
	if !ok {
		return
	}

	doc, err := s.Resend(r.Context(), entityID, esfID, body.Operator, auth.ActorFromContext(r.Context()))
	if err != nil {
		if errors.Is(err, ErrUpstreamSubmission) && doc.EsfID != "" {
			afterJSON, _ := json.Marshal(doc)
			audit.SetMutationContext(r.Context(), audit.MutationContext{
				TenantID:   entityID,
				Action:     "fiscal.esf.resend_failed",
				ObjectType: "EsfDocument",
				ObjectID:   doc.EsfID,
				AfterJSON:  string(afterJSON),
			})
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error":    err.Error(),
				"code":     "esf_upstream_failed",
				"message":  err.Error(),
				"document": doc,
			})
			return
		}
		writeDomainError(w, err)
		return
	}

	afterJSON, _ := json.Marshal(doc)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "fiscal.esf.resend",
		ObjectType: "EsfDocument",
		ObjectID:   doc.EsfID,
		AfterJSON:  string(afterJSON),
	})
	writeJSON(w, http.StatusOK, doc)
}

// HandleCreateInbound serves POST /v1/fiscal/esf/inbound.
func (s *Service) HandleCreateInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}

	var req CreateInboundRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	entityID, ok := resolveEntityScope(w, r, req.EntityID)
	if !ok {
		return
	}
	req.EntityID = entityID
	req.ActorID = auth.ActorFromContext(r.Context())

	doc, err := s.CreateInbound(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	afterJSON, _ := json.Marshal(doc)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "fiscal.esf.create_inbound",
		ObjectType: "EsfDocument",
		ObjectID:   doc.EsfID,
		AfterJSON:  string(afterJSON),
	})
	writeJSON(w, http.StatusCreated, doc)
}

// HandleTransitionStatus serves POST /v1/fiscal/esf/{id}/status.
func (s *Service) HandleTransitionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	esfID := strings.TrimSpace(chi.URLParam(r, "id"))
	if esfID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_esf_id", "esf id is required")
		return
	}

	var req TransitionStatusRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	entityID, ok := resolveEntityScope(w, r, firstNonEmptyStr(req.EntityID, r.URL.Query().Get("entity_id")))
	if !ok {
		return
	}
	req.EntityID = entityID
	req.ActorID = auth.ActorFromContext(r.Context())

	res, err := s.TransitionStatus(r.Context(), entityID, esfID, req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	afterJSON, _ := json.Marshal(res)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "fiscal.esf.status_transition",
		ObjectType: "EsfDocument",
		ObjectID:   res.Document.EsfID,
		AfterJSON:  string(afterJSON),
	})
	writeJSON(w, http.StatusOK, res)
}

// HandleGetVATSummary serves GET /v1/fiscal/vat-summary?entity_id=&period_id=.
func (s *Service) HandleGetVATSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	q := r.URL.Query()
	entityID, ok := resolveEntityScope(w, r, firstNonEmptyStr(q.Get("entity_id"), q.Get("supplier_id")))
	if !ok {
		return
	}

	report, err := s.GetVATSummary(r.Context(), entityID, strings.TrimSpace(q.Get("period_id")))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// HandleListMissingIKPU serves GET /v1/fiscal/missing-ikpu?supplier_id=&entity_id=.
func (s *Service) HandleListMissingIKPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "esf_unavailable", "ESF service unavailable")
		return
	}
	q := r.URL.Query()
	supplierID, ok := resolveEntityScope(w, r, firstNonEmptyStr(q.Get("supplier_id"), q.Get("entity_id")))
	if !ok {
		return
	}

	res, err := s.ListMissingIKPU(r.Context(), supplierID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func resolveEntityScope(w http.ResponseWriter, r *http.Request, requestedEntity string) (string, bool) {
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
		target := firstNonEmptyStr(requested, callerEntity)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "entity_required", "entity_id query/body parameter is required for platform admin")
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
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_esf_input", err.Error())
	case errors.Is(err, ErrOrderForbidden):
		writeJSONError(w, http.StatusForbidden, "order_forbidden", err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrOrderNotFound):
		writeJSONError(w, http.StatusNotFound, "esf_not_found", err.Error())
	case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrAlreadyExists):
		writeJSONError(w, http.StatusConflict, "esf_conflict", err.Error())
	case errors.Is(err, fiscal.ErrEDSSignerNotConfigured), errors.Is(err, fiscal.ErrSoliqGatewayNotConfigured), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "esf_not_configured", err.Error())
	case errors.Is(err, ErrUpstreamSubmission):
		writeJSONError(w, http.StatusBadGateway, "esf_upstream_failed", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "esf_internal_error", err.Error())
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
