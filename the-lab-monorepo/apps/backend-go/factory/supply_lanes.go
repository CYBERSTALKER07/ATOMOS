package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Supply Lanes — Factory→Warehouse Routing Edges ────────────────────────────
// Full CRUD for SupplyLanes table plus EMA-dampened transit time updates.
//
// Dampening strategy:
//   - DampenedTransitHours = old * 0.8 + new * 0.2 (EMA alpha=0.2)
//   - Only propagate (emit event) if delta > 15% from current dampened value
//   - Minimum update interval: 1 hour per lane (prevents thrash)

const (
	emaDampenAlpha       = 0.2
	propagationThreshold = 0.15 // 15% delta triggers event
	minUpdateInterval    = 1 * time.Hour
)

// SupplyLaneResponse is the JSON shape for a SupplyLane.
type SupplyLaneResponse struct {
	LaneId               string  `json:"lane_id"`
	SupplierId           string  `json:"supplier_id"`
	FactoryId            string  `json:"factory_id"`
	WarehouseId          string  `json:"warehouse_id"`
	TransitTimeHours     float64 `json:"transit_time_hours"`
	DampenedTransitHours float64 `json:"dampened_transit_hours"`
	FreightCostMinor     int64   `json:"freight_cost_minor"`
	CarbonScoreKg        float64 `json:"carbon_score_kg"`
	IsActive             bool    `json:"is_active"`
	Priority             int64   `json:"priority"`
}

// SupplyLanesService holds deps for supply lane operations.
type SupplyLanesService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// HandleSupplyLanes handles GET (list) and POST (create) for /v1/supplier/supply-lanes.
// POST (create) is a SOVEREIGN ACTION requiring GLOBAL_ADMIN.
func (s *SupplyLanesService) HandleSupplyLanes(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listSupplyLanes(w, r, claims.ResolveSupplierID())
	case http.MethodPost:
		if err := auth.RequireGlobalAdmin(w, claims); err != nil {
			return
		}
		s.createSupplyLane(w, r, claims.ResolveSupplierID())
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSupplyLaneAction handles PUT (update), DELETE (deactivate), and
// PUT with /transit suffix (dampened transit update) for /v1/supplier/supply-lanes/{id}.
// DELETE (deactivate) is a SOVEREIGN ACTION requiring GLOBAL_ADMIN.
func (s *SupplyLanesService) HandleSupplyLaneAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Extract lane ID from path: /v1/supplier/supply-lanes/{id}[/transit]
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/v1/supplier/supply-lanes/"), "/")
	laneID := parts[0]
	isTransitUpdate := len(parts) > 1 && parts[1] == "transit"

	if laneID == "" {
		http.Error(w, `{"error":"lane_id required"}`, http.StatusBadRequest)
		return
	}

	switch {
	case r.Method == http.MethodPut && isTransitUpdate:
		s.updateTransitTime(w, r, claims.UserID, laneID)
	case r.Method == http.MethodPut:
		s.updateSupplyLane(w, r, claims.UserID, laneID)
	case r.Method == http.MethodDelete:
		if err := auth.RequireGlobalAdmin(w, claims); err != nil {
			return
		}
		s.deactivateSupplyLane(w, r, claims.UserID, laneID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *SupplyLanesService) listSupplyLanes(w http.ResponseWriter, r *http.Request, supplierID string) {
	warehouseFilter := r.URL.Query().Get("warehouse_id")
	factoryFilter := r.URL.Query().Get("factory_id")

	sql := `SELECT LaneId, SupplierId, FactoryId, WarehouseId, TransitTimeHours,
	               DampenedTransitHours, FreightCostMinor, CarbonScoreKg, IsActive, Priority
	        FROM SupplyLanes
	        WHERE SupplierId = @supplierID`
	params := map[string]interface{}{"supplierID": supplierID}

	if warehouseFilter != "" {
		sql += ` AND WarehouseId = @warehouseID`
		params["warehouseID"] = warehouseFilter
	}
	if factoryFilter != "" {
		sql += ` AND FactoryId = @factoryID`
		params["factoryID"] = factoryFilter
	}

	sql += ` ORDER BY Priority DESC, DampenedTransitHours ASC`

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	lanes := []SupplyLaneResponse{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
			return
		}
		var lane SupplyLaneResponse
		if err := row.Columns(&lane.LaneId, &lane.SupplierId, &lane.FactoryId, &lane.WarehouseId,
			&lane.TransitTimeHours, &lane.DampenedTransitHours, &lane.FreightCostMinor,
			&lane.CarbonScoreKg, &lane.IsActive, &lane.Priority); err != nil {
			continue
		}
		lanes = append(lanes, lane)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"lanes": lanes})
}

