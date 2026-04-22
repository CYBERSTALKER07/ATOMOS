package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/kafka"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Pull Matrix — Automated Replenishment Aggregator ──────────────────────────
// Scans SupplierInventory for SKUs that have breached SafetyStockLevel.
// Groups by factory via SupplyLanes and generates InternalTransferOrders.
//
// Two execution modes:
//   - CRON: full sweep every 4 hours (all suppliers, all warehouses)
//   - EVENT_TRIGGERED: single-SKU recalc on OUT_OF_STOCK Kafka event
//   - MANUAL: POST /v1/admin/replenishment/trigger (existing endpoint delegates here)

// PullMatrixService orchestrates replenishment aggregation.
type PullMatrixService struct {
	Spanner   *spanner.Client
	Producer  *kafkago.Writer
	LockSvc   *ReplenishmentLockService
	Optimizer *NetworkOptimizerService
}

// breachedSKU represents a single SKU that has fallen below its safety stock.
type breachedSKU struct {
	SupplierId  string
	WarehouseId string
	ProductId   string
	CurrentQty  int64
	SafetyLevel int64
	Deficit     int64 // SafetyLevel - CurrentQty (units to request)
}

// RunPullMatrix executes a full sweep of all breached SKUs across all suppliers.
// Called by the 4-hour cron and by the manual trigger endpoint.
func (s *PullMatrixService) RunPullMatrix(ctx context.Context, source string) error {
	startTime := time.Now()

	// 1. Check kill switch — if Mode=MANUAL_ONLY for a supplier, skip automated transfers
	// (Per-supplier filtering happens inside the loop)

	// 2. Find all breached SKUs across all warehouses
	breached, err := s.findBreachedSKUs(ctx, "")
	if err != nil {
		return fmt.Errorf("pull matrix scan failed: %w", err)
	}

	if len(breached) == 0 {
		log.Printf("[PULL_MATRIX] No breached SKUs found — no action needed")
		return nil
	}

	log.Printf("[PULL_MATRIX] Found %d breached SKUs across all warehouses", len(breached))

	// 3. Group breached SKUs by supplier for per-supplier processing
	bySupplier := map[string][]breachedSKU{}
	for _, b := range breached {
		bySupplier[b.SupplierId] = append(bySupplier[b.SupplierId], b)
	}

	var totalTransfers int64
	var totalSKUs int64

	for supplierID, skus := range bySupplier {
		// Check network mode — skip if MANUAL_ONLY
		mode, err := s.Optimizer.GetNetworkMode(ctx, supplierID)
		if err != nil {
			log.Printf("[PULL_MATRIX] Failed to get network mode for %s: %v", supplierID, err)
			continue
		}
		if mode == "MANUAL_ONLY" {
			log.Printf("[PULL_MATRIX] Supplier %s in MANUAL_ONLY mode — skipping", supplierID)
			continue
		}

		transfers, processed, err := s.processSupplierBreaches(ctx, supplierID, skus, source, mode)
		if err != nil {
			log.Printf("[PULL_MATRIX] Error processing supplier %s: %v", supplierID, err)
			continue
		}
		totalTransfers += transfers
		totalSKUs += processed

		// Write audit row
		runID := uuid.New().String()
		durationMs := time.Since(startTime).Milliseconds()
		_, _ = s.Spanner.Apply(ctx, []*spanner.Mutation{
			spanner.Insert("PullMatrixRuns",
				[]string{"RunId", "SupplierId", "RunAt", "TransfersGenerated", "SKUsProcessed", "DurationMs", "Source"},
				[]interface{}{runID, supplierID, spanner.CommitTimestamp, transfers, processed, durationMs, source},
			),
		})
	}

	// Emit completion event
	if s.Producer != nil {
		evt := kafka.PullMatrixCompletedEvent{
			RunId:              uuid.New().String(),
			SupplierId:         "GLOBAL",
			TransfersGenerated: totalTransfers,
			SKUsProcessed:      totalSKUs,
			DurationMs:         time.Since(startTime).Milliseconds(),
			Source:             source,
			Timestamp:          time.Now().UTC(),
		}
		payload, _ := json.Marshal(evt)
		_ = s.Producer.WriteMessages(ctx, kafkago.Message{
			Key:   []byte(kafka.EventPullMatrixCompleted),
			Value: payload,
		})
	}

	log.Printf("[PULL_MATRIX] Completed: %d transfers generated for %d SKUs (source=%s, duration=%dms)",
		totalTransfers, totalSKUs, source, time.Since(startTime).Milliseconds())

	return nil
}

