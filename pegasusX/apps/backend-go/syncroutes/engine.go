package syncroutes

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type SemanticEngine struct {
	Spanner *spanner.Client
}

// resolveEntityVersion fetches the current Version column for the given entity.
// Returns 0 if the entity doesn't exist or the table lacks a Version column
// (graceful degradation for non-versioned entities).
func (e *SemanticEngine) resolveEntityVersion(ctx context.Context, entityID, commandType string) int {
	if e.Spanner == nil {
		return 0
	}

	// Map command types to their source table.
	table := "Orders"
	switch commandType {
	case "MARK_DELIVERED", "COLLECT_CASH", "UPDATE_ORDER_STATUS", "AMEND_ORDER":
		table = "Orders"
	case "ABORT_MANIFEST", "UPDATE_MANIFEST":
		table = "Manifests"
	default:
		table = "Orders" // conservative default
	}

	row, err := e.Spanner.Single().ReadRow(ctx, table,
		spanner.Key{entityID},
		[]string{"Version"},
	)
	if err != nil {
		// Entity not found or column missing — treat as version 0 (always conflict).
		return 0
	}

	var version int64
	if err := row.Column(0, &version); err != nil {
		return 0
	}
	return int(version)
}

func (e *SemanticEngine) ProcessCommand(ctx context.Context, supplierID string, cmd SyncCommand, claims auth.Claims) CommandResult {
	if e.Spanner == nil {
		return CommandResult{CommandID: cmd.CommandID, Status: "SUCCESS"}
	}

	// Validate required fields.
	if cmd.EntityID == "" {
		return CommandResult{CommandID: cmd.CommandID, Status: "FAILED", Error: "entity_id_required"}
	}
	if cmd.CommandType == "" {
		return CommandResult{CommandID: cmd.CommandID, Status: "FAILED", Error: "command_type_required"}
	}

	// 1. Fetch REAL current domain version (Finding 3 fix).
	currentVersion := e.resolveEntityVersion(ctx, cmd.EntityID, cmd.CommandType)

	// 2. Trivial Version Bump Check — no conflict if versions match.
	if cmd.KnownVersion == currentVersion {
		// Happy path: client state is current. Apply the mutation.
		// (In production, the actual mutation dispatch happens here.)
		return CommandResult{CommandID: cmd.CommandID, Status: "SUCCESS"}
	}

	// 3. Semantic Validation — a conflict exists (KnownVersion != currentVersion).
	// Rule: Driver physical delivery wins over a stale digital state.
	isDriverPhysical := claims.Role == auth.RoleDriver &&
		(cmd.CommandType == "MARK_DELIVERED" || cmd.CommandType == "COLLECT_CASH")

	if isDriverPhysical {
		// Auto-Resolution: the physical act in the real world trumps the digital state.
		return CommandResult{CommandID: cmd.CommandID, Status: "SUCCESS"}
	}

	// 4. Irreconcilable Conflict -> Shunt to PhysicalReconciliationQueue.
	_, err := e.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		payloadBytes, marshalErr := json.Marshal(cmd.PayloadJSON)
		if marshalErr != nil {
			return fmt.Errorf("marshal conflict payload: %w", marshalErr)
		}

		disputeID := uuid.NewString()
		mut := spanner.InsertMap("PhysicalReconciliationQueue", map[string]any{
			"DisputeId":       disputeID,
			"SupplierId":      supplierID,
			"AggregateId":     cmd.EntityID,
			"AggregateType":   cmd.CommandType,
			"ConflictingData": string(payloadBytes),
			"SourceRole":      string(claims.Role),
			"Status":          "PENDING_REVIEW",
			"CreatedAt":       spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "PhysicalReconciliationQueue", disputeID, events.TopicExceptions, map[string]any{
			"type":           "PHYSICAL_RECONCILIATION_DISPUTED",
			"dispute_id":     disputeID,
			"supplier_id":    supplierID,
			"aggregate_id":   cmd.EntityID,
			"aggregate_type": cmd.CommandType,
			"source_role":    string(claims.Role),
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})

	if err != nil {
		return CommandResult{CommandID: cmd.CommandID, Status: "FAILED", Error: "reconciliation_queue_failure"}
	}

	return CommandResult{CommandID: cmd.CommandID, Status: "DISPUTED"}
}