func (s *SupplyLanesService) createSupplyLane(w http.ResponseWriter, r *http.Request, supplierID string) {
	var req struct {
		FactoryId        string  `json:"factory_id"`
		WarehouseId      string  `json:"warehouse_id"`
		TransitTimeHours float64 `json:"transit_time_hours"`
		FreightCostMinor int64   `json:"freight_cost_minor"`
		CarbonScoreKg    float64 `json:"carbon_score_kg"`
		Priority         int64   `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.FactoryId == "" || req.WarehouseId == "" {
		http.Error(w, `{"error":"factory_id and warehouse_id required"}`, http.StatusBadRequest)
		return
	}

	// Validate factory and warehouse belong to this supplier
	if err := validateWarehousesBelongToSupplier(r.Context(), s.Spanner, []string{req.WarehouseId}, supplierID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusForbidden)
		return
	}
	fRow, fErr := s.Spanner.Single().ReadRow(r.Context(), "Factories", spanner.Key{req.FactoryId}, []string{"SupplierId"})
	if fErr != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var fOwner string
	if cErr := fRow.Column(0, &fOwner); cErr != nil || fOwner != supplierID {
		http.Error(w, `{"error":"factory does not belong to your organization"}`, http.StatusForbidden)
		return
	}

	if req.TransitTimeHours <= 0 {
		req.TransitTimeHours = 24
	}

	// ── Carbon Auto-Seeding: Haversine distance between factory & warehouse ──
	var directDistanceKm float64
	coordsStmt := spanner.Statement{
		SQL: `SELECT f.Lat, f.Lng, w.Lat, w.Lng
		      FROM Factories f, Warehouses w
		      WHERE f.FactoryId = @factoryID AND w.WarehouseId = @warehouseID`,
		Params: map[string]interface{}{
			"factoryID":   req.FactoryId,
			"warehouseID": req.WarehouseId,
		},
	}
	coordsIter := s.Spanner.Single().Query(r.Context(), coordsStmt)
	coordsRow, err := coordsIter.Next()
	if err == nil {
		var fLat, fLng, wLat, wLng spanner.NullFloat64
		if err := coordsRow.Columns(&fLat, &fLng, &wLat, &wLng); err == nil {
			if fLat.Valid && fLng.Valid && wLat.Valid && wLng.Valid {
				directDistanceKm = proximity.HaversineKm(fLat.Float64, fLng.Float64, wLat.Float64, wLng.Float64)
			}
		}
	}
	coordsIter.Stop()

	// If caller didn't provide carbon score, auto-seed from distance: 0.1 kg CO2/km
	if req.CarbonScoreKg <= 0 && directDistanceKm > 0 {
		req.CarbonScoreKg = directDistanceKm * 0.1
	}

	laneID := uuid.New().String()
	_, err = s.Spanner.Apply(r.Context(), []*spanner.Mutation{
		spanner.Insert("SupplyLanes",
			[]string{"LaneId", "SupplierId", "FactoryId", "WarehouseId", "TransitTimeHours",
				"DampenedTransitHours", "FreightCostMinor", "CarbonScoreKg", "IsActive", "Priority",
				"DirectDistanceKm", "ExternalEnrichmentEnabled", "CreatedAt"},
			[]interface{}{laneID, supplierID, req.FactoryId, req.WarehouseId, req.TransitTimeHours,
				req.TransitTimeHours, req.FreightCostMinor, req.CarbonScoreKg, true, req.Priority,
				directDistanceKm, false, spanner.CommitTimestamp},
		),
	})
	if err != nil {
		log.Printf("[SUPPLY_LANES] Create error: %v", err)
		http.Error(w, `{"error":"create_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lane_id":            laneID,
		"factory_id":         req.FactoryId,
		"warehouse_id":       req.WarehouseId,
		"direct_distance_km": directDistanceKm,
		"carbon_score_kg":    req.CarbonScoreKg,
		"carbon_source": func() string {
			if directDistanceKm > 0 && req.CarbonScoreKg > 0 {
				return "ESTIMATED"
			}
			return "MANUAL"
		}(),
	})
}