// RunSingleSKU is the event-driven fast path — triggered by OUT_OF_STOCK Kafka event.
// Only recalculates replenishment for one specific SKU at one warehouse.
func (s *PullMatrixService) RunSingleSKU(ctx context.Context, supplierID, warehouseID, productID string) error {
	mode, err := s.Optimizer.GetNetworkMode(ctx, supplierID)
	if err != nil || mode == "MANUAL_ONLY" {
		return nil
	}

	breached, err := s.findBreachedSKUsForProduct(ctx, supplierID, warehouseID, productID)
	if err != nil || len(breached) == 0 {
		return err
	}

	transfers, processed, err := s.processSupplierBreaches(ctx, supplierID, breached, "EVENT_TRIGGERED", mode)
	if err != nil {
		return err
	}

	if transfers > 0 {
		log.Printf("[PULL_MATRIX] Event-triggered: %d transfers for SKU %s at warehouse %s",
			transfers, productID, warehouseID)

		// Audit row
		runID := uuid.New().String()
		_, _ = s.Spanner.Apply(ctx, []*spanner.Mutation{
			spanner.Insert("PullMatrixRuns",
				[]string{"RunId", "SupplierId", "RunAt", "TransfersGenerated", "SKUsProcessed", "DurationMs", "Source", "Notes"},
				[]interface{}{runID, supplierID, spanner.CommitTimestamp, transfers, processed, int64(0), "EVENT_TRIGGERED",
					fmt.Sprintf("SKU=%s WH=%s", productID, warehouseID)},
			),
		})
	}

	return nil
}

