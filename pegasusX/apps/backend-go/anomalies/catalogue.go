package anomalies

import "strings"

var catalogueRules = []CatalogueRule{
	{
		RuleCode:              RuleCashGeofenceMismatch,
		Name:                  "Cash Shortfall & Geofence Mismatch",
		Category:              CategoryCashFraud,
		DefaultSeverity:       SeverityHigh,
		Description:           "Detects depot blind-count cash shortfalls, vault ledger balance discrepancies, and shop-closed geofence mismatches across field cash collections.",
		DefaultThresholdBps:   200,     // 2.00% variance
		DefaultThresholdMinor: 500_000, // 5,000 UZS in tiyin
		PlaybookAction:        "REQUIRE_DEPOT_RECOUNT",
		SourceTables:          []string{"CashierBlindCountSessions", "DepotVaultLedgerEntries", "OrderShopClosedLog", "Orders"},
	},
	{
		RuleCode:              RuleBasketSpikeSharedINN,
		Name:                  "Basket Spike & Shared INN Collusion",
		Category:              CategoryCollusionFraud,
		DefaultSeverity:       SeverityHigh,
		Description:           "Identifies multiple retailer accounts sharing a NormalizedTaxId (INN) and abnormal order basket size spikes against baseline order averages.",
		DefaultThresholdBps:   20000,     // 200% (2.0x baseline basket)
		DefaultThresholdMinor: 5_000_000, // 50,000 UZS in tiyin
		PlaybookAction:        "HOLD_ORDER_VERIFY_INN",
		SourceTables:          []string{"Orders", "Retailers"},
	},
	{
		RuleCode:              RuleStockCountVariance,
		Name:                  "Stock-to-GL & Cycle Count Variance",
		Category:              CategoryInventoryLoss,
		DefaultSeverity:       SeverityHigh,
		Description:           "Detects unresolved STOCK_GL reconciliation breaks, cycle count quantity variances, and unapproved inventory adjustments.",
		DefaultThresholdBps:   500,       // 5.00% quantity variance
		DefaultThresholdMinor: 1_000_000, // 10,000 UZS in tiyin
		PlaybookAction:        "CREATE_CYCLE_COUNT_AUDIT",
		SourceTables:          []string{"ReconciliationBreaks", "CycleCounts", "InventoryAdjustments", "InventoryCostLayers"},
	},
	{
		RuleCode:              RulePricingDiscountAbuse,
		Name:                  "Below-Cost & Extreme Discount Abuse",
		Category:              CategoryPricingFraud,
		DefaultSeverity:       SeverityHigh,
		Description:           "Flags order lines priced below FEFO inventory unit cost or discounted beyond approved catalog thresholds.",
		DefaultThresholdBps:   3000, // 30.00% discount vs catalog/price list
		DefaultThresholdMinor: 0,
		PlaybookAction:        "FLAG_BELOW_COST_ORDER",
		SourceTables:          []string{"Orders", "Products", "PriceListItems", "InventoryCostLayers"},
	},
	{
		RuleCode:              RuleManualJournalAfterHours,
		Name:                  "After-Hours & Split Manual Journals",
		Category:              CategoryGLControls,
		DefaultSeverity:       SeverityHigh,
		Description:           "Flags manual GL journal entries posted outside 08:00–20:00 UZT (UTC+5) or structured into same-day split batches under approval thresholds.",
		DefaultThresholdBps:   9000,       // 90.00% split structuring sensitivity
		DefaultThresholdMinor: 10_000_000, // 100,000 UZS in tiyin
		PlaybookAction:        "ESCALATE_GL_REVIEW",
		SourceTables:          []string{"JournalEntries", "JournalPostings"},
	},
}

// DefaultCatalogue returns a deterministic copy of all 5 §7.7 detector rules.
func DefaultCatalogue() []CatalogueRule {
	out := make([]CatalogueRule, len(catalogueRules))
	for i, r := range catalogueRules {
		cp := r
		if len(r.SourceTables) > 0 {
			cp.SourceTables = append([]string(nil), r.SourceTables...)
		}
		out[i] = cp
	}
	return out
}

// GetRule looks up a rule by canonical RuleCode.
func GetRule(code string) (CatalogueRule, bool) {
	norm := strings.ToUpper(strings.TrimSpace(code))
	for _, r := range catalogueRules {
		if r.RuleCode == norm {
			cp := r
			cp.SourceTables = append([]string(nil), r.SourceTables...)
			return cp, true
		}
	}
	return CatalogueRule{}, false
}
