package whatif

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type simulateHTTPReq struct {
	TenantID             string `json:"tenant_id,omitempty"`
	Name                 string `json:"name"`
	ScenarioID           string `json:"scenario_id,omitempty"`
	EntityID             string `json:"entity_id,omitempty"`
	HorizonDays          int64  `json:"horizon_days,omitempty"`
	DemandDeltaBps       int64  `json:"demand_delta_bps"`
	FactoryDowntimeHours int64  `json:"factory_downtime_hours"`
	FxRateShockBps       int64  `json:"fx_rate_shock_bps"`
	CreditLimitDeltaBps  int64  `json:"credit_limit_delta_bps"`
	UnitPriceDeltaBps    int64  `json:"unit_price_delta_bps"`
	Currency             string `json:"currency,omitempty"`
	IdempotencyKey       string `json:"idempotency_key,omitempty"`
}

type commitProposalsHTTPReq struct {
	TenantID      string   `json:"tenant_id,omitempty"`
	ProposalTypes []string `json:"proposal_types,omitempty"`
	AgentID       string   `json:"agent_id,omitempty"`
}

// HandleSimulate serves POST /v1/whatif/simulate.
func (s *Service) HandleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "whatif_unavailable", "whatif service unavailable")
		return
	}
	var req simulateHTTPReq
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	tenantID, ok := resolveTenantScope(w, r, nonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	actorID, actorType, actorRole := callerIdentity(r)
	idem := nonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	sim, replayed, err := s.Simulate(r.Context(), SimulateInput{
		TenantID:             tenantID,
		Name:                 req.Name,
		ScenarioID:           req.ScenarioID,
		EntityID:             req.EntityID,
		HorizonDays:          req.HorizonDays,
		DemandDeltaBps:       req.DemandDeltaBps,
		FactoryDowntimeHours: req.FactoryDowntimeHours,
		FxRateShockBps:       req.FxRateShockBps,
		CreditLimitDeltaBps:  req.CreditLimitDeltaBps,
		UnitPriceDeltaBps:    req.UnitPriceDeltaBps,
		Currency:             req.Currency,
		IdempotencyKey:       idem,
		ActorID:              actorID,
		ActorType:            actorType,
		ActorRole:            actorRole,
		TraceID:              strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:                   strings.TrimSpace(r.RemoteAddr),
		DeviceID:             strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "whatif.simulate",
		ObjectType: "WhatIfSimulation",
		ObjectID:   sim.SimulationID,
	})
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"simulation": sim,
		"replayed":   replayed,
	})
}

// HandleListSimulations serves GET /v1/whatif/simulations.
func (s *Service) HandleListSimulations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "whatif_unavailable", "whatif service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	sims, err := s.ListSimulations(r.Context(), ListFilter{
		TenantID: tenantID,
		Status:   r.URL.Query().Get("status"),
		Limit:    parseQueryLimit(r),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if sims == nil {
		sims = []WhatIfSimulation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":   tenantID,
		"simulations": sims,
		"count":       len(sims),
	})
}

// HandleGetSimulation serves GET /v1/whatif/simulations/{simID}.
func (s *Service) HandleGetSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "whatif_unavailable", "whatif service unavailable")
		return
	}
	tenantID, ok := resolveTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	simID := strings.TrimSpace(chi.URLParam(r, "simID"))
	sim, err := s.GetSimulation(r.Context(), tenantID, simID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"simulation": sim,
	})
}

// HandleCommitProposals serves POST /v1/whatif/simulations/{simID}/commit-proposals.
func (s *Service) HandleCommitProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if !ensureAuditorActive(w, r) {
		return
	}
	if !rejectAuditorOrAgentMutation(w, r) {
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "whatif_unavailable", "whatif service unavailable")
		return
	}
	var req commitProposalsHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveTenantScope(w, r, nonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	simID := strings.TrimSpace(chi.URLParam(r, "simID"))
	actorID, actorType, actorRole := callerIdentity(r)
	res, err := s.CommitProposals(r.Context(), CommitProposalsInput{
		TenantID:       tenantID,
		SimulationID:   simID,
		ProposalTypes:  req.ProposalTypes,
		AgentID:        req.AgentID,
		ActorID:        actorID,
		ActorType:      actorType,
		ActorRole:      actorRole,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		TraceID:        strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		IP:             strings.TrimSpace(r.RemoteAddr),
		DeviceID:       strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "whatif.commit_proposals",
		ObjectType: "WhatIfSimulation",
		ObjectID:   res.Simulation.SimulationID,
	})
	writeJSON(w, http.StatusOK, res)
}

func ensureAuditorActive(w http.ResponseWriter, r *http.Request) bool {
	if auth.IsAuditorExpiredInContext(r.Context()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	if claims, ok := auth.FromContext(r.Context()); ok && auth.IsAuditorExpired(claims, time.Now().UTC()) {
		writeJSONError(w, http.StatusUnauthorized, "auditor_access_expired", "auditor access has expired")
		return false
	}
	return true
}

func rejectAuditorOrAgentMutation(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_approval_forbidden", "AGENT actor cannot execute what-if mutations")
		return false
	}
	return true
}

func resolveTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
	req := strings.TrimSpace(requested)
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
		target := nonEmpty(req, callerTenant)
		if target == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_required", "tenant_id parameter is required for platform admin")
			return "", false
		}
		return target, true
	}
	if callerTenant == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if req != "" && req != callerTenant {
		writeJSONError(w, http.StatusForbidden, "tenant_scope_mismatch", "requested tenant_id does not match authenticated supplier tenant")
		return "", false
	}
	return callerTenant, true
}

func callerIdentity(r *http.Request) (actorID, actorType, actorRole string) {
	actorID = auth.ActorFromContext(r.Context())
	if c, ok := auth.FromContext(r.Context()); ok {
		actorType = auth.EffectiveActorType(c)
		actorRole = auth.EffectiveSupplierRole(c)
	}
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	return actorID, actorType, actorRole
}

func parseQueryLimit(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return DefaultListLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return DefaultListLimit
	}
	return clampLimit(n)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	if r.Body == nil {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512*1024))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		if allowEmpty {
			return true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return false
	}
	var generic any
	gdec := json.NewDecoder(bytes.NewReader(raw))
	gdec.UseNumber()
	if err := gdec.Decode(&generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	if err := validateNoFloats(generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "float_forbidden", err.Error())
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrFloatForbidden):
		writeJSONError(w, http.StatusBadRequest, "float_forbidden", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrConflict):
		writeJSONError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, ErrProposalUnavailable), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "whatif_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "whatif_internal_error", err.Error())
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
