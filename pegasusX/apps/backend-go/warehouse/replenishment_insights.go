package warehouse

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type replenishmentInsight struct {
	ID                string  `json:"id"`
	WarehouseID       string  `json:"warehouse_id"`
	WarehouseName     string  `json:"warehouse_name"`
	ProductID         string  `json:"product_id"`
	ProductName       string  `json:"product_name"`
	Urgency           string  `json:"urgency"`
	CurrentStock      int64   `json:"current_stock"`
	AvgDailyVelocity  float64 `json:"avg_daily_velocity"`
	DaysUntilStockout int     `json:"days_until_stockout"`
	ReorderQuantity   int64   `json:"reorder_quantity"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
}

func (s *Service) ensureReplenishmentInsightsLocked(warehouseID string) {
	if len(s.insights) > 0 {
		return
	}
	now := s.now().Format("2006-01-02T15:04:05.999999999Z07:00")
	s.insights = []replenishmentInsight{{
		ID:                "ins_wh_1",
		WarehouseID:       warehouseID,
		WarehouseName:     "Warehouse",
		ProductID:         "prod_demo_1",
		ProductName:       "Demo SKU",
		Urgency:           "HIGH",
		CurrentStock:      12,
		AvgDailyVelocity:  4.5,
		DaysUntilStockout: 3,
		ReorderQuantity:   48,
		Status:            "OPEN",
		CreatedAt:         now,
	}}
}

// HandleReplenishmentInsights serves GET /v1/warehouse/replenishment/insights.
func (s *Service) HandleReplenishmentInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	s.mu.Lock()
	s.ensureReplenishmentInsightsLocked(whID)
	rows := make([]replenishmentInsight, 0, len(s.insights))
	for _, row := range s.insights {
		if whID == "" || row.WarehouseID == whID {
			rows = append(rows, row)
		}
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"insights": rows, "data": rows})
}

// HandleReplenishmentInsightAction serves POST /v1/warehouse/replenishment/insights/{id}/{action}.
func (s *Service) HandleReplenishmentInsightAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	insightID := strings.TrimSpace(chi.URLParam(r, "id"))
	action := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "action")))
	if insightID == "" || action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path must be /insights/{id}/{approve|dismiss}"})
		return
	}
	if action != "approve" && action != "dismiss" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action must be approve or dismiss"})
		return
	}

	whID := warehouseIDFromRequest(r)
	s.mu.Lock()
	s.ensureReplenishmentInsightsLocked(whID)
	idx := -1
	for i := range s.insights {
		if s.insights[i].ID == insightID {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "insight_not_found"})
		return
	}
	if whID != "" && s.insights[idx].WarehouseID != whID {
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	nextStatus := "DISMISSED"
	transferID := ""
	if action == "approve" {
		nextStatus = "APPROVED"
		transferID = uuid.NewString()
	}
	s.insights[idx].Status = nextStatus
	s.mu.Unlock()

	resp := map[string]any{
		"insight_id": insightID,
		"status":     nextStatus,
	}
	if transferID != "" {
		resp["transfer_id"] = transferID
	}
	writeJSON(w, http.StatusOK, resp)
}
