package cashrecon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

type Handlers struct {
	Svc  *Service
	Idem idempotency.Store
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func idempotencyKey(r *http.Request) string {
	k := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if k == "" {
		k = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	}
	return k
}

type wireReconciliation struct {
	ReconciliationID  string  `json:"reconciliation_id"`
	SupplierID        string  `json:"supplier_id,omitempty"`
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
		SupplierID:        cr.SupplierId,
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
	claims, _ := auth.FromContext(r.Context())
	supplierID := auth.PreferTenantSupplierID(r.Context(), strings.TrimSpace(claims.SupplierID))
	if supplierID == "" {
		supplierID = strings.TrimSpace(claims.SupplierID)
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()

	bodyHash := sha256Hex(bodyBytes)
	idemKey := idempotencyKey(r)
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:driver:cashrecon:" + driverID + ":" + idemKey
		replay, hit, gErr := idempotency.Guard(r.Context(), h.Idem, fullKey, bodyHash)
		if errors.Is(gErr, idempotency.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_conflict"})
			return
		}
		if hit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(replay.StatusCode)
			_, _ = w.Write(replay.Response)
			return
		}
	}

	var body struct {
		RouteID           *string `json:"route_id"`
		ShiftDate         string  `json:"shift_date"`
		DeclaredCashMinor int64   `json:"declared_cash_minor"`
		DriverNote        *string `json:"driver_note"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
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
		SupplierId:        supplierID,
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

	respBytes, _ := json.Marshal(toWire(*cr))
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:driver:cashrecon:" + driverID + ":" + idemKey
		_ = h.Idem.Save(r.Context(), fullKey, idempotency.Record{
			BodyHash:   bodyHash,
			StatusCode: http.StatusOK,
			Response:   respBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
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

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()

	bodyHash := sha256Hex(bodyBytes)
	idemKey := idempotencyKey(r)
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:supplier:cashrecon:accept:" + id + ":" + idemKey
		replay, hit, gErr := idempotency.Guard(r.Context(), h.Idem, fullKey, bodyHash)
		if errors.Is(gErr, idempotency.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_conflict"})
			return
		}
		if hit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(replay.StatusCode)
			_, _ = w.Write(replay.Response)
			return
		}
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = json.Unmarshal(bodyBytes, &body)
	if err := h.Svc.Accept(r.Context(), id, claims.Subject, strings.TrimSpace(body.Note)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respBytes, _ := json.Marshal(map[string]string{"status": "accepted"})
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:supplier:cashrecon:accept:" + id + ":" + idemKey
		_ = h.Idem.Save(r.Context(), fullKey, idempotency.Record{
			BodyHash:   bodyHash,
			StatusCode: http.StatusOK,
			Response:   respBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
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

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()

	bodyHash := sha256Hex(bodyBytes)
	idemKey := idempotencyKey(r)
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:supplier:cashrecon:writeoff:" + id + ":" + idemKey
		replay, hit, gErr := idempotency.Guard(r.Context(), h.Idem, fullKey, bodyHash)
		if errors.Is(gErr, idempotency.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_conflict"})
			return
		}
		if hit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(replay.StatusCode)
			_, _ = w.Write(replay.Response)
			return
		}
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = json.Unmarshal(bodyBytes, &body)
	if err := h.Svc.WriteOff(r.Context(), id, claims.Subject, strings.TrimSpace(body.Note)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respBytes, _ := json.Marshal(map[string]string{"status": "written_off"})
	if idemKey != "" && h.Idem != nil {
		fullKey := "idem:supplier:cashrecon:writeoff:" + id + ":" + idemKey
		_ = h.Idem.Save(r.Context(), fullKey, idempotency.Record{
			BodyHash:   bodyHash,
			StatusCode: http.StatusOK,
			Response:   respBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
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
