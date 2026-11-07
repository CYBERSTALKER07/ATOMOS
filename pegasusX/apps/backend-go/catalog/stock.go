package catalog

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/api/iterator"
)

// StockSnapshot is warehouse-scoped availability for one SKU.
type StockSnapshot struct {
	AvailableStock   int64
	IsOutOfStock     bool
	AcceptsBackorder bool
	ShowStockCounts  bool
	WarehouseID      string
}

// StockEnricher resolves per-retailer warehouse stock for catalog rows.
type StockEnricher struct {
	client *spanner.Client
}

// NewStockEnricher builds a Spanner-backed stock enricher.
func NewStockEnricher(client *spanner.Client) *StockEnricher {
	if client == nil {
		return nil
	}
	return &StockEnricher{client: client}
}

func (e *StockEnricher) Enrich(
	ctx context.Context,
	retailerID string,
	products []Product,
) map[string]StockSnapshot {
	out := make(map[string]StockSnapshot)
	if e == nil || e.client == nil || retailerID == "" || len(products) == 0 {
		return out
	}

	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return out
	}
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return out
	}

	activeLocationID := ""
	if claims, ok := auth.FromContext(ctx); ok {
		activeLocationID = strings.TrimSpace(claims.ActiveLocationID)
	}
	store, err := (&proximity.CoverageStore{Client: e.client}).LoadStore(ctx, retailerID, activeLocationID)
	if err != nil || (store.Lat == 0 && store.Lng == 0) {
		return out
	}

	bySupplier := make(map[string][]string)
	for _, p := range products {
		sid := strings.TrimSpace(p.SupplierID)
		pid := strings.TrimSpace(p.ProductID)
		if sid == "" || pid == "" {
			continue
		}
		bySupplier[sid] = append(bySupplier[sid], pid)
	}

	cov := &proximity.CoverageStore{Client: e.client}
	for supplierID, productIDs := range bySupplier {
		warehouses, listErr := cov.ListWarehouses(ctx, supplierID)
		if listErr != nil {
			continue
		}
		pins, pinErr := cov.ListPins(ctx, supplierID)
		if pinErr != nil {
			pins = nil
		}
		warehouseID, resolveErr := proximity.ResolveServingWarehouse(packCountry, store, warehouses, pins)
		if resolveErr != nil || warehouseID == "" {
			continue
		}
		policy := "REJECT"
		showCounts := false
		for _, wh := range warehouses {
			if wh.WarehouseID != warehouseID {
				continue
			}
			if strings.TrimSpace(wh.DefaultOutOfStockPolicy) != "" {
				policy = wh.DefaultOutOfStockPolicy
			}
			showCounts = wh.ShowStockCounts
			break
		}
		snaps, snapErr := loadStockSnapshots(ctx, e.client, supplierID, warehouseID, policy, showCounts, productIDs)
		if snapErr != nil {
			continue
		}
		for pid, snap := range snaps {
			out[supplierID+":"+pid] = snap
		}
	}
	return out
}

func loadStockSnapshots(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID, warehousePolicy string,
	showCounts bool,
	productIDs []string,
) (map[string]StockSnapshot, error) {
	keys := make([]spanner.KeySet, 0, len(productIDs))
	for _, pid := range productIDs {
		keys = append(keys, spanner.Key{supplierID, warehouseID, pid})
	}
	iter := client.Single().Read(ctx, "SupplierInventoryV2", spanner.KeySets(keys...),
		[]string{"ProductId", "QuantityOnHand", "QuantityReserved", "OutOfStockPolicy"})
	defer iter.Stop()

	out := make(map[string]StockSnapshot, len(productIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read supplier inventory: %w", err)
		}
		var productID string
		var qoh, qr int64
		var policy spanner.NullString
		if err := row.Columns(&productID, &qoh, &qr, &policy); err != nil {
			continue
		}
		avail := qoh - qr
		if avail < 0 {
			avail = 0
		}
		productPolicy := ""
		if policy.Valid {
			productPolicy = policy.StringVal
		}
		effectivePolicy := resolvePolicy(warehousePolicy, productPolicy)
		out[productID] = StockSnapshot{
			AvailableStock:   avail,
			IsOutOfStock:     avail <= 0,
			AcceptsBackorder: effectivePolicy == "ACCEPT_BACKORDER",
			ShowStockCounts:  showCounts,
			WarehouseID:      warehouseID,
		}
	}
	for _, pid := range productIDs {
		if _, ok := out[pid]; ok {
			continue
		}
		effectivePolicy := resolvePolicy(warehousePolicy, "")
		out[pid] = StockSnapshot{
			AvailableStock:   0,
			IsOutOfStock:     true,
			AcceptsBackorder: effectivePolicy == "ACCEPT_BACKORDER",
			WarehouseID:      warehouseID,
		}
	}
	return out, nil
}

func resolvePolicy(warehouseDefault, productOverride string) string {
	override := strings.ToUpper(strings.TrimSpace(productOverride))
	if override == "ACCEPT_BACKORDER" || override == "REJECT" {
		return override
	}
	def := strings.ToUpper(strings.TrimSpace(warehouseDefault))
	if def == "ACCEPT_BACKORDER" {
		return "ACCEPT_BACKORDER"
	}
	return "REJECT"
}
