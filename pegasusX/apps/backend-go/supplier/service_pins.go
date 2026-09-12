package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/api/iterator"
)

type servicePinInput struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Priority   int64  `json:"priority"`
}

type warehouseCoverageRequest struct {
	Cities []order.CoverageCity `json:"cities"`
	Pins   []servicePinInput    `json:"pins"`
}

func normalizePinType(raw string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case proximity.PinTargetLocation, proximity.PinTargetRetailer, proximity.PinTargetRegion, proximity.PinTargetCity:
		return strings.ToUpper(strings.TrimSpace(raw)), nil
	default:
		return "", fmt.Errorf("invalid_pin_target_type")
	}
}

func assertPinSameMarket(packCountry, targetCountry string) error {
	targetCountry = auth.NormalizeCountryCode(targetCountry)
	if targetCountry == "" {
		return auth.ErrGeographyIncomplete
	}
	return auth.AssertSameMarket(packCountry, targetCountry)
}

// HandleWarehousePins serves GET/PUT /v1/supplier/warehouses/{warehouseID}/pins.
func (s *Service) HandleWarehousePins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetWarehousePins(w, r)
	case http.MethodPut:
		s.handlePutWarehousePins(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleWarehouseCoverage serves GET/PUT /v1/supplier/warehouses/{warehouseID}/coverage.
func (s *Service) HandleWarehouseCoverage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetWarehouseCoverage(w, r)
	case http.MethodPut:
		s.handlePutWarehouseCoverage(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetWarehousePins(w http.ResponseWriter, r *http.Request) {
	sid, warehouseID, ok := s.pinScope(w, r)
	if !ok {
		return
	}
	pins, err := s.listWarehousePins(r.Context(), sid, warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pins_failed"})
		return
	}
	mode, _ := s.coverageMode(r.Context(), sid, warehouseID, pins)
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id": warehouseID,
		"mode":         mode,
		"pins":         pins,
	})
}

