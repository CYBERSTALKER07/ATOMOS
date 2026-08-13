package stocklots

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

// Handler serves bin/lot HTTP endpoints using a Spanner client + supplier seed.
type Handler struct {
	Spanner    *spanner.Client
	SupplierID string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func warehouseID(r *http.Request) string {
	if id := auth.EffectiveWarehouseID(r.Context()); id != "" {
		return id
	}
	return strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
}

func supplierID(r *http.Request, fallback string) string {
	if ops := auth.GetWarehouseOps(r.Context()); ops != nil && strings.TrimSpace(ops.SupplierID) != "" {
		return strings.TrimSpace(ops.SupplierID)
	}
	if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
		return sid
	}
	return strings.TrimSpace(fallback)
}

// assertResourceWarehouse enforces B7 WH-P0-4: by-id resources must belong to
// EffectiveWarehouseID. Empty resource warehouse is treated as unknown → forbid.
func assertResourceWarehouse(w http.ResponseWriter, r *http.Request, resourceWarehouseID string) bool {
	wh := warehouseID(r)
	if wh == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return false
	}
	resourceWarehouseID = strings.TrimSpace(resourceWarehouseID)
	if resourceWarehouseID == "" || resourceWarehouseID != wh {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_forbidden"})
		return false
	}
	return true
}

// HandleBins GET/POST /v1/warehouse/ops/bins
func (h *Handler) HandleBins(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	whID := warehouseID(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		bins, err := ListBins(r.Context(), h.Spanner, whID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_bins_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"bins": bins, "lots_enabled": LotsEnabled()})
	case http.MethodPost:
		var body struct {
			LocationID   string  `json:"location_id"`
			Zone         string  `json:"zone"`
			Aisle        string  `json:"aisle"`
			Rack         string  `json:"rack"`
			Level        string  `json:"level"`
			Bin          string  `json:"bin"`
			LocationType string  `json:"location_type"`
			PickSequence int64   `json:"pick_sequence"`
			MaxVolumeVU  float64 `json:"max_volume_vu"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		var created *BinLocation
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			created, err = UpsertBinInTxn(ctx, txn, CreateBinRequest{
				WarehouseID: whID, LocationID: body.LocationID, Zone: body.Zone, Aisle: body.Aisle,
				Rack: body.Rack, Level: body.Level, Bin: body.Bin, LocationType: body.LocationType,
				PickSequence: body.PickSequence, MaxVolumeVU: body.MaxVolumeVU,
			})
			return err
		})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleBinByID GET/PATCH /v1/warehouse/ops/bins/{locationID}
func (h *Handler) HandleBinByID(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	whID := warehouseID(r)
	locID := chi.URLParam(r, "locationID")
	if whID == "" || locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_and_location_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		b, err := GetBin(r.Context(), h.Spanner, whID, locID)
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_bin_failed"})
			return
		}
		writeJSON(w, http.StatusOK, b)
	case http.MethodPatch:
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		var updated *BinLocation
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			updated, err = PatchBinInTxn(ctx, txn, whID, locID, patch)
			return err
		})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
				return
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleLots GET /v1/warehouse/ops/lots
func (h *Handler) HandleLots(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	whID := warehouseID(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	lots, err := ListLots(r.Context(), h.Spanner, whID,
		r.URL.Query().Get("product_id"),
		r.URL.Query().Get("location_id"),
		r.URL.Query().Get("status"),
		0,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_lots_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lots": lots, "lots_enabled": LotsEnabled()})
}

// HandleLotByID GET /v1/warehouse/ops/lots/{lotID}
func (h *Handler) HandleLotByID(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	lotID := chi.URLParam(r, "lotID")
	lot, err := GetLot(r.Context(), h.Spanner, lotID)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_lot_failed"})
		return
	}
	// B7 WH-P0-4: lot must belong to caller's warehouse scope.
	if !assertResourceWarehouse(w, r, lot.WarehouseID) {
		return
	}
	writeJSON(w, http.StatusOK, lot)
}

// HandlePutaway POST /v1/warehouse/ops/lots/putaway
func (h *Handler) HandlePutaway(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !LotsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_lots_disabled"})
		return
	}
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	if whID == "" || sid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	var body struct {
		ProductID        string  `json:"product_id"`
		LocationID       string  `json:"location_id"`
		LotCode          string  `json:"lot_code"`
		Quantity         int64   `json:"quantity"`
		ExpiryDate       string  `json:"expiry_date"`
		ManufacturedDate string  `json:"manufactured_date"`
		LotID            string  `json:"lot_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req := PutawayRequest{
		SupplierID:  sid,
		WarehouseID: whID,
		ProductID:   body.ProductID,
		LocationID:  body.LocationID,
		LotCode:     body.LotCode,
		Quantity:    body.Quantity,
		LotID:       body.LotID,
	}
	if body.ExpiryDate != "" {
		if t, err := time.Parse("2006-01-02", body.ExpiryDate); err == nil {
			req.ExpiryDate = &t
		} else if t, err := time.Parse(time.RFC3339, body.ExpiryDate); err == nil {
			req.ExpiryDate = &t
		}
	}
	if body.ManufacturedDate != "" {
		if t, err := time.Parse("2006-01-02", body.ManufacturedDate); err == nil {
			req.ManufacturedDate = &t
		}
	}
	var result *PutawayResult
	err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		result, err = PutawayInTxn(ctx, txn, req)
		if err != nil {
			return err
		}
		lotID := ""
		qty := int64(0)
		if result != nil {
			lotID = result.LotID
			qty = result.QuantityOnHand
		}
		return emitWMSEvent(ctx, txn, events.EventWMSPutaway, warehouseID(r), supplierID(r, h.SupplierID), lotID, map[string]any{
			"product_id":  req.ProductID,
			"location_id": req.LocationID,
			"lot_id":      lotID,
			"quantity":    qty,
		})
	})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandlePickWaves GET/POST /v1/warehouse/ops/pick-waves
