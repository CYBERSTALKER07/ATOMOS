package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ReconcileService handles depot-level reverse-logistics reconciliation.
//
// Two surfaces:
//   - GET  /v1/supplier/quarantine-stock        — list QUARANTINE vehicles + orders + unresolved items
//   - POST /v1/inventory/reconcile-returns      — bulk RESTOCK or WRITE_OFF a vehicle's returned load
type ReconcileService struct {
	Client   *spanner.Client
	Producer *kafka.Writer
}

func NewReconcileService(client *spanner.Client, producer *kafka.Writer) *ReconcileService {
	return &ReconcileService{Client: client, Producer: producer}
}

// ── DTOs ────────────────────────────────────────────────────────────────────

type QuarantineLineItem struct {
	LineItemID   string `json:"line_item_id"`
	SkuID        string `json:"sku_id"`
	ProductName  string `json:"product_name"`
	Quantity     int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

type QuarantineOrder struct {
	OrderID      string               `json:"order_id"`
	RetailerName string               `json:"retailer_name"`
	Items        []QuarantineLineItem `json:"items"`
}

type QuarantineVehicle struct {
	VehicleID    string            `json:"vehicle_id"`
	VehicleClass string            `json:"vehicle_class"`
	DriverName   string            `json:"driver_name"`
	RouteID      string            `json:"route_id"`
	Orders       []QuarantineOrder `json:"orders"`
}

type ReconcileRequest struct {
	LineItemIDs []string `json:"line_item_ids"`
	Action      string   `json:"action"` // RESTOCK | WRITE_OFF_DAMAGED
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// HandleQuarantineStock — GET /v1/supplier/quarantine-stock
// Returns vehicles that have QUARANTINE orders with unresolved REJECTED_DAMAGED line items.
func (s *ReconcileService) HandleQuarantineStock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	sql := `SELECT
	            COALESCE(v.VehicleId, ''), COALESCE(v.VehicleClass, ''),
	            COALESCE(d.Name, 'Unknown Driver'),
	            COALESCE(o.RouteId, ''), o.OrderId,
	            COALESCE(ret.Name, ''),
	            li.LineItemId, li.SkuId, sp.Name, li.Quantity, li.UnitPrice
	        FROM Orders o
	        JOIN Drivers d ON o.DriverId = d.DriverId
	        LEFT JOIN Vehicles v ON d.VehicleId = v.VehicleId
	        JOIN OrderLineItems li ON o.OrderId = li.OrderId
	        JOIN SupplierProducts sp ON li.SkuId = sp.SkuId AND sp.SupplierId = o.SupplierId
	        LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
	        WHERE o.SupplierId = @sid
	          AND o.State = 'QUARANTINE'
	          AND li.Status = 'REJECTED_DAMAGED'`

	params := map[string]interface{}{"sid": supplierID}

	// Apply warehouse scope if present
	if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
		sql += " AND o.WarehouseId = @warehouseId"
		params["warehouseId"] = whID
	}

	sql += " ORDER BY v.VehicleId, o.OrderId, li.LineItemId"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	// Assemble vehicle → order → line item hierarchy
	vehicleMap := make(map[string]*QuarantineVehicle)
	var vehicleOrder []string // preserve insertion order for deterministic output

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[QUARANTINE STOCK] Query error: %v", err)
			break
		}

		var vehicleID, vehicleClass, driverName, routeID, orderID, retailerName string
		var lineItemID, skuID, productName string
		var quantity, unitPrice int64

		if err := row.Columns(
			&vehicleID, &vehicleClass, &driverName, &routeID, &orderID, &retailerName,
			&lineItemID, &skuID, &productName, &quantity, &unitPrice,
		); err != nil {
			log.Printf("[QUARANTINE STOCK] Row parse error: %v", err)
			continue
		}

		vehicleKey := vehicleID
		if vehicleKey == "" {
			vehicleKey = "NO_VEHICLE"
		}

		if vehicleMap[vehicleKey] == nil {
			vehicleMap[vehicleKey] = &QuarantineVehicle{
				VehicleID:    vehicleID,
				VehicleClass: vehicleClass,
				DriverName:   driverName,
				RouteID:      routeID,
				Orders:       []QuarantineOrder{},
			}
			vehicleOrder = append(vehicleOrder, vehicleKey)
		}

		veh := vehicleMap[vehicleKey]

		var targetOrder *QuarantineOrder
		for i := range veh.Orders {
			if veh.Orders[i].OrderID == orderID {
				targetOrder = &veh.Orders[i]
				break
			}
		}
		if targetOrder == nil {
			veh.Orders = append(veh.Orders, QuarantineOrder{
				OrderID:      orderID,
				RetailerName: retailerName,
				Items:        []QuarantineLineItem{},
			})
			targetOrder = &veh.Orders[len(veh.Orders)-1]
		}

		targetOrder.Items = append(targetOrder.Items, QuarantineLineItem{
			LineItemID:   lineItemID,
			SkuID:        skuID,
			ProductName:  productName,
			Quantity:     quantity,
			UnitPrice: unitPrice,
		})
	}

	result := make([]QuarantineVehicle, 0, len(vehicleOrder))
	for _, k := range vehicleOrder {
		result = append(result, *vehicleMap[k])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": result})
}

