package warehouseops

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// LWWInventorySet represents an offline-capable Last-Write-Wins CRDT update for warehouse inventory.
type LWWInventorySet struct {
	ProductID    string
	Quantity     int64
	WriteClock   int64
	WorkerNodeID string
}

// MergeOfflineAdjustments safely merges offline LWW updates into Spanner without blindly overwriting
// concurrent mutations that occurred with a higher logical clock.
func MergeOfflineAdjustments(ctx context.Context, txn *spanner.ReadWriteTransaction, updates []LWWInventorySet) error {
	if len(updates) == 0 {
		return nil
	}

	var productIDs []string
	updatesByProduct := make(map[string]LWWInventorySet)
	for _, u := range updates {
		pid := strings.TrimSpace(u.ProductID)
		if pid == "" {
			continue
		}
		productIDs = append(productIDs, pid)
		// If multiple updates for same product in this batch, keep latest clock
		if existing, ok := updatesByProduct[pid]; ok {
			if u.WriteClock > existing.WriteClock {
				updatesByProduct[pid] = u
			}
		} else {
			updatesByProduct[pid] = u
		}
	}

	if len(productIDs) == 0 {
		return nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT ProductID, Quantity, WriteClock, WorkerNodeID 
		      FROM InventoryCRDT 
		      WHERE ProductID IN UNNEST(@ids)`,
		Params: map[string]interface{}{
			"ids": productIDs,
		},
	}
	
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var mutations []*spanner.Mutation
	existingClocks := make(map[string]int64)

	for {
		row, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query existing CRDTs: %w", err)
		}
		var pid string
		var qty, clock int64
		var nodeID string
		if err := row.Columns(&pid, &qty, &clock, &nodeID); err != nil {
			return fmt.Errorf("parse CRDT row: %w", err)
		}
		existingClocks[pid] = clock
	}

	for pid, update := range updatesByProduct {
		currentClock := existingClocks[pid]
		// LWW merge rule: Only apply if incoming WriteClock is strictly greater
		if update.WriteClock > currentClock {
			mutations = append(mutations, spanner.InsertOrUpdate("InventoryCRDT",
				[]string{"ProductID", "Quantity", "WriteClock", "WorkerNodeID"},
				[]interface{}{pid, update.Quantity, update.WriteClock, update.WorkerNodeID},
			))
		}
	}

	if len(mutations) > 0 {
		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer CRDT mutations: %w", err)
		}
	}

	return nil
}