func (h *Handler) HandlePickWaves(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !PickWavesEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_pick_waves_disabled"})
		return
	}
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		waves, err := ListPickWaves(r.Context(), h.Spanner, whID, r.URL.Query().Get("manifest_id"), r.URL.Query().Get("status"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_pick_waves_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"waves": waves, "pick_waves_enabled": true})
	case http.MethodPost:
		var body struct {
			ManifestID string `json:"manifest_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		var created *PickWaveView
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			created, err = CreatePickWaveInTxn(ctx, txn, sid, whID, body.ManifestID)
			return err
		})
		if err != nil {
			code := http.StatusUnprocessableEntity
			if strings.Contains(err.Error(), "pick_wave_exists") {
				code = http.StatusConflict
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePickWaveByID GET /v1/warehouse/ops/pick-waves/{waveID}
func (h *Handler) HandlePickWaveByID(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	waveID := chi.URLParam(r, "waveID")
	wave, err := GetPickWave(r.Context(), h.Spanner, waveID, true)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_pick_wave_failed"})
		return
	}
	// B7 WH-P0-4: wave must belong to caller's warehouse scope.
	if !assertResourceWarehouse(w, r, wave.WarehouseID) {
		return
	}
	writeJSON(w, http.StatusOK, wave)
}

// HandleConfirmPickTask POST /v1/warehouse/ops/pick-waves/{waveID}/tasks/{taskID}/confirm
func (h *Handler) HandleConfirmPickTask(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !PickWavesEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_pick_waves_disabled"})
		return
	}
	waveID := chi.URLParam(r, "waveID")
	taskID := chi.URLParam(r, "taskID")
	var body struct {
		QuantityPicked int64  `json:"quantity_picked"`
		PickerID       string `json:"picker_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	// B7 WH-P0-4: pre-load wave membership before mutation.
	pre, err := GetPickWave(r.Context(), h.Spanner, waveID, false)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_pick_wave_failed"})
		return
	}
	if !assertResourceWarehouse(w, r, pre.WarehouseID) {
		return
	}
	var wave *PickWaveView
	err = spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		wave, err = ConfirmPickTaskInTxn(ctx, txn, waveID, taskID, body.PickerID, body.QuantityPicked)
		if err != nil {
			return err
		}
		return emitWMSEvent(ctx, txn, events.EventWMSPickConfirmed, whID, sid, taskID, map[string]any{
			"wave_id":         waveID,
			"task_id":         taskID,
			"quantity_picked": body.QuantityPicked,
			"picker_id":       body.PickerID,
		})
	})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, wave)
}

