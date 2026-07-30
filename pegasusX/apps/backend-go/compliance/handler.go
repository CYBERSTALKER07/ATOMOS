package compliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	supplierID := r.URL.Query().Get("supplierId")
	if supplierID == "" {
		http.Error(w, "supplierId is required", http.StatusBadRequest)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var from, to time.Time
	var err error
	if fromStr != "" {
		if from, err = time.Parse(time.RFC3339, fromStr); err != nil {
			http.Error(w, "invalid from date format", http.StatusBadRequest)
			return
		}
	} else {
		from = time.Now().AddDate(0, -1, 0) // default last 30 days
	}
	if toStr != "" {
		if to, err = time.Parse(time.RFC3339, toStr); err != nil {
			http.Error(w, "invalid to date format", http.StatusBadRequest)
			return
		}
	} else {
		to = time.Now()
	}

	filter := DashboardFilter{
		SupplierID: supplierID,
		From:       from,
		To:         to,
	}

	stats, orders, err := h.svc.GetDashboard(ctx, filter)
	if err != nil {
		http.Error(w, "failed to get dashboard", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Stats  DashboardStats `json:"stats"`
		Orders []ProblemOrder `json:"orders"`
	}{
		Stats:  stats,
		Orders: orders,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	supplierID := r.URL.Query().Get("supplierId")
	if supplierID == "" {
		http.Error(w, "supplierId is required", http.StatusBadRequest)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var from, to time.Time
	var err error
	if fromStr != "" {
		if from, err = time.Parse(time.RFC3339, fromStr); err != nil {
			http.Error(w, "invalid from date format", http.StatusBadRequest)
			return
		}
	} else {
		from = time.Now().AddDate(0, -1, 0)
	}
	if toStr != "" {
		if to, err = time.Parse(time.RFC3339, toStr); err != nil {
			http.Error(w, "invalid to date format", http.StatusBadRequest)
			return
		}
	} else {
		to = time.Now()
	}

	filter := DashboardFilter{
		SupplierID: supplierID,
		From:       from,
		To:         to,
	}

	csvData, err := h.svc.ExportCSV(ctx, filter)
	if err != nil {
		http.Error(w, "failed to generate csv", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("compliance_export_%s.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = w.Write(csvData)
}
