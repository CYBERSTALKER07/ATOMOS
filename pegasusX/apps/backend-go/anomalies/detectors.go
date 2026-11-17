package anomalies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var uztZone = time.FixedZone("UZT", 5*3600)

// CashierBlindCountRecord captures a live CashierBlindCountSessions row.
type CashierBlindCountRecord struct {
	SessionID         string
	SupplierID        string
	DepotID           string
	DriverID          string
	RouteID           string
	CountedCashMinor  int64
	ExpectedCashMinor int64
	VarianceMinor     int64
	Status            string
	CreatedAt         time.Time
}

// DepotVaultLedgerRecord captures a live DepotVaultLedgerEntries row.
type DepotVaultLedgerRecord struct {
	VaultEntryID        string
	DepotID             string
	SupplierID          string
	SessionID           string
	EntryType           string
	AmountMinor         int64
	RunningBalanceMinor int64
	CreatedAt           time.Time
}

// ShopClosedLogRecord captures a live OrderShopClosedLog + Orders + Retailers join row.
type ShopClosedLogRecord struct {
	OrderID         string
	EventID         string
	Actor           string
	Action          string
	PayloadJSON     string
	OrderTotalMinor int64
	OrderH3Cell     string
	RetailerID      string
	RetailerH3Cell  string
	ProximityMethod string
	ShopClosedRes   string
	CreatedAt       time.Time
}

// OrderRecord captures a live Orders row joined with Retailer tax metadata.
type OrderRecord struct {
	OrderID            string
	SupplierID         string
	RetailerID         string
	RetailerName       string
	NormalizedTaxID    string
	Status             string
	TotalMinor         int64
	OriginalTotalMinor int64
	Currency           string
	LineItemsJSON      string
	CreatedAt          time.Time
}