func (s *Service) handlePutWarehousePins(w http.ResponseWriter, r *http.Request) {
	sid, warehouseID, ok := s.pinScope(w, r)
	if !ok {
		return
	}
	var req struct {
		Pins []servicePinInput `json:"pins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if err := s.replaceWarehousePins(r.Context(), sid, warehouseID, req.Pins); err != nil {
		s.writePinError(w, err)
		return
	}
	pins, _ := s.listWarehousePins(r.Context(), sid, warehouseID)
	mode, _ := s.coverageMode(r.Context(), sid, warehouseID, pins)
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id": warehouseID,
		"mode":         mode,
		"pins":         pins,
	})
}

func (s *Service) handleGetWarehouseCoverage(w http.ResponseWriter, r *http.Request) {
	sid, warehouseID, ok := s.pinScope(w, r)
	if !ok {
		return
	}
	cities := s.loadCoverageCities(r.Context(), warehouseID)
	pins, err := s.listWarehousePins(r.Context(), sid, warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pins_failed"})
		return
	}
	mode, _ := s.coverageMode(r.Context(), sid, warehouseID, pins)
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id": warehouseID,
		"mode":         mode,
		"cities":       cities,
		"pins":         pins,
	})
}

func (s *Service) handlePutWarehouseCoverage(w http.ResponseWriter, r *http.Request) {
	sid, warehouseID, ok := s.pinScope(w, r)
	if !ok {
		return
	}
	var req warehouseCoverageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if err := s.replaceWarehouseCoverage(r.Context(), sid, warehouseID, req); err != nil {
		s.writePinError(w, err)
		return
	}
	s.handleGetWarehouseCoverage(w, r)
}

func (s *Service) pinScope(w http.ResponseWriter, r *http.Request) (supplierID, warehouseID string, ok bool) {
	supplierID = strings.TrimSpace(s.scopedSupplierID(r))
	warehouseID = strings.TrimSpace(chi.URLParam(r, "warehouseID"))
	if supplierID == "" || warehouseID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return "", "", false
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return "", "", false
	}
	return supplierID, warehouseID, true
}

func (s *Service) replaceWarehousePins(ctx context.Context, supplierID, warehouseID string, in []servicePinInput) error {
	pack, err := auth.FiscalPackForSupplier(supplierID)
	if err != nil {
		return err
	}
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return err
	}
	if err := s.assertWarehouseOwned(ctx, supplierID, warehouseID); err != nil {
		return err
	}
	pins, err := s.normalizePins(ctx, supplierID, packCountry, in)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return s.bufferPinReplace(ctx, txn, supplierID, warehouseID, pins, now, nil)
	})
	if err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, supplierCacheKey(supplierID))
	}
	slog.InfoContext(ctx, "supplier.service_pins_replaced", "supplier_id", supplierID, "warehouse_id", warehouseID, "pin_count", len(pins))
	return nil
}

func (s *Service) replaceWarehouseCoverage(ctx context.Context, supplierID, warehouseID string, req warehouseCoverageRequest) error {
	if err := s.assertWarehouseOwned(ctx, supplierID, warehouseID); err != nil {
		return err
	}
	pack, err := auth.FiscalPackForSupplier(supplierID)
	if err != nil {
		return err
	}
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return err
	}
	pins, err := s.normalizePins(ctx, supplierID, packCountry, req.Pins)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := order.CoverageMutations(supplierID, warehouseID, req.Cities, now)
		return s.bufferPinReplace(ctx, txn, supplierID, warehouseID, pins, now, muts)
	})
	if err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, supplierCacheKey(supplierID))
	}
	slog.InfoContext(ctx, "supplier.warehouse_coverage_replaced", "supplier_id", supplierID, "warehouse_id", warehouseID)
	return nil
}

func (s *Service) bufferPinReplace(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID string,
	pins []proximity.ServicePin,
	now time.Time,
	muts []*spanner.Mutation,
) error {
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT PinId FROM ServicePins WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var pinID string
		if err := row.Columns(&pinID); err != nil {
			continue
		}
		muts = append(muts, spanner.Delete("ServicePins", spanner.Key{pinID}))
	}
	for _, pin := range pins {
		muts = append(muts, spanner.InsertMap("ServicePins", map[string]any{
			"PinId":       uuid.NewString(),
			"SupplierId":  supplierID,
			"WarehouseId": warehouseID,
			"TargetType":  pin.TargetType,
			"TargetId":    pin.TargetID,
			"Priority":    pin.Priority,
			"CreatedAt":   now,
			"UpdatedAt":   now,
		}))
	}
	buf := outbox.NewSpannerTxnBuffer(txn)
	if err := outbox.EmitJSON(ctx, buf, events.AggregateSupplier, supplierID, events.TopicMain, events.SupplierEvent{
		BaseEvent:           events.BaseEvent{Type: events.EventSupplierUpdated, Timestamp: now.UTC().Format(time.RFC3339Nano)},
		SupplierID:          supplierID,
		Action:              "service_pins_replaced",
		AssignedWarehouseID: warehouseID,
	}); err != nil {
		return err
	}
	if err := buf.Flush(ctx); err != nil {
		return err
	}
	if len(muts) == 0 {
		return nil
	}
	return txn.BufferWrite(muts)
}

func (s *Service) normalizePins(ctx context.Context, supplierID, packCountry string, in []servicePinInput) ([]proximity.ServicePin, error) {
	out := make([]proximity.ServicePin, 0, len(in))
	for _, raw := range in {
		targetType, err := normalizePinType(raw.TargetType)
		if err != nil {
			return nil, err
		}
		targetID := strings.TrimSpace(raw.TargetID)
		if targetID == "" {
			return nil, errors.New("pin_target_id_required")
		}
		targetCountry, err := s.lookupPinTargetCountry(ctx, supplierID, targetType, targetID)
		if err != nil {
			return nil, err
		}
		if err := assertPinSameMarket(packCountry, targetCountry); err != nil {
			return nil, err
		}
		out = append(out, proximity.ServicePin{
			TargetType: targetType,
			TargetID:   targetID,
			Priority:   raw.Priority,
		})
	}
	return out, nil
}

func (s *Service) lookupPinTargetCountry(ctx context.Context, supplierID, targetType, targetID string) (string, error) {
	if s.lookupPinCountry != nil {
		return s.lookupPinCountry(ctx, supplierID, targetType, targetID)
	}
	switch targetType {
	case proximity.PinTargetLocation:
		row, err := s.portalSpanner.Single().ReadRow(ctx, "RetailerLocations", spanner.Key{targetID}, []string{"CountryCode"})
		if err != nil {
			return "", fmt.Errorf("pin_target_not_found")
		}
		var country spanner.NullString
		if err := row.Columns(&country); err != nil || !country.Valid {
			return "", auth.ErrGeographyIncomplete
		}
		return country.StringVal, nil
	case proximity.PinTargetRetailer:
		row, err := s.portalSpanner.Single().ReadRow(ctx, "Retailers", spanner.Key{targetID}, []string{"CountryCode"})
		if err != nil {
			return "", fmt.Errorf("pin_target_not_found")
		}
		var country spanner.NullString
		if err := row.Columns(&country); err != nil || !country.Valid {
			return "", auth.ErrGeographyIncomplete
		}
		return country.StringVal, nil
	case proximity.PinTargetRegion:
		row, err := s.portalSpanner.Single().ReadRow(ctx, "SupplierRegions", spanner.Key{supplierID, targetID}, []string{"CountryCode"})
		if err != nil {
			return "", fmt.Errorf("pin_target_not_found")
		}
		var country string
		if err := row.Columns(&country); err != nil {
			return "", auth.ErrGeographyIncomplete
		}
		return country, nil
	case proximity.PinTargetCity:
		pack, err := auth.FiscalPackForSupplier(supplierID)
		if err != nil {
			return "", err
		}
		return auth.PackCountryCode(pack)
	default:
		return "", fmt.Errorf("invalid_pin_target_type")
	}
}

func (s *Service) assertWarehouseOwned(ctx context.Context, supplierID, warehouseID string) error {
	row, err := s.portalSpanner.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
	if err != nil {
		return fmt.Errorf("warehouse_not_found")
	}
	var owner string
	if err := row.Columns(&owner); err != nil {
		return fmt.Errorf("warehouse_not_found")
	}
	if strings.TrimSpace(owner) != supplierID {
		return fmt.Errorf("warehouse_not_found")
	}
	return nil
}

func (s *Service) listWarehousePins(ctx context.Context, supplierID, warehouseID string) ([]proximity.ServicePin, error) {
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, TargetType, TargetId, Priority
		      FROM ServicePins WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	out := make([]proximity.ServicePin, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pin proximity.ServicePin
		if err := row.Columns(&pin.WarehouseID, &pin.TargetType, &pin.TargetID, &pin.Priority); err != nil {
			continue
		}
		out = append(out, pin)
	}
	return out, nil
}

func (s *Service) coverageMode(ctx context.Context, supplierID, warehouseID string, pins []proximity.ServicePin) (string, error) {
	cells := []string{}
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT H3Cell FROM WarehouseCoverageCells WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var cell string
		if err := row.Columns(&cell); err == nil && cell != "" {
			cells = append(cells, cell)
		}
	}
	return proximity.EffectiveCoverageMode(proximity.WarehouseCandidate{
		WarehouseID:   warehouseID,
		CoverageCells: cells,
	}, pins), nil
}

func (s *Service) loadCoverageCities(ctx context.Context, warehouseID string) []order.CoverageCity {
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT CityName, Lat, Lng FROM WarehouseCoverageCities WHERE WarehouseId = @wid ORDER BY CityName`,
		Params: map[string]any{"wid": warehouseID},
	})
	defer iter.Stop()
	out := make([]order.CoverageCity, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var city order.CoverageCity
		if err := row.Columns(&city.Name, &city.Lat, &city.Lng); err != nil {
			continue
		}
		out = append(out, city)
	}
	return out
}

func (s *Service) writePinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrCrossMarketDeferred), errors.Is(err, auth.ErrGeographyIncomplete),
		errors.Is(err, auth.ErrMarketPackNotShipped), errors.Is(err, auth.ErrMarketPackUnknown):
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
	case strings.Contains(err.Error(), "invalid_pin_target_type"),
		strings.Contains(err.Error(), "pin_target_id_required"):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case strings.Contains(err.Error(), "warehouse_not_found"),
		strings.Contains(err.Error(), "pin_target_not_found"):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}
