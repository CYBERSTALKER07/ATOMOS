package catalog

import (
	"context"
	"fmt"
	"math"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// StockSnapshot is warehouse-scoped availability for one SKU.
type StockSnapshot struct {
	AvailableStock   int64
	IsOutOfStock     bool
	AcceptsBackorder bool
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

	lat, lng, err := loadRetailerCoordinates(ctx, e.client, retailerID)
	if err != nil || (lat == 0 && lng == 0) {
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

	for supplierID, productIDs := range bySupplier {
		warehouseID, warehousePolicy, err := resolveNearestWarehouse(ctx, e.client, supplierID, lat, lng)
		if err != nil || warehouseID == "" {
			continue
		}
		snaps, err := loadStockSnapshots(ctx, e.client, supplierID, warehouseID, warehousePolicy, productIDs)
		if err != nil {
			continue
		}
		for pid, snap := range snaps {
			out[supplierID+":"+pid] = snap
		}
	}
	return out
}

func loadRetailerCoordinates(ctx context.Context, client *spanner.Client, retailerID string) (float64, float64, error) {
	row, err := client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Latitude", "Longitude"})
	if err != nil {
		return 0, 0, err
	}
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return 0, 0, err
	}
	if lat.Valid && lng.Valid {
		return lat.Float64, lng.Float64, nil
	}
	return 0, 0, nil
}

func resolveNearestWarehouse(ctx context.Context, client *spanner.Client, supplierID string, lat, lng float64) (string, string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, CoverageRadiusKm, COALESCE(DefaultOutOfStockPolicy, 'REJECT')
		      FROM Warehouses
		      WHERE SupplierId = @supplierId AND IsActive = true`,
		Params: map[string]any{"supplierId": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	bestID := ""
	bestPolicy := "REJECT"
	bestDist := math.MaxFloat64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", "", err
		}
		var warehouseID, policy string
		var wLat, wLng, radius spanner.NullFloat64
		if err := row.Columns(&warehouseID, &wLat, &wLng, &radius, &policy); err != nil {
			continue
		}
		if !wLat.Valid || !wLng.Valid {
			continue
		}
		dist := haversineKm(lat, lng, wLat.Float64, wLng.Float64)
		effectiveRadius := math.MaxFloat64
		if radius.Valid && radius.Float64 > 0 {
			effectiveRadius = radius.Float64
		}
		if dist <= effectiveRadius && dist < bestDist {
			bestDist = dist
			bestID = warehouseID
			bestPolicy = strings.ToUpper(strings.TrimSpace(policy))
			if bestPolicy == "" {
				bestPolicy = "REJECT"
			}
		}
	}
	return bestID, bestPolicy, nil
}

func loadStockSnapshots(
	ctx context.Context,
	client *spanner.Client,
	supplierID, warehouseID, warehousePolicy string,
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

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
