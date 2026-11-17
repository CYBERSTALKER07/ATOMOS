package forecastv2

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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeNoFloatJSON decodes JSON strictly rejecting any floating-point literals.
func decodeNoFloatJSON(body io.Reader, dst any) error {
	raw, err := io.ReadAll(io.LimitReader(body, 2<<20))
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var probe any
	if err := dec.Decode(&probe); err != nil {
		return err
	}
	if err := checkNoFloats(probe); err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func checkNoFloats(v any) error {
	switch tv := v.(type) {
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return ErrFloatForbidden
		}
	case float64, float32:
		return ErrFloatForbidden
	case map[string]any:
		for _, val := range tv {
			if err := checkNoFloats(val); err != nil {
				return err
			}
		}
	case []any:
		for _, val := range tv {
			if err := checkNoFloats(val); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveSupplierEntity(r *http.Request, claims auth.Claims) (string, int) {
	claimed := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	queried := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if queried == "" {
		queried = strings.TrimSpace(r.URL.Query().Get("entity_id"))
	}
	if claims.Role == auth.RolePlatformAdmin {
		if queried != "" {
			return queried, 0
		}
		if claimed != "" {
			return claimed, 0
		}
		return "", http.StatusBadRequest
	}
	if claims.Role == auth.RoleRetailer {
		if queried != "" {
			return queried, 0
		}
		if claimed != "" {
			return claimed, 0
		}
		return "", http.StatusBadRequest
	}
	if claimed == "" {
		return "", http.StatusForbidden
	}
	return claimed, 0
}

// HandleGetForecast serves GET /v1/planning/forecast.
func (s *Service) HandleGetForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "forecast_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse, auth.RoleRetailer:
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	entityID, code := resolveSupplierEntity(r, claims)
	if entityID == "" {
		if claims.Role == auth.RoleRetailer {
			writeJSON(w, http.StatusOK, ForecastView{
				Forecasts:    []ForecastValue{},
				Explanations: []ForecastExplanation{},
			})
			return
		}
		writeJSON(w, code, map[string]string{"error": "supplier_scope_required"})
		return
	}

	if claims.Role == auth.RoleRetailer {
		retailerID := auth.ResolveRetailerOrgID(claims)
		if retailerID == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_scope_required"})
			return
		}
		linked, lErr := s.repo.IsRetailerLinked(r.Context(), entityID, retailerID)
		if lErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "retailer_link_check_failed"})
			return
		}
		if !linked {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_not_linked"})
			return
		}
	}

	horizon := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("horizon_days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			horizon = n
		}
	} else if raw := strings.TrimSpace(r.URL.Query().Get("horizon")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			horizon = n
		}
	}
	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if locID == "" {
		locID = strings.TrimSpace(r.URL.Query().Get("location"))
	}

	view, err := s.GetForecast(r.Context(), ForecastQuery{
		EntityID:    entityID,
		RunID:       strings.TrimSpace(r.URL.Query().Get("run_id")),
		LocationID:  locID,
		SKU:         strings.TrimSpace(r.URL.Query().Get("sku")),
		HorizonDays: horizon,
		Limit:       limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleRunForecast serves POST /v1/planning/forecast/run.
func (s *Service) HandleRunForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "forecast_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin && claims.Role != auth.RoleWarehouseAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "auditor_read_only"})
		return
	}

	var req RunForecastInput
	if err := decodeNoFloatJSON(r.Body, &req); err != nil {
		status := http.StatusBadRequest
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	entityID, code := resolveSupplierEntity(r, claims)
	if strings.TrimSpace(req.EntityID) != "" && claims.Role == auth.RolePlatformAdmin {
		entityID = strings.TrimSpace(req.EntityID)
	}
	if entityID == "" {
		writeJSON(w, code, map[string]string{"error": "supplier_scope_required"})
		return
	}
	req.EntityID = entityID
	if strings.TrimSpace(req.SupplierID) == "" {
		req.SupplierID = entityID
	}
	req.TriggeredBy = auth.ActorLabel(claims)
	req.ActorType = auth.EffectiveActorType(claims)

	out, err := s.RunForecast(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrFloatForbidden):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrGeminiExplainerNotConfigured), errors.Is(err, ErrGeminiExplainerFailed):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "forecast.run.created",
		ObjectType: "ForecastRun",
		ObjectID:   out.Run.RunID,
	})
	writeJSON(w, http.StatusCreated, out)
}

// HandleApproveExplanation serves POST /v1/planning/forecast/explanations/{id}/approve.
func (s *Service) HandleApproveExplanation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "forecast_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin && claims.Role != auth.RoleWarehouseAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if auth.IsAgentActor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": ErrAgentForbidden.Error()})
		return
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": ErrAuditorReadOnly.Error()})
		return
	}
	if (claims.SupplierRole != "" || len(claims.SupplierPerms) > 0) &&
		!auth.HasSupplierPerm(claims, auth.PermAgentProposalApprove) &&
		!auth.HasSupplierPerm(claims, auth.PermDraftApprove) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing_permission", "permission": auth.PermAgentProposalApprove})
		return
	}

	expID := strings.TrimSpace(chi.URLParam(r, "id"))
	if expID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "explanation_id_required"})
		return
	}

	var body struct {
		Decision       string `json:"decision,omitempty"`
		AdjustmentBps  *int64 `json:"adjustment_bps,omitempty"`
		ActualQtyMilli *int64 `json:"actual_qty_milli,omitempty"`
		Comment        string `json:"comment,omitempty"`
	}
	if err := decodeNoFloatJSON(r.Body, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	entityID, code := resolveSupplierEntity(r, claims)
	if entityID == "" {
		writeJSON(w, code, map[string]string{"error": "supplier_scope_required"})
		return
	}

	exp, err := s.ApproveDemandAdjustment(r.Context(), ApproveAdjustmentInput{
		EntityID:       entityID,
		ExplanationID:  expID,
		Decision:       body.Decision,
		AdjustmentBps:  body.AdjustmentBps,
		ActualQtyMilli: body.ActualQtyMilli,
		Comment:        body.Comment,
		ActorID:        auth.ActorLabel(claims),
		ActorType:      auth.EffectiveActorType(claims),
		ActorRole:      auth.EffectiveSupplierRole(claims),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrExplanationNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrExplanationAlreadyDecided):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrAgentForbidden), errors.Is(err, ErrAuditorReadOnly), errors.Is(err, ErrSoDViolation):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrFloatForbidden):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "forecast.explanation." + strings.ToLower(exp.Status),
		ObjectType: "ForecastExplanation",
		ObjectID:   exp.ExplanationID,
		ReasonCode: exp.AdjustmentReason,
	})
	writeJSON(w, http.StatusOK, map[string]any{"explanation": exp})
}
