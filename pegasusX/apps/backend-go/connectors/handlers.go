package connectors

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type triggerConnectorHTTPReq struct {
	TenantID       string `json:"tenant_id,omitempty"`
	AccountLabel   string `json:"account_label,omitempty"`
	CredentialID   string `json:"credential_id,omitempty"`
	Stream         string `json:"stream,omitempty"`
	RawXML         string `json:"raw_xml,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type pushConnectorHTTPReq struct {
	TenantID       string            `json:"tenant_id,omitempty"`
	AccountLabel   string            `json:"account_label,omitempty"`
	CredentialID   string            `json:"credential_id,omitempty"`
	Stream         string            `json:"stream,omitempty"`
	ApprovalID     string            `json:"approval_id"`
	Records        []CanonicalRecord `json:"records,omitempty"`
	RawPayload     json.RawMessage   `json:"raw_payload,omitempty"`
	RawXML         string            `json:"raw_xml,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// HandleListConnectors serves GET /v1/connectors (readable by AUDITOR).
func (s *Service) HandleListConnectors(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	overview, err := s.GetOverview(r.Context(), tenantID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// HandleUpsertCredential serves PUT /v1/connectors/credentials (PermConfigManage; blocks AUDITOR & AGENT 403).
func (s *Service) HandleUpsertCredential(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, auth.PermConfigManage) {
		return
	}
	var req UpsertCredentialInput
	if !decodeJSONBody(w, r, &req, false) {
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	req.TenantID = tenantID
	req.CreatedBy = auth.ActorFromContext(r.Context())
	req.ActorType = auth.EffectiveActorType(claims)
	req.ActorRole = auth.EffectiveSupplierRole(claims)
	req.TraceID = claims.TraceID
	req.IP = r.RemoteAddr

	summary, err := s.UpsertCredential(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	afterRaw, _ := json.Marshal(summary)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "connector.credential.upsert",
		ObjectType: "ExternalCredential",
		ObjectID:   summary.CredentialID,
		AfterJSON:  string(afterRaw),
	})
	writeJSON(w, http.StatusOK, map[string]any{"credential": summary})
}

// HandleHealth serves POST /v1/connectors/{provider}/health (blocks AUDITOR & AGENT 403).
func (s *Service) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	var req triggerConnectorHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	res, err := s.CheckHealth(r.Context(), TriggerConnectorInput{
		TenantID:     tenantID,
		Provider:     provider,
		AccountLabel: req.AccountLabel,
		CredentialID: req.CredentialID,
		ActorID:      auth.ActorFromContext(r.Context()),
		ActorType:    auth.EffectiveActorType(claims),
		ActorRole:    auth.EffectiveSupplierRole(claims),
		TraceID:      claims.TraceID,
		IP:           r.RemoteAddr,
	})
	if res.Run != nil {
		audit.SetMutationContext(r.Context(), audit.MutationContext{
			TenantID:   tenantID,
			Action:     "connector.health",
			ObjectType: "ConnectorSyncRun",
			ObjectID:   res.Run.RunID,
			ReasonCode: res.Run.Status,
		})
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandlePull serves POST /v1/connectors/{provider}/pull (idempotent; blocks AUDITOR & AGENT 403).
func (s *Service) HandlePull(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	var req triggerConnectorHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	res, err := s.Pull(r.Context(), TriggerConnectorInput{
		TenantID:       tenantID,
		Provider:       provider,
		AccountLabel:   req.AccountLabel,
		CredentialID:   req.CredentialID,
		Stream:         req.Stream,
		RawPayload:     []byte(req.RawXML),
		IdempotencyKey: idem,
		ActorID:        auth.ActorFromContext(r.Context()),
		ActorType:      auth.EffectiveActorType(claims),
		ActorRole:      auth.EffectiveSupplierRole(claims),
		TraceID:        claims.TraceID,
		IP:             r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "connector.pull",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   res.Run.RunID,
		ReasonCode: res.Run.Status,
	})
	writeJSON(w, http.StatusOK, res)
}

// HandlePush serves POST /v1/connectors/{provider}/push (requires PermConfigManage/PermJournalPost + verified approval_id; blocks AUDITOR & AGENT 403).
func (s *Service) HandlePush(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	rawBytes, ok := readAndValidateNoFloatJSONBody(w, r, false)
	if !ok {
		return
	}
	var req pushConnectorHTTPReq
	if err := decodeJSONWithNumbers(rawBytes, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	var rawMap map[string]any
	_ = decodeJSONWithNumbers(rawBytes, &rawMap)

	tenantID, ok := resolveConnectorTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}

	var rawPayloadBytes []byte
	var rawPayloadForApproval any
	if trimmedRaw := bytes.TrimSpace(req.RawPayload); len(trimmedRaw) > 0 && string(trimmedRaw) != "null" {
		if trimmedRaw[0] == '"' {
			var strVal string
			if err := json.Unmarshal(trimmedRaw, &strVal); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid raw_payload string")
				return
			}
			rawPayloadBytes = []byte(strVal)
			rawPayloadForApproval = strVal
		} else {
			rawPayloadBytes = trimmedRaw
			if rawMap != nil {
				rawPayloadForApproval = rawMap["raw_payload"]
			}
		}
	} else if strings.TrimSpace(req.RawXML) != "" {
		rawPayloadBytes = []byte(req.RawXML)
		rawPayloadForApproval = req.RawXML
	}

	acctLabel := normalizeAccountLabel(req.AccountLabel)
	stream := firstNonEmpty(strings.ToLower(strings.TrimSpace(req.Stream)), defaultPushStreamForProvider(provider))
	approvalPayload := map[string]any{
		"provider":      normalizeProvider(provider),
		"account_label": acctLabel,
		"stream":        stream,
	}
	if rawMap != nil {
		if rawRecs, hasRecs := rawMap["records"]; hasRecs && rawRecs != nil {
			approvalPayload["records"] = rawRecs
		}
	}
	if rawPayloadForApproval != nil {
		approvalPayload["raw_payload"] = rawPayloadForApproval
	}

	claims, _ := auth.FromContext(r.Context())
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	res, err := s.Push(r.Context(), TriggerConnectorInput{
		TenantID:        tenantID,
		Provider:        provider,
		AccountLabel:    acctLabel,
		CredentialID:    req.CredentialID,
		Stream:          stream,
		ApprovalID:      req.ApprovalID,
		Records:         req.Records,
		ApprovalPayload: approvalPayload,
		RawPayload:      rawPayloadBytes,
		IdempotencyKey:  idem,
		ActorID:         auth.ActorFromContext(r.Context()),
		ActorType:       auth.EffectiveActorType(claims),
		ActorRole:       auth.EffectiveSupplierRole(claims),
		TraceID:         claims.TraceID,
		IP:              r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "connector.push",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   res.Run.RunID,
		ApprovalID: res.ApprovalID,
		ReasonCode: res.Run.Status,
	})
	writeJSON(w, http.StatusOK, res)
}

// HandleShadowCompare serves POST /v1/connectors/{provider}/shadow-compare (idempotent; blocks AUDITOR & AGENT 403).
func (s *Service) HandleShadowCompare(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	if !requireNonAuditorNonAgent(w, r, "") {
		return
	}
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	var req triggerConnectorHTTPReq
	if !decodeJSONBody(w, r, &req, true) {
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, coalesceNonEmpty(req.TenantID, r.URL.Query().Get("tenant_id")))
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	idem := coalesceNonEmpty(req.IdempotencyKey, r.Header.Get("Idempotency-Key"))
	res, err := s.ShadowCompare(r.Context(), TriggerConnectorInput{
		TenantID:       tenantID,
		Provider:       provider,
		AccountLabel:   req.AccountLabel,
		CredentialID:   req.CredentialID,
		Stream:         req.Stream,
		RawPayload:     []byte(req.RawXML),
		IdempotencyKey: idem,
		ActorID:        auth.ActorFromContext(r.Context()),
		ActorType:      auth.EffectiveActorType(claims),
		ActorRole:      auth.EffectiveSupplierRole(claims),
		TraceID:        claims.TraceID,
		IP:             r.RemoteAddr,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   tenantID,
		Action:     "connector.shadow_compare",
		ObjectType: "ConnectorSyncRun",
		ObjectID:   res.Run.RunID,
		ReasonCode: res.Run.Status,
	})
	writeJSON(w, http.StatusOK, res)
}

// HandleListRuns serves GET /v1/connectors/runs?provider=&limit= (readable by AUDITOR).
func (s *Service) HandleListRuns(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	runs, err := s.ListRuns(r.Context(), tenantID, r.URL.Query().Get("provider"), limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"runs":      runs,
		"count":     len(runs),
	})
}

// HandleGetRun serves GET /v1/connectors/runs/{runID} (readable by AUDITOR).
func (s *Service) HandleGetRun(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "connectors_unavailable", "connectors service unavailable")
		return
	}
	runID := strings.TrimSpace(chi.URLParam(r, "runID"))
	if runID == "" {
		writeJSONError(w, http.StatusBadRequest, "run_id_required", "runID is required")
		return
	}
	tenantID, ok := resolveConnectorTenantScope(w, r, r.URL.Query().Get("tenant_id"))
	if !ok {
		return
	}
	run, err := s.GetRun(r.Context(), tenantID, runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	resp := map[string]any{"run": run}
	if run.Summary != nil {
		resp["summary"] = run.Summary
	}
	if run.Mismatches != nil {
		resp["mismatches"] = run.Mismatches
	}
	writeJSON(w, http.StatusOK, resp)
}

