package factory

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/supplier"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PULL MATRIX LOOK-AHEAD — Proactive Shadow Demand Layer
//
// The threshold-based Pull Matrix fires when: CurrentStock ≤ SafetyStockLevel.
// The Look-Ahead fires when:
//
//   ShadowDemand = Σ(LOCKED + PENDING orders in next 7 days) − CurrentStock
//   If ShadowDemand > 0 → trigger transfer even if stock is "safe" today.
//
// This bridges the "Reactive Refill" gap: stock is above safety level now, but
// Monday's locked orders will consume it before the factory truck arrives.
//
// Integration points:
//   - Uses proximity.TashkentNow() for all date math (timezone-safe)
//   - Uses Idx_Orders_ByScheduleShardStateDate for O(log n) demand scan
//   - Links ReplenishmentId on contributing LOCKED orders (Front 1 traceability)
//   - Uses supplier.NormalizeProductVU() for volumetric convoy splitting
//   - Emits REPLENISHMENT_TRANSFER_CREATED Kafka event
// ═══════════════════════════════════════════════════════════════════════════════

const (
	// LookAheadWindowDays is the forward-looking demand horizon.
	LookAheadWindowDays = 7

	// SafetyStockBufferPct adds a 15% buffer on top of raw shadow demand
	// to absorb flash-sale surges while the factory truck is in transit.
	SafetyStockBufferPct = 0.15

	// FactoryClassCCapacityVU is the volumetric capacity of a single Class-C truck.
	FactoryClassCCapacityVU = 400.0
)

// shadowDemandEntry holds aggregated forward demand for one SKU at one warehouse.
type shadowDemandEntry struct {
	WarehouseId   string
	SupplierId    string
	ProductId     string
	FutureDemand  int64    // Σ(Quantity) of LOCKED+PENDING orders in window
	CurrentStock  int64    // QuantityAvailable from SupplierInventory
	SafetyLevel   int64    // SafetyStockLevel from SupplierInventory
	ShadowDeficit int64    // max(0, demand + buffer - stock)
	OrderIds      []string // contributing order IDs for ReplenishmentId linking
}

