package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// DispatchLockService manages dispatch locks for supplier/warehouse/factory scopes.
// When a lock is active, new orders insert normally but are hidden from the active dispatch batch.
type DispatchLockService struct {
	Spanner *spanner.Client
}

type DispatchLockResponse struct {
	LockID      string     `json:"lock_id"`
	SupplierID  string     `json:"supplier_id"`
	WarehouseID string     `json:"warehouse_id,omitempty"`
	FactoryID   string     `json:"factory_id,omitempty"`
	LockType    string     `json:"lock_type"`
	LockedAt    time.Time  `json:"locked_at"`
	UnlockedAt  *time.Time `json:"unlocked_at,omitempty"`
	LockedBy    string     `json:"locked_by"`
}

// HandleAcquireDispatchLock acquires a new dispatch lock.
// POST /v1/warehouse/dispatch-lock
func (d *DispatchLockService) HandleAcquireDispatchLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		WarehouseID string `json:"warehouse_id,omitempty"`
		FactoryID   string `json:"factory_id,omitempty"`
		LockType    string `json:"lock_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	validTypes := map[string]bool{"AUTO_DISPATCH": true, "MANUAL_DISPATCH": true, "FACTORY_DISPATCH": true, "SPATIAL_UPDATE": true}
	if !validTypes[req.LockType] {
		req.LockType = "MANUAL_DISPATCH"
	}

	supplierID := claims.ResolveSupplierID()
	if claims.Role == "WAREHOUSE" || claims.Role == "FACTORY" {
		// Resolve supplier from warehouse/factory staff
		supplierID = resolveSupplierFromClaims(r.Context(), d.Spanner, claims)
	}

	// Scope-pin: derive warehouse/factory IDs from auth context, not body.
	warehouseID := req.WarehouseID
	factoryID := req.FactoryID
	switch claims.Role {
	case "WAREHOUSE":
		if ops := auth.GetWarehouseOps(r.Context()); ops != nil {
			warehouseID = ops.WarehouseID
		}
		factoryID = "" // warehouse role cannot lock factories
	case "FACTORY":
		if fs := auth.GetFactoryScope(r.Context()); fs != nil {
			factoryID = fs.FactoryID
		}
		warehouseID = "" // factory role cannot lock warehouses
	default: // SUPPLIER / ADMIN — validate body IDs belong to this supplier
		if warehouseID != "" {
			if err := validateEntityOwnership(r.Context(), d.Spanner, "Warehouses", "WarehouseId", warehouseID, supplierID); err != nil {
				http.Error(w, `{"error":"warehouse_id does not belong to your organization"}`, http.StatusForbidden)
				return
			}
		}
		if factoryID != "" {
			if err := validateEntityOwnership(r.Context(), d.Spanner, "Factories", "FactoryId", factoryID, supplierID); err != nil {
				http.Error(w, `{"error":"factory_id does not belong to your organization"}`, http.StatusForbidden)
				return
			}
		}
	}

	// Check for existing active lock at same scope
	lockExists, err := d.hasActiveLock(r.Context(), supplierID, warehouseID, factoryID)
	if err != nil {
		log.Printf("[DISPATCH LOCK] check error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if lockExists {
		http.Error(w, `{"error":"active dispatch lock already exists at this scope"}`, http.StatusConflict)
		return
	}

	lockID := uuid.New().String()
	payload := internalKafka.DispatchLockEvent{
		LockID:      lockID,
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		FactoryID:   factoryID,
		LockType:    req.LockType,
		LockedBy:    claims.UserID,
		Timestamp:   time.Now().UTC(),
	}
	_, err = d.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("DispatchLocks",
				[]string{"LockId", "SupplierId", "WarehouseId", "FactoryId", "LockType", "LockedAt", "LockedBy"},
				[]interface{}{lockID, supplierID, nullStr(warehouseID), nullStr(factoryID), req.LockType, spanner.CommitTimestamp, claims.UserID}),
		}); err != nil {
			return fmt.Errorf("buffer lock insert: %w", err)
		}

		if err := outbox.EmitJSON(txn, "DispatchLock", lockID, internalKafka.EventDispatchLockAcquired, internalKafka.TopicMain, payload, telemetry.TraceIDFromContext(ctx)); err != nil {
			return fmt.Errorf("outbox dispatch-lock-acquired: %w", err)
		}

		// Freeze lock for manual dispatch — AI worker must stop touching this scope.
		if req.LockType == "MANUAL_DISPATCH" {
			if err := outbox.EmitJSON(txn, "DispatchLock", lockID, internalKafka.EventFreezeLockAcquired, internalKafka.TopicFreezeLocks, payload, telemetry.TraceIDFromContext(ctx)); err != nil {
				return fmt.Errorf("outbox freeze-lock-acquired: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[DISPATCH LOCK] insert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Post-commit notification fan-out. Outbox above is the durable spine
	// (AI worker consumes TopicFreezeLocks keyed by LockID); these parallel
	// EventType-keyed pushes drive the notification dispatcher (supplier admin).
	internalKafka.EmitNotification(internalKafka.EventDispatchLockAcquired, payload)
	if req.LockType == "MANUAL_DISPATCH" {
		internalKafka.EmitNotification(internalKafka.EventFreezeLockAcquired, payload)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"lock_id":   lockID,
		"lock_type": req.LockType,
		"status":    "LOCKED",
	})
}

// HandleReleaseDispatchLock releases an active dispatch lock.
// DELETE /v1/warehouse/dispatch-lock?lock_id=X
func (d *DispatchLockService) HandleReleaseDispatchLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lockID := r.URL.Query().Get("lock_id")
	if lockID == "" {
		http.Error(w, `{"error":"lock_id required"}`, http.StatusBadRequest)
		return
	}

	var releasedPayload internalKafka.DispatchLockEvent
	_, err := d.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "DispatchLocks",
			spanner.Key{lockID},
			[]string{"SupplierId", "WarehouseId", "FactoryId", "LockType", "UnlockedAt"})
		if err != nil {
			return fmt.Errorf("lock not found")
		}

		var supplierID, lockType string
		var warehouseID, factoryID spanner.NullString
		var unlockedAt spanner.NullTime
		if err := row.Columns(&supplierID, &warehouseID, &factoryID, &lockType, &unlockedAt); err != nil {
			return err
		}
		if unlockedAt.Valid {
			return fmt.Errorf("lock already released")
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("DispatchLocks",
				[]string{"LockId", "UnlockedAt"},
				[]interface{}{lockID, spanner.CommitTimestamp}),
		}); err != nil {
			return err
		}

		releasedPayload = internalKafka.DispatchLockEvent{
			LockID:      lockID,
			SupplierID:  supplierID,
			WarehouseID: warehouseID.StringVal,
			FactoryID:   factoryID.StringVal,
			LockType:    lockType,
			LockedBy:    claims.UserID,
			Timestamp:   time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "DispatchLock", lockID, internalKafka.EventFreezeLockReleased, internalKafka.TopicFreezeLocks, releasedPayload, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		errMsg := err.Error()
		if errMsg == "lock not found" {
			http.Error(w, `{"error":"lock not found"}`, http.StatusNotFound)
		} else if errMsg == "lock already released" {
			http.Error(w, `{"error":"lock already released"}`, http.StatusConflict)
		} else {
			log.Printf("[DISPATCH LOCK] release error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Post-commit notification fan-out with real supplier/warehouse/lockType
	// captured inside the transaction (previous implementation passed empty
	// IDs to emitLockEvent, producing unroutable payloads).
	internalKafka.EmitNotification(internalKafka.EventDispatchLockReleased, releasedPayload)
	if releasedPayload.LockType == "MANUAL_DISPATCH" {
		internalKafka.EmitNotification(internalKafka.EventFreezeLockReleased, releasedPayload)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"lock_id": lockID,
		"status":  "RELEASED",
	})
}

// HandleListDispatchLocks returns active locks for the supplier scope.
// GET /v1/warehouse/dispatch-locks
func (d *DispatchLockService) HandleListDispatchLocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	supplierID := claims.ResolveSupplierID()
	if claims.Role == "WAREHOUSE" || claims.Role == "FACTORY" {
		supplierID = resolveSupplierFromClaims(r.Context(), d.Spanner, claims)
	}

	stmt := spanner.Statement{
		SQL: `SELECT LockId, SupplierId, WarehouseId, FactoryId, LockType, LockedAt, UnlockedAt, LockedBy
		      FROM DispatchLocks WHERE SupplierId = @sid AND UnlockedAt IS NULL
		      ORDER BY LockedAt DESC LIMIT 50`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := d.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var results []DispatchLockResponse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[DISPATCH LOCK] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var resp DispatchLockResponse
		var whID, fID spanner.NullString
		var unlocked spanner.NullTime
		if err := row.Columns(&resp.LockID, &resp.SupplierID, &whID, &fID,
			&resp.LockType, &resp.LockedAt, &unlocked, &resp.LockedBy); err != nil {
			continue
		}
		if whID.Valid {
			resp.WarehouseID = whID.StringVal
		}
		if fID.Valid {
			resp.FactoryID = fID.StringVal
		}
		if unlocked.Valid {
			resp.UnlockedAt = &unlocked.Time
		}
		results = append(results, resp)
	}

	if results == nil {
		results = []DispatchLockResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// IsDispatchLocked checks if there's an active dispatch lock at the given scope.
// Used by the dispatch engine to skip locked scopes.
func (d *DispatchLockService) IsDispatchLocked(ctx context.Context, supplierID, warehouseID, factoryID string) bool {
	locked, _ := d.hasActiveLock(ctx, supplierID, warehouseID, factoryID)
	return locked
}

func (d *DispatchLockService) hasActiveLock(ctx context.Context, supplierID, warehouseID, factoryID string) (bool, error) {
	sql := `SELECT LockId FROM DispatchLocks
	        WHERE SupplierId = @sid AND UnlockedAt IS NULL`
	params := map[string]interface{}{"sid": supplierID}

	if warehouseID != "" {
		sql += ` AND (WarehouseId = @wid OR WarehouseId IS NULL)`
		params["wid"] = warehouseID
	}
	if factoryID != "" {
		sql += ` AND (FactoryId = @fid OR FactoryId IS NULL)`
		params["fid"] = factoryID
	}
	sql += ` LIMIT 1`

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := d.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func resolveSupplierFromClaims(ctx context.Context, client *spanner.Client, claims *auth.LabClaims) string {
	if claims.Role == "WAREHOUSE" && claims.WarehouseID != "" {
		row, err := client.Single().ReadRow(ctx, "Warehouses",
			spanner.Key{claims.WarehouseID}, []string{"SupplierId"})
		if err == nil {
			var sid string
			if row.Columns(&sid) == nil {
				return sid
			}
		}
	}
	if claims.Role == "FACTORY" && claims.FactoryID != "" {
		row, err := client.Single().ReadRow(ctx, "Factories",
			spanner.Key{claims.FactoryID}, []string{"SupplierId"})
		if err == nil {
			var sid string
			if row.Columns(&sid) == nil {
				return sid
			}
		}
	}
	return claims.ResolveSupplierID()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// validateEntityOwnership checks that a given entity (warehouse or factory) belongs
// to the expected supplier. Returns nil on success, error if not owned or not found.
func validateEntityOwnership(ctx context.Context, client *spanner.Client, table, pkCol, entityID, expectedSupplierID string) error {
	row, err := client.Single().ReadRow(ctx, table, spanner.Key{entityID}, []string{"SupplierId"})
	if err != nil {
		return fmt.Errorf("entity %s not found in %s: %w", entityID, table, err)
	}
	var ownerID string
	if err := row.Column(0, &ownerID); err != nil {
		return fmt.Errorf("read SupplierId from %s: %w", table, err)
	}
	if ownerID != expectedSupplierID {
		return fmt.Errorf("entity %s in %s belongs to another supplier", entityID, table)
	}
	return nil
}
