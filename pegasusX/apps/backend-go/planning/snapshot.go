package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// NetworkSnapshot is a read-only supplier network state for scenario projection.
type NetworkSnapshot struct {
	SupplierID      string
	WarehouseCount  int
	Inventory       map[string]int64 // sku -> available qty (summed across warehouses)
	OpenDemand      map[string]int64 // sku -> open order qty
	UnitValueMinor  map[string]int64 // sku -> Products.PriceMinor
	OpenOrderCount  int64
	ActiveRoutes    int64
	CapturedAt      time.Time
	UnitValueSource string // products | fallback | mixed (set during projection)
}

const snapshotMaxSKUs = 5000

// LoadNetworkSnapshot reads live Spanner state without mutating it.
func LoadNetworkSnapshot(ctx context.Context, client *spanner.Client, supplierID string) (NetworkSnapshot, error) {
	snap := NetworkSnapshot{
		SupplierID:     supplierID,
		Inventory:      make(map[string]int64),
		OpenDemand:     make(map[string]int64),
		UnitValueMinor: make(map[string]int64),
		CapturedAt:     time.Now().UTC(),
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

	skuSet := make(map[string]struct{}, len(snap.Inventory)+len(snap.OpenDemand))
	for sku := range snap.Inventory {
		skuSet[sku] = struct{}{}
	}
	for sku := range snap.OpenDemand {
		skuSet[sku] = struct{}{}
	}
	values, err := loadProductUnitValues(ctx, client, supplierID, skuSet)
	if err != nil {
		return snap, err
	}
	snap.UnitValueMinor = values

	return snap, nil
}

func (s NetworkSnapshot) TooLarge() bool {
	return len(s.Inventory) > snapshotMaxSKUs || len(s.OpenDemand) > snapshotMaxSKUs
}

func loadProductUnitValues(ctx context.Context, client *spanner.Client, supplierID string, skus map[string]struct{}) (map[string]int64, error) {
	out := make(map[string]int64)
	if client == nil || strings.TrimSpace(supplierID) == "" || len(skus) == 0 {
		return out, nil
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ProductId, PriceMinor FROM Products WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, err
		}
		var pid string
		var price int64
		if err := row.Columns(&pid, &price); err != nil {
			return out, err
		}
		if _, ok := skus[pid]; !ok || price <= 0 {
			continue
		}
		out[pid] = price
	}
	return out, nil
}

func scenarioUnitValueFallbackMinor() int64 {
	v := strings.TrimSpace(os.Getenv("SCENARIO_UNIT_VALUE_MINOR"))
	if v == "" {
		v = strings.TrimSpace(os.Getenv("MEIO_UNIT_VALUE_MINOR"))
	}
	if v == "" {
		return 10000
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 10000
	}
	return n
}

func unitValueForSKU(values map[string]int64, sku string, fallback int64) (int64, bool) {
	if values != nil {
		if v := values[sku]; v > 0 {
			return v, true
		}
	}
	if fallback <= 0 {
		fallback = 10000
	}
	return fallback, false
}

func classifyUnitValueSource(usedProduct, usedFallback bool) string {
	switch {
	case usedProduct && usedFallback:
		return "mixed"
	case usedProduct:
		return "products"
	default:
		return "fallback"
	}
}