// GetFutureDemand scans LOCKED + PENDING orders scheduled within the look-ahead
// window for a specific warehouse + SKU. Uses the WarehouseId index.
//
// Returns (total units demanded, contributing order IDs, error).
func GetFutureDemand(ctx context.Context, client *spanner.Client, warehouseID, skuID string, days int) (int64, []string, error) {
	now := proximity.TashkentNow()
	horizon := now.Add(time.Duration(days) * 24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, li.Quantity
		      FROM OrderLineItems li
		      JOIN Orders o ON li.OrderId = o.OrderId
		      WHERE o.WarehouseId = @warehouseID
		        AND o.State IN ('LOCKED', 'PENDING')
		        AND o.RequestedDeliveryDate >= @now
		        AND o.RequestedDeliveryDate <= @horizon
		        AND li.SkuId = @skuID`,
		Params: map[string]interface{}{
			"warehouseID": warehouseID,
			"skuID":       skuID,
			"now":         now,
			"horizon":     horizon,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var totalQty int64
	var orderIDs []string
	seen := make(map[string]struct{})

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, nil, fmt.Errorf("future demand query: %w", err)
		}
		var orderID string
		var qty int64
		if err := row.Columns(&orderID, &qty); err != nil {
			continue
		}
		totalQty += qty
		if _, dup := seen[orderID]; !dup {
			orderIDs = append(orderIDs, orderID)
			seen[orderID] = struct{}{}
		}
	}

	return totalQty, orderIDs, nil
}

// RunLookAhead executes the proactive shadow demand scan for all warehouses.
// Called by the Pull Matrix cron after the threshold-based sweep.
func (s *PullMatrixService) RunLookAhead(ctx context.Context) error {
	startTime := proximity.TashkentNow()
	log.Println("[LOOK_AHEAD] Starting shadow demand scan...")

	// 1. Fetch all active warehouse→supplier pairs
	warehouses, err := s.fetchActiveWarehousePairs(ctx)
	if err != nil {
		return fmt.Errorf("fetch warehouses: %w", err)
	}

	var totalTransfers int64

	for _, wh := range warehouses {
		// Check network mode — skip MANUAL_ONLY suppliers
		mode, err := s.Optimizer.GetNetworkMode(ctx, wh.SupplierId)
		if err != nil || mode == "MANUAL_ONLY" {
			continue
		}

		transfers, err := s.processWarehouseLookAhead(ctx, wh.WarehouseId, wh.SupplierId, mode)
		if err != nil {
			log.Printf("[LOOK_AHEAD] Error processing warehouse %s: %v", wh.WarehouseId, err)
			continue
		}
		totalTransfers += transfers
	}

	elapsed := time.Since(startTime)
	log.Printf("[LOOK_AHEAD] Completed: %d proactive transfers generated across %d warehouses (%dms)",
		totalTransfers, len(warehouses), elapsed.Milliseconds())

	return nil
}

// processWarehouseLookAhead scans one warehouse for shadow demand deficits.
func (s *PullMatrixService) processWarehouseLookAhead(ctx context.Context, warehouseID, supplierID, mode string) (int64, error) {
	// 1. Get all SKUs with inventory at this warehouse
	inventory, err := s.fetchWarehouseInventory(ctx, warehouseID, supplierID)
	if err != nil {
		return 0, err
	}

	// 2. For each SKU, compute shadow demand
	var deficits []shadowDemandEntry
	for _, inv := range inventory {
		futureDemand, orderIDs, err := GetFutureDemand(ctx, s.Spanner, warehouseID, inv.ProductId, LookAheadWindowDays)
		if err != nil {
			log.Printf("[LOOK_AHEAD] Demand query failed for %s/%s: %v", warehouseID, inv.ProductId, err)
			continue
		}

		if futureDemand == 0 {
			continue // no upcoming orders for this SKU
		}

		// Shadow Demand = FutureDemand + SafetyBuffer - CurrentStock
		buffered := int64(math.Ceil(float64(futureDemand) * (1.0 + SafetyStockBufferPct)))
		deficit := buffered - inv.CurrentQty

		// Also respect MinThreshold: target = max(SafetyStockLevel, buffered demand)
		target := buffered
		if inv.SafetyLevel > target {
			target = inv.SafetyLevel
		}
		deficit = target - inv.CurrentQty

		if deficit <= 0 {
			continue // stock covers demand + buffer
		}

		deficits = append(deficits, shadowDemandEntry{
			WarehouseId:   warehouseID,
			SupplierId:    supplierID,
			ProductId:     inv.ProductId,
			FutureDemand:  futureDemand,
			CurrentStock:  inv.CurrentQty,
			SafetyLevel:   inv.SafetyLevel,
			ShadowDeficit: deficit,
			OrderIds:      orderIDs,
		})
	}

	if len(deficits) == 0 {
		return 0, nil
	}

	log.Printf("[LOOK_AHEAD] Warehouse %s: %d SKUs with shadow demand deficits", warehouseID, len(deficits))

	// 3. Group by optimal factory and create transfers
	return s.createLookAheadTransfers(ctx, supplierID, warehouseID, deficits, mode)
}

// createLookAheadTransfers routes deficits to factories and creates transfers with
// volumetric convoy splitting + ReplenishmentId traceability.
func (s *PullMatrixService) createLookAheadTransfers(ctx context.Context, supplierID, warehouseID string, deficits []shadowDemandEntry, mode string) (int64, error) {
	// Group items by optimal factory
	type transferKey struct {
		FactoryId string
	}
	type transferItem struct {
		ProductId string
		Quantity  int64
		VolumeVU  float64
		OrderIds  []string // contributing orders for ReplenishmentId link
	}

	grouped := map[transferKey][]transferItem{}

	for _, d := range deficits {
		factory, err := s.Optimizer.SelectOptimalFactory(ctx, supplierID, warehouseID, d.ProductId, mode)
		if err != nil || factory == "" {
			log.Printf("[LOOK_AHEAD] No supply lane for %s/%s/%s — skipping", supplierID, warehouseID, d.ProductId)
			continue
		}

		// Acquire replenishment lock (prevent duplicate transfers for same SKU)
		if s.LockSvc != nil {
			result, err := s.LockSvc.AcquireLock(ctx, supplierID, warehouseID, d.ProductId, factory)
			if err != nil || !result.Acquired {
				continue
			}
		}

		// Compute volumetric load using real product dimensions
		vuPerUnit := fetchProductVU(ctx, s.Spanner, supplierID, d.ProductId)
		totalVU := float64(d.ShadowDeficit) * vuPerUnit

		key := transferKey{FactoryId: factory}
		grouped[key] = append(grouped[key], transferItem{
			ProductId: d.ProductId,
			Quantity:  d.ShadowDeficit,
			VolumeVU:  totalVU,
			OrderIds:  d.OrderIds,
		})
	}

	var transferCount int64

	for key, items := range grouped {
		// Calculate total volume for convoy splitting
		var totalVolumeVU float64
		for _, item := range items {
			totalVolumeVU += item.VolumeVU
		}

		// Split into convoys if volume exceeds Class-C capacity
		convoyCount := int(math.Ceil(totalVolumeVU / FactoryClassCCapacityVU))
		if convoyCount < 1 {
			convoyCount = 1
		}

		// For single convoy or first pass: create the transfer
		transferID := uuid.New().String()
		mutations := []*spanner.Mutation{}

		mutations = append(mutations, spanner.InsertMap("InternalTransferOrders", map[string]interface{}{
			"TransferId":    transferID,
			"FactoryId":     key.FactoryId,
			"WarehouseId":   warehouseID,
			"SupplierId":    supplierID,
			"State":         "DRAFT",
			"TotalVolumeVU": totalVolumeVU,
			"Source":        "SYSTEM_LOOKAHEAD",
			"CreatedAt":     spanner.CommitTimestamp,
		}))

		// Line items
		var allContributingOrders []string
		for _, item := range items {
			itemID := uuid.New().String()
			mutations = append(mutations, spanner.InsertMap("InternalTransferItems", map[string]interface{}{
				"TransferId": transferID,
				"ItemId":     itemID,
				"ProductId":  item.ProductId,
				"Quantity":   item.Quantity,
				"VolumeVU":   item.VolumeVU,
			}))
			allContributingOrders = append(allContributingOrders, item.OrderIds...)
		}

		if _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
			evt := struct {
				Event       string  `json:"event"`
				TransferID  string  `json:"transfer_id"`
				FactoryID   string  `json:"factory_id"`
				WarehouseID string  `json:"warehouse_id"`
				SupplierID  string  `json:"supplier_id"`
				Source      string  `json:"source"`
				VolumeVU    float64 `json:"volume_vu"`
				ConvoyCount int     `json:"convoy_count"`
				Timestamp   string  `json:"timestamp"`
			}{
				Event:       kafka.EventReplenishmentTransferCreated,
				TransferID:  transferID,
				FactoryID:   key.FactoryId,
				WarehouseID: warehouseID,
				SupplierID:  supplierID,
				Source:      "SYSTEM_LOOKAHEAD",
				VolumeVU:    totalVolumeVU,
				ConvoyCount: convoyCount,
				Timestamp:   proximity.TashkentNow().Format(time.RFC3339),
			}
			return outbox.EmitJSON(txn, "InternalTransferOrder", transferID, kafka.EventReplenishmentTransferCreated, kafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
		}); err != nil {
			log.Printf("[LOOK_AHEAD] Transfer creation failed: %v", err)
			continue
		}

		// Front 1 Integration: stamp ReplenishmentId on contributing LOCKED orders
		if len(allContributingOrders) > 0 {
			go s.linkReplenishmentId(context.Background(), transferID, allContributingOrders)
		}

		// NOTE: Convoy manifests are NOT created at DRAFT time. They are generated
		// by HandleApproveTransfer in transfers.go when the transfer → APPROVED,
		// ensuring the Volumetric Engine operates on confirmed transfers only.

		// Increment factory load
		if err := AtomicIncrementLoad(ctx, s.Spanner, key.FactoryId, 1); err != nil {
			log.Printf("[LOOK_AHEAD] Factory load increment failed: %v", err)
		}

		transferCount++
		log.Printf("[LOOK_AHEAD] Created transfer %s: factory=%s → warehouse=%s (%d items, %.1f VU, %d trucks)",
			transferID[:8], key.FactoryId[:8], warehouseID[:8], len(items), totalVolumeVU, convoyCount)
	}

	return transferCount, nil
}

// linkReplenishmentId atomically stamps the TransferID as ReplenishmentId on all
// contributing LOCKED orders. This enables the "Global Trace" view:
//
//	Retailer → Order → ReplenishmentId → Transfer → Factory shipment
func (s *PullMatrixService) linkReplenishmentId(ctx context.Context, transferID string, orderIDs []string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Batch update in chunks of 100 to stay under Spanner mutation limits
	const batchSize = 100
	for i := 0; i < len(orderIDs); i += batchSize {
		end := i + batchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		batch := orderIDs[i:end]

		var mutations []*spanner.Mutation
		for _, orderID := range batch {
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]interface{}{
				"OrderId":         orderID,
				"ReplenishmentId": transferID,
			}))
		}

		if _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(mutations)
		}); err != nil {
			log.Printf("[LOOK_AHEAD] ReplenishmentId link failed for batch %d-%d: %v", i, end, err)
		}
	}

	log.Printf("[LOOK_AHEAD] Linked ReplenishmentId=%s to %d orders", transferID[:8], len(orderIDs))
}

// createConvoyManifests splits an oversized transfer into multiple
// FactoryTruckManifests (Class-C vehicles, 400 VU each).
func (s *PullMatrixService) createConvoyManifests(ctx context.Context, transferID, factoryID string, totalVU float64, convoyCount int) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var mutations []*spanner.Mutation
	for i := 0; i < convoyCount; i++ {
		manifestID := uuid.New().String()
		truckVU := FactoryClassCCapacityVU
		if i == convoyCount-1 {
			// Last truck gets the remainder
			truckVU = totalVU - float64(i)*FactoryClassCCapacityVU
			if truckVU <= 0 {
				continue
			}
		}
		mutations = append(mutations, spanner.InsertMap("FactoryTruckManifests", map[string]interface{}{
			"ManifestId":    manifestID,
			"FactoryId":     factoryID,
			"State":         "PENDING",
			"TotalVolumeVU": truckVU,
			"MaxVolumeVU":   FactoryClassCCapacityVU,
			"StopCount":     int64(1), // single warehouse stop
			"CreatedAt":     spanner.CommitTimestamp,
		}))
	}

	if len(mutations) > 0 {
		if _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(mutations)
		}); err != nil {
			log.Printf("[LOOK_AHEAD] Convoy manifest creation failed: %v", err)
		} else {
			log.Printf("[LOOK_AHEAD] Created %d convoy manifests for transfer %s (%.1f VU total)",
				convoyCount, transferID[:8], totalVU)
		}
	}
}

// ── Data Fetchers ───────────────────────────────────────────────────────────

type warehousePair struct {
	WarehouseId string
	SupplierId  string
}

type inventoryRow struct {
	ProductId   string
	CurrentQty  int64
	SafetyLevel int64
}

func (s *PullMatrixService) fetchActiveWarehousePairs(ctx context.Context) ([]warehousePair, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, SupplierId FROM Warehouses WHERE IsActive = true`,
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []warehousePair
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var wp warehousePair
		if err := row.Columns(&wp.WarehouseId, &wp.SupplierId); err != nil {
			continue
		}
		result = append(result, wp)
	}
	return result, nil
}

