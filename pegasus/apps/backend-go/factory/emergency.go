package factory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"
	bkafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

// ── Emergency Transfer ────────────────────────────────────────────────────────
// POST /v1/warehouse/transfers/emergency — warehouse admin creates urgent
// InternalTransferOrder. Routes to PrimaryFactoryId, fallback to SecondaryFactoryId.

// EmergencyTransferService holds dependencies for emergency transfer creation.
type EmergencyTransferService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

const emergencyTransferCreatedEventName = "EMERGENCY_TRANSFER_CREATED"

// HandleEmergencyTransfer creates an urgent InternalTransferOrder from a warehouse.
func (s *EmergencyTransferService) HandleEmergencyTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	whID := auth.EffectiveWarehouseID(r.Context())
	if whID == "" {
		http.Error(w, `{"error":"warehouse_id scope required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Items []struct {
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

	// Resolve factory from warehouse's PrimaryFactoryId / SecondaryFactoryId
	whRow, err := s.Spanner.Single().ReadRow(r.Context(), "Warehouses",
		spanner.Key{whID}, []string{"SupplierId", "PrimaryFactoryId", "SecondaryFactoryId"})
	if err != nil {
		http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
		return
	}

	var supplierID string
	var primaryFactory, secondaryFactory spanner.NullString
	if err := whRow.Columns(&supplierID, &primaryFactory, &secondaryFactory); err != nil {
		log.Printf("[EMERGENCY TRANSFER] warehouse read error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	factoryID := ""
	if primaryFactory.Valid && primaryFactory.StringVal != "" {
		factoryID = primaryFactory.StringVal
	} else if secondaryFactory.Valid && secondaryFactory.StringVal != "" {
		factoryID = secondaryFactory.StringVal
	}
	if factoryID == "" {
		http.Error(w, `{"error":"no factory assigned to this warehouse — configure PrimaryFactoryId"}`, http.StatusPreconditionFailed)
		return
	}

	// Create the emergency transfer order
	transferID := uuid.New().String()
	var totalVolumeVU float64
	mutations := []*spanner.Mutation{}

	for _, item := range req.Items {
		totalVolumeVU += item.VolumeVU
		itemID := uuid.New().String()
		mutations = append(mutations, spanner.Insert("InternalTransferItems",
			[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
			[]interface{}{transferID, itemID, item.ProductId, item.Quantity, item.VolumeVU},
		))
	}

	// Header mutation (must come first for interleave constraint)
	headerMutation := spanner.Insert("InternalTransferOrders",
		[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State",
			"TotalVolumeVU", "Source", "CreatedAt"},
		[]interface{}{transferID, factoryID, whID, supplierID, "DRAFT",
			totalVolumeVU, "MANUAL_EMERGENCY", spanner.CommitTimestamp},
	)
	mutations = append([]*spanner.Mutation{headerMutation}, mutations...)

	_, err = s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		evt := struct {
			Event       string  `json:"event"`
			TransferID  string  `json:"transfer_id"`
			FactoryID   string  `json:"factory_id"`
			WarehouseID string  `json:"warehouse_id"`
			SupplierID  string  `json:"supplier_id"`
			ItemsCount  int     `json:"items_count"`
			Source      string  `json:"source"`
			VolumeVU    float64 `json:"total_volume_vu"`
		}{
			Event:       emergencyTransferCreatedEventName,
			TransferID:  transferID,
			FactoryID:   factoryID,
			WarehouseID: whID,
			SupplierID:  supplierID,
			ItemsCount:  len(req.Items),
			Source:      "MANUAL_EMERGENCY",
			VolumeVU:    totalVolumeVU,
		}

		return outbox.EmitJSON(txn, "InternalTransferOrder", transferID, bkafka.EventReplenishmentTransferCreated, bkafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[EMERGENCY TRANSFER] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer_id":     transferID,
		"factory_id":      factoryID,
		"warehouse_id":    whID,
		"state":           "DRAFT",
		"total_volume_vu": totalVolumeVU,
		"source":          "MANUAL_EMERGENCY",
		"items_count":     len(req.Items),
	})
}
