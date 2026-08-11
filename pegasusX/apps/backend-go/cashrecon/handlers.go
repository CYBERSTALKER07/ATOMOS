package cashrecon

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type Handlers struct {
	Svc *Service
}

type wireReconciliation struct {
	ReconciliationID  string  `json:"reconciliation_id"`
	DriverID          string  `json:"driver_id"`
	RouteID           *string `json:"route_id,omitempty"`
	ShiftDate         string  `json:"shift_date"`
	ExpectedCashMinor int64   `json:"expected_cash_minor"`
	DeclaredCashMinor int64   `json:"declared_cash_minor"`
	DifferenceMinor   int64   `json:"difference_minor"`
	Status            string  `json:"status"`
	DriverNote        *string `json:"driver_note,omitempty"`
	FinanceNote       *string `json:"finance_note,omitempty"`
	CreatedAt         string  `json:"created_at"`
	ResolvedAt        *string `json:"resolved_at,omitempty"`
	ResolvedBy        *string `json:"resolved_by,omitempty"`
}

func toWire(cr CashReconciliation) wireReconciliation {
	w := wireReconciliation{
		ReconciliationID:  cr.ReconciliationId,
		DriverID:          cr.DriverId,
		RouteID:           cr.RouteId,
		ShiftDate:         civil.DateOf(cr.ShiftDate.UTC()).String(),
		ExpectedCashMinor: cr.ExpectedCashMinor,
		DeclaredCashMinor: cr.DeclaredCashMinor,
		DifferenceMinor:   cr.DifferenceMinor,
		Status:            string(cr.Status),
		DriverNote:        cr.DriverNote,
		FinanceNote:       cr.FinanceNote,
		CreatedAt:         cr.CreatedAt.UTC().Format(time.RFC3339Nano),
		ResolvedBy:        cr.ResolvedBy,
	}
	if cr.ResolvedAt != nil {
		s := cr.ResolvedAt.UTC().Format(time.RFC3339Nano)
		w.ResolvedAt = &s
	}
	return w
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) HandleDriverSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "service_unavailable"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var body struct {
		RouteID           *string `json:"route_id"`
		ShiftDate         string  `json:"shift_date"`
		DeclaredCashMinor int64   `json:"declared_cash_minor"`
		DriverNote        *string `json:"driver_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	shiftDate := time.Now().UTC()
	if strings.TrimSpace(body.ShiftDate) != "" {
		d, err := civil.ParseDate(strings.TrimSpace(body.ShiftDate))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_shift_date"})
			return
		}
		shiftDate = d.In(time.UTC)
	}
	cr, err := h.Svc.SubmitReconciliation(r.Context(), SubmitReconciliationRequest{
		DriverId:          driverID,
		RouteId:           body.RouteID,
		ShiftDate:         shiftDate,
		DeclaredCashMinor: body.DeclaredCashMinor,
		DriverNote:        body.DriverNote,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toWire(*cr))
}

func (h *Handlers) HandleDriverList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "service_unavailable"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	shiftDate := time.Now().UTC()
	if raw := strings.TrimSpace(r.URL.Query().Get("shift_date")); raw != "" {
		d, err := civil.ParseDate(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_shift_date"})
			return
		}
		shiftDate = d.In(time.UTC)
	}
	list, err := h.Svc.ListByDriver(r.Context(), driverID, shiftDate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	rows := make([]wireReconciliation, 0, len(list))
	for _, cr := range list {
		rows = append(rows, toWire(cr))
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliations": rows})
}

func (h *Handlers) HandleSupplierList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	supplierID := auth.PreferTenantSupplierID(r.Context(), strings.TrimSpace(claims.SupplierID))
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	status := ReconciliationStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = ReconciliationStatusPending
	}
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
	rows := make([]wireReconciliation, 0, len(list))
	for _, cr := range list {
		rows = append(rows, toWire(cr))
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliations": rows, "supplier_id": supplierID})
}

func (h *Handlers) HandleSupplierAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.Svc.Accept(r.Context(), id, claims.Subject, strings.TrimSpace(body.Note)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *Handlers) HandleSupplierWriteOff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.Svc.WriteOff(r.Context(), id, claims.Subject, strings.TrimSpace(body.Note)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "written_off"})
}

func driverIDFromRequest(r *http.Request) string {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return ""
	}
	if claims.Role != auth.RoleDriver {
		return ""
	}
	return strings.TrimSpace(claims.Subject)
}
