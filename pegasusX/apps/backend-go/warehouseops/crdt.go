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

// ResolveLatestUpdates deduplicates a batch of offline updates by ProductID, keeping
// only the update with the strictly highest logical WriteClock.
func ResolveLatestUpdates(updates []LWWInventorySet) map[string]LWWInventorySet {
	updatesByProduct := make(map[string]LWWInventorySet)
	for _, u := range updates {
		pid := strings.TrimSpace(u.ProductID)
		if pid == "" {
			continue
		}
		u.ProductID = pid
		if existing, ok := updatesByProduct[pid]; ok {
			if u.WriteClock > existing.WriteClock {
				updatesByProduct[pid] = u
			}
		} else {
			updatesByProduct[pid] = u
		}
	}
	return updatesByProduct
}

// FilterStrictlyNewer filters updates to only those whose WriteClock strictly exceeds
// the corresponding clock in existingClocks.
func FilterStrictlyNewer(existingClocks map[string]int64, updatesByProduct map[string]LWWInventorySet) []LWWInventorySet {
	var newer []LWWInventorySet
	for pid, update := range updatesByProduct {
		currentClock := existingClocks[pid]
		if update.WriteClock > currentClock {
			newer = append(newer, update)
		}
	}
	return newer
}

// MergeOfflineAdjustments safely merges offline LWW updates into Spanner without blindly overwriting
// concurrent mutations that occurred with a higher logical clock.
func MergeOfflineAdjustments(ctx context.Context, txn *spanner.ReadWriteTransaction, updates []LWWInventorySet) error {
	if len(updates) == 0 {
		return nil
	}

	updatesByProduct := ResolveLatestUpdates(updates)
	if len(updatesByProduct) == 0 {
		return nil
	}

	productIDs := make([]string, 0, len(updatesByProduct))
	for pid := range updatesByProduct {
		productIDs = append(productIDs, pid)
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

	newer := FilterStrictlyNewer(existingClocks, updatesByProduct)
	var mutations []*spanner.Mutation
	for _, update := range newer {
		mutations = append(mutations, spanner.InsertOrUpdate("InventoryCRDT",
			[]string{"ProductID", "Quantity", "WriteClock", "WorkerNodeID"},
			[]interface{}{update.ProductID, update.Quantity, update.WriteClock, update.WorkerNodeID},
		))
	}

	if len(mutations) > 0 {
		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer CRDT mutations: %w", err)
		}
	}

	return nil
}
