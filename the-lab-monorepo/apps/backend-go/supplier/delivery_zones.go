// Package supplier — DeliveryZones CRUD (Phase VII: Cross-Warehouse Freight Surcharge)
// Suppliers define distance-based delivery fee bands per warehouse.
package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// DeliveryZone represents a distance-based fee band for a supplier.
type DeliveryZone struct {
	ZoneId        string  `json:"zone_id"`
	SupplierId    string  `json:"supplier_id"`
	WarehouseId   string  `json:"warehouse_id,omitempty"`
	ZoneName      string  `json:"zone_name"`
	MinDistanceKm float64 `json:"min_distance_km"`
	MaxDistanceKm float64 `json:"max_distance_km"`
	FeeMinor      int64   `json:"fee_minor"`
	Priority      int64   `json:"priority"`
	IsActive      bool    `json:"is_active"`
}

// HandleDeliveryZones handles GET (list) and POST (create) for /v1/supplier/delivery-zones.
func HandleDeliveryZones(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)

		switch r.Method {
		case http.MethodGet:
			listDeliveryZones(w, r, client, claims.ResolveSupplierID())
		case http.MethodPost:
			createDeliveryZone(w, r, client, claims.ResolveSupplierID())
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleDeliveryZoneAction handles PUT (update) and DELETE (deactivate) for /v1/supplier/delivery-zones/{id}.
func HandleDeliveryZoneAction(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		zoneID := ExtractPathParam(r.URL.Path, "delivery-zones")
		if zoneID == "" {
			http.Error(w, `{"error":"zone_id required"}`, http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			updateDeliveryZone(w, r, client, claims.ResolveSupplierID(), zoneID)
		case http.MethodDelete:
			deactivateDeliveryZone(w, r, client, claims.ResolveSupplierID(), zoneID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listDeliveryZones(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	warehouseFilter := r.URL.Query().Get("warehouse_id")

	sql := `SELECT ZoneId, SupplierId, COALESCE(WarehouseId, ''), ZoneName,
	               MinDistanceKm, MaxDistanceKm, FeeMinor, Priority, IsActive
	        FROM DeliveryZones
	        WHERE SupplierId = @sid`
	params := map[string]interface{}{"sid": supplierID}

	if warehouseFilter != "" {
		sql += " AND (WarehouseId = @wid OR WarehouseId IS NULL)"
		params["wid"] = warehouseFilter
	}
	sql += " ORDER BY Priority DESC, MinDistanceKm ASC"

	iter := client.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var zones []DeliveryZone
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[ZONES] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var z DeliveryZone
		if err := row.Columns(&z.ZoneId, &z.SupplierId, &z.WarehouseId,
			&z.ZoneName, &z.MinDistanceKm, &z.MaxDistanceKm,
			&z.FeeMinor, &z.Priority, &z.IsActive); err != nil {
			continue
		}
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []DeliveryZone{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"zones": zones})
}

type createZoneReq struct {
	WarehouseId   string  `json:"warehouse_id"`
	ZoneName      string  `json:"zone_name"`
	MinDistanceKm float64 `json:"min_distance_km"`
	MaxDistanceKm float64 `json:"max_distance_km"`
	FeeMinor      int64   `json:"fee_minor"`
	Priority      int64   `json:"priority"`
}

func createDeliveryZone(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	var req createZoneReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ZoneName == "" || req.MaxDistanceKm <= 0 {
		http.Error(w, `{"error":"zone_name and max_distance_km required"}`, http.StatusUnprocessableEntity)
		return
	}
	if req.MinDistanceKm >= req.MaxDistanceKm {
		http.Error(w, `{"error":"min_distance_km must be less than max_distance_km"}`, http.StatusUnprocessableEntity)
		return
	}

	zoneID := uuid.New().String()
	var whID spanner.NullString
	if req.WarehouseId != "" {
		// Validate warehouse belongs to this supplier before writing
		whRow, lookupErr := client.Single().ReadRow(r.Context(), "Warehouses", spanner.Key{req.WarehouseId}, []string{"SupplierId"})
		if lookupErr != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}
		var whOwner string
		if cErr := whRow.Column(0, &whOwner); cErr != nil || whOwner != supplierID {
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		}
		whID = spanner.NullString{StringVal: req.WarehouseId, Valid: true}
	}

	_, err := client.Apply(r.Context(), []*spanner.Mutation{
		spanner.Insert("DeliveryZones",
			[]string{"ZoneId", "SupplierId", "WarehouseId", "ZoneName",
				"MinDistanceKm", "MaxDistanceKm", "FeeMinor", "Priority",
				"IsActive", "CreatedAt", "UpdatedAt"},
			[]interface{}{zoneID, supplierID, whID, req.ZoneName,
				req.MinDistanceKm, req.MaxDistanceKm, req.FeeMinor, req.Priority,
				true, spanner.CommitTimestamp, spanner.CommitTimestamp}),
	})
	if err != nil {
		log.Printf("[ZONES] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"zone_id": zoneID,
		"message": "Delivery zone created",
	})
}

func updateDeliveryZone(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID, zoneID string) {
	var req createZoneReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify ownership
		row, err := txn.ReadRow(ctx, "DeliveryZones",
			spanner.Key{supplierID, zoneID},
			[]string{"SupplierId"})
		if err != nil {
			return err
		}
		var owner string
		if err := row.Columns(&owner); err != nil {
			return err
		}
		if owner != supplierID {
			return fmt.Errorf("FORBIDDEN")
		}

		cols := []string{"SupplierId", "ZoneId", "UpdatedAt"}
		vals := []interface{}{supplierID, zoneID, spanner.CommitTimestamp}

		if req.ZoneName != "" {
			cols = append(cols, "ZoneName")
			vals = append(vals, req.ZoneName)
		}
		if req.MaxDistanceKm > 0 {
			cols = append(cols, "MinDistanceKm", "MaxDistanceKm")
			vals = append(vals, req.MinDistanceKm, req.MaxDistanceKm)
		}
		if req.FeeMinor > 0 {
			cols = append(cols, "FeeMinor")
			vals = append(vals, req.FeeMinor)
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("DeliveryZones", cols, vals),
		})
	})

	if err != nil {
		log.Printf("[ZONES] update error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Zone updated"})
}

func deactivateDeliveryZone(w http.ResponseWriter, _ *http.Request, client *spanner.Client, supplierID, zoneID string) {
	_, err := client.Apply(context.Background(), []*spanner.Mutation{
		spanner.Update("DeliveryZones",
			[]string{"SupplierId", "ZoneId", "IsActive", "UpdatedAt"},
			[]interface{}{supplierID, zoneID, false, spanner.CommitTimestamp}),
	})
	if err != nil {
		log.Printf("[ZONES] deactivate error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Zone deactivated"})
}
