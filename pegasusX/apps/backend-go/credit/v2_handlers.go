package credit

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func decodeCreditNoFloatJSON(body io.Reader, dst any) error {
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return ErrInvalidInput
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var probe any
	if err := dec.Decode(&probe); err != nil {
		return err
	}
	if err := checkCreditNoFloats(probe); err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func checkCreditNoFloats(v any) error {
	switch tv := v.(type) {
	case json.Number:
		if strings.ContainsAny(tv.String(), ".eE") {
			return ErrFloatForbidden
		}
	case float64, float32:
		return ErrFloatForbidden
	case map[string]any:
		for _, val := range tv {
			if err := checkCreditNoFloats(val); err != nil {
				return err
			}
		}
	case []any:
		for _, val := range tv {
			if err := checkCreditNoFloats(val); err != nil {
				return err
			}
		}
	}
	return nil
}

// HandleGetRetailerScore serves GET /v1/credit/retailers/{id}/score?scope=RELATIONSHIP|NETWORK|BOTH&supplier_id=&recompute=true.
func (s *Service) HandleGetRetailerScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}

	var supplierID string
	switch claims.Role {
	case auth.RoleRetailer:
		ownOrg := strings.TrimSpace(auth.ResolveRetailerOrgID(claims))
		if ownOrg == "" || ownOrg != retailerID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		supplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		supplierID = strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
		if supplierID == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
			return
		}
		if qSup := strings.TrimSpace(r.URL.Query().Get("supplier_id")); qSup != "" && qSup != supplierID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	case auth.RolePlatformAdmin:
		supplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
		if supplierID == "" {
			supplierID = strings.TrimSpace(claims.SupplierID)
		}
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	recomputeRaw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("recompute")))
	recompute := recomputeRaw == "true" || recomputeRaw == "1"

	view, err := s.GetRetailerScoreView(r.Context(), retailerID, supplierID, scope, recompute)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrMLScorerNotConfigured), errors.Is(err, ErrMLScorerFailed):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error(), "code": "ml_scorer_unavailable"})
		default:
			slog.ErrorContext(r.Context(), "get retailer score v2 failed", "err", err, "retailer_id", retailerID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleGetPortfolio serves GET /v1/credit/portfolio?status=&risk_tier=&limit=.
func (s *Service) HandleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var supplierID string
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		supplierID = strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	case auth.RolePlatformAdmin:
		supplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
		if supplierID == "" {
			supplierID = strings.TrimSpace(claims.SupplierID)
		}
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_limit"})
			return
		}
		limit = n
	}

	resp, err := s.GetSupplierPortfolio(
		r.Context(),
		supplierID,
		strings.TrimSpace(r.URL.Query().Get("status")),
		strings.TrimSpace(r.URL.Query().Get("risk_tier")),
		limit,
	)
	if err != nil {
		slog.ErrorContext(r.Context(), "get credit portfolio failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandlePostCreditLimit serves POST /v1/credit/limits/{id} with strict no-float parsing and CEL approval gating.
func (s *Service) HandlePostCreditLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin && claims.Role != auth.RolePlatformAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !requireCreditWritePerm(w, claims, auth.PermCreditLimitChange) {
		return
	}

	var supplierID string
	if claims.Role == auth.RolePlatformAdmin {
		supplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
		if supplierID == "" {
			supplierID = strings.TrimSpace(claims.SupplierID)
		}
	} else {
		supplierID = strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	retailerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}

	var req struct {
		NewLimitMinor   *int64 `json:"new_limit_minor"`
		LimitMinor      *int64 `json:"limit_minor"`
		Reason          string `json:"reason"`
		ConsentVerified *bool  `json:"consent_verified"`
		ApprovalID      string `json:"approval_id"`
		IdempotencyKey  string `json:"idempotency_key"`
	}
	if err := decodeCreditNoFloatJSON(r.Body, &req); err != nil {
		if errors.Is(err, ErrFloatForbidden) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "float_forbidden"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	limitPtr := req.NewLimitMinor
	if limitPtr == nil {
		limitPtr = req.LimitMinor
	}
	if limitPtr == nil || *limitPtr < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new_limit_minor_required"})
		return
	}
	if s == nil || s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credit_unavailable"})
		return
	}

	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		idemKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	s.setMutationContext(
		r.Context(),
		supplierID,
		"CREDIT_LIMIT_CHANGE",
		"credit_profile",
		profileID(retailerID, supplierID),
		strings.TrimSpace(req.Reason),
		strings.TrimSpace(req.ApprovalID),
	)

	out, err := s.RequestOrApplyLimitChange(r.Context(), LimitChangeInput{
		RetailerID:      retailerID,
		SupplierID:      supplierID,
		NewLimitMinor:   *limitPtr,
		Reason:          strings.TrimSpace(req.Reason),
		ConsentVerified: req.ConsentVerified,
		ActorID:         auth.ActorLabel(claims),
		ActorType:       auth.EffectiveActorType(claims),
		ActorRole:       auth.EffectiveSupplierRole(claims),
		IsAuditor:       auth.IsReadOnlyAuditor(claims),
		ApprovalID:      strings.TrimSpace(req.ApprovalID),
		IdempotencyKey:  idemKey,
	})
	if err != nil {
		low := strings.ToLower(err.Error())
		switch {
		case errors.Is(err, ErrAgentApprovalForbidden) || strings.Contains(low, "agent actor cannot"):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "agent_approval_forbidden"})
		case errors.Is(err, ErrAuditorReadOnly) || strings.Contains(low, "auditor") && strings.Contains(low, "read-only"):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "auditor_read_only"})
		case errors.Is(err, ErrSoDViolation) || strings.Contains(low, "segregation of duties"):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "sod_violation"})
		case errors.Is(err, ErrApprovalPayloadMismatch) || strings.Contains(low, "payload hash mismatch"):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "payload_hash_mismatch"})
		case errors.Is(err, ErrApprovalRequired) || strings.Contains(low, "not approved") || strings.Contains(low, "approval_required") || strings.Contains(low, "mfa verification required") || strings.Contains(low, "role not authorized"):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error(), "code": "approval_pending"})
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, ErrMLScorerNotConfigured), errors.Is(err, ErrMLScorerFailed):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error(), "code": "ml_scorer_unavailable"})
		default:
			slog.ErrorContext(r.Context(), "post credit limit failed", "err", err, "retailer_id", retailerID, "supplier_id", supplierID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		}
		return
	}
	if out.ApprovalRequired {
		writeJSON(w, http.StatusAccepted, out)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
