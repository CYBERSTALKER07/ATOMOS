package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// NetworkSnapshot is a read-only supplier network state for scenario projection.
type NetworkSnapshot struct {
	SupplierID     string
	WarehouseCount int
	Inventory      map[string]int64 // sku -> available qty (summed across warehouses)
	OpenDemand     map[string]int64 // sku -> open order qty
	OpenOrderCount int64
	ActiveRoutes   int64
	CapturedAt     time.Time
}

const snapshotMaxSKUs = 5000

// LoadNetworkSnapshot reads live Spanner state without mutating it.
func LoadNetworkSnapshot(ctx context.Context, client *spanner.Client, supplierID string) (NetworkSnapshot, error) {
	snap := NetworkSnapshot{
		SupplierID: supplierID,
		Inventory:  make(map[string]int64),
		OpenDemand: make(map[string]int64),
		CapturedAt: time.Now().UTC(),
	}
	if client == nil || supplierID == "" {
		return snap, fmt.Errorf("snapshot unavailable")
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	if row, err := iter.Next(); err == nil {
		_ = row.Column(0, &snap.WarehouseCount)
	}

	iterInv := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, SUM(QuantityOnHand - QuantityReserved) AS Avail
		      FROM SupplierInventoryV2 WHERE SupplierId = @sid GROUP BY ProductId LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": snapshotMaxSKUs},
	})
	defer iterInv.Stop()
	for {
		row, err := iterInv.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return snap, err
		}
		var sku string
		var avail int64
		if err := row.Columns(&sku, &avail); err != nil {
			return snap, err
		}
		if avail > 0 {
			snap.Inventory[sku] = avail
		}
	}

	iterOrd := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT LineItemsJson FROM Orders
		      WHERE SupplierId = @sid AND Status IN ('PENDING','LOADED','IN_TRANSIT') LIMIT 2000`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iterOrd.Stop()
	for {
		row, err := iterOrd.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return snap, err
		}
		var linesJSON spanner.NullJSON
		if err := row.Column(0, &linesJSON); err != nil {
			continue
		}
		snap.OpenOrderCount++
		if !linesJSON.Valid {
			continue
		}
		var items []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		}
		raw, _ := json.Marshal(linesJSON.Value)
		if json.Unmarshal(raw, &items) != nil {
			continue
		}
		for _, it := range items {
			if it.SKU == "" || it.Quantity <= 0 {
				continue
			}
			snap.OpenDemand[it.SKU] += it.Quantity
		}
	}

	iterRoutes := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Routes WHERE SupplierId = @sid AND Status IN ('ACTIVE','IN_PROGRESS')`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iterRoutes.Stop()
	if row, err := iterRoutes.Next(); err == nil {
		_ = row.Column(0, &snap.ActiveRoutes)
	}

	return snap, nil
}

func (s NetworkSnapshot) TooLarge() bool {
	return len(s.Inventory) > snapshotMaxSKUs || len(s.OpenDemand) > snapshotMaxSKUs
}