// HandleManualPullMatrix is the HTTP handler for POST /v1/supplier/replenishment/pull-matrix.
func (s *PullMatrixService) HandleManualPullMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := s.RunPullMatrix(r.Context(), "MANUAL"); err != nil {
		log.Printf("[PULL_MATRIX] Manual trigger failed: %v", err)
		http.Error(w, `{"error":"pull_matrix_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

// processSupplierBreaches groups breached SKUs by optimal factory and creates transfers.
func (s *PullMatrixService) processSupplierBreaches(ctx context.Context, supplierID string, breached []breachedSKU, source, mode string) (int64, int64, error) {
	// Group items by (warehouseID, factoryID) for efficient transfer creation
	type transferKey struct {
		WarehouseId string
		FactoryId   string
	}
	type transferItem struct {
		ProductId string
		Quantity  int64
	}

	grouped := map[transferKey][]transferItem{}

	for _, b := range breached {
		// ── Look-Ahead Enhancement: upgrade deficit with shadow demand ──
		// Target = max(SafetyLevel, FutureDemand × 1.15)
		// Deficit = Target - CurrentStock
		futureDemand, _, _ := GetFutureDemand(ctx, s.Spanner, b.WarehouseId, b.ProductId, LookAheadWindowDays)
		bufferedDemand := int64(math.Ceil(float64(futureDemand) * (1.0 + SafetyStockBufferPct)))
		target := b.SafetyLevel
		if bufferedDemand > target {
			target = bufferedDemand
		}
		effectiveDeficit := target - b.CurrentQty
		if effectiveDeficit <= 0 {
			continue // stock covers both threshold and shadow demand
		}
		b.Deficit = effectiveDeficit

		// Find optimal factory for this SKU via SupplyLanes
		factory, err := s.Optimizer.SelectOptimalFactory(ctx, supplierID, b.WarehouseId, b.ProductId, mode)
		if err != nil || factory == "" {
			log.Printf("[PULL_MATRIX] No supply lane for %s/%s/%s — skipping", supplierID, b.WarehouseId, b.ProductId)
			continue
		}

		// Try to acquire replenishment lock (concurrency arbitration)
		if s.LockSvc != nil {
			result, err := s.LockSvc.AcquireLock(ctx, supplierID, b.WarehouseId, b.ProductId, factory)
			if err != nil {
				log.Printf("[PULL_MATRIX] Lock acquisition error: %v", err)
				continue
			}
			if !result.Acquired {
				log.Printf("[PULL_MATRIX] Lock denied for %s/%s — held by %s (priority %.1f > %.1f)",
					b.WarehouseId, b.ProductId, result.HeldBy, result.HeldPriority, result.Priority)
				continue
			}
		}

		key := transferKey{WarehouseId: b.WarehouseId, FactoryId: factory}
		grouped[key] = append(grouped[key], transferItem{
			ProductId: b.ProductId,
			Quantity:  b.Deficit,
		})
	}

	var transferCount int64
	for key, items := range grouped {
		transferID := uuid.New().String()
		var totalVolumeVU float64
		mutations := []*spanner.Mutation{}

		// Header
		mutations = append(mutations, spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State",
				"TotalVolumeVU", "Source", "CreatedAt"},
			[]interface{}{transferID, key.FactoryId, key.WarehouseId, supplierID, "DRAFT",
				totalVolumeVU, sourceToDBValue(source), spanner.CommitTimestamp},
		))

		for _, item := range items {
			itemID := uuid.New().String()
			// Real volumetric load from product catalog (LWH or VolumetricUnit)
			vuPerUnit := fetchProductVU(ctx, s.Spanner, supplierID, item.ProductId)
			volVU := float64(item.Quantity) * vuPerUnit
			totalVolumeVU += volVU
			mutations = append(mutations, spanner.Insert("InternalTransferItems",
				[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
				[]interface{}{transferID, itemID, item.ProductId, item.Quantity, volVU},
			))
		}

		if _, err := s.Spanner.Apply(ctx, mutations); err != nil {
			log.Printf("[PULL_MATRIX] Failed to create transfer %s: %v", transferID, err)
			continue
		}

		// Increment factory CurrentLoad via JIT self-healing (date-aware reset)
		if err := AtomicIncrementLoad(ctx, s.Spanner, key.FactoryId, 1); err != nil {
			log.Printf("[PULL_MATRIX] CurrentLoad increment failed for factory %s: %v", key.FactoryId, err)
		}

		transferCount++
		log.Printf("[PULL_MATRIX] Created transfer %s: factory=%s → warehouse=%s (%d items)",
			transferID[:8], key.FactoryId[:8], key.WarehouseId[:8], len(items))
	}

	return transferCount, int64(len(breached)), nil
}

// findBreachedSKUs returns all inventory rows where current stock <= safety level.
// If supplierID is empty, scans all suppliers.
func (s *PullMatrixService) findBreachedSKUs(ctx context.Context, supplierID string) ([]breachedSKU, error) {
	sql := `SELECT si.SupplierId, si.WarehouseId, si.ProductId,
	               si.QuantityAvailable, si.SafetyStockLevel
	        FROM SupplierInventory si
	        WHERE si.SafetyStockLevel > 0
	          AND si.QuantityAvailable <= si.SafetyStockLevel`
	params := map[string]interface{}{}

	if supplierID != "" {
		sql += ` AND si.SupplierId = @supplierID`
		params["supplierID"] = supplierID
	}

	sql += ` LIMIT 1000`

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []breachedSKU
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var b breachedSKU
		var warehouseID spanner.NullString
		if err := row.Columns(&b.SupplierId, &warehouseID, &b.ProductId, &b.CurrentQty, &b.SafetyLevel); err != nil {
			continue
		}
		b.WarehouseId = warehouseID.StringVal
		if b.WarehouseId == "" {
			continue // Skip inventory without warehouse assignment
		}
		b.Deficit = b.SafetyLevel - b.CurrentQty
		if b.Deficit <= 0 {
			continue
		}
		results = append(results, b)
	}

	return results, nil
}

// findBreachedSKUsForProduct returns breached inventory for a specific product at a warehouse.
func (s *PullMatrixService) findBreachedSKUsForProduct(ctx context.Context, supplierID, warehouseID, productID string) ([]breachedSKU, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, WarehouseId, ProductId, QuantityAvailable, SafetyStockLevel
		      FROM SupplierInventory
		      WHERE SupplierId = @supplierID
		        AND WarehouseId = @warehouseID
		        AND ProductId = @productID
		        AND SafetyStockLevel > 0
		        AND QuantityAvailable <= SafetyStockLevel`,
		Params: map[string]interface{}{
			"supplierID":  supplierID,
			"warehouseID": warehouseID,
			"productID":   productID,
		},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []breachedSKU
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var b breachedSKU
		var whID spanner.NullString
		if err := row.Columns(&b.SupplierId, &whID, &b.ProductId, &b.CurrentQty, &b.SafetyLevel); err != nil {
			continue
		}
		b.WarehouseId = whID.StringVal
		b.Deficit = b.SafetyLevel - b.CurrentQty
		if b.Deficit > 0 {
			results = append(results, b)
		}
	}
	return results, nil
}

func sourceToDBValue(source string) string {
	switch source {
	case "CRON", "EVENT_TRIGGERED":
		return "SYSTEM_THRESHOLD"
	case "MANUAL":
		return "MANUAL_EMERGENCY"
	default:
		return "SYSTEM_THRESHOLD"
	}
}