// HandleWaivePickShorts POST /v1/warehouse/ops/pick-waves/{waveID}/waive-shorts
func (h *Handler) HandleWaivePickShorts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !PickWavesEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_pick_waves_disabled"})
		return
	}
	waveID := chi.URLParam(r, "waveID")
	pre, err := GetPickWave(r.Context(), h.Spanner, waveID, false)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_pick_wave_failed"})
		return
	}
	if !assertResourceWarehouse(w, r, pre.WarehouseID) {
		return
	}
	var wave *PickWaveView
	err = spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		wave, err = WaiveShortsInTxn(ctx, txn, waveID)
		return err
	})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, wave)
}

func actorSubject(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok {
		return strings.TrimSpace(claims.Subject)
	}
	return ""
}

// HandleCycleCounts GET/POST /v1/warehouse/ops/cycle-counts
func (h *Handler) HandleCycleCounts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	whID := warehouseID(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		counts, err := ListCycleCounts(r.Context(), h.Spanner, whID, r.URL.Query().Get("status"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_cycle_counts_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"counts": counts, "cycle_counts_enabled": true})
	case http.MethodPost:
		var body struct {
			LocationID  string `json:"location_id"`
			ProductID   string `json:"product_id"`
			LotID       string `json:"lot_id"`
			ExpectedQty *int64 `json:"expected_qty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		var created *CycleCountView
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			created, err = CreateCycleCountInTxn(ctx, txn, whID, CreateCycleCountRequest{
				LocationID: body.LocationID, ProductID: body.ProductID, LotID: body.LotID, ExpectedQty: body.ExpectedQty,
			})
			return err
		})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleCycleCountByID GET /v1/warehouse/ops/cycle-counts/{countID}
func (h *Handler) HandleCycleCountByID(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	countID := chi.URLParam(r, "countID")
	view, err := GetCycleCount(r.Context(), h.Spanner, countID)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_cycle_count_failed"})
		return
	}
	// B7 WH-P0-4: cycle count must belong to caller's warehouse scope.
	if !assertResourceWarehouse(w, r, view.WarehouseID) {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleSubmitCycleCount POST /v1/warehouse/ops/cycle-counts/{countID}/submit
func (h *Handler) HandleSubmitCycleCount(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	countID := chi.URLParam(r, "countID")
	var body struct {
		CountedQty int64  `json:"counted_qty"`
		ReasonCode string `json:"reason_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	pre, err := GetCycleCount(r.Context(), h.Spanner, countID)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_cycle_count_failed"})
		return
	}
	if !assertResourceWarehouse(w, r, pre.WarehouseID) {
		return
	}
	var view *CycleCountView
	err = spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		view, err = SubmitCycleCountInTxn(ctx, txn, countID, actorSubject(r), body.CountedQty, body.ReasonCode)
		return err
	})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleInventoryAdjustments GET /v1/warehouse/ops/inventory-adjustments
func (h *Handler) HandleInventoryAdjustments(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	whID := warehouseID(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rows, err := ListInventoryAdjustments(r.Context(), h.Spanner, whID, r.URL.Query().Get("status"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_adjustments_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"adjustments": rows, "cycle_counts_enabled": true})
}

// HandleApproveInventoryAdjustment POST /v1/warehouse/ops/inventory-adjustments/{adjustmentID}/approve
func (h *Handler) HandleApproveInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	adjID := chi.URLParam(r, "adjustmentID")
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	// B7 WH-P0-4: pin adjustment to caller's warehouse before approve mutates stock.
	pre, err := GetInventoryAdjustment(r.Context(), h.Spanner, adjID)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_adjustment_failed"})
		return
	}
	if !assertResourceWarehouse(w, r, pre.WarehouseID) {
		return
	}
	var view *InventoryAdjustmentView
	err = spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		view, err = ApproveAdjustmentInTxn(ctx, txn, adjID, actorSubject(r))
		if err != nil {
			return err
		}
		return emitWMSEvent(ctx, txn, events.EventWMSCycleApproved, whID, sid, adjID, map[string]any{
			"adjustment_id": adjID,
			"actor":         actorSubject(r),
		})
	})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleRejectInventoryAdjustment POST /v1/warehouse/ops/inventory-adjustments/{adjustmentID}/reject
