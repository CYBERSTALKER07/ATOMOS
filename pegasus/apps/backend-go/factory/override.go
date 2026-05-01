package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FACTORY PAYLOAD OVERRIDE — MANIFEST REBALANCING DURING LOADING
//
// When a manifest is in LOADING state, factory admin/payloader can:
//   1. Reassign transfer orders between manifests (capacity mismatch)
//   2. Cancel individual transfers from a manifest
//   3. Cancel an entire manifest
//
// This is the "miscalculation override" surface — the bin-packing algorithm
// estimated volumes, but real-world loading reveals mismatches.
// ═══════════════════════════════════════════════════════════════════════════════

// OverrideService handles manifest rebalancing during the LOADING phase.
// All state-change events are emitted via the transactional outbox inside the
// owning ReadWriteTransaction — there is no in-process Kafka writer here.
type OverrideService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer // retained for legacy wiring; no longer used internally
}

// ReassignRequest moves transfer orders from one manifest to another.
type ReassignRequest struct {
	SourceManifestID string   `json:"source_manifest_id"`
	TargetManifestID string   `json:"target_manifest_id"`
	TransferIDs      []string `json:"transfer_ids"`
	TransferID       string   `json:"transfer_id,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

// HandleManifestRebalance reassigns transfers between LOADING-state manifests.
// POST /v1/factory/manifests/rebalance
func (o *OverrideService) HandleManifestRebalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	factoryScope := auth.GetFactoryScope(r.Context())
	if factoryScope == nil {
		http.Error(w, "Factory scope required", http.StatusForbidden)
		return
	}

	var req ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if len(req.TransferIDs) == 0 && req.TransferID != "" {
		req.TransferIDs = []string{req.TransferID}
	}
	if req.SourceManifestID == "" || req.TargetManifestID == "" || len(req.TransferIDs) == 0 {
		http.Error(w, `{"error":"source_manifest_id, target_manifest_id, and transfer_ids are required"}`, http.StatusBadRequest)
		return
	}
	if req.SourceManifestID == req.TargetManifestID {
		http.Error(w, `{"error":"source and target manifest must be different"}`, http.StatusBadRequest)
		return
	}

	var movedVolume float64
	supplierID := claims.ResolveSupplierID()
	reason := req.Reason
	if reason == "" {
		reason = "MANUAL_OVERRIDE"
	}
	_, err := o.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify source manifest is in LOADING state and belongs to this factory
		srcRow, err := txn.ReadRow(ctx, "FactoryTruckManifests",
			spanner.Key{req.SourceManifestID},
			[]string{"FactoryId", "State", "TotalVolumeVU"})
		if err != nil {
			return fmt.Errorf("source manifest not found")
		}
		var srcFactoryID, srcState string
		var srcVolume float64
		if err := srcRow.Columns(&srcFactoryID, &srcState, &srcVolume); err != nil {
			return err
		}
		if srcFactoryID != factoryScope.FactoryID {
			return fmt.Errorf("source manifest does not belong to this factory")
		}
		if srcState != "LOADING" {
			return fmt.Errorf("source manifest must be in LOADING state (current: %s)", srcState)
		}

		// Verify target manifest is in LOADING or READY_FOR_LOADING state
		tgtRow, err := txn.ReadRow(ctx, "FactoryTruckManifests",
			spanner.Key{req.TargetManifestID},
			[]string{"FactoryId", "State", "TotalVolumeVU", "MaxVolumeVU"})
		if err != nil {
			return fmt.Errorf("target manifest not found")
		}
		var tgtFactoryID, tgtState string
		var tgtVolume, tgtMaxVolume float64
		if err := tgtRow.Columns(&tgtFactoryID, &tgtState, &tgtVolume, &tgtMaxVolume); err != nil {
			return err
		}
		if tgtFactoryID != factoryScope.FactoryID {
			return fmt.Errorf("target manifest does not belong to this factory")
		}
		if tgtState != "LOADING" && tgtState != "READY_FOR_LOADING" {
			return fmt.Errorf("target manifest must be in LOADING or READY_FOR_LOADING state (current: %s)", tgtState)
		}

		// Validate and move each transfer
		movedVolume = 0
		mutations := make([]*spanner.Mutation, 0, len(req.TransferIDs)*2+2)

		for _, transferID := range req.TransferIDs {
			transferRow, err := txn.ReadRow(ctx, "InternalTransferOrders",
				spanner.Key{transferID},
				[]string{"ManifestId", "TotalVolumeVU", "State"})
			if err != nil {
				return fmt.Errorf("transfer %s not found", transferID)
			}
			var manifestID spanner.NullString
			var volume float64
			var tState string
			if err := transferRow.Columns(&manifestID, &volume, &tState); err != nil {
				return err
			}
			if !manifestID.Valid || manifestID.StringVal != req.SourceManifestID {
				return fmt.Errorf("transfer %s is not assigned to source manifest", transferID)
			}
			if tState != "LOADING" {
				return fmt.Errorf("transfer %s must be in LOADING state (current: %s)", transferID, tState)
			}

			movedVolume += volume

			// Update transfer's manifest assignment
			mutations = append(mutations, spanner.Update("InternalTransferOrders",
				[]string{"TransferId", "ManifestId", "UpdatedAt"},
				[]interface{}{transferID, req.TargetManifestID, spanner.CommitTimestamp}))
		}

		// Check target capacity
		if tgtMaxVolume > 0 && (tgtVolume+movedVolume) > tgtMaxVolume {
			return fmt.Errorf("target manifest capacity exceeded: would be %.1f / %.1f VU", tgtVolume+movedVolume, tgtMaxVolume)
		}

		// Update source manifest volume
		mutations = append(mutations, spanner.Update("FactoryTruckManifests",
			[]string{"ManifestId", "TotalVolumeVU", "UpdatedAt"},
			[]interface{}{req.SourceManifestID, srcVolume - movedVolume, spanner.CommitTimestamp}))

		// Update target manifest volume
		mutations = append(mutations, spanner.Update("FactoryTruckManifests",
			[]string{"ManifestId", "TotalVolumeVU", "UpdatedAt"},
			[]interface{}{req.TargetManifestID, tgtVolume + movedVolume, spanner.CommitTimestamp}))

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Emit MANIFEST_REBALANCED and PAYLOAD_SYNC via outbox — atomic with the mutation.
		if err := outbox.EmitJSON(txn, "Manifest", req.SourceManifestID, internalKafka.EventManifestRebalanced, internalKafka.TopicMain,
			internalKafka.ManifestRebalancedEvent{
				FactoryID:        factoryScope.FactoryID,
				SupplierID:       supplierID,
				SourceManifestID: req.SourceManifestID,
				TargetManifestID: req.TargetManifestID,
				TransferIDs:      req.TransferIDs,
				Reason:           reason,
				RebalancedBy:     claims.UserID,
				Timestamp:        time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return err
		}

		if err := outbox.EmitJSON(txn, "Manifest", req.SourceManifestID, internalKafka.EventPayloadSync, internalKafka.TopicMain,
			internalKafka.PayloadSyncEvent{
				SupplierID: supplierID,
				ManifestID: req.SourceManifestID,
				Reason:     "REBALANCED",
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Manifest", req.TargetManifestID, internalKafka.EventPayloadSync, internalKafka.TopicMain,
			internalKafka.PayloadSyncEvent{
				SupplierID: supplierID,
				ManifestID: req.TargetManifestID,
				Reason:     "REBALANCED",
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not assigned") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusNotFound)
		} else if strings.Contains(errMsg, "capacity exceeded") || strings.Contains(errMsg, "must be in") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusConflict)
		} else {
			log.Printf("[MANIFEST REBALANCE] error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// MANIFEST_REBALANCED event was emitted via outbox inside the transaction.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"source_manifest_id": req.SourceManifestID,
		"target_manifest_id": req.TargetManifestID,
		"transfers_moved":    len(req.TransferIDs),
		"volume_moved_vu":    movedVolume,
		"reason":             reason,
	})
}

// HandleCancelManifestTransfer removes a transfer from a LOADING manifest.
// POST /v1/factory/manifests/cancel-transfer
func (o *OverrideService) HandleCancelManifestTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	factoryScope := auth.GetFactoryScope(r.Context())
	if factoryScope == nil {
		http.Error(w, "Factory scope required", http.StatusForbidden)
		return
	}

	var req struct {
		ManifestID string `json:"manifest_id"`
		TransferID string `json:"transfer_id"`
		Reason     string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.ManifestID == "" || req.TransferID == "" {
		http.Error(w, `{"error":"manifest_id and transfer_id are required"}`, http.StatusBadRequest)
		return
	}

	supplierID := claims.ResolveSupplierID()
	_, err := o.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify manifest belongs to factory and is in LOADING
		mRow, err := txn.ReadRow(ctx, "FactoryTruckManifests",
			spanner.Key{req.ManifestID},
			[]string{"FactoryId", "State", "TotalVolumeVU"})
		if err != nil {
			return fmt.Errorf("manifest not found")
		}
		var mFactoryID, mState string
		var mVolume float64
		if err := mRow.Columns(&mFactoryID, &mState, &mVolume); err != nil {
			return err
		}
		if mFactoryID != factoryScope.FactoryID {
			return fmt.Errorf("manifest does not belong to this factory")
		}
		if mState != "LOADING" {
			return fmt.Errorf("manifest must be in LOADING state (current: %s)", mState)
		}

		// Read transfer and unassign from manifest
		tRow, err := txn.ReadRow(ctx, "InternalTransferOrders",
			spanner.Key{req.TransferID},
			[]string{"ManifestId", "TotalVolumeVU", "State"})
		if err != nil {
			return fmt.Errorf("transfer not found")
		}
		var tManifest spanner.NullString
		var tVolume float64
		var tState string
		if err := tRow.Columns(&tManifest, &tVolume, &tState); err != nil {
			return err
		}
		if !tManifest.Valid || tManifest.StringVal != req.ManifestID {
			return fmt.Errorf("transfer not assigned to this manifest")
		}

		mutations := []*spanner.Mutation{
			// Unassign transfer from manifest, revert to APPROVED
			spanner.Update("InternalTransferOrders",
				[]string{"TransferId", "ManifestId", "State", "UpdatedAt"},
				[]interface{}{req.TransferID, nil, "APPROVED", spanner.CommitTimestamp}),
			// Reduce manifest volume
			spanner.Update("FactoryTruckManifests",
				[]string{"ManifestId", "TotalVolumeVU", "UpdatedAt"},
				[]interface{}{req.ManifestID, mVolume - tVolume, spanner.CommitTimestamp}),
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Emit TRANSFER_UNASSIGNED and PAYLOAD_SYNC via outbox — atomic with the mutation.
		if err := outbox.EmitJSON(txn, "Manifest", req.ManifestID, internalKafka.EventTransferUnassigned, internalKafka.TopicMain,
			internalKafka.TransferUnassignedEvent{
				ManifestID:   req.ManifestID,
				TransferID:   req.TransferID,
				FactoryID:    factoryScope.FactoryID,
				SupplierID:   supplierID,
				Reason:       req.Reason,
				UnassignedBy: claims.UserID,
				Timestamp:    time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Manifest", req.ManifestID, internalKafka.EventPayloadSync, internalKafka.TopicMain,
			internalKafka.PayloadSyncEvent{
				SupplierID: supplierID,
				ManifestID: req.ManifestID,
				Reason:     "REBALANCED",
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not assigned") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusNotFound)
		} else if strings.Contains(errMsg, "must be in") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusConflict)
		} else {
			log.Printf("[CANCEL MANIFEST TRANSFER] error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// TRANSFER_UNASSIGNED event was emitted via outbox inside the transaction.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"manifest_id": req.ManifestID,
		"transfer_id": req.TransferID,
		"status":      "UNASSIGNED",
	})
}

// HandleCancelManifest cancels an entire LOADING manifest, returning all transfers to APPROVED.
// POST /v1/factory/manifests/cancel
func (o *OverrideService) HandleCancelManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	factoryScope := auth.GetFactoryScope(r.Context())
	if factoryScope == nil {
		http.Error(w, "Factory scope required", http.StatusForbidden)
		return
	}

	var req struct {
		ManifestID string `json:"manifest_id"`
		Reason     string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.ManifestID == "" {
		http.Error(w, `{"error":"manifest_id is required"}`, http.StatusBadRequest)
		return
	}

	var transferIDs []string
	supplierID := claims.ResolveSupplierID()
	cancelReason := req.Reason
	if cancelReason == "" {
		cancelReason = "MANUAL_OVERRIDE"
	}
	_, err := o.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify manifest
		mRow, err := txn.ReadRow(ctx, "FactoryTruckManifests",
			spanner.Key{req.ManifestID},
			[]string{"FactoryId", "State"})
		if err != nil {
			return fmt.Errorf("manifest not found")
		}
		var mFactoryID, mState string
		if err := mRow.Columns(&mFactoryID, &mState); err != nil {
			return err
		}
		if mFactoryID != factoryScope.FactoryID {
			return fmt.Errorf("manifest does not belong to this factory")
		}
		if mState != "LOADING" && mState != "READY_FOR_LOADING" {
			return fmt.Errorf("manifest must be in LOADING or READY_FOR_LOADING state (current: %s)", mState)
		}

		// Find all transfers assigned to this manifest
		stmt := spanner.Statement{
			SQL:    `SELECT TransferId FROM InternalTransferOrders WHERE ManifestId = @mid`,
			Params: map[string]interface{}{"mid": req.ManifestID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		mutations := make([]*spanner.Mutation, 0)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var tid string
			if err := row.Columns(&tid); err != nil {
				return err
			}
			transferIDs = append(transferIDs, tid)
			mutations = append(mutations, spanner.Update("InternalTransferOrders",
				[]string{"TransferId", "ManifestId", "State", "UpdatedAt"},
				[]interface{}{tid, nil, "APPROVED", spanner.CommitTimestamp}))
		}

		// Mark manifest as CANCELLED
		mutations = append(mutations, spanner.Update("FactoryTruckManifests",
			[]string{"ManifestId", "State", "UpdatedAt"},
			[]interface{}{req.ManifestID, "CANCELLED", spanner.CommitTimestamp}))

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Emit MANIFEST_CANCELLED and PAYLOAD_SYNC via outbox — atomic with the mutation.
		if err := outbox.EmitJSON(txn, "Manifest", req.ManifestID, internalKafka.EventManifestCancelled, internalKafka.TopicMain,
			internalKafka.ManifestCancelledEvent{
				ManifestID:   req.ManifestID,
				SupplierID:   supplierID,
				FactoryID:    factoryScope.FactoryID,
				ReleasedIDs:  transferIDs,
				ReleasedKind: "TRANSFER",
				Reason:       cancelReason,
				CancelledBy:  claims.UserID,
				Timestamp:    time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return err
		}

		return outbox.EmitJSON(txn, "Manifest", req.ManifestID, internalKafka.EventPayloadSync, internalKafka.TopicMain,
			internalKafka.PayloadSyncEvent{
				SupplierID: supplierID,
				ManifestID: req.ManifestID,
				Reason:     "CANCELLED",
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusNotFound)
		} else if strings.Contains(errMsg, "must be in") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusConflict)
		} else {
			log.Printf("[CANCEL MANIFEST] error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// MANIFEST_CANCELLED event was emitted via outbox inside the transaction.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"manifest_id":        req.ManifestID,
		"status":             "CANCELLED",
		"transfers_released": len(transferIDs),
	})
}