// HandleReconcile — POST /v1/inventory/reconcile-returns
// Bulk-resolves a set of REJECTED_DAMAGED line items returned to the depot.
func (s *ReconcileService) HandleReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req ReconcileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if len(req.LineItemIDs) == 0 {
		http.Error(w, `{"error":"line_item_ids required"}`, http.StatusBadRequest)
		return
	}
	if req.Action != "RESTOCK" && req.Action != "WRITE_OFF_DAMAGED" {
		http.Error(w, `{"error":"action must be RESTOCK or WRITE_OFF_DAMAGED"}`, http.StatusBadRequest)
		return
	}

	newStatus := "WRITE_OFF"
	if req.Action == "RESTOCK" {
		newStatus = "RETURNED_TO_STOCK"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	var resolvedCount int64

	_, txnErr := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify supplier ownership and collect records for mutation building
		verifyStmt := spanner.Statement{
			SQL: `SELECT li.LineItemId, li.SkuId, li.Quantity, o.SupplierId
			      FROM OrderLineItems li
			      JOIN Orders o ON li.OrderId = o.OrderId
			      WHERE li.LineItemId IN UNNEST(@ids)
			        AND li.Status = 'REJECTED_DAMAGED'`,
			Params: map[string]interface{}{"ids": req.LineItemIDs},
		}
		verifyIter := txn.Query(ctx, verifyStmt)
		defer verifyIter.Stop()

		type itemRecord struct {
			lineItemID string
			skuID      string
			qty        int64
		}
		var records []itemRecord

		for {
			row, err := verifyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("verify query error: %w", err)
			}
			var rec itemRecord
			var ownerSupplier string
			if err := row.Columns(&rec.lineItemID, &rec.skuID, &rec.qty, &ownerSupplier); err != nil {
				return err
			}
			if ownerSupplier != supplierID {
				return fmt.Errorf("access denied: item %s belongs to a different supplier", rec.lineItemID)
			}
			records = append(records, rec)
		}

		if len(records) == 0 {
			return fmt.Errorf("no eligible REJECTED_DAMAGED items found")
		}

		muts := make([]*spanner.Mutation, 0, len(records))
		for _, rec := range records {
			muts = append(muts, spanner.Update("OrderLineItems",
				[]string{"LineItemId", "Status"},
				[]interface{}{rec.lineItemID, newStatus},
			))
		}

		// For RESTOCK, restore inventory quantities on the SupplierInventory table
		if req.Action == "RESTOCK" {
			skuQtyMap := make(map[string]int64)
			for _, rec := range records {
				skuQtyMap[rec.skuID] += rec.qty
			}
			for skuID, qty := range skuQtyMap {
				invRow, err := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{skuID}, []string{"QuantityAvailable"})
				if err != nil {
					return fmt.Errorf("RESTOCK blocked: cannot read inventory for SKU %s: %w", skuID, err)
				}
				var current int64
				invRow.Columns(&current)
				muts = append(muts, spanner.Update("SupplierInventory",
					[]string{"ProductId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{skuID, current + qty, spanner.CommitTimestamp},
				))
			}
		}

		txn.BufferWrite(muts)
		resolvedCount = int64(len(records))
		return nil
	})

	if txnErr != nil {
		log.Printf("[RECONCILE] supplier=%s action=%s failed: %v", supplierID, req.Action, txnErr)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, txnErr.Error()), http.StatusInternalServerError)
		return
	}

	log.Printf("[RECONCILE] supplier=%s | action=%s | resolved=%d items", supplierID, req.Action, resolvedCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "RETURNS_RECONCILED",
		"resolved": resolvedCount,
		"action":   req.Action,
	})
}
