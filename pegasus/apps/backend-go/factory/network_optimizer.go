package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Network Optimizer — Multi-Objective Factory Selection ─────────────────────
// Selects the optimal factory for a (warehouse, SKU) pair based on the supplier's
// current NetworkOptimizationMode:
//
//   - SPEED:      Sort by DampenedTransitHours ASC
//   - ECONOMY:    Sort by FreightCostMinor ASC
//   - BALANCED:   Weighted score: 0.5*transit + 0.3*cost + 0.2*carbon (normalized)
//   - LOW_CARBON: Sort by CarbonScoreKg ASC
//   - MANUAL_ONLY: Returns "" (caller must skip automated actions)

// NetworkOptimizerService handles multi-objective factory routing.
type NetworkOptimizerService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// GetNetworkMode reads the current optimization mode for a supplier.
// Returns "BALANCED" as default if no row exists.
func (s *NetworkOptimizerService) GetNetworkMode(ctx context.Context, supplierID string) (string, error) {
	row, err := s.Spanner.Single().ReadRow(ctx, "NetworkOptimizationMode",
		spanner.Key{supplierID}, []string{"Mode"})
	if err != nil {
		return "BALANCED", nil // default
	}
	var mode string
	if err := row.Columns(&mode); err != nil {
		return "BALANCED", nil
	}
	return mode, nil
}

// FactorySelection is the result of the multi-objective factory selector.
// It returns the chosen factory along with its current load ratio so the
// caller (and the UI) can surface overage warnings without hard-blocking.
type FactorySelection struct {
	FactoryID     string  `json:"factory_id"`
	LoadPercent   float64 `json:"load_percent"`   // CurrentLoad / DailyOutputCapacity * 100 (0 if unlimited)
	IsOverloaded  bool    `json:"is_overloaded"`  // true when LoadPercent > 100
	CapacityLabel string  `json:"capacity_label"` // "OK", "WARNING", "OVERLOADED", "UNLIMITED"
}

// SelectOptimalFactory returns the best factory ID for the given warehouse+SKU pair.
// Uses SupplyLanes table filtered by IsActive=true, JOINed with Factories.
//
// OBSERVER PROTOCOL: Capacity is tracked but never blocks selection.
// DailyOutputCapacity is a reference benchmark — overages are surfaced as
// telemetry warnings, not hard gates. The system tells you "how much" and
// "how fast", never "no".
func (s *NetworkOptimizerService) SelectOptimalFactory(ctx context.Context, supplierID, warehouseID, productID, mode string) (string, error) {
	sel, err := s.SelectOptimalFactoryWithTelemetry(ctx, supplierID, warehouseID, productID, mode)
	if err != nil {
		return "", err
	}
	return sel.FactoryID, nil
}

