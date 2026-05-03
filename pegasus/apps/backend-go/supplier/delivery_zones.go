// Package supplier — DeliveryZones CRUD (Phase VII: Cross-Warehouse Freight Surcharge)
// Suppliers define distance-based delivery fee bands per warehouse.
package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

var (
	errDeliveryZoneWarehouseNotFound  = errors.New("warehouse not found")
	errDeliveryZoneWarehouseForbidden = errors.New("warehouse does not belong to your organization")
	errDeliveryZoneNotFound           = errors.New("delivery zone not found")
	errDeliveryZoneInvalidRange       = errors.New("min_distance_km must be less than max_distance_km")
	errDeliveryZoneMaxDistance        = errors.New("max_distance_km must be greater than zero")
	errDeliveryZoneNameRequired       = errors.New("zone_name required")
	errDeliveryZoneNegativeFee        = errors.New("fee_minor must be non-negative")
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

type deliveryZoneRecord struct {
	ZoneId        string
	SupplierId    string
	WarehouseId   spanner.NullString
	ZoneName      string
	MinDistanceKm float64
	MaxDistanceKm float64
	FeeMinor      int64
	Priority      int64
	IsActive      bool
}

// HandleDeliveryZones handles GET (list) and POST (create) for /v1/supplier/delivery-zones.
func HandleDeliveryZones(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)

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
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
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

type updateZoneReq struct {
	WarehouseId   *string  `json:"warehouse_id,omitempty"`
	ZoneName      *string  `json:"zone_name,omitempty"`
	MinDistanceKm *float64 `json:"min_distance_km,omitempty"`
	MaxDistanceKm *float64 `json:"max_distance_km,omitempty"`
	FeeMinor      *int64   `json:"fee_minor,omitempty"`
	Priority      *int64   `json:"priority,omitempty"`
}

func validateDeliveryZoneWarehouse(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string) error {
	if warehouseID == "" {
		return nil
	}

	whRow, err := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
	switch {
	case errors.Is(err, spanner.ErrRowNotFound):
		return errDeliveryZoneWarehouseNotFound
	case err != nil:
		return fmt.Errorf("read warehouse %s: %w", warehouseID, err)
	}

	var whOwner string
	if err := whRow.Column(0, &whOwner); err != nil {
		return fmt.Errorf("read warehouse %s owner: %w", warehouseID, err)
	}
	if whOwner != supplierID {
		return errDeliveryZoneWarehouseForbidden
	}

	return nil
}

func readDeliveryZone(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, zoneID string) (deliveryZoneRecord, error) {
	row, err := txn.ReadRow(ctx, "DeliveryZones", spanner.Key{supplierID, zoneID}, []string{
		"ZoneId",
		"SupplierId",
		"WarehouseId",
		"ZoneName",
		"MinDistanceKm",
		"MaxDistanceKm",
		"FeeMinor",
		"Priority",
		"IsActive",
	})
	if errors.Is(err, spanner.ErrRowNotFound) {
		return deliveryZoneRecord{}, errDeliveryZoneNotFound
	}
	if err != nil {
		return deliveryZoneRecord{}, fmt.Errorf("read delivery zone %s: %w", zoneID, err)
	}

	var zone deliveryZoneRecord
	if err := row.Columns(
		&zone.ZoneId,
		&zone.SupplierId,
		&zone.WarehouseId,
		&zone.ZoneName,
		&zone.MinDistanceKm,
		&zone.MaxDistanceKm,
		&zone.FeeMinor,
		&zone.Priority,
		&zone.IsActive,
	); err != nil {
		return deliveryZoneRecord{}, fmt.Errorf("decode delivery zone %s: %w", zoneID, err)
	}

	return zone, nil
}

func createDeliveryZone(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	var req createZoneReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ZoneName = strings.TrimSpace(req.ZoneName)
	if req.ZoneName == "" {
		http.Error(w, `{"error":"zone_name required"}`, http.StatusUnprocessableEntity)
		return
	}
	if req.MaxDistanceKm <= 0 {
		http.Error(w, `{"error":"max_distance_km must be greater than zero"}`, http.StatusUnprocessableEntity)
		return
	}
	if req.FeeMinor < 0 {
		http.Error(w, `{"error":"fee_minor must be non-negative"}`, http.StatusUnprocessableEntity)
		return
	}
	if req.MinDistanceKm >= req.MaxDistanceKm {
		http.Error(w, `{"error":"min_distance_km must be less than max_distance_km"}`, http.StatusUnprocessableEntity)
		return
	}

	zoneID := uuid.New().String()
	var whID spanner.NullString
	if req.WarehouseId != "" {
		whID = spanner.NullString{StringVal: req.WarehouseId, Valid: true}
	}

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if whID.Valid {
			if err := validateDeliveryZoneWarehouse(ctx, txn, supplierID, req.WarehouseId); err != nil {
				return err
			}
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("DeliveryZones",
				[]string{"ZoneId", "SupplierId", "WarehouseId", "ZoneName",
					"MinDistanceKm", "MaxDistanceKm", "FeeMinor", "Priority",
					"IsActive", "CreatedAt", "UpdatedAt"},
				[]interface{}{zoneID, supplierID, whID, req.ZoneName,
					req.MinDistanceKm, req.MaxDistanceKm, req.FeeMinor, req.Priority,
					true, spanner.CommitTimestamp, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		switch {
		case errors.Is(err, errDeliveryZoneWarehouseNotFound):
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		case errors.Is(err, errDeliveryZoneWarehouseForbidden):
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		}
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
	var req updateZoneReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.WarehouseId == nil && req.ZoneName == nil && req.MinDistanceKm == nil && req.MaxDistanceKm == nil && req.FeeMinor == nil && req.Priority == nil {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		current, err := readDeliveryZone(ctx, txn, supplierID, zoneID)
		if err != nil {
			return err
		}

		nextWarehouse := current.WarehouseId
		nextZoneName := current.ZoneName
		nextMinDistance := current.MinDistanceKm
		nextMaxDistance := current.MaxDistanceKm
		nextFee := current.FeeMinor
		nextPriority := current.Priority

		if req.WarehouseId != nil {
			warehouseID := strings.TrimSpace(*req.WarehouseId)
			if warehouseID == "" {
				nextWarehouse = spanner.NullString{}
			} else {
				if err := validateDeliveryZoneWarehouse(ctx, txn, supplierID, warehouseID); err != nil {
					return err
				}
				nextWarehouse = spanner.NullString{StringVal: warehouseID, Valid: true}
			}
		}

		if req.ZoneName != nil {
			nextZoneName = strings.TrimSpace(*req.ZoneName)
			if nextZoneName == "" {
				return errDeliveryZoneNameRequired
			}
		}

		if req.MinDistanceKm != nil {
			nextMinDistance = *req.MinDistanceKm
		}
		if req.MaxDistanceKm != nil {
			nextMaxDistance = *req.MaxDistanceKm
		}
		if nextMaxDistance <= 0 {
			return errDeliveryZoneMaxDistance
		}
		if nextMinDistance >= nextMaxDistance {
			return errDeliveryZoneInvalidRange
		}

		if req.FeeMinor != nil {
			if *req.FeeMinor < 0 {
				return errDeliveryZoneNegativeFee
			}
			nextFee = *req.FeeMinor
		}
		if req.Priority != nil {
			nextPriority = *req.Priority
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("DeliveryZones",
				[]string{"SupplierId", "ZoneId", "WarehouseId", "ZoneName", "MinDistanceKm", "MaxDistanceKm", "FeeMinor", "Priority", "UpdatedAt"},
				[]interface{}{supplierID, zoneID, nextWarehouse, nextZoneName, nextMinDistance, nextMaxDistance, nextFee, nextPriority, spanner.CommitTimestamp}),
		})
	})

	if err != nil {
		switch {
		case errors.Is(err, errDeliveryZoneNotFound):
			http.Error(w, `{"error":"delivery zone not found"}`, http.StatusNotFound)
			return
		case errors.Is(err, errDeliveryZoneWarehouseNotFound):
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		case errors.Is(err, errDeliveryZoneWarehouseForbidden):
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		case errors.Is(err, errDeliveryZoneInvalidRange):
			http.Error(w, `{"error":"min_distance_km must be less than max_distance_km"}`, http.StatusUnprocessableEntity)
			return
		case errors.Is(err, errDeliveryZoneMaxDistance):
			http.Error(w, `{"error":"max_distance_km must be greater than zero"}`, http.StatusUnprocessableEntity)
			return
		case errors.Is(err, errDeliveryZoneNameRequired):
			http.Error(w, `{"error":"zone_name required"}`, http.StatusUnprocessableEntity)
			return
		case errors.Is(err, errDeliveryZoneNegativeFee):
			http.Error(w, `{"error":"fee_minor must be non-negative"}`, http.StatusUnprocessableEntity)
			return
		}
		log.Printf("[ZONES] update zone %s error: %v", zoneID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Zone updated"})
}

func deactivateDeliveryZone(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID, zoneID string) {
	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := readDeliveryZone(ctx, txn, supplierID, zoneID); err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("DeliveryZones",
				[]string{"SupplierId", "ZoneId", "IsActive", "UpdatedAt"},
				[]interface{}{supplierID, zoneID, false, spanner.CommitTimestamp}),
		})
	})
	if err != nil {
		if errors.Is(err, errDeliveryZoneNotFound) {
			http.Error(w, `{"error":"delivery zone not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[ZONES] deactivate zone %s error: %v", zoneID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Zone deactivated"})
}
