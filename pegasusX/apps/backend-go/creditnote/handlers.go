package creditnote

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type Handlers struct {
	Svc        *Service
	SupplierID func() string
}

type wireCreditNote struct {
	CreditNoteID    string  `json:"credit_note_id"`
	OrderID         string  `json:"order_id"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	ReasonCode      string  `json:"reason_code"`
	ReasonText      *string `json:"reason_text,omitempty"`
	TotalNetMinor   int64   `json:"total_net_minor"`
	TotalVatMinor   int64   `json:"total_vat_minor"`
	TotalGrossMinor int64   `json:"total_gross_minor"`
	CreatedBy       string  `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
}

func toWire(cn CreditNote) wireCreditNote {
	return wireCreditNote{
		CreditNoteID:    cn.CreditNoteId,
		OrderID:         cn.OrderId,
		Type:            string(cn.Type),
		Status:          string(cn.Status),
		ReasonCode:      cn.ReasonCode,
		ReasonText:      cn.ReasonText,
		TotalNetMinor:   cn.TotalNetMinor,
		TotalVatMinor:   cn.TotalVatMinor,
		TotalGrossMinor: cn.TotalGrossMinor,
		CreatedBy:       cn.CreatedBy,
		CreatedAt:       cn.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) HandleList(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	seed := ""
	if h.SupplierID != nil {
		seed = strings.TrimSpace(h.SupplierID())
	}
	// PreferTenant already fail-closes when enforced + authenticated with no tenant;
	// do not undo that with an unconditional seed fallback.
	supplierID := auth.PreferTenantSupplierID(r.Context(), seed)
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant_required", "code": "TENANT_REQUIRED"})
		return
	}
	status := CreditNoteStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.Svc.ListBySupplier(r.Context(), supplierID, status, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	rows := make([]wireCreditNote, 0, len(list))
	for _, cn := range list {
		rows = append(rows, toWire(cn))
	}
	writeJSON(w, http.StatusOK, map[string]any{"credit_notes": rows})
}

func (h *Handlers) HandleCreateManual(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req CreateManualCreditNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	cn, err := h.Svc.CreateManual(r.Context(), req, claims.Subject)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toWire(*cn))
}

func (h *Handlers) HandleIssue(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if err := h.Svc.Issue(r.Context(), id, claims.Subject); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "issued"})
}

// reverseWarehouseStaff allows native warehouse staff and warehouse admins (B7 WH-P0-3).
func reverseWarehouseStaff(claims auth.Claims) bool {
	if claims.Role == auth.RoleWarehouse || claims.Role == auth.RoleWarehouseAdmin {
		return true
	}
	return claims.Role == auth.RoleAdmin && claims.SupplierRole == auth.RoleWarehouseAdmin
}

// resolveReverseWarehouseID pins body warehouse_id to JWT home node / ops scope.
// Body may only restate the home node; mismatch → scope violation.
func resolveReverseWarehouseID(r *http.Request, claims auth.Claims, bodyWarehouseID string) (string, string) {
	home := strings.TrimSpace(auth.EffectiveWarehouseOpsID(r.Context()))
	if home == "" {
		if claims.HomeNodeType == auth.HomeNodeWarehouse {
			home = strings.TrimSpace(claims.HomeNodeID)
		}
	}
	bodyWH := strings.TrimSpace(bodyWarehouseID)
	if home != "" {
		if bodyWH != "" && bodyWH != home {
			return "", "warehouse_scope_violation"
		}
		return home, ""
	}
	// No home node: only unscoped supplier ADMIN may use body warehouse_id.
	if claims.Role == auth.RoleAdmin && claims.SupplierRole != auth.RoleWarehouseAdmin {
		if bodyWH == "" {
			return "", "warehouse_id_required"
		}
		return bodyWH, ""
	}
	return "", "warehouse_scope_missing"
}

func (h *Handlers) HandleReceiveReverse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !reverseWarehouseStaff(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	taskID := strings.TrimSpace(chi.URLParam(r, "taskId"))
	var body struct {
		WarehouseID string           `json:"warehouse_id"`
		ReceivedQty map[string]int64 `json:"received_qty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	// B7 WH-P0-3: never trust body warehouse_id over JWT home node.
	wh, scopeErr := resolveReverseWarehouseID(r, claims, body.WarehouseID)
	if scopeErr != "" {
		code := http.StatusForbidden
		if scopeErr == "warehouse_id_required" {
			code = http.StatusBadRequest
		}
		writeJSON(w, code, map[string]string{"error": scopeErr})
		return
	}
	if err := h.Svc.ReceiveReverseTask(r.Context(), taskID, wh, body.ReceivedQty, claims.Subject); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (h *Handlers) HandleOrderLines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orderID := strings.TrimSpace(r.URL.Query().Get("order_id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if t, ok := auth.TenantFromContext(r.Context()); ok {
		supplierID = t.SupplierID
	}
	if supplierID == "" && h.SupplierID != nil {
		supplierID = strings.TrimSpace(h.SupplierID())
	}
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant_required"})
		return
	}
	owned, err := h.Svc.OrderOwnedBySupplier(r.Context(), orderID, supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ownership_check_failed"})
		return
	}
	if !owned {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	lines, err := h.Svc.OrderLinesForCredit(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	type wireLine struct {
		OrderLineID string `json:"order_line_id"`
		Sku         string `json:"sku"`
		Qty         int64  `json:"qty"`
		GrossMinor  int64  `json:"gross_minor"`
	}
	out := make([]wireLine, 0, len(lines))
	for _, ln := range lines {
		out = append(out, wireLine{
			OrderLineID: ln.OrderLineId,
			Sku:         ln.Sku,
			Qty:         ln.Qty,
			GrossMinor:  ln.LineGrossMinor,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": out})
}

func (h *Handlers) HandleListReverseTasks(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !reverseWarehouseStaff(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	warehouseID := strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "OPEN"
	}
	// B7 WH-P0-3: pin list to home node; query warehouse_id may restate it only.
	home := strings.TrimSpace(auth.EffectiveWarehouseOpsID(r.Context()))
	if home == "" && claims.HomeNodeType == auth.HomeNodeWarehouse {
		home = strings.TrimSpace(claims.HomeNodeID)
	}
	if home == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_missing"})
		return
	}
	if warehouseID == "" {
		warehouseID = home
	} else if warehouseID != home {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_violation"})
		return
	}
	tasks, err := h.Svc.ListReverseTasks(r.Context(), warehouseID, status, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	type wireTask struct {
		TaskID    string `json:"task_id"`
		OrderID   string `json:"order_id"`
		Status    string `json:"status"`
		Warehouse string `json:"warehouse_id,omitempty"`
	}
	out := make([]wireTask, 0, len(tasks))
	for _, t := range tasks {
		wh := ""
		if t.WarehouseId != nil {
			wh = *t.WarehouseId
		}
		out = append(out, wireTask{
			TaskID:    t.TaskId,
			OrderID:   t.OrderId,
			Status:    string(t.Status),
			Warehouse: wh,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
}
