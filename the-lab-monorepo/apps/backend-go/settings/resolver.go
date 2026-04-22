package settings

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"backend-go/cache"

	"cloud.google.com/go/spanner"
)

// IsAutoOrderEnabled resolves the 5-level hierarchy for a specific SKU:
// Variant (SkuId) > Product > Category > Supplier > Global > false
//
// When rc is non-nil and Redis is reachable, the resolved boolean is cached
// under cache.SettingsKey(retailerID, "auto_order:"+supplierID+":"+categoryID+":"+skuID)
// for cache.TTLSettings. Callers that mutate RetailerXxxSettings tables MUST
// invalidate the affected keys via cache.Invalidate.
func IsAutoOrderEnabled(ctx context.Context, client *spanner.Client, rc *cache.Cache, retailerID, supplierID, categoryID, skuID string) bool {
	// Composite key that captures the full resolution scope.
	cacheKey := cache.SettingsKey(retailerID, fmt.Sprintf("auto_order:%s:%s:%s", supplierID, categoryID, skuID))

	// Read-through
	if rc != nil && rc.Client() != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		val, err := rc.Client().Get(cacheCtx, cacheKey).Result()
		cancel()
		if err == nil {
			b, _ := strconv.ParseBool(val)
			return b
		}
	}

	result := isAutoOrderEnabledFromSpanner(ctx, client, retailerID, supplierID, categoryID, skuID)

	// Backfill
	if rc != nil && rc.Client() != nil {
		go func() {
			setCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			rc.Client().Set(setCtx, cacheKey, strconv.FormatBool(result), cache.TTLSettings)
			cancel()
		}()
	}
	return result
}

func isAutoOrderEnabledFromSpanner(ctx context.Context, client *spanner.Client, retailerID, supplierID, categoryID, skuID string) bool {
	txn := client.Single()

	// 1. Check Variant-level override (highest precedence)
	if skuID != "" {
		row, err := txn.ReadRow(ctx, "RetailerVariantSettings",
			spanner.Key{retailerID, skuID}, []string{"AutoOrderEnabled"})
		if err == nil {
			var enabled bool
			if row.Columns(&enabled) == nil {
				return enabled
			}
		}
	}

	// 2. Check Product-level override — skuID maps to a product via SupplierProducts
	if skuID != "" {
		stmt := spanner.Statement{
			SQL: `SELECT rps.AutoOrderEnabled
			      FROM SupplierProducts sp
			      JOIN RetailerProductSettings rps ON rps.RetailerId = @rid AND rps.ProductId = sp.SkuId
			      WHERE sp.SkuId = @skuId
			      LIMIT 1`,
			Params: map[string]interface{}{"rid": retailerID, "skuId": skuID},
		}
		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		if err == nil {
			var enabled bool
			if row.Columns(&enabled) == nil {
				iter.Stop()
				return enabled
			}
		}
		iter.Stop()
	}

	// 3. Check Category-level override
	if categoryID != "" {
		row, err := txn.ReadRow(ctx, "RetailerCategorySettings",
			spanner.Key{retailerID, categoryID}, []string{"AutoOrderEnabled"})
		if err == nil {
			var enabled bool
			if row.Columns(&enabled) == nil {
				return enabled
			}
		}
	}

	// 4. Check Supplier-level override
	if supplierID != "" {
		row, err := txn.ReadRow(ctx, "RetailerSupplierSettings",
			spanner.Key{retailerID, supplierID}, []string{"AutoOrderEnabled"})
		if err == nil {
			var enabled bool
			if row.Columns(&enabled) == nil {
				return enabled
			}
		}
	}

	// 5. Fallback to Global master switch
	row, err := txn.ReadRow(ctx, "RetailerGlobalSettings",
		spanner.Key{retailerID}, []string{"GlobalAutoOrderEnabled"})
	if err == nil {
		var enabled bool
		if row.Columns(&enabled) == nil {
			return enabled
		}
	}

	// No settings exist → default OFF
	return false
}

// GetAnalyticsStartDate returns the analytics cut-off date for a retailer (global level only).
// Returns zero NullTime if no cut-off (use all history).
func GetAnalyticsStartDate(ctx context.Context, client *spanner.Client, retailerID string) (spanner.NullTime, error) {
	row, err := client.Single().ReadRow(ctx, "RetailerGlobalSettings",
		spanner.Key{retailerID}, []string{"AnalyticsStartDate"})
	if err != nil {
		return spanner.NullTime{}, nil // No settings = use all history
	}
	var startDate spanner.NullTime
	if err := row.Columns(&startDate); err != nil {
		log.Printf("[EMPATHY ENGINE] Failed to read AnalyticsStartDate for %s: %v", retailerID, err)
		return spanner.NullTime{}, nil
	}
	return startDate, nil
}

// GetAnalyticsStartDateForSku resolves the most specific AnalyticsStartDate for a given SKU.
// Hierarchy: Variant → Product → Category → Supplier → Global
// Returns zero time if no cut-off is set at any level (use all history).
func GetAnalyticsStartDateForSku(
	ctx context.Context,
	client *spanner.Client,
	retailerID, supplierID, categoryID, productID, skuID string,
) time.Time {
	txn := client.Single()
	var t spanner.NullTime

	// 1. Variant level
	if skuID != "" {
		row, err := txn.ReadRow(ctx, "RetailerVariantSettings",
			spanner.Key{retailerID, skuID}, []string{"AnalyticsStartDate"})
		if err == nil && row.Columns(&t) == nil && t.Valid {
			return t.Time
		}
	}

	// 2. Product level
	if productID != "" {
		row, err := txn.ReadRow(ctx, "RetailerProductSettings",
			spanner.Key{retailerID, productID}, []string{"AnalyticsStartDate"})
		if err == nil && row.Columns(&t) == nil && t.Valid {
			return t.Time
		}
	}

	// 3. Category level
	if categoryID != "" {
		row, err := txn.ReadRow(ctx, "RetailerCategorySettings",
			spanner.Key{retailerID, categoryID}, []string{"AnalyticsStartDate"})
		if err == nil && row.Columns(&t) == nil && t.Valid {
			return t.Time
		}
	}

	// 4. Supplier level
	if supplierID != "" {
		row, err := txn.ReadRow(ctx, "RetailerSupplierSettings",
			spanner.Key{retailerID, supplierID}, []string{"AnalyticsStartDate"})
		if err == nil && row.Columns(&t) == nil && t.Valid {
			return t.Time
		}
	}

	// 5. Global level
	row, err := txn.ReadRow(ctx, "RetailerGlobalSettings",
		spanner.Key{retailerID}, []string{"AnalyticsStartDate"})
	if err == nil && row.Columns(&t) == nil && t.Valid {
		return t.Time
	}

	// No cut-off configured — use all available history
	log.Printf("[EMPATHY ENGINE] No AnalyticsStartDate for retailer %s sku %s — using full history", retailerID, skuID)
	return time.Time{}
}