// SelectOptimalFactoryWithTelemetry is the telemetry-rich variant that returns
// the full FactorySelection including load percentage and overage status.
func (s *NetworkOptimizerService) SelectOptimalFactoryWithTelemetry(ctx context.Context, supplierID, warehouseID, productID, mode string) (*FactorySelection, error) {
	if mode == "MANUAL_ONLY" {
		return &FactorySelection{}, nil
	}

	var orderClause string
	switch mode {
	case "SPEED":
		orderClause = "ORDER BY sl.DampenedTransitHours ASC"
	case "ECONOMY":
		orderClause = "ORDER BY sl.FreightCostMinor ASC"
	case "LOW_CARBON":
		orderClause = "ORDER BY sl.CarbonScoreKg ASC"
	case "BALANCED":
		// Weighted composite: normalize each dimension 0-1 within the result set
		// For simplicity, use a linear combo of rank positions
		orderClause = "ORDER BY (sl.DampenedTransitHours * 0.5 + sl.FreightCostMinor * 0.0003 + sl.CarbonScoreKg * 0.2) ASC"
	default:
		orderClause = "ORDER BY sl.DampenedTransitHours ASC"
	}

	// OBSERVER PROTOCOL: No capacity filter. All active factories are candidates.
	// We fetch CurrentLoad and DailyOutputCapacity for telemetry purposes only.
	sql := `SELECT sl.FactoryId, f.CurrentLoad, f.DailyOutputCapacity
	        FROM SupplyLanes sl
	        JOIN Factories f ON f.FactoryId = sl.FactoryId
	        WHERE sl.SupplierId = @supplierID
	          AND sl.WarehouseId = @warehouseID
	          AND sl.IsActive = TRUE
	          AND f.IsActive = TRUE
	        ` + orderClause + `
	        LIMIT 1`

	stmt := spanner.Statement{
		SQL: sql,
		Params: map[string]interface{}{
			"supplierID":  supplierID,
			"warehouseID": warehouseID,
		},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return &FactorySelection{}, err
	}
	var factoryID string
	var currentLoad, dailyCapacity int64
	if err := row.Columns(&factoryID, &currentLoad, &dailyCapacity); err != nil {
		return &FactorySelection{}, err
	}

	sel := &FactorySelection{FactoryID: factoryID}
	if dailyCapacity == 0 {
		sel.CapacityLabel = "UNLIMITED"
	} else {
		sel.LoadPercent = float64(currentLoad) / float64(dailyCapacity) * 100
		if sel.LoadPercent > 100 {
			sel.IsOverloaded = true
			sel.CapacityLabel = "OVERLOADED"
		} else if sel.LoadPercent > 80 {
			sel.CapacityLabel = "WARNING"
		} else {
			sel.CapacityLabel = "OK"
		}
	}

	return sel, nil
}

// HandleGetNetworkMode returns the current network optimization mode.
// GET /v1/supplier/network-mode
func (s *NetworkOptimizerService) HandleGetNetworkMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	mode, _ := s.GetNetworkMode(r.Context(), claims.ResolveSupplierID())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"mode": mode, "supplier_id": claims.ResolveSupplierID()})
}

// HandleSetNetworkMode updates the supplier's optimization mode.
// PUT /v1/supplier/network-mode
// SOVEREIGN ACTION: Requires GLOBAL_ADMIN supplier role.
func (s *NetworkOptimizerService) HandleSetNetworkMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req struct {
		Mode   string `json:"mode"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	validModes := map[string]bool{
		"SPEED": true, "ECONOMY": true, "BALANCED": true, "LOW_CARBON": true, "MANUAL_ONLY": true,
	}
	if !validModes[req.Mode] {
		http.Error(w, `{"error":"invalid mode — must be SPEED, ECONOMY, BALANCED, LOW_CARBON, or MANUAL_ONLY"}`, http.StatusBadRequest)
		return
	}

	// Get old mode for event
	oldMode, _ := s.GetNetworkMode(r.Context(), supplierID)

	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("NetworkOptimizationMode",
				[]string{"SupplierId", "Mode", "UpdatedAt", "UpdatedBy"},
				[]interface{}{supplierID, req.Mode, spanner.CommitTimestamp, claims.UserID},
			),
		}); err != nil {
			return err
		}

		evt := kafka.NetworkModeChangedEvent{
			SupplierId: supplierID,
			OldMode:    oldMode,
			NewMode:    req.Mode,
			ChangedBy:  claims.UserID,
			Reason:     req.Reason,
			Timestamp:  time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "NetworkOptimizationMode", supplierID, kafka.EventNetworkModeChanged, kafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"old_mode": oldMode,
		"new_mode": req.Mode,
		"status":   "updated",
	})
}

// HandleNetworkAnalytics returns supply lane stats for the optimization dashboard.
// GET /v1/supplier/network-analytics
func (s *NetworkOptimizerService) HandleNetworkAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	mode, _ := s.GetNetworkMode(r.Context(), supplierID)

	// Get supply lane summary
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, WarehouseId, DampenedTransitHours, FreightCostMinor,
		             CarbonScoreKg, IsActive, Priority
		      FROM SupplyLanes WHERE SupplierId = @supplierID
		      ORDER BY Priority DESC`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}

	type laneAnalytic struct {
		FactoryId        string  `json:"factory_id"`
		WarehouseId      string  `json:"warehouse_id"`
		TransitHours     float64 `json:"dampened_transit_hours"`
		FreightCostMinor int64   `json:"freight_cost_minor"`
		CarbonScoreKg    float64 `json:"carbon_score_kg"`
		IsActive         bool    `json:"is_active"`
		Priority         int64   `json:"priority"`
	}

	var lanes []laneAnalytic
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var la laneAnalytic
		if err := row.Columns(&la.FactoryId, &la.WarehouseId, &la.TransitHours,
			&la.FreightCostMinor, &la.CarbonScoreKg, &la.IsActive, &la.Priority); err != nil {
			continue
		}
		lanes = append(lanes, la)
	}

	// Get recent SLA events
	slaStmt := spanner.Statement{
		SQL: `SELECT EventId, TransferId, FactoryId, WarehouseId, EscalationLevel, SLABreachMinutes
		      FROM FactorySLAEvents WHERE SupplierId = @supplierID
		      ORDER BY CreatedAt DESC LIMIT 20`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}

	type slaEvt struct {
		EventId         string `json:"event_id"`
		TransferId      string `json:"transfer_id"`
		FactoryId       string `json:"factory_id"`
		WarehouseId     string `json:"warehouse_id"`
		EscalationLevel string `json:"escalation_level"`
		BreachMinutes   int64  `json:"breach_minutes"`
	}

	var slaEvents []slaEvt
	slaIter := s.Spanner.Single().Query(r.Context(), slaStmt)
	defer slaIter.Stop()
	for {
		row, err := slaIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var se slaEvt
		if err := row.Columns(&se.EventId, &se.TransferId, &se.FactoryId, &se.WarehouseId,
			&se.EscalationLevel, &se.BreachMinutes); err != nil {
			continue
		}
		slaEvents = append(slaEvents, se)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"network_mode": mode,
		"supply_lanes": lanes,
		"sla_events":   slaEvents,
	})
}