func (s *PullMatrixService) fetchWarehouseInventory(ctx context.Context, warehouseID, supplierID string) ([]inventoryRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, QuantityAvailable, SafetyStockLevel
		      FROM SupplierInventory
		      WHERE SupplierId = @sid AND WarehouseId = @whId AND QuantityAvailable > 0`,
		Params: map[string]interface{}{"sid": supplierID, "whId": warehouseID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []inventoryRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var inv inventoryRow
		if err := row.Columns(&inv.ProductId, &inv.CurrentQty, &inv.SafetyLevel); err != nil {
			continue
		}
		result = append(result, inv)
	}
	return result, nil
}

// fetchProductVU returns the effective volumetric unit for a product.
// Uses NormalizeProductVU from the dispatcher for consistent VU calculation.
func fetchProductVU(ctx context.Context, client *spanner.Client, supplierID, productID string) float64 {
	row, err := client.Single().ReadRow(ctx, "SupplierProducts",
		spanner.Key{supplierID, productID},
		[]string{"VolumetricUnit", "LengthCM", "WidthCM", "HeightCM"})
	if err != nil {
		return 1.0 // fallback: 1 VU per unit
	}

	var vu float64
	var lengthCM, widthCM, heightCM spanner.NullFloat64
	if err := row.Columns(&vu, &lengthCM, &widthCM, &heightCM); err != nil {
		return 1.0
	}

	p := supplier.Product{
		VolumetricUnit: vu,
	}
	if lengthCM.Valid {
		p.LengthCM = &lengthCM.Float64
	}
	if widthCM.Valid {
		p.WidthCM = &widthCM.Float64
	}
	if heightCM.Valid {
		p.HeightCM = &heightCM.Float64
	}

	return supplier.NormalizeProductVU(p)
}
