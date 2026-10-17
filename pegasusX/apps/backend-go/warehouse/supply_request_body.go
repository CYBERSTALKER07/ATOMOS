package warehouse

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// createSupplyRequestBody is POST /v1/warehouse/supply-requests JSON body (portal + native).
type createSupplyRequestBody struct {
	FactoryID             string                   `json:"factory_id"`
	Priority              string                   `json:"priority,omitempty"`
	RequestedDeliveryDate string                   `json:"requested_delivery_date,omitempty"`
	Notes                 string                   `json:"notes,omitempty"`
	RegionID              string                   `json:"region_id,omitempty"`
	Items                 []supplyRequestItemInput `json:"items"`
	UseDemandForecast     bool                     `json:"use_demand_forecast"`
}

type supplyRequestItemInput struct {
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	Quantity          int64   `json:"quantity"`
	RecommendedQty    int64   `json:"recommended_qty,omitempty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu,omitempty"`
}

// SupplyRequestItem is one SKU line on a factory replenishment request.
type SupplyRequestItem struct {
	ItemID            string  `json:"item_id"`
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	ShippedQuantity   int64   `json:"shipped_quantity,omitempty"`
	ReceivedQuantity  int64   `json:"received_quantity,omitempty"`
	VarianceReason    string  `json:"variance_reason,omitempty"`
	RecommendedQty    int64   `json:"recommended_qty,omitempty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu,omitempty"`
}

func (s *Service) handleCreateSupplyRequestFromBody(w http.ResponseWriter, r *http.Request, warehouseID string) bool {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil || len(raw) == 0 {
		return false
	}

	var body createSupplyRequestBody
	if err := json.Unmarshal(raw, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return true
	}

	if body.UseDemandForecast {
		products, _ := s.productDemandForecast(r.Context(), warehouseID, 7)
		body.Items = make([]supplyRequestItemInput, 0, len(products))
		for _, p := range products {
			qty := p.RecommendedQty
			if qty <= 0 {
				continue
			}
			body.Items = append(body.Items, supplyRequestItemInput{
				ProductID:         p.ProductID,
				RequestedQuantity: qty,
				RecommendedQty:    qty,
			})
		}
	}

	if len(body.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supply_request_items_required"})
		return true
	}

	topology, err := s.resolveWarehouseSupplyContext(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_topology_unconfigured"})
		return true
	}
	factoryID := strings.TrimSpace(body.FactoryID)
	if factoryID == "" {
		factoryID = topology.FactoryID
	}

	if err := s.validateSupplyCycle(r.Context(), warehouseID, factoryID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return true
	}

	claims, _ := auth.FromContext(r.Context())
	requestedBy := strings.TrimSpace(claims.Subject)
	nowTS := s.now().UTC().Format(time.RFC3339Nano)
	requestID := uuid.NewString()
	priority := strings.ToUpper(strings.TrimSpace(body.Priority))
	if priority == "" {
		priority = "NORMAL"
	}

	var projected int64
	items := make([]SupplyRequestItem, 0, len(body.Items))
	for _, row := range body.Items {
		pid := strings.TrimSpace(row.ProductID)
		if pid == "" {
			continue
		}
		qty := row.RequestedQuantity
		if qty <= 0 {
			qty = row.Quantity
		}
		if qty <= 0 {
			continue
		}
		projected += qty
		items = append(items, SupplyRequestItem{
			ItemID:            uuid.NewString(),
			ProductID:         pid,
			RequestedQuantity: qty,
			RecommendedQty:    row.RecommendedQty,
			UnitVolumeVU:      row.UnitVolumeVU,
		})
	}
	if len(items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supply_request_items_required"})
		return true
	}

	req := SupplyRequest{
		RequestID:      requestID,
		SupplierID:     s.resolveSupplierScope(r.Context()),
		WarehouseID:    warehouseID,
		FactoryID:      factoryID,
		TransferMode:   topology.TransferMode,
		Status:         "SUBMITTED",
		State:          "SUBMITTED",
		RequestedBy:    requestedBy,
		Priority:       priority,
		Notes:          strings.TrimSpace(body.Notes),
		RegionID:       strings.TrimSpace(body.RegionID),
		ProjectedUnits: projected,
		TotalVolumeVU:  float64(projected),
		Items:          items,
		CreatedAt:      nowTS,
		UpdatedAt:      nowTS,
	}
	if body.RequestedDeliveryDate != "" {
		if t, err := time.Parse(time.RFC3339, body.RequestedDeliveryDate); err == nil {
			req.RequestedDeliveryDate = t.UTC().Format(time.RFC3339Nano)
		}
	}

	eventPayload := events.WarehouseEvent{
		BaseEvent:    events.BaseEvent{Type: events.EventWarehouseSupplyRequestOpened},
		RequestID:    req.RequestID,
		SupplierID:   s.resolveSupplierScope(r.Context()),
		WarehouseID:  req.WarehouseID,
		FactoryID:    req.FactoryID,
		TransferMode: req.TransferMode,
		Status:       req.Status,
		Projected:    req.ProjectedUnits,
		RequestedBy:  req.RequestedBy,
	}

	if err := s.repo.CreateSupplyRequest(r.Context(), req, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateWarehouse, req.RequestID, events.TopicMain, eventPayload)
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supply_request_failed"})
		return true
	}

	if s.cache != nil {
		s.cache.Invalidate(r.Context(), warehouseSupplyRequestsKey(s.resolveSupplierScope(r.Context()), warehouseID))
	}
	s.broadcastSupplyRequestUpdate(r.Context(), warehouseID, req)

	writeJSON(w, http.StatusCreated, map[string]any{
		"request_id":      req.RequestID,
		"status":          req.Status,
		"item_count":      len(items),
		"projected_units": projected,
	})
	return true
}