// ── JIT Self-Healing: Temporal Load Balancing ─────────────────────────────────
// AtomicIncrementLoad atomically increments a factory's CurrentLoad counter.
// If the stored LastLoadUpdate is from a previous calendar day (in Tashkent TZ),
// it resets CurrentLoad to 0 before incrementing — so pod restarts, clock drift,
// and missed cron ticks can never brick the counter.

// tashkentLocation delegates to the canonical proximity.TashkentLocation.
var tashkentLocation = proximity.TashkentLocation

// AtomicIncrementLoad increments Factories.CurrentLoad by delta inside a
// ReadWriteTransaction. If LastLoadUpdate is stale (different day in Tashkent TZ),
// it resets CurrentLoad to 0 before incrementing and stamps today's date.
func AtomicIncrementLoad(ctx context.Context, client *spanner.Client, factoryID string, delta int64) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Factories", spanner.Key{factoryID},
			[]string{"CurrentLoad", "LastLoadUpdate"})
		if err != nil {
			return fmt.Errorf("read factory %s: %w", factoryID, err)
		}

		var currentLoad spanner.NullInt64
		var lastUpdate spanner.NullDate
		if err := row.Columns(&currentLoad, &lastUpdate); err != nil {
			return fmt.Errorf("scan factory %s: %w", factoryID, err)
		}

		now := time.Now().In(tashkentLocation)
		today := civil.DateOf(now)

		load := currentLoad.Int64
		if !lastUpdate.Valid || lastUpdate.Date != today {
			// New calendar day (or first-ever write) — reset
			log.Printf("[JIT_RESET] Factory %s: resetting CurrentLoad from %d → 0 (date: %v → %v)",
				factoryID, load, lastUpdate.Date, today)
			load = 0
		}

		load += delta

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Factories",
				[]string{"FactoryId", "CurrentLoad", "LastLoadUpdate"},
				[]interface{}{factoryID, load, today}),
		})
	})
	return err
}