func requireNonAuditorNonAgent(w http.ResponseWriter, r *http.Request, perm string) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return true
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return false
	}
	if auth.IsAgentActor(claims) {
		writeJSONError(w, http.StatusForbidden, "agent_forbidden", "AGENT actor cannot mutate connector credentials or trigger syncs")
		return false
	}
	if perm != "" && !auth.HasSupplierPerm(claims, perm) {
		writeJSONError(w, http.StatusForbidden, "missing_permission", "missing_permission: "+perm)
		return false
	}
	return true
}

func resolveConnectorTenantScope(w http.ResponseWriter, r *http.Request, requested string) (string, bool) {
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
		target := coalesceNonEmpty(req, callerTenant)
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

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
	raw, ok := readAndValidateNoFloatJSONBody(w, r, allowEmpty)
	if !ok {
		return false
	}
	if len(raw) == 0 {
		return true
	}
	if err := decodeJSONWithNumbers(raw, dst); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return false
	}
	return true
}

func readAndValidateNoFloatJSONBody(w http.ResponseWriter, r *http.Request, allowEmpty bool) ([]byte, bool) {
	if r.Body == nil {
		if allowEmpty {
			return nil, true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return nil, false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return nil, false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		if allowEmpty {
			return nil, true
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "request body is required")
		return nil, false
	}
	var generic any
	if err := decodeJSONWithNumbers(raw, &generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return nil, false
	}
	if err := validateNoFloats(generic); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return nil, false
	}
	return raw, true
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAuditorReadOnly):
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", err.Error())
	case errors.Is(err, ErrAgentForbidden):
		writeJSONError(w, http.StatusForbidden, "agent_forbidden", err.Error())
	case errors.Is(err, ErrApprovalRequired):
		writeJSONError(w, http.StatusForbidden, "approval_required", err.Error())
	case errors.Is(err, ErrApprovalPayloadMismatch):
		writeJSONError(w, http.StatusConflict, "payload_hash_mismatch", err.Error())
	case errors.Is(err, ErrFloatForbidden), errors.Is(err, ErrInvalidInput), errors.Is(err, ErrUnsupportedProvider):
		writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, ErrCredentialNotFound), errors.Is(err, ErrRunNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrConnectorReadOnly):
		writeJSONError(w, http.StatusMethodNotAllowed, "connector_read_only", err.Error())
	case errors.Is(err, ErrConnectorNotConfigured), errors.Is(err, ErrSecretNotFound), errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "connector_not_configured", err.Error())
	case errors.Is(err, ErrConnectorAuthFailed):
		writeJSONError(w, http.StatusBadGateway, "connector_upstream_auth_failed", err.Error())
	case errors.Is(err, ErrConnectorTimeout):
		writeJSONError(w, http.StatusGatewayTimeout, "connector_timeout", err.Error())
	case errors.Is(err, ErrConnectorUpstreamError), errors.Is(err, ErrConnectorBadPayload):
		writeJSONError(w, http.StatusBadGateway, "connector_upstream_error", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "connectors_internal_error", err.Error())
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

func coalesceNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
