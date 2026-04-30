// Package cart implements the B2B Dynamic Pricing Engine.
// All arithmetic is exact INT64 — zero floating-point drift.
// Discount formula: (Quantity * UnitPrice) * (100 - DiscountPct) / 100
package cart

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Domain Types ──────────────────────────────────────────────────────────────

// OrderLineItem represents a single SKU in a B2B checkout cart.
type OrderLineItem struct {
	SkuId     string `json:"sku_id"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// Tier mirrors a PricingTiers row from Spanner.
type Tier struct {
	SkuId       string
	MinPallets  int64
	DiscountPct int64
}

// ── Pricing Engine ────────────────────────────────────────────────────────────

// CalculateB2BTotal queries live PricingTiers from Spanner, applies the best
// eligible discount bracket for each line item, and returns the exact INT64
// total in minor currency units. Zero floats anywhere in the pipeline.
func CalculateB2BTotal(ctx context.Context, client *spanner.Client, items []OrderLineItem) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	// Build a set of SkuIds from the cart for the IN clause.
	skuSet := make(map[string]struct{}, len(items))
	for _, item := range items {
		skuSet[item.SkuId] = struct{}{}
	}

	skuList := make([]string, 0, len(skuSet))
	for sku := range skuSet {
		skuList = append(skuList, sku)
	}

	// Fetch all tiers for the SKUs in this cart, ordered so the highest bracket
	// comes first. We keep all rows; the selection logic below picks the best fit.
	stmt := spanner.Statement{
		SQL: `SELECT SkuId, MinPallets, DiscountPct
		      FROM PricingTiers
		      WHERE SkuId IN UNNEST(@skus)
		      ORDER BY SkuId, MinPallets DESC`,
		Params: map[string]interface{}{
			"skus": skuList,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	// Group tiers by SkuId (already sorted DESC by MinPallets from Spanner).
	tiersBysku := make(map[string][]Tier)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("cart: tier query failed: %w", err)
		}

		var t Tier
		if err := row.Columns(&t.SkuId, &t.MinPallets, &t.DiscountPct); err != nil {
			return 0, fmt.Errorf("cart: tier row parse failed: %w", err)
		}
		tiersBysku[t.SkuId] = append(tiersBysku[t.SkuId], t)
	}

	var total int64
	for _, item := range items {
		lineTotal := item.Quantity * item.UnitPrice

		discountPct := int64(0)
		if tiers, ok := tiersBysku[item.SkuId]; ok {
			// Tiers are DESC by MinPallets — first qualifying tier wins.
			for _, tier := range tiers {
				if item.Quantity >= tier.MinPallets {
					discountPct = tier.DiscountPct
					break
				}
			}
		}

		// Apply discount: avoid any float division.
		discounted := lineTotal * (100 - discountPct) / 100

		log.Printf("[PricingEngine] SkuId=%s Qty=%d UnitPrice=%d Discount=%d%% LineTotal=%d",
			item.SkuId, item.Quantity, item.UnitPrice, discountPct, discounted)

		total += discounted
	}

	return total, nil
}

// ── Per-Retailer Pricing Resolution ───────────────────────────────────────────

// ResolveRetailerPrice resolves the effective price for a retailer + SKU.
// Resolution order: per-retailer override → tier discount → base price.
// Returns (effectivePrice, isSpecialPrice, error).
func ResolveRetailerPrice(ctx context.Context, client *spanner.Client, supplierID, retailerID, skuID string, basePrice int64) (int64, bool, error) {
	// Priority 1: Per-retailer override (most specific)
	overrideStmt := spanner.Statement{
		SQL: `SELECT OverridePrice, ExpiresAt FROM RetailerPricingOverrides
		      WHERE SupplierId = @sid AND RetailerId = @rid AND SkuId = @skuId
		        AND IsActive = true
		      LIMIT 1`,
		Params: map[string]interface{}{
			"sid":   supplierID,
			"rid":   retailerID,
			"skuId": skuID,
		},
	}
	iter := client.Single().Query(ctx, overrideStmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == nil {
		var overridePrice int64
		var expiresAt spanner.NullTime
		if err := row.Columns(&overridePrice, &expiresAt); err == nil {
			// Check expiry
			if !expiresAt.Valid || expiresAt.Time.After(time.Now()) {
				return overridePrice, true, nil
			}
		}
	}

	// Priority 2: Tier-based discount (existing PricingTiers)
	tierStmt := spanner.Statement{
		SQL: `SELECT RetailerTier FROM SupplierRetailerClients
		      WHERE SupplierId = @sid AND RetailerId = @rid`,
		Params: map[string]interface{}{"sid": supplierID, "rid": retailerID},
	}
	tierIter := client.Single().Query(ctx, tierStmt)
	defer tierIter.Stop()

	retailerTier := "BRONZE" // default
	if tRow, err := tierIter.Next(); err == nil {
		var tier string
		if err := tRow.Columns(&tier); err == nil && tier != "" {
			retailerTier = tier
		}
	}

	discountStmt := spanner.Statement{
		SQL: `SELECT DiscountPct FROM PricingTiers
		      WHERE SupplierId = @sid AND SkuId = @skuId AND IsActive = true
		        AND (TargetRetailerTier = 'ALL' OR TargetRetailerTier = @tier)
		      ORDER BY MinPallets ASC LIMIT 1`,
		Params: map[string]interface{}{
			"sid":   supplierID,
			"skuId": skuID,
			"tier":  retailerTier,
		},
	}
	dIter := client.Single().Query(ctx, discountStmt)
	defer dIter.Stop()
	if dRow, err := dIter.Next(); err == nil {
		var discountPct int64
		if err := dRow.Columns(&discountPct); err == nil && discountPct > 0 {
			discounted := basePrice - (basePrice * discountPct / 100)
			if discounted > 0 {
				return discounted, true, nil
			}
		}
	}

	// Priority 3: Base price (no special pricing)
	return basePrice, false, nil
}

// ── Delivery Fee Calculation ──────────────────────────────────────────────────

// CalculateDeliveryFee resolves the delivery fee for an order based on supplier-defined
// DeliveryZones. Matches the retailer's distance from the fulfillment warehouse to the
// appropriate zone band. Returns 0 if no zones are configured (free delivery default).
func CalculateDeliveryFee(ctx context.Context, client *spanner.Client, supplierID, warehouseID string, distanceKm float64) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FeeMinor FROM DeliveryZones
		      WHERE SupplierId = @sid
		        AND (WarehouseId = @wid OR WarehouseId IS NULL)
		        AND IsActive = true
		        AND MinDistanceKm <= @dist
		        AND MaxDistanceKm > @dist
		      ORDER BY Priority DESC, WarehouseId DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"sid":  supplierID,
			"wid":  warehouseID,
			"dist": distanceKm,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		// No matching zone = free delivery
		return 0, nil
	}

	var fee int64
	if err := row.Columns(&fee); err != nil {
		return 0, fmt.Errorf("cart: delivery zone fee parse failed: %w", err)
	}

	log.Printf("[PricingEngine] DeliveryFee supplier=%s warehouse=%s dist=%.1fkm fee=%d",
		supplierID, warehouseID, distanceKm, fee)

	return fee, nil
}
