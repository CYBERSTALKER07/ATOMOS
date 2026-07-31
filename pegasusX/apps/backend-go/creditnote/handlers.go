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
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" && h.SupplierID != nil {
		supplierID = strings.TrimSpace(h.SupplierID())
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

func (h *Handlers) HandleReceiveReverse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleWarehouse {
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
	wh := strings.TrimSpace(body.WarehouseID)
	if wh == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	if err := h.Svc.ReceiveReverseTask(r.Context(), taskID, wh, body.ReceivedQty, claims.Subject); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}