// OrderLineRecord captures line-level pricing extracted from Orders.LineItemsJson.
type OrderLineRecord struct {
	SKU            string `json:"sku"`
	ProductID      string `json:"product_id,omitempty"`
	Name           string `json:"name,omitempty"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	ListPriceMinor int64  `json:"list_price_minor,omitempty"`
	DiscountBps    int64  `json:"discount_bps,omitempty"`
}

// ReconBreakRecord captures a live ReconciliationBreaks row.
type ReconBreakRecord struct {
	BreakID       string
	RunID         string
	Dimension     string
	ObjectType    string
	ObjectID      string
	ExpectedMinor int64
	ActualMinor   int64
	DeltaMinor    int64
	Currency      string
	Severity      string
	Status        string
	CreatedAt     time.Time
}

// CycleCountRecord captures a live CycleCounts row scoped to tenant warehouses.
type CycleCountRecord struct {
	CountID     string
	WarehouseID string
	LocationID  string
	ProductID   string
	LotID       string
	ExpectedQty int64
	CountedQty  int64
	VarianceQty int64
	HasCounted  bool
	ReasonCode  string
	Status      string
	CreatedAt   time.Time
}

// InventoryAdjustmentRecord captures a live InventoryAdjustments row scoped to tenant warehouses.
type InventoryAdjustmentRecord struct {
	AdjustmentID string
	WarehouseID  string
	ProductID    string
	LotID        string
	DeltaQty     int64
	ReasonCode   string
	Status       string
	CreatedAt    time.Time
}

// CostLayerSummary captures weighted unit cost per SKU/ProductId from InventoryCostLayers.
type CostLayerSummary struct {
	ProductID     string
	UnitCostMinor int64
}

// ProductPriceRecord captures catalog/price-list pricing for a SKU/ProductId.
type ProductPriceRecord struct {
	ProductID      string
	SKU            string
	UnitPriceMinor int64
}

// ManualJournalRecord captures a live JournalEntries row with summed debit AmountMinor.
type ManualJournalRecord struct {
	EntityID        string
	EntryID         string
	PeriodID        string
	SourceType      string
	SourceID        string
	Memo            string
	Currency        string
	PostedBy        string
	DebitTotalMinor int64
	OccurredAt      time.Time
	PostedAt        time.Time
}

// DetectorDataset holds a consistent point-in-time snapshot across Spanner tables.
type DetectorDataset struct {
	TenantID             string
	EntityID             string
	CashierSessions      []CashierBlindCountRecord
	DepotVaultEntries    []DepotVaultLedgerRecord
	ShopClosedLogs       []ShopClosedLogRecord
	Orders               []OrderRecord
	ReconBreaks          []ReconBreakRecord
	CycleCounts          []CycleCountRecord
	InventoryAdjustments []InventoryAdjustmentRecord
	CostLayers           []CostLayerSummary
	ProductPrices        []ProductPriceRecord
	ManualJournals       []ManualJournalRecord
}

// DetectedFinding represents a candidate AnomalyCase produced by a deterministic rule.
type DetectedFinding struct {
	RuleCode       string
	Category       string
	Severity       string
	ObjectType     string
	ObjectID       string
	ExposureMinor  int64
	ScoreBps       int64
	PlaybookAction string
	Evidence       map[string]any
	DetectedAt     time.Time
}

// EvalOptions overrides optional thresholds when running detectors.
type EvalOptions struct {
	RuleCodes      []string
	ThresholdBps   *int64
	ThresholdMinor *int64
	Now            time.Time
}

// EvaluateDetectors executes the selected §7.7 detector rules deterministically in pure int64 arithmetic.
func EvaluateDetectors(ds DetectorDataset, opts EvalOptions) ([]DetectedFinding, []string) {
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	selected := normalizeRuleSelection(opts.RuleCodes)
	var findings []DetectedFinding
	var scanned []string

	for _, rule := range catalogueRules {
		if !selected[rule.RuleCode] {
			continue
		}
		scanned = append(scanned, rule.RuleCode)
		bps := rule.DefaultThresholdBps
		if opts.ThresholdBps != nil && *opts.ThresholdBps >= 0 {
			bps = *opts.ThresholdBps
		}
		minor := rule.DefaultThresholdMinor
		if opts.ThresholdMinor != nil && *opts.ThresholdMinor >= 0 {
			minor = *opts.ThresholdMinor
		}

		switch rule.RuleCode {
		case RuleCashGeofenceMismatch:
			findings = append(findings, evalCashGeofenceMismatch(ds, rule, bps, minor, now)...)
		case RuleBasketSpikeSharedINN:
			findings = append(findings, evalBasketSpikeSharedINN(ds, rule, bps, minor, now)...)
		case RuleStockCountVariance:
			findings = append(findings, evalStockCountVariance(ds, rule, bps, minor, now)...)
		case RulePricingDiscountAbuse:
			findings = append(findings, evalPricingDiscountAbuse(ds, rule, bps, minor, now)...)
		case RuleManualJournalAfterHours:
			findings = append(findings, evalManualJournalAfterHoursOrSplit(ds, rule, bps, minor, now)...)
		}
	}

	// Deduplicate within the scan by (RuleCode, ObjectType, ObjectID), keeping highest severity/score.
	deduped := make(map[string]DetectedFinding, len(findings))
	for _, f := range findings {
		key := FindingKey(f.RuleCode, f.ObjectType, f.ObjectID)
		if existing, ok := deduped[key]; ok {
			if f.ScoreBps > existing.ScoreBps || (f.ScoreBps == existing.ScoreBps && f.ExposureMinor > existing.ExposureMinor) {
				deduped[key] = f
			}
			continue
		}
		deduped[key] = f
	}

	out := make([]DetectedFinding, 0, len(deduped))
	for _, f := range deduped {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScoreBps != out[j].ScoreBps {
			return out[i].ScoreBps > out[j].ScoreBps
		}
		if out[i].ExposureMinor != out[j].ExposureMinor {
			return out[i].ExposureMinor > out[j].ExposureMinor
		}
		if out[i].RuleCode != out[j].RuleCode {
			return out[i].RuleCode < out[j].RuleCode
		}
		return out[i].ObjectID < out[j].ObjectID
	})
	return out, scanned
}

// FindingKey builds the canonical deduplication key for a rule and target object.
func FindingKey(ruleCode, objectType, objectID string) string {
	return strings.ToUpper(strings.TrimSpace(ruleCode)) + "|" +
		strings.TrimSpace(objectType) + "|" +
		strings.TrimSpace(objectID)
}

func normalizeRuleSelection(codes []string) map[string]bool {
	selected := make(map[string]bool, len(catalogueRules))
	if len(codes) == 0 {
		for _, r := range catalogueRules {
			selected[r.RuleCode] = true
		}
		return selected
	}
	for _, c := range codes {
		if r, ok := GetRule(c); ok {
			selected[r.RuleCode] = true
		}
	}
	return selected
}

func evalCashGeofenceMismatch(ds DetectorDataset, rule CatalogueRule, thresholdBps, thresholdMinor int64, now time.Time) []DetectedFinding {
	var out []DetectedFinding

	// 1. Blind count session variances / quarantined sessions.
	for _, s := range ds.CashierSessions {
		varianceAbs := abs64(s.VarianceMinor)
		diffAbs := abs64(s.ExpectedCashMinor - s.CountedCashMinor)
		if diffAbs > varianceAbs {
			varianceAbs = diffAbs
		}
		var varianceBps int64
		if s.ExpectedCashMinor > 0 {
			varianceBps = mulDivInt64(varianceAbs, 10000, s.ExpectedCashMinor)
		} else if varianceAbs > 0 {
			varianceBps = 10000
		}
		st := strings.ToUpper(strings.TrimSpace(s.Status))
		statusFlag := st == "DISCREPANT_QUARANTINE" ||
			st == "TOLERANCE_VARIANCE" ||
			st == "BLIND_RECOUNT_REQUIRED" ||
			st == "REGULATORY_AML_HOLD"
		if !statusFlag && varianceAbs <= thresholdMinor && varianceBps <= thresholdBps {
			continue
		}
		score := clampBps(5500 + varianceBps)
		severity := SeverityHigh
		if st == "DISCREPANT_QUARANTINE" || st == "REGULATORY_AML_HOLD" || varianceBps >= 1000 {
			severity = SeverityCritical
			score = clampBps(8500 + varianceBps/2)
		}
		detectedAt := s.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       severity,
			ObjectType:     "CashierBlindCountSession",
			ObjectID:       s.SessionID,
			ExposureMinor:  varianceAbs,
			ScoreBps:       score,
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":              "CASHIER_BLIND_COUNT_VARIANCE",
				"depot_id":            s.DepotID,
				"driver_id":           s.DriverID,
				"route_id":            s.RouteID,
				"session_status":      s.Status,
				"expected_cash_minor": s.ExpectedCashMinor,
				"counted_cash_minor":  s.CountedCashMinor,
				"variance_minor":      s.VarianceMinor,
				"variance_bps":        varianceBps,
			},
			DetectedAt: detectedAt,
		})
	}

	// 2. Depot vault ledger shortfalls, negative balances, or running balance drift.
	byDepot := make(map[string][]DepotVaultLedgerRecord)
	for _, v := range ds.DepotVaultEntries {
		byDepot[v.DepotID] = append(byDepot[v.DepotID], v)
	}
	for depotID, entries := range byDepot {
		sort.Slice(entries, func(i, j int) bool {
			if !entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
				return entries[i].CreatedAt.Before(entries[j].CreatedAt)
			}
			return entries[i].VaultEntryID < entries[j].VaultEntryID
		})
		var prevBal int64
		for idx, e := range entries {
			et := strings.ToUpper(strings.TrimSpace(e.EntryType))
			isShortfallType := strings.Contains(et, "SHORTFALL") ||
				strings.Contains(et, "VARIANCE") ||
				strings.Contains(et, "DISCREPANCY") ||
				strings.Contains(et, "MISSING")
			isNegativeBal := e.RunningBalanceMinor < 0
			isNegativeInflow := (et == "DRIVER_DROP_INFLOW" || et == "DEPOSIT") && e.AmountMinor < 0

			signedDelta := e.AmountMinor
			if strings.Contains(et, "OUTFLOW") || strings.Contains(et, "WITHDRAW") {
				if signedDelta > 0 {
					signedDelta = -signedDelta
				}
			}
			balanceDrift := int64(0)
			if idx > 0 {
				expectedBal := prevBal + signedDelta
				if e.RunningBalanceMinor != expectedBal {
					balanceDrift = abs64(e.RunningBalanceMinor - expectedBal)
				}
			}
			prevBal = e.RunningBalanceMinor

			if !isShortfallType && !isNegativeBal && !isNegativeInflow && balanceDrift == 0 {
				continue
			}
			exp := abs64(e.AmountMinor)
			if balanceDrift > exp {
				exp = balanceDrift
			}
			if isNegativeBal && abs64(e.RunningBalanceMinor) > exp {
				exp = abs64(e.RunningBalanceMinor)
			}
			detectedAt := e.CreatedAt
			if detectedAt.IsZero() {
				detectedAt = now
			}
			out = append(out, DetectedFinding{
				RuleCode:       rule.RuleCode,
				Category:       rule.Category,
				Severity:       SeverityCritical,
				ObjectType:     "DepotVaultLedgerEntry",
				ObjectID:       e.VaultEntryID,
				ExposureMinor:  exp,
				ScoreBps:       9000,
				PlaybookAction: rule.PlaybookAction,
				Evidence: map[string]any{
					"signal":                "DEPOT_VAULT_DISCREPANCY",
					"depot_id":              depotID,
					"session_id":            e.SessionID,
					"entry_type":            e.EntryType,
					"amount_minor":          e.AmountMinor,
					"running_balance_minor": e.RunningBalanceMinor,
					"balance_drift_minor":   balanceDrift,
				},
				DetectedAt: detectedAt,
			})
		}
	}

	// 3. Shop-closed geofence mismatch or repeated shop-closed evasion / cash shortfall.
	logsByOrder := make(map[string][]ShopClosedLogRecord)
	for _, l := range ds.ShopClosedLogs {
		logsByOrder[l.OrderID] = append(logsByOrder[l.OrderID], l)
	}
	for orderID, logs := range logsByOrder {
		var markedClosedCount int
		var geofenceMismatch bool
		var distanceMeters int64
		var shortfallMinor int64
		var latest ShopClosedLogRecord

		for _, l := range logs {
			if l.CreatedAt.After(latest.CreatedAt) || latest.OrderID == "" {
				latest = l
			}
			act := strings.ToUpper(strings.TrimSpace(l.Action))
			if act == "MARKED_CLOSED" {
				markedClosedCount++
			}
			if strings.Contains(act, "GEOFENCE") || strings.Contains(act, "SHORTFALL") {
				geofenceMismatch = true
			}
			if l.OrderH3Cell != "" && l.RetailerH3Cell != "" && l.OrderH3Cell != l.RetailerH3Cell {
				geofenceMismatch = true
			}
			pMeta := parseJSONIntAndBool(l.PayloadJSON)
			if pMeta.geofenceMismatch {
				geofenceMismatch = true
			}
			if pMeta.distanceMeters > distanceMeters {
				distanceMeters = pMeta.distanceMeters
			}
			if pMeta.distanceMeters > 100 {
				geofenceMismatch = true
			}
			if pMeta.shortfallMinor > shortfallMinor {
				shortfallMinor = pMeta.shortfallMinor
			}
		}

		if !geofenceMismatch && markedClosedCount < 2 && shortfallMinor <= thresholdMinor {
			continue
		}
		exp := shortfallMinor
		if exp == 0 {
			exp = abs64(latest.OrderTotalMinor)
		}
		score := int64(7200)
		if geofenceMismatch && shortfallMinor > 0 {
			score = 9200
		} else if markedClosedCount >= 2 {
			score = 7800
		}
		detectedAt := latest.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       SeverityHigh,
			ObjectType:     "Order",
			ObjectID:       orderID,
			ExposureMinor:  exp,
			ScoreBps:       clampBps(score),
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":              "SHOP_CLOSED_GEOFENCE_MISMATCH",
				"order_id":            orderID,
				"retailer_id":         latest.RetailerID,
				"order_h3_cell":       latest.OrderH3Cell,
				"retailer_h3_cell":    latest.RetailerH3Cell,
				"distance_meters":     distanceMeters,
				"marked_closed_count": markedClosedCount,
				"shortfall_minor":     shortfallMinor,
			},
			DetectedAt: detectedAt,
		})
	}

	return out
}

func evalBasketSpikeSharedINN(ds DetectorDataset, rule CatalogueRule, thresholdBps, thresholdMinor int64, now time.Time) []DetectedFinding {
	var out []DetectedFinding
	if thresholdBps <= 0 {
		thresholdBps = 20000
	}

	// Track retailers sharing the same NormalizedTaxId and per-retailer/per-TIN basket baselines.
	tinRetailers := make(map[string]map[string]struct{})
	tinOrders := make(map[string][]OrderRecord)
	retailerOrders := make(map[string][]OrderRecord)
	var tenantTotalMinor int64
	var tenantOrderCount int64

	for _, o := range ds.Orders {
		if strings.EqualFold(o.Status, "CANCELLED") {
			continue
		}
		retailerOrders[o.RetailerID] = append(retailerOrders[o.RetailerID], o)
		tenantTotalMinor += o.TotalMinor
		tenantOrderCount++

		tin := strings.TrimSpace(o.NormalizedTaxID)
		if tin != "" {
			if tinRetailers[tin] == nil {
				tinRetailers[tin] = make(map[string]struct{})
			}
			tinRetailers[tin][o.RetailerID] = struct{}{}
			tinOrders[tin] = append(tinOrders[tin], o)
		}
	}

	tenantAvgMinor := int64(0)
	if tenantOrderCount > 0 {
		tenantAvgMinor = tenantTotalMinor / tenantOrderCount
	}

	// 1. Shared INN across multiple distinct RetailerIds.
	for tin, rset := range tinRetailers {
		if len(rset) < 2 {
			continue
		}
		orders := tinOrders[tin]
		var totalExposure int64
		retailerIDs := make([]string, 0, len(rset))
		for rid := range rset {
			retailerIDs = append(retailerIDs, rid)
		}
		sort.Strings(retailerIDs)
		latestAt := now
		for _, o := range orders {
			totalExposure += o.TotalMinor
			if o.CreatedAt.After(latestAt) || latestAt.Equal(now) {
				if !o.CreatedAt.IsZero() {
					latestAt = o.CreatedAt
				}
			}
		}
		score := clampBps(7500 + int64(len(rset)-1)*750)
		sev := SeverityHigh
		if len(rset) >= 3 {
			sev = SeverityCritical
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "NormalizedTaxId",
			ObjectID:       tin,
			ExposureMinor:  totalExposure,
			ScoreBps:       score,
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":             "SHARED_NORMALIZED_TAX_ID",
				"normalized_tax_id":  tin,
				"retailer_ids":       retailerIDs,
				"retailer_count":     len(retailerIDs),
				"order_count":        len(orders),
				"total_basket_minor": totalExposure,
			},
			DetectedAt: latestAt,
		})
	}

	// 2. Basket spike detection per retailer (or against tenant baseline).
	for rid, orders := range retailerOrders {
		if len(orders) == 0 {
			continue
		}
		var sumMinor int64
		for _, o := range orders {
			sumMinor += o.TotalMinor
		}
		retailerAvg := sumMinor / int64(len(orders))

		for _, o := range orders {
			if o.TotalMinor <= 0 || o.TotalMinor < thresholdMinor {
				continue
			}
			// Compare against other orders for the same retailer (leave-one-out mean when >= 2 orders),
			// falling back to tenant average when single order.
			baseline := tenantAvgMinor
			if len(orders) >= 2 {
				otherAvg := (sumMinor - o.TotalMinor) / int64(len(orders)-1)
				if otherAvg > 0 {
					baseline = otherAvg
				} else if retailerAvg > 0 {
					baseline = retailerAvg
				}
			}
			if baseline <= 0 {
				continue
			}
			ratioBps := mulDivInt64(o.TotalMinor, 10000, baseline)
			if ratioBps < thresholdBps {
				continue
			}
			sharedINN := len(tinRetailers[strings.TrimSpace(o.NormalizedTaxID)]) >= 2
			score := clampBps(6500 + (ratioBps-thresholdBps)/10)
			if sharedINN {
				score = clampBps(score + 1500)
			}
			sev := SeverityHigh
			if score >= 9000 {
				sev = SeverityCritical
			}
			detectedAt := o.CreatedAt
			if detectedAt.IsZero() {
				detectedAt = now
			}
			out = append(out, DetectedFinding{
				RuleCode:       rule.RuleCode,
				Category:       rule.Category,
				Severity:       sev,
				ObjectType:     "Order",
				ObjectID:       o.OrderID,
				ExposureMinor:  o.TotalMinor - baseline,
				ScoreBps:       score,
				PlaybookAction: rule.PlaybookAction,
				Evidence: map[string]any{
					"signal":              "BASKET_SIZE_SPIKE",
					"order_id":            o.OrderID,
					"retailer_id":         rid,
					"normalized_tax_id":   o.NormalizedTaxID,
					"shared_inn":          sharedINN,
					"order_total_minor":   o.TotalMinor,
					"baseline_avg_minor":  baseline,
					"spike_ratio_bps":     ratioBps,
					"threshold_ratio_bps": thresholdBps,
				},
				DetectedAt: detectedAt,
			})
		}
	}

	return out
}

func evalStockCountVariance(ds DetectorDataset, rule CatalogueRule, thresholdBps, thresholdMinor int64, now time.Time) []DetectedFinding {
	var out []DetectedFinding

	costByProduct := buildCostMap(ds.CostLayers, ds.ProductPrices)

	// 1. Open STOCK_GL ReconciliationBreaks.
	for _, b := range ds.ReconBreaks {
		if !strings.EqualFold(b.Dimension, "STOCK_GL") {
			continue
		}
		if strings.EqualFold(b.Status, "RESOLVED") {
			continue
		}
		deltaAbs := abs64(b.DeltaMinor)
		if deltaAbs == 0 {
			deltaAbs = abs64(b.ExpectedMinor - b.ActualMinor)
		}
		if deltaAbs == 0 {
			continue
		}
		var ratioBps int64
		if b.ExpectedMinor > 0 {
			ratioBps = mulDivInt64(deltaAbs, 10000, b.ExpectedMinor)
		} else {
			ratioBps = 10000
		}
		sev := strings.ToUpper(strings.TrimSpace(b.Severity))
		if sev == "" {
			sev = SeverityHigh
		}
		score := clampBps(7000 + ratioBps/4)
		detectedAt := b.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "ReconciliationBreak",
			ObjectID:       b.BreakIdSafe(),
			ExposureMinor:  deltaAbs,
			ScoreBps:       score,
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":         "STOCK_GL_RECON_BREAK",
				"run_id":         b.RunID,
				"dimension":      b.Dimension,
				"object_type":    b.ObjectType,
				"object_id":      b.ObjectID,
				"expected_minor": b.ExpectedMinor,
				"actual_minor":   b.ActualMinor,
				"delta_minor":    b.DeltaMinor,
				"variance_bps":   ratioBps,
			},
			DetectedAt: detectedAt,
		})
	}

	// 2. CycleCounts with significant quantity variance.
	for _, c := range ds.CycleCounts {
		varianceQty := abs64(c.VarianceQty)
		if varianceQty == 0 && c.HasCounted {
			varianceQty = abs64(c.ExpectedQty - c.CountedQty)
		}
		if varianceQty == 0 {
			continue
		}
		var varianceBps int64
		if c.ExpectedQty > 0 {
			varianceBps = mulDivInt64(varianceQty, 10000, c.ExpectedQty)
		} else {
			varianceBps = 10000
		}
		unitCost := costByProduct[c.ProductID]
		expMinor := varianceQty * unitCost
		if varianceBps < thresholdBps && expMinor < thresholdMinor {
			continue
		}
		sev := SeverityMedium
		if varianceBps >= 1500 || expMinor >= thresholdMinor*5 {
			sev = SeverityHigh
		}
		score := clampBps(6000 + varianceBps/2)
		detectedAt := c.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "CycleCount",
			ObjectID:       c.CountID,
			ExposureMinor:  expMinor,
			ScoreBps:       score,
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":          "CYCLE_COUNT_VARIANCE",
				"warehouse_id":    c.WarehouseID,
				"location_id":     c.LocationID,
				"product_id":      c.ProductID,
				"expected_qty":    c.ExpectedQty,
				"counted_qty":     c.CountedQty,
				"variance_qty":    c.VarianceQty,
				"variance_bps":    varianceBps,
				"unit_cost_minor": unitCost,
			},
			DetectedAt: detectedAt,
		})
	}

	// 3. Large negative or unapproved InventoryAdjustments.
	for _, a := range ds.InventoryAdjustments {
		if strings.EqualFold(a.Status, "REJECTED") {
			continue
		}
		qtyAbs := abs64(a.DeltaQty)
		if qtyAbs == 0 {
			continue
		}
		unitCost := costByProduct[a.ProductID]
		expMinor := qtyAbs * unitCost
		pendingNegative := a.DeltaQty < 0 && !strings.EqualFold(a.Status, "APPROVED")
		if !pendingNegative && expMinor < thresholdMinor {
			continue
		}
		sev := SeverityMedium
		if expMinor >= thresholdMinor {
			sev = SeverityHigh
		}
		score := clampBps(6200 + qtyAbs*100)
		detectedAt := a.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "InventoryAdjustment",
			ObjectID:       a.AdjustmentID,
			ExposureMinor:  expMinor,
			ScoreBps:       score,
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":          "INVENTORY_ADJUSTMENT_VARIANCE",
				"warehouse_id":    a.WarehouseID,
				"product_id":      a.ProductID,
				"delta_qty":       a.DeltaQty,
				"reason_code":     a.ReasonCode,
				"status":          a.Status,
				"unit_cost_minor": unitCost,
			},
			DetectedAt: detectedAt,
		})
	}

	return out
}

func (r ReconBreakRecord) BreakIdSafe() string {
	return strings.TrimSpace(r.BreakID)
}

func evalPricingDiscountAbuse(ds DetectorDataset, rule CatalogueRule, thresholdBps, _ int64, now time.Time) []DetectedFinding {
	var out []DetectedFinding
	if thresholdBps <= 0 {
		thresholdBps = 3000
	}

	costByKey := buildCostMap(ds.CostLayers, nil)
	listPriceByKey := make(map[string]int64, len(ds.ProductPrices)*2)
	for _, p := range ds.ProductPrices {
		if p.UnitPriceMinor <= 0 {
			continue
		}
		if p.ProductID != "" {
			listPriceByKey[p.ProductID] = p.UnitPriceMinor
		}
		if p.SKU != "" {
			listPriceByKey[p.SKU] = p.UnitPriceMinor
		}
	}

	for _, o := range ds.Orders {
		if strings.EqualFold(o.Status, "CANCELLED") {
			continue
		}
		lines := parseOrderLines(o.LineItemsJSON)
		var maxExposure int64
		var maxScore int64
		var belowCostLines []map[string]any
		var extremeDiscountLines []map[string]any

		for _, line := range lines {
			if line.Quantity <= 0 || line.UnitPriceMinor < 0 {
				continue
			}
			key := nonEmpty(line.ProductID, line.SKU)
			unitCost := costByKey[key]
			if unitCost == 0 && line.SKU != "" {
				unitCost = costByKey[line.SKU]
			}
			listPrice := line.ListPriceMinor
			if listPrice == 0 {
				listPrice = listPriceByKey[key]
			}
			if listPrice == 0 && line.SKU != "" {
				listPrice = listPriceByKey[line.SKU]
			}

			// 1. Below-cost check against InventoryCostLayers.
			if unitCost > 0 && line.UnitPriceMinor < unitCost {
				lossMinor := (unitCost - line.UnitPriceMinor) * line.Quantity
				lossBps := mulDivInt64(unitCost-line.UnitPriceMinor, 10000, unitCost)
				score := clampBps(8000 + lossBps/5)
				if lossMinor > maxExposure {
					maxExposure = lossMinor
				}
				if score > maxScore {
					maxScore = score
				}
				belowCostLines = append(belowCostLines, map[string]any{
					"sku":              key,
					"quantity":         line.Quantity,
					"unit_price_minor": line.UnitPriceMinor,
					"unit_cost_minor":  unitCost,
					"loss_minor":       lossMinor,
					"loss_bps":         lossBps,
				})
			}

			// 2. Extreme discount check against catalog/list price or explicit discount_bps.
			discountBps := line.DiscountBps
			var discountExposure int64
			if listPrice > 0 && line.UnitPriceMinor < listPrice {
				calcBps := mulDivInt64(listPrice-line.UnitPriceMinor, 10000, listPrice)
				if calcBps > discountBps {
					discountBps = calcBps
				}
				discountExposure = (listPrice - line.UnitPriceMinor) * line.Quantity
			}
			if discountBps >= thresholdBps {
				if discountExposure == 0 && line.UnitPriceMinor > 0 {
					discountExposure = mulDivInt64(line.UnitPriceMinor*line.Quantity, discountBps, 10000)
				}
				score := clampBps(6500 + (discountBps-thresholdBps)/3)
				if discountExposure > maxExposure {
					maxExposure = discountExposure
				}
				if score > maxScore {
					maxScore = score
				}
				extremeDiscountLines = append(extremeDiscountLines, map[string]any{
					"sku":              key,
					"quantity":         line.Quantity,
					"unit_price_minor": line.UnitPriceMinor,
					"list_price_minor": listPrice,
					"discount_bps":     discountBps,
					"discount_minor":   discountExposure,
				})
			}
		}

		// 3. Order-level markdown vs OriginalTotalMinor.
		var orderDiscountBps int64
		var orderDiscountMinor int64
		if o.OriginalTotalMinor > 0 && o.TotalMinor < o.OriginalTotalMinor {
			orderDiscountMinor = o.OriginalTotalMinor - o.TotalMinor
			orderDiscountBps = mulDivInt64(orderDiscountMinor, 10000, o.OriginalTotalMinor)
			if orderDiscountBps >= thresholdBps {
				if orderDiscountMinor > maxExposure {
					maxExposure = orderDiscountMinor
				}
				score := clampBps(6800 + (orderDiscountBps-thresholdBps)/4)
				if score > maxScore {
					maxScore = score
				}
			}
		}

		if len(belowCostLines) == 0 && len(extremeDiscountLines) == 0 && orderDiscountBps < thresholdBps {
			continue
		}
		sev := SeverityHigh
		if len(belowCostLines) > 0 {
			sev = SeverityCritical
		}
		detectedAt := o.CreatedAt
		if detectedAt.IsZero() {
			detectedAt = now
		}
		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "Order",
			ObjectID:       o.OrderID,
			ExposureMinor:  maxExposure,
			ScoreBps:       clampBps(maxScore),
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":                 "PRICING_OR_BELOW_COST_ABUSE",
				"order_id":               o.OrderID,
				"retailer_id":            o.RetailerID,
				"order_total_minor":      o.TotalMinor,
				"original_total_minor":   o.OriginalTotalMinor,
				"order_discount_bps":     orderDiscountBps,
				"order_discount_minor":   orderDiscountMinor,
				"below_cost_lines":       belowCostLines,
				"extreme_discount_lines": extremeDiscountLines,
			},
			DetectedAt: detectedAt,
		})
	}

	return out
}

func evalManualJournalAfterHoursOrSplit(ds DetectorDataset, rule CatalogueRule, _ int64, thresholdMinor int64, now time.Time) []DetectedFinding {
	var out []DetectedFinding
	if thresholdMinor <= 0 {
		thresholdMinor = 10_000_000
	}

	type dayUserKey struct {
		day      string
		postedBy string
	}
	dayGroups := make(map[dayUserKey][]ManualJournalRecord)
	manualEntries := make([]ManualJournalRecord, 0, len(ds.ManualJournals))

	for _, j := range ds.ManualJournals {
		st := strings.ToUpper(strings.TrimSpace(j.SourceType))
		if st != "MANUAL" && st != "MANUAL_JOURNAL" {
			continue
		}
		manualEntries = append(manualEntries, j)
		eventAt := j.OccurredAt
		if eventAt.IsZero() {
			eventAt = j.PostedAt
		}
		if !eventAt.IsZero() {
			day := eventAt.In(uztZone).Format("2006-01-02")
			key := dayUserKey{day: day, postedBy: strings.ToLower(strings.TrimSpace(j.PostedBy))}
			dayGroups[key] = append(dayGroups[key], j)
		}
	}

	// Identify entries belonging to same-day split structuring batches.
	splitBatchByEntry := make(map[string]struct {
		day        string
		count      int
		batchTotal int64
	})
	for k, grp := range dayGroups {
		if len(grp) < 2 {
			continue
		}
		var subThresholdCount int
		var batchTotal int64
		for _, j := range grp {
			amt := abs64(j.DebitTotalMinor)
			batchTotal += amt
			if amt > 0 && amt < thresholdMinor {
				subThresholdCount++
			}
		}
		if subThresholdCount >= 2 && batchTotal >= thresholdMinor {
			for _, j := range grp {
				if abs64(j.DebitTotalMinor) > 0 && abs64(j.DebitTotalMinor) < thresholdMinor {
					splitBatchByEntry[j.EntryID] = struct {
						day        string
						count      int
						batchTotal int64
					}{day: k.day, count: len(grp), batchTotal: batchTotal}
				}
			}
		}
	}

	for _, j := range manualEntries {
		eventAt := j.OccurredAt
		if eventAt.IsZero() {
			eventAt = j.PostedAt
		}
		if eventAt.IsZero() {
			eventAt = now
		}
		uztTime := eventAt.In(uztZone)
		hour := uztTime.Hour()
		afterHours := hour < 8 || hour >= 20
		splitInfo, isSplit := splitBatchByEntry[j.EntryID]
		if !afterHours && !isSplit {
			continue
		}

		reasons := make([]string, 0, 2)
		score := int64(6800)
		exposure := abs64(j.DebitTotalMinor)
		if afterHours {
			reasons = append(reasons, "AFTER_HOURS_UZT")
			score += 1200
		}
		if isSplit {
			reasons = append(reasons, "SAME_DAY_THRESHOLD_SPLIT")
			score += 1500
			if splitInfo.batchTotal > exposure {
				exposure = splitInfo.batchTotal
			}
		}
		sev := SeverityHigh
		if afterHours && isSplit {
			sev = SeverityCritical
		}

		out = append(out, DetectedFinding{
			RuleCode:       rule.RuleCode,
			Category:       rule.Category,
			Severity:       sev,
			ObjectType:     "JournalEntry",
			ObjectID:       j.EntryID,
			ExposureMinor:  exposure,
			ScoreBps:       clampBps(score),
			PlaybookAction: rule.PlaybookAction,
			Evidence: map[string]any{
				"signal":                strings.Join(reasons, "+"),
				"reasons":               reasons,
				"entry_id":              j.EntryID,
				"entity_id":             j.EntityID,
				"period_id":             j.PeriodID,
				"source_type":           j.SourceType,
				"posted_by":             j.PostedBy,
				"debit_total_minor":     j.DebitTotalMinor,
				"uzt_timestamp":         uztTime.Format(time.RFC3339),
				"uzt_hour":              hour,
				"after_hours_uzt":       afterHours,
				"split_day":             splitInfo.day,
				"split_entry_count":     splitInfo.count,
				"split_batch_total":     splitInfo.batchTotal,
				"split_threshold_minor": thresholdMinor,
			},
			DetectedAt: eventAt.UTC(),
		})
	}

	return out
}

func buildCostMap(layers []CostLayerSummary, fallbackPrices []ProductPriceRecord) map[string]int64 {
	m := make(map[string]int64, len(layers)+len(fallbackPrices))
	// layers arrive ordered newest-first (ReceivedAt DESC); keep the newest layer per SKU.
	for _, c := range layers {
		if c.UnitCostMinor > 0 && c.ProductID != "" && m[c.ProductID] == 0 {
			m[c.ProductID] = c.UnitCostMinor
		}
	}
	for _, p := range fallbackPrices {
		if p.UnitPriceMinor > 0 && p.ProductID != "" && m[p.ProductID] == 0 {
			m[p.ProductID] = p.UnitPriceMinor
		}
	}
	return m
}

func parseOrderLines(raw string) []OrderLineRecord {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	var lines []OrderLineRecord
	if err := dec.Decode(&lines); err != nil {
		return nil
	}
	return lines
}

type parsedShopClosedMeta struct {
	geofenceMismatch bool
	distanceMeters   int64
	shortfallMinor   int64
}

func parseJSONIntAndBool(raw string) parsedShopClosedMeta {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return parsedShopClosedMeta{}
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return parsedShopClosedMeta{}
	}
	var out parsedShopClosedMeta
	if b, ok := m["geofence_mismatch"].(bool); ok {
		out.geofenceMismatch = b
	}
	if d := parseAnyInt64(m["distance_m"]); d > out.distanceMeters {
		out.distanceMeters = d
	}
	if d := parseAnyInt64(m["distance_meters"]); d > out.distanceMeters {
		out.distanceMeters = d
	}
	if d := parseAnyInt64(m["geofence_distance_m"]); d > out.distanceMeters {
		out.distanceMeters = d
	}
	if s := parseAnyInt64(m["shortfall_minor"]); s > out.shortfallMinor {
		out.shortfallMinor = s
	}
	if s := parseAnyInt64(m["variance_minor"]); abs64(s) > out.shortfallMinor {
		out.shortfallMinor = abs64(s)
	}
	return out
}

func parseAnyInt64(v any) int64 {
	switch tv := v.(type) {
	case int64:
		return tv
	case int:
		return int64(tv)
	case json.Number:
		if n, err := tv.Int64(); err == nil {
			return n
		}
		s := tv.String()
		if dot := strings.IndexByte(s, '.'); dot > 0 {
			if n, err := strconv.ParseInt(s[:dot], 10, 64); err == nil {
				return n
			}
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func fmtEvidenceJSON(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf(`{"marshal_error":%q}`, err.Error())
	}
	return string(b)
}