func (s *SupplyLanesService) updateSupplyLane(w http.ResponseWriter, r *http.Request, supplierID, laneID string) {
	var req struct {
		FreightCostMinor *int64   `json:"freight_cost_minor"`
		CarbonScoreKg    *float64 `json:"carbon_score_kg"`
		Priority         *int64   `json:"priority"`
		IsActive         *bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify ownership
		row, err := txn.ReadRow(ctx, "SupplyLanes", spanner.Key{supplierID, laneID}, []string{"SupplierId"})
		if err != nil {
			return fmt.Errorf("lane not found")
		}
		var owner string
		row.Columns(&owner)
		if owner != supplierID {
			return fmt.Errorf("not_owner")
		}

		cols := []string{"SupplierId", "LaneId", "UpdatedAt"}
		vals := []interface{}{supplierID, laneID, spanner.CommitTimestamp}

		if req.FreightCostMinor != nil {
			cols = append(cols, "FreightCostMinor")
			vals = append(vals, *req.FreightCostMinor)
		}
		if req.CarbonScoreKg != nil {
			cols = append(cols, "CarbonScoreKg")
			vals = append(vals, *req.CarbonScoreKg)
		}
		if req.Priority != nil {
			cols = append(cols, "Priority")
			vals = append(vals, *req.Priority)
		}
		if req.IsActive != nil {
			cols = append(cols, "IsActive")
			vals = append(vals, *req.IsActive)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("SupplyLanes", cols, vals),
		})
	})
	if err != nil {
		if err.Error() == "lane not found" || err.Error() == "not_owner" {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *SupplyLanesService) deactivateSupplyLane(w http.ResponseWriter, r *http.Request, supplierID, laneID string) {
	_, err := s.Spanner.Apply(r.Context(), []*spanner.Mutation{
		spanner.Update("SupplyLanes",
			[]string{"SupplierId", "LaneId", "IsActive", "UpdatedAt"},
			[]interface{}{supplierID, laneID, false, spanner.CommitTimestamp},
		),
	})
	if err != nil {
		http.Error(w, `{"error":"deactivate_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated"})
}

// ── Dampened Transit Time Update ──────────────────────────────────────────────
// PUT /v1/supplier/supply-lanes/{id}/transit
// Updates TransitTimeHours and recalculates DampenedTransitHours using EMA.
// Only emits SUPPLY_LANE_TRANSIT_UPDATED event if change exceeds propagation threshold.

func (s *SupplyLanesService) updateTransitTime(w http.ResponseWriter, r *http.Request, supplierID, laneID string) {
	var req struct {
		TransitTimeHours float64 `json:"transit_time_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TransitTimeHours <= 0 {
		http.Error(w, `{"error":"transit_time_hours required and must be positive"}`, http.StatusBadRequest)
		return
	}

	var oldDampened, newDampened float64
	var factoryID, warehouseID string
	propagated := false
	var event *kafka.SupplyLaneTransitUpdatedEvent

	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplyLanes", spanner.Key{supplierID, laneID},
			[]string{"FactoryId", "WarehouseId", "DampenedTransitHours", "LastTransitUpdate"})
		if err != nil {
			return fmt.Errorf("lane not found")
		}

		var lastUpdate spanner.NullTime
		if err := row.Columns(&factoryID, &warehouseID, &oldDampened, &lastUpdate); err != nil {
			return err
		}

		// Check minimum update interval (1 hour cooldown)
		if lastUpdate.Valid && time.Since(lastUpdate.Time) < minUpdateInterval {
			return fmt.Errorf("cooldown: last update was %v ago", time.Since(lastUpdate.Time).Round(time.Minute))
		}

		// Calculate EMA-dampened value
		newDampened = oldDampened*(1-emaDampenAlpha) + req.TransitTimeHours*emaDampenAlpha

		// Check propagation threshold
		if oldDampened > 0 {
			delta := math.Abs(newDampened-oldDampened) / oldDampened
			propagated = delta > propagationThreshold
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("SupplyLanes",
				[]string{"SupplierId", "LaneId", "TransitTimeHours", "DampenedTransitHours",
					"LastTransitUpdate", "UpdatedAt"},
				[]interface{}{supplierID, laneID, req.TransitTimeHours, newDampened,
					spanner.CommitTimestamp, spanner.CommitTimestamp},
			),
		}); err != nil {
			return err
		}

		if propagated {
			e := kafka.SupplyLaneTransitUpdatedEvent{
				LaneId:           laneID,
				SupplierId:       supplierID,
				FactoryId:        factoryID,
				WarehouseId:      warehouseID,
				OldDampenedHours: oldDampened,
				NewDampenedHours: newDampened,
				RawTransitHours:  req.TransitTimeHours,
				Timestamp:        time.Now().UTC(),
			}
			event = &e
			return outbox.EmitJSON(txn, "SupplyLane", laneID, kafka.EventSupplyLaneTransitUpdated, kafka.TopicMain, e, telemetry.TraceIDFromContext(ctx))
		}

		return nil
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "lane not found") || strings.HasPrefix(err.Error(), "cooldown") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	// Only log propagation when the dampened change exceeds threshold.
	if propagated && event != nil {
		log.Printf("[SUPPLY_LANES] Transit update propagated for lane %s: %.2f → %.2f (raw=%.2f)",
			laneID[:8], event.OldDampenedHours, event.NewDampenedHours, event.RawTransitHours)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lane_id":            laneID,
		"old_dampened_hours": oldDampened,
		"new_dampened_hours": newDampened,
		"raw_transit_hours":  req.TransitTimeHours,
		"propagated":         propagated,
	})
}
