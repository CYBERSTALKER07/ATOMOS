package factory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Predictive Push — AI Predictions → Factory Transfers Bridge ───────────────
// Reads from AIPredictions table (written by the AI Worker) and generates
// SYSTEM_PREDICTED InternalTransferOrders when a projected stock breach is
// detected within the supplier's SafetyStockDays horizon.
//
// Runs inside the PullMatrix cron (piggybacked) OR via manual endpoint.
// Does NOT replace the threshold-based Pull Matrix — it augments it by acting
// BEFORE stock actually breaches safety level.

// PredictivePushService generates preemptive transfers from AI forecasts.
type PredictivePushService struct {
	Spanner   *spanner.Client
	Producer  *kafkago.Writer
	Optimizer *NetworkOptimizerService
}

// AIPredictionRow represents aggregated demand from AIPredictions + AIPredictionItems.
type AIPredictionRow struct {
	PredictionId string
	ProductId    string // SkuId from AIPredictionItems
	Quantity     int64  // PredictedQuantity from AIPredictionItems
}

// RunPredictivePush scans AIPredictions for imminent demand and creates preemptive transfers.
func (s *PredictivePushService) RunPredictivePush(ctx context.Context, supplierID string) (int64, error) {
	// 1. Get supplier's safety stock horizon
	safetyDays := int64(3) // default

	// Query warehouses for this supplier to find SafetyStockDays
	whStmt := spanner.Statement{
		SQL:    `SELECT SafetyStockDays FROM Warehouses WHERE SupplierId = @supplierID AND IsActive = TRUE LIMIT 1`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}
	whIter := s.Spanner.Single().Query(ctx, whStmt)
	whRow, err := whIter.Next()
	if err == nil {
		var sd spanner.NullInt64
		if err := whRow.Columns(&sd); err == nil && sd.Valid && sd.Int64 > 0 {
			safetyDays = sd.Int64
		}
	}
	whIter.Stop()

	// 2. Query WAITING AIPredictions within the horizon, joined with items and retailer→supplier mapping
	// AIPredictions are retailer-scoped. We find predictions for retailers that order from this supplier,
	// then aggregate their predicted SKU demand.
	stmt := spanner.Statement{
		SQL: `SELECT p.PredictionId, pi.SkuId, pi.PredictedQuantity
		      FROM AIPredictions p
		      JOIN AIPredictionItems pi ON pi.PredictionId = p.PredictionId
		      JOIN RetailerSuppliers rs ON rs.RetailerId = p.RetailerId
		      WHERE rs.SupplierId = @supplierID
		        AND p.Status = 'WAITING'
		        AND p.TriggerDate IS NOT NULL
		        AND p.TriggerDate <= TIMESTAMP_ADD(CURRENT_TIMESTAMP(), INTERVAL @horizonDays DAY)`,
		Params: map[string]interface{}{
			"supplierID":  supplierID,
			"horizonDays": safetyDays,
		},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	// Aggregate demand by SKU (SkuId = ProductId in our context)
	type skuDemand struct {
		PredictionIds []string
		TotalQty      int64
	}
	demandBySKU := map[string]*skuDemand{}

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}

		var predID, skuID string
		var qty int64
		if err := row.Columns(&predID, &skuID, &qty); err != nil {
			continue
		}

		if d, ok := demandBySKU[skuID]; ok {
			d.TotalQty += qty
			d.PredictionIds = append(d.PredictionIds, predID)
		} else {
			demandBySKU[skuID] = &skuDemand{
				PredictionIds: []string{predID},
				TotalQty:      qty,
			}
		}
	}

	if len(demandBySKU) == 0 {
		return 0, nil
	}

	// 3. Cross-reference with current inventory to find projected breaches
	var transferCount int64
	for skuID, demand := range demandBySKU {
		// Find warehouses with this product
		invStmt := spanner.Statement{
			SQL: `SELECT WarehouseId, QuantityAvailable, SafetyStockLevel
			      FROM SupplierInventory
			      WHERE SupplierId = @supplierID AND ProductId = @productID
			        AND SafetyStockLevel > 0`,
			Params: map[string]interface{}{
				"supplierID": supplierID,
				"productID":  skuID,
			},
		}

		invIter := s.Spanner.Single().Query(ctx, invStmt)
		for {
			invRow, err := invIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}

			var warehouseID spanner.NullString
			var currentQty, safetyLevel int64
			if err := invRow.Columns(&warehouseID, &currentQty, &safetyLevel); err != nil {
				continue
			}
			whID := warehouseID.StringVal
			if whID == "" {
				continue
			}

			// Project: after predicted demand, will stock breach safety level?
			projectedQty := currentQty - demand.TotalQty
			if projectedQty > safetyLevel {
				continue // Still above safety — no action needed
			}

			deficit := safetyLevel - projectedQty
			if deficit <= 0 {
				deficit = demand.TotalQty // At minimum, replenish predicted demand
			}

			// Get optimal factory
			mode, _ := s.Optimizer.GetNetworkMode(ctx, supplierID)
			if mode == "MANUAL_ONLY" {
				continue
			}
			factory, err := s.Optimizer.SelectOptimalFactory(ctx, supplierID, whID, skuID, mode)
			if err != nil || factory == "" {
				continue
			}

			// Create preemptive transfer
			transferID := uuid.New().String()
			itemID := uuid.New().String()
			volVU := float64(deficit) * 0.01

			_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Insert("InternalTransferOrders",
						[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State",
							"TotalVolumeVU", "Source", "CreatedAt"},
						[]interface{}{transferID, factory, whID, supplierID, "DRAFT",
							volVU, "SYSTEM_PREDICTED", spanner.CommitTimestamp},
					),
					spanner.Insert("InternalTransferItems",
						[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
						[]interface{}{transferID, itemID, skuID, deficit, volVU},
					),
				})
			})
			if err != nil {
				log.Printf("[PREDICTIVE_PUSH] Failed to create transfer for %s: %v", skuID, err)
				continue
			}

			// Increment factory CurrentLoad via JIT self-healing (date-aware reset)
			if err := AtomicIncrementLoad(ctx, s.Spanner, factory, 1); err != nil {
				log.Printf("[PREDICTIVE_PUSH] CurrentLoad increment failed for factory %s: %v", factory, err)
			}

			transferCount++
			log.Printf("[PREDICTIVE_PUSH] Created preemptive transfer %s: factory=%s → WH=%s for SKU=%s (deficit=%d)",
				transferID[:8], factory[:8], whID[:8], skuID, deficit)
		}
		invIter.Stop()
	}

	return transferCount, nil
}

// HandleManualPredictivePush triggers a predictive push scan for the authenticated supplier.
func (s *PredictivePushService) HandleManualPredictivePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	count, err := s.RunPredictivePush(r.Context(), claims.ResolveSupplierID())
	if err != nil {
		log.Printf("[PREDICTIVE_PUSH] Manual trigger failed: %v", err)
		http.Error(w, `{"error":"predictive_push_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfers_created": count,
		"source":            "MANUAL_PREDICTIVE",
	})
}
