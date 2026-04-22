package factory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/kafka"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

// ── Force-Receive — DLQ Reconciliation Handler ────────────────────────────────
// POST /v1/warehouse/transfers/force-receive
//
// When a Kafka INBOUND_FREIGHT_DISPATCHED event is dropped or ends up in the DLQ,
// the physical truck arrives at a warehouse that has no record of the transfer.
// The warehouse manager uses this endpoint to retroactively reconcile the freight
// into Spanner without corrupting the factory's ledger.
//
// Creates an InternalTransferOrder in RECEIVED state (Source=MANUAL_EMERGENCY),
// updates SupplierInventory stock levels (additive), and emits an audit event.

// ForceReceiveService handles unannounced freight reconciliation.
type ForceReceiveService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// HandleForceReceive creates a retroactive transfer and updates inventory.
func (s *ForceReceiveService) HandleForceReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	whID := auth.EffectiveWarehouseID(r.Context())
	if whID == "" {
		http.Error(w, `{"error":"warehouse_id scope required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		FactoryId string `json:"factory_id"` // optional — may be unknown
		Notes     string `json:"notes"`      // reason / truck plate / driver name
		Items     []struct {
			ProductId string  `json:"product_id"`
			Quantity  int64   `json:"quantity"`
			VolumeVU  float64 `json:"volume_vu"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, `{"error":"at least one item is required"}`, http.StatusBadRequest)
		return
	}

	// Resolve supplier from warehouse
	whRow, err := s.Spanner.Single().ReadRow(r.Context(), "Warehouses",
		spanner.Key{whID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
		return
	}
	var supplierID string
	if err := whRow.Columns(&supplierID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ── Tenant isolation: validate every ProductId belongs to this supplier ──
	for _, item := range req.Items {
		var prodSupplierID string
		prodIter := s.Spanner.Single().Query(r.Context(), spanner.Statement{
			SQL:    "SELECT SupplierId FROM SupplierProducts WHERE SkuId = @skuId LIMIT 1",
			Params: map[string]interface{}{"skuId": item.ProductId},
		})
		prodRow, prodErr := prodIter.Next()
		if prodErr == nil {
			_ = prodRow.Columns(&prodSupplierID)
		}
		prodIter.Stop()

		if prodSupplierID != "" && prodSupplierID != supplierID {
			log.Printf("[FORCE_RECEIVE] Cross-tenant ProductId injection blocked: product=%s belongs to %s, caller is %s",
				item.ProductId, prodSupplierID, supplierID)
			http.Error(w, `{"error":"product does not belong to your organization"}`, http.StatusForbidden)
			return
		}
	}

	// If factory unknown, use "UNKNOWN" as placeholder
	factoryID := req.FactoryId
	if factoryID == "" {
		factoryID = "UNKNOWN"
	}

	transferID := uuid.New().String()
	var totalVolumeVU float64
	mutations := []*spanner.Mutation{}

	// Header — directly in RECEIVED state (retroactive reconciliation)
	for _, item := range req.Items {
		totalVolumeVU += item.VolumeVU
	}

	mutations = append(mutations, spanner.Insert("InternalTransferOrders",
		[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State",
			"TotalVolumeVU", "Source", "CreatedAt", "UpdatedAt"},
		[]interface{}{transferID, factoryID, whID, supplierID, "RECEIVED",
			totalVolumeVU, "MANUAL_EMERGENCY", spanner.CommitTimestamp, spanner.CommitTimestamp},
	))

	// Items
	for _, item := range req.Items {
		itemID := uuid.New().String()
		mutations = append(mutations, spanner.Insert("InternalTransferItems",
			[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
			[]interface{}{transferID, itemID, item.ProductId, item.Quantity, item.VolumeVU},
		))
	}

	// Update SupplierInventory stock levels (additive)
	// Use ReadWriteTransaction to atomically read current stock and add received qty
	_, err = s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Update inventory for each item
		for _, item := range req.Items {
			invRow, err := txn.ReadRow(ctx, "SupplierInventory",
				spanner.Key{supplierID, item.ProductId},
				[]string{"QuantityAvailable"})
			if err != nil {
				// Inventory row doesn't exist — skip (warehouse manager should add it separately)
				log.Printf("[FORCE_RECEIVE] No inventory row for %s/%s — skipping stock update", supplierID, item.ProductId)
				continue
			}

			var currentQty int64
			if err := invRow.Columns(&currentQty); err != nil {
				continue
			}

			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("SupplierInventory",
					[]string{"SupplierId", "ProductId", "QuantityAvailable"},
					[]interface{}{supplierID, item.ProductId, currentQty + item.Quantity},
				),
			})
		}

		return nil
	})
	if err != nil {
		log.Printf("[FORCE_RECEIVE] Transaction failed: %v", err)
		http.Error(w, `{"error":"reconciliation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Create balancing SLA event for audit trail
	_, _ = s.Spanner.Apply(r.Context(), []*spanner.Mutation{
		spanner.Insert("FactorySLAEvents",
			[]string{"EventId", "TransferId", "SupplierId", "FactoryId", "WarehouseId",
				"EscalationLevel", "SLABreachMinutes", "CreatedAt"},
			[]interface{}{uuid.New().String(), transferID, supplierID, factoryID, whID,
				"FORCE_RECEIVED", int64(0), spanner.CommitTimestamp},
		),
	})

	// Emit Kafka audit event
	if s.Producer != nil {
		evt := kafka.InboundFreightUnannouncedEvent{
			TransferId:  transferID,
			WarehouseId: whID,
			SupplierId:  supplierID,
			ItemsCount:  len(req.Items),
			ReceivedBy:  claims.UserID,
			Timestamp:   time.Now().UTC(),
		}
		payload, _ := json.Marshal(evt)
		_ = s.Producer.WriteMessages(r.Context(), kafkago.Message{
			Key:   []byte(kafka.EventInboundFreightUnannounced),
			Value: payload,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer_id":     transferID,
		"factory_id":      factoryID,
		"warehouse_id":    whID,
		"state":           "RECEIVED",
		"total_volume_vu": totalVolumeVU,
		"source":          "MANUAL_EMERGENCY",
		"items_count":     len(req.Items),
		"notes":           req.Notes,
	})
}
