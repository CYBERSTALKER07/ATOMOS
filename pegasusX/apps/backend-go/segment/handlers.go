package segment

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes supplier-portal segmentation APIs.
type Handlers struct {
	Service    *Service
	SupplierID func(r *http.Request) string
}

// HandleBootstrap serves POST /v1/supplier/segmentation/bootstrap.
func (h *Handlers) HandleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "segmentation_unavailable"})
		return
	}
	sid := strings.TrimSpace(h.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	actor := "supplier_admin"
	result, err := h.Service.BootstrapSegments(r.Context(), sid, actor)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "segmentation_bootstrap_failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleRetailerSegments serves GET /v1/supplier/segmentation/retailers.
func (h *Handlers) HandleRetailerSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "segmentation_unavailable"})
		return
	}
	sid := strings.TrimSpace(h.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := h.Service.ListRetailerSegments(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_retailer_segments_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"retailers": rows})
}

// HandleRetailerSegmentByID serves PATCH /v1/supplier/segmentation/retailers/{retailerID}.
func (h *Handlers) HandleRetailerSegmentByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "segmentation_unavailable"})
		return
	}
	sid := strings.TrimSpace(h.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := strings.TrimSpace(chi.URLParam(r, "retailerID"))
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	var in SetRetailerSegmentInput
	if err := json.Unmarshal(body, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if err := h.Service.SetRetailerSegment(r.Context(), retailerID, "supplier_admin", in); err != nil {
		if strings.Contains(err.Error(), "invalid_segment") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_segment"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "set_retailer_segment_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// HandleSkuClasses serves GET /v1/supplier/segmentation/sku-classes.
func (h *Handlers) HandleSkuClasses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "segmentation_unavailable"})
		return
	}
	sid := strings.TrimSpace(h.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	rows, err := h.Service.ListSkuClasses(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_sku_classes_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sku_classes": rows})
}

// HandleSkuClassBySKU serves PATCH /v1/supplier/segmentation/sku-classes/{sku}.
func (h *Handlers) HandleSkuClassBySKU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if h == nil || h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "segmentation_unavailable"})
		return
	}
	sid := strings.TrimSpace(h.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sku := strings.TrimSpace(chi.URLParam(r, "sku"))
	if sku == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku_required"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	var in SetSkuClassInput
	if err := json.Unmarshal(body, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if err := h.Service.SetSkuClass(r.Context(), sid, sku, "supplier_admin", in); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "set_sku_class_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
