package valuation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleGetValuation serves GET /v1/inventory/valuation?entity_id=&warehouse_id=&product_id=&expiry_risk_days=.
func (s *Service) HandleGetValuation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "valuation_unavailable", "Valuation repository unavailable")
		return
	}
	q := r.URL.Query()
	entityID, ok := resolveEntityScope(w, r, firstNonEmpty(q.Get("entity_id"), q.Get("entity")))
	if !ok {
		return
	}
	var riskDays int64 = DefaultExpiryRiskWindowDays
	if raw := strings.TrimSpace(q.Get("expiry_risk_days")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 || n > 3650 {
			writeJSONError(w, http.StatusBadRequest, "invalid_expiry_risk_days", "expiry_risk_days must be between 1 and 3650")
			return
		}
		riskDays = n
	}
	includeZero := strings.EqualFold(strings.TrimSpace(q.Get("include_zero")), "true")
	res, err := s.GetValuationSummary(r.Context(), ValuationFilter{
		EntityID:       entityID,
		WarehouseID:    strings.TrimSpace(q.Get("warehouse_id")),
		ProductID:      strings.TrimSpace(q.Get("product_id")),
		IncludeZeroQty: includeZero,
		ExpiryRiskDays: riskDays,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleListMovements serves GET /v1/inventory/movements?entity_id=&warehouse_id=&product_id=&movement_type=&limit=.
func (s *Service) HandleListMovements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "valuation_unavailable", "Valuation repository unavailable")
		return
	}
	q := r.URL.Query()
	entityID, ok := resolveEntityScope(w, r, firstNonEmpty(q.Get("entity_id"), q.Get("entity")))
	if !ok {
		return
	}
	limit := 100
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	rows, err := s.ListMovements(r.Context(), MovementsFilter{
		EntityID:     entityID,
		WarehouseID:  strings.TrimSpace(q.Get("warehouse_id")),
		ProductID:    strings.TrimSpace(q.Get("product_id")),
		MovementType: strings.TrimSpace(q.Get("movement_type")),
		Limit:        limit,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if rows == nil {
		rows = []InventoryMovement{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entity_id": entityID,
		"movements": rows,
		"count":     len(rows),
	})
}

// HandleRecordReceipt serves POST /v1/inventory/cost-layers/receipt.
func (s *Service) HandleRecordReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "valuation_unavailable", "Valuation repository unavailable")
		return
	}
	if claims, ok := auth.FromContext(r.Context()); ok && auth.IsReadOnlyAuditor(claims) {
		writeJSONError(w, http.StatusForbidden, "auditor_read_only", "AUDITOR role is strictly read-only")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	var req ReceiptInput
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	q := r.URL.Query()
	entityID, ok := resolveEntityScope(w, r, firstNonEmpty(req.EntityID, q.Get("entity_id"), q.Get("entity")))
	if !ok {
		return
	}
	req.EntityID = entityID
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		req.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" && strings.TrimSpace(req.ReceiptRefID) != "" {
		refType := strings.ToUpper(strings.TrimSpace(req.ReceiptRefType))
		if refType == "" {
			refType = "MANUAL_RECEIPT"
		}
		req.IdempotencyKey = "receipt:" + entityID + ":" + strings.TrimSpace(req.WarehouseID) + ":" + strings.TrimSpace(req.ProductID) + ":" + refType + ":" + strings.TrimSpace(req.ReceiptRefID)
	}
	req.ActorID = auth.ActorFromContext(r.Context())

	res, err := s.RecordReceipt(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	afterJSON, _ := json.Marshal(res)
	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   entityID,
		Action:     "inventory.cost_layer.receipt",
		ObjectType: "InventoryCostLayer",
		ObjectID:   res.Layer.LayerID,
		AfterJSON:  string(afterJSON),
	})

	status := http.StatusCreated
	if res.AlreadyProcessed {
		status = http.StatusOK
	}
	writeJSON(w, status, res)
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
		target := firstNonEmpty(requested, callerEntity)
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
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrUnknownCreditAccount):
		writeJSONError(w, http.StatusBadRequest, "invalid_valuation_input", err.Error())
	case errors.Is(err, ErrInsufficientCostLayers):
		writeJSONError(w, http.StatusConflict, "insufficient_cost_layers", err.Error())
	case errors.Is(err, ErrRepoUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, "valuation_unavailable", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "valuation_internal_error", err.Error())
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
