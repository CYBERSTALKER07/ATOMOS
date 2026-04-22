package factory

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Kill Switch — Global Replenishment Halt ───────────────────────────────────
// POST /v1/supplier/replenishment/kill-switch
//
// Immediately halts all automated replenishment for a supplier:
// 1. Cancels all DRAFT/APPROVED InternalTransferOrders with Source=SYSTEM_THRESHOLD or SYSTEM_PREDICTED
// 2. Sets NetworkOptimizationMode.Mode to MANUAL_ONLY
// 3. Emits NETWORK_MODE_CHANGED event for audit
//
// While in MANUAL_ONLY mode:
// - Pull Matrix cron skips this supplier
// - MANUAL_EMERGENCY transfers still work (analog fallback preserved)
// - Supplier can re-enable via PUT /v1/supplier/network-mode

// KillSwitchService handles emergency replenishment halt.
type KillSwitchService struct {
	Spanner  *spanner.Client
	Producer *kafkago.Writer
}

// HandleKillSwitch immediately halts automated replenishment.
// SOVEREIGN ACTION: Requires GLOBAL_ADMIN supplier role.
func (s *KillSwitchService) HandleKillSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req struct {
		Reason string `json:"reason"` // mandatory audit trail
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		http.Error(w, `{"error":"reason is required for audit trail"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Read current mode (pre-mutation snapshot for audit).
	oldMode := "BALANCED"
	modeRow, err := s.Spanner.Single().ReadRow(ctx, "NetworkOptimizationMode",
		spanner.Key{supplierID}, []string{"Mode"})
	if err == nil {
		modeRow.Columns(&oldMode)
	}

	// Read every automated DRAFT/APPROVED transfer that needs cancelling.
	cancelledIDs, err := s.listAutomatedTransfers(ctx, supplierID)
	if err != nil {
		log.Printf("[KILL_SWITCH] list transfers for %s: %v", supplierID, err)
		http.Error(w, `{"error":"cancel_failed"}`, http.StatusInternalServerError)
		return
	}

	// Atomic: cancel transfers + flip mode + emit both events through outbox.
	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(cancelledIDs)+1)
		for _, id := range cancelledIDs {
			mutations = append(mutations, spanner.Update("InternalTransferOrders",
				[]string{"TransferId", "State", "UpdatedAt"},
				[]interface{}{id, "CANCELLED", spanner.CommitTimestamp},
			))
		}
		mutations = append(mutations, spanner.InsertOrUpdate("NetworkOptimizationMode",
			[]string{"SupplierId", "Mode", "UpdatedAt", "UpdatedBy"},
			[]interface{}{supplierID, "MANUAL_ONLY", spanner.CommitTimestamp, claims.UserID},
		))
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		// Emit NETWORK_MODE_CHANGED via outbox.
		if err := outbox.EmitJSON(txn, "Supplier", supplierID, kafka.EventNetworkModeChanged, kafka.TopicMain,
			kafka.NetworkModeChangedEvent{
				SupplierId: supplierID,
				OldMode:    oldMode,
				NewMode:    "MANUAL_ONLY",
				ChangedBy:  claims.UserID,
				Reason:     req.Reason,
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
			return err
		}

		// Emit MANIFEST_CANCELLED for the batch of cancelled transfers.
		if len(cancelledIDs) > 0 {
			return outbox.EmitJSON(txn, "Transfer", supplierID, kafka.EventManifestCancelled, kafka.TopicMain,
				kafka.ManifestCancelledEvent{
					ManifestID:   "", // batch-cancel — no manifest scope
					SupplierID:   supplierID,
					ReleasedIDs:  cancelledIDs,
					ReleasedKind: "TRANSFER",
					Reason:       "KILL_SWITCH",
					CancelledBy:  claims.UserID,
					Timestamp:    time.Now().UTC(),
				}, telemetry.TraceIDFromContext(ctx))
		}
		return nil
	})
	if err != nil {
		log.Printf("[KILL_SWITCH] commit failed for %s: %v", supplierID, err)
		http.Error(w, `{"error":"commit_failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[KILL_SWITCH] Supplier %s: killed automated replenishment. Cancelled %d transfers. Reason: %s",
		supplierID, len(cancelledIDs), req.Reason)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":              "killed",
		"mode":                "MANUAL_ONLY",
		"transfers_cancelled": len(cancelledIDs),
		"reason":              req.Reason,
	})
}

// listAutomatedTransfers returns the IDs of all DRAFT/APPROVED automated transfers
// for the supplier — read-only helper used by the kill-switch transaction.
func (s *KillSwitchService) listAutomatedTransfers(ctx context.Context, supplierID string) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TransferId FROM InternalTransferOrders
		      WHERE SupplierId = @supplierID
		        AND State IN ('DRAFT', 'APPROVED')
		        AND Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED')`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}

	var ids []string
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		if err := row.Columns(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// HandleListKillSwitchAudit returns the PullMatrixRuns and recent FactorySLAEvents
// for a supplier — useful for understanding what the system has been doing.
func (s *KillSwitchService) HandleListKillSwitchAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Get current mode
	mode := "BALANCED"
	modeRow, err := s.Spanner.Single().ReadRow(r.Context(), "NetworkOptimizationMode",
		spanner.Key{supplierID}, []string{"Mode"})
	if err == nil {
		modeRow.Columns(&mode)
	}

	// Get recent pull matrix runs
	stmt := spanner.Statement{
		SQL: `SELECT RunId, RunAt, TransfersGenerated, SKUsProcessed, DurationMs, Source
		      FROM PullMatrixRuns WHERE SupplierId = @supplierID
		      ORDER BY RunAt DESC LIMIT 20`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}

	type runRow struct {
		RunId              string `json:"run_id"`
		RunAt              string `json:"run_at"`
		TransfersGenerated int64  `json:"transfers_generated"`
		SKUsProcessed      int64  `json:"skus_processed"`
		DurationMs         int64  `json:"duration_ms"`
		Source             string `json:"source"`
	}

	var runs []runRow
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var rr runRow
		var runAt time.Time
		if err := row.Columns(&rr.RunId, &runAt, &rr.TransfersGenerated, &rr.SKUsProcessed, &rr.DurationMs, &rr.Source); err != nil {
			continue
		}
		rr.RunAt = runAt.Format(time.RFC3339)
		runs = append(runs, rr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_mode":     mode,
		"pull_matrix_runs": runs,
	})
}