func (h *Handler) HandleRejectInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	adjID := chi.URLParam(r, "adjustmentID")
	pre, err := GetInventoryAdjustment(r.Context(), h.Spanner, adjID)
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get_adjustment_failed"})
		return
	}
	if !assertResourceWarehouse(w, r, pre.WarehouseID) {
		return
	}
	var view *InventoryAdjustmentView
	err = spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		view, err = RejectAdjustmentInTxn(ctx, txn, adjID, actorSubject(r))
		return err
	})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// HandleEnqueueABCCounts POST /v1/warehouse/ops/cycle-counts/enqueue-abc
func (h *Handler) HandleEnqueueABCCounts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	var created []CycleCountView
	err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		created, err = EnqueueABCCountsInTxn(ctx, txn, whID, sid, 20)
		return err
	})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"counts": created})
}

// HandleInventoryAccuracy GET /v1/warehouse/ops/inventory-accuracy
func (h *Handler) HandleInventoryAccuracy(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !CycleCountsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cycle_counts_disabled"})
		return
	}
	whID := warehouseID(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	kpi, err := ComputeInventoryAccuracy(r.Context(), h.Spanner, whID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "accuracy_failed"})
		return
	}
	writeJSON(w, http.StatusOK, kpi)
}

// HandleReconcileInventoryV2 GET /v1/warehouse/ops/inventory-reconcile
func (h *Handler) HandleReconcileInventoryV2(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !LotsEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_lots_disabled"})
		return
	}
	whID := warehouseID(r)
	sid := supplierID(r, h.SupplierID)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	rep, err := ReconcileInventoryV2(r.Context(), h.Spanner, sid, whID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reconcile_failed"})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// HandleTemperatureReadings GET/POST /v1/warehouse/ops/temperature-readings
func (h *Handler) HandleTemperatureReadings(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if !ColdChainEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "wms_cold_chain_disabled"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		mid := strings.TrimSpace(r.URL.Query().Get("manifest_id"))
		if mid == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id required"})
			return
		}
		rows, err := ListTemperatureReadings(r.Context(), h.Spanner, mid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_temp_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"readings": rows})
	case http.MethodPost:
		var body struct {
			ManifestID string   `json:"manifest_id"`
			SensorID   string   `json:"sensor_id"`
			TempC      float64  `json:"temp_c"`
			Lat        float64  `json:"lat"`
			Lng        float64  `json:"lng"`
			MinC       *float64 `json:"min_c"`
			MaxC       *float64 `json:"max_c"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		var bandOverride *TempBand
		if body.MinC != nil && body.MaxC != nil {
			bandOverride = &TempBand{MinC: *body.MinC, MaxC: *body.MaxC}
		}
		whID := warehouseID(r)
		sid := supplierID(r, h.SupplierID)
		var view *TemperatureReadingView
		err := spannerutils.RunReadWriteTransaction(r.Context(), h.Spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			view, err = IngestTemperatureInTxn(ctx, txn, body.ManifestID, body.SensorID, body.TempC, body.Lat, body.Lng, bandOverride)
			if err != nil {
				return err
			}
			// Emit only when an excursion/quarantine was recorded.
			if view != nil && view.Excursion {
				return emitWMSEvent(ctx, txn, events.EventWMSTemperatureBreach, whID, sid, body.ManifestID, map[string]any{
					"manifest_id": body.ManifestID,
					"sensor_id":   body.SensorID,
					"temp_c":      body.TempC,
				})
			}
			return nil
		})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, view)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}
