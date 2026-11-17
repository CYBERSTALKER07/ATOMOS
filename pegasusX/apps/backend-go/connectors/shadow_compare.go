package connectors

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ComputeShadowCompare deterministically diffs external CanonicalRecords against
// Spanner Products, SupplierInventoryV2, Orders, and ArInvoices.
func ComputeShadowCompare(
	tenantID, provider, accountLabel, runID string,
	syncedAt time.Time,
	external []CanonicalRecord,
	internal InternalSnapshot,
) (ShadowCompareSummary, []ShadowCompareMismatch, []ConnectorEntityMap) {
	summary := ShadowCompareSummary{
		Provider:     normalizeProvider(provider),
		AccountLabel: normalizeAccountLabel(accountLabel),
		RunID:        runID,
		Currency:     "UZS",
	}

	// 1. Index existing ConnectorEntityMaps.
	extToInt := make(map[string]string, len(internal.EntityMaps))
	for _, em := range internal.EntityMaps {
		extToInt[em.ObjectType+"|"+em.ExternalID] = em.InternalID
	}

	// 2. Index internal Spanner objects.
	prodByID := make(map[string]InternalProduct, len(internal.Products))
	prodByBarcode := make(map[string]InternalProduct, len(internal.Products))
	for _, p := range internal.Products {
		prodByID[p.ProductID] = p
		if bc := strings.TrimSpace(p.Barcode); bc != "" {
			prodByBarcode[bc] = p
		}
		if strings.TrimSpace(p.Currency) != "" {
			summary.Currency = strings.TrimSpace(p.Currency)
		}
	}
	summary.InternalProducts = int64(len(internal.Products))

	stockTotalByProduct := make(map[string]int64, len(internal.Stocks))
	hasInventoryRowsByProduct := make(map[string]bool, len(internal.Stocks))
	stockByWhProd := make(map[string]int64, len(internal.Stocks))
	for _, st := range internal.Stocks {
		stockTotalByProduct[st.ProductID] += st.QuantityOnHand
		hasInventoryRowsByProduct[st.ProductID] = true
		if st.WarehouseID != "" {
			stockByWhProd[st.WarehouseID+"|"+st.ProductID] = st.QuantityOnHand
		}
	}
	summary.InternalStocks = int64(len(internal.Stocks))

	orderByID := make(map[string]InternalOrder, len(internal.Orders))
	for _, ord := range internal.Orders {
		orderByID[ord.OrderID] = ord
	}
	summary.InternalOrders = int64(len(internal.Orders))

	invByID := make(map[string]InternalInvoice, len(internal.Invoices))
	invByOrderID := make(map[string]InternalInvoice, len(internal.Invoices))
	invTotalByRetailer := make(map[string]int64, len(internal.Invoices))
	invCurrencyByRetailer := make(map[string]string, len(internal.Invoices))
	for _, inv := range internal.Invoices {
		invByID[inv.InvoiceID] = inv
		if inv.OrderID != "" {
			invByOrderID[inv.OrderID] = inv
		}
		if inv.RetailerID != "" {
			invTotalByRetailer[inv.RetailerID] += inv.BalanceMinor
			if inv.Currency != "" {
				invCurrencyByRetailer[inv.RetailerID] = inv.Currency
			}
		}
	}
	summary.InternalInvoices = int64(len(internal.Invoices))

	// Aggregate external stocks per SKU/product so multi-row stocks roll up deterministically.
	extStockByKey := make(map[string]CanonicalRecord)
	extProducts := make([]CanonicalRecord, 0)
	extOrders := make([]CanonicalRecord, 0)
	extClients := make([]CanonicalRecord, 0)

	for _, rec := range external {
		switch rec.ObjectType {
		case ObjectTypeProduct:
			summary.ExternalProducts++
			extProducts = append(extProducts, rec)
		case ObjectTypeInventory:
			summary.ExternalStocks++
			k := rec.SKU
			if k == "" {
				k = rec.ExternalID
			}
			prev, exists := extStockByKey[k]
			if !exists {
				extStockByKey[k] = rec
			} else {
				prev.QuantityOnHand += rec.QuantityOnHand
				extStockByKey[k] = prev
			}
		case ObjectTypeOrder:
			summary.ExternalOrders++
			extOrders = append(extOrders, rec)
		case ObjectTypeClient, ObjectTypeInvoice:
			summary.ExternalClients++
			extClients = append(extClients, rec)
		}
	}

	mismatches := make([]ShadowCompareMismatch, 0)
	mapsToSave := make([]ConnectorEntityMap, 0)
	matchedProducts := make(map[string]bool)
	matchedOrders := make(map[string]bool)
	matchedInvoices := make(map[string]bool)
	matchedStocks := make(map[string]bool)

	recordMap := func(objType, extID, intID, hash string) {
		mapsToSave = append(mapsToSave, ConnectorEntityMap{
			TenantID:     tenantID,
			Provider:     summary.Provider,
			ObjectType:   objType,
			ExternalID:   extID,
			InternalID:   intID,
			ContentHash:  hash,
			LastRunID:    runID,
			LastSyncedAt: syncedAt,
		})
	}

	resolveProduct := func(extID, sku, barcode string) (InternalProduct, bool) {
		for _, candidate := range []string{
			extToInt[ObjectTypeProduct+"|"+extID],
			extToInt[ObjectTypeProduct+"|"+sku],
			extID,
			sku,
		} {
			if candidate != "" {
				if p, ok := prodByID[candidate]; ok {
					return p, true
				}
			}
		}
		if barcode != "" {
			if p, ok := prodByBarcode[barcode]; ok {
				return p, true
			}
		}
		return InternalProduct{}, false
	}

	// 3. Compare external products against internal Products (+ embedded stock if no separate stock stream).
	for _, ext := range extProducts {
		p, found := resolveProduct(ext.ExternalID, ext.SKU, ext.Barcode)
		if !found {
			summary.MissingInternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType: MismatchMissingInternal,
				ObjectType:   ObjectTypeProduct,
				ExternalID:   ext.ExternalID,
				Name:         ext.Name,
				Currency:     firstNonEmpty(ext.Currency, summary.Currency),
				Detail:       fmt.Sprintf("external product %q (%s) missing in Spanner Products", ext.ExternalID, ext.Name),
			})
			continue
		}
		matchedProducts[p.ProductID] = true
		extToInt[ObjectTypeProduct+"|"+ext.ExternalID] = p.ProductID
		if ext.SKU != "" {
			extToInt[ObjectTypeProduct+"|"+ext.SKU] = p.ProductID
		}
		summary.MatchedCount++
		recordMap(ObjectTypeProduct, ext.ExternalID, p.ProductID, ext.ContentHash)

		if _, hasSepStock := extStockByKey[ext.ExternalID]; !hasSepStock {
			if _, hasSKUStock := extStockByKey[ext.SKU]; !hasSKUStock && ext.QuantityOnHand > 0 {
				intQty := p.StockQuantity
				if hasInventoryRowsByProduct[p.ProductID] {
					intQty = stockTotalByProduct[p.ProductID]
				}
				if ext.QuantityOnHand != intQty {
					delta := ext.QuantityOnHand - intQty
					summary.StockDeltaCount++
					summary.NetStockDelta += delta
					mismatches = append(mismatches, ShadowCompareMismatch{
						MismatchType:  MismatchStockDelta,
						ObjectType:    ObjectTypeInventory,
						ExternalID:    ext.ExternalID,
						InternalID:    p.ProductID,
						Name:          p.Name,
						ExternalStock: ext.QuantityOnHand,
						InternalStock: intQty,
						StockDelta:    delta,
						Detail:        fmt.Sprintf("stock mismatch for product %s: external=%d internal=%d delta=%d", p.ProductID, ext.QuantityOnHand, intQty, delta),
					})
				}
				matchedStocks[p.ProductID] = true
			}
		}
	}

	for _, p := range internal.Products {
		if !matchedProducts[p.ProductID] {
			summary.MissingExternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType: MismatchMissingExternal,
				ObjectType:   ObjectTypeProduct,
				InternalID:   p.ProductID,
				Name:         p.Name,
				Currency:     firstNonEmpty(p.Currency, summary.Currency),
				Detail:       fmt.Sprintf("internal product %q (%s) missing in %s pull", p.ProductID, p.Name, summary.Provider),
			})
		}
	}

	// 4. Compare external stocks against SupplierInventoryV2 (or Products.StockQuantity fallback).
	extStockKeys := make([]string, 0, len(extStockByKey))
	for k := range extStockByKey {
		extStockKeys = append(extStockKeys, k)
	}
	sort.Strings(extStockKeys)
	for _, k := range extStockKeys {
		ext := extStockByKey[k]
		p, found := resolveProduct(ext.ExternalID, ext.SKU, ext.Barcode)
		if !found {
			if _, alreadyMissing := extToInt[ObjectTypeProduct+"|"+ext.ExternalID]; !alreadyMissing {
				summary.MissingInternal++
				mismatches = append(mismatches, ShadowCompareMismatch{
					MismatchType:  MismatchMissingInternal,
					ObjectType:    ObjectTypeInventory,
					ExternalID:    ext.ExternalID,
					WarehouseID:   ext.WarehouseID,
					ExternalStock: ext.QuantityOnHand,
					Detail:        fmt.Sprintf("external stock %q missing matching Spanner product", ext.ExternalID),
				})
			}
			continue
		}
		matchedStocks[p.ProductID] = true
		recordMap(ObjectTypeInventory, ext.ExternalID, p.ProductID, ext.ContentHash)

		intQty := p.StockQuantity
		if hasInventoryRowsByProduct[p.ProductID] {
			if ext.WarehouseID != "" {
				if whQty, ok := stockByWhProd[ext.WarehouseID+"|"+p.ProductID]; ok {
					intQty = whQty
				} else {
					intQty = stockTotalByProduct[p.ProductID]
				}
			} else {
				intQty = stockTotalByProduct[p.ProductID]
			}
		}
		if ext.QuantityOnHand != intQty {
			delta := ext.QuantityOnHand - intQty
			summary.StockDeltaCount++
			summary.NetStockDelta += delta
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:  MismatchStockDelta,
				ObjectType:    ObjectTypeInventory,
				ExternalID:    ext.ExternalID,
				InternalID:    p.ProductID,
				WarehouseID:   ext.WarehouseID,
				Name:          p.Name,
				ExternalStock: ext.QuantityOnHand,
				InternalStock: intQty,
				StockDelta:    delta,
				Detail:        fmt.Sprintf("stock mismatch for %s: external=%d internal=%d delta=%d", p.ProductID, ext.QuantityOnHand, intQty, delta),
			})
		} else {
			summary.MatchedCount++
		}
	}

	// Report internal SupplierInventoryV2 rows that had no external stock or product match when stock stream existed.
	if len(extStockByKey) > 0 {
		for pid, intQty := range stockTotalByProduct {
			if !matchedStocks[pid] {
				summary.MissingExternal++
				mismatches = append(mismatches, ShadowCompareMismatch{
					MismatchType:  MismatchMissingExternal,
					ObjectType:    ObjectTypeInventory,
					InternalID:    pid,
					InternalStock: intQty,
					Detail:        fmt.Sprintf("internal inventory for %s (qty=%d) missing in %s stocks pull", pid, intQty, summary.Provider),
				})
			}
		}
	}

	// 5. Compare external orders against Spanner Orders.
	for _, ext := range extOrders {
		var ord InternalOrder
		found := false
		for _, cand := range []string{
			extToInt[ObjectTypeOrder+"|"+ext.ExternalID],
			ext.OrderID,
			ext.ExternalID,
		} {
			if cand != "" {
				if o, ok := orderByID[cand]; ok {
					ord = o
					found = true
					break
				}
			}
		}
		if !found {
			summary.MissingInternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:        MismatchMissingInternal,
				ObjectType:          ObjectTypeOrder,
				ExternalID:          ext.ExternalID,
				ExternalAmountMinor: ext.TotalMinor,
				Currency:            firstNonEmpty(ext.Currency, summary.Currency),
				Detail:              fmt.Sprintf("external order %q missing in Spanner Orders", ext.ExternalID),
			})
			continue
		}
		matchedOrders[ord.OrderID] = true
		summary.MatchedCount++
		recordMap(ObjectTypeOrder, ext.ExternalID, ord.OrderID, ext.ContentHash)

		if ext.TotalMinor != ord.TotalMinor {
			delta := ext.TotalMinor - ord.TotalMinor
			summary.BalanceDeltaCount++
			summary.NetBalanceDeltaMinor += delta
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:        MismatchBalanceDeltaMinor,
				ObjectType:          ObjectTypeOrder,
				ExternalID:          ext.ExternalID,
				InternalID:          ord.OrderID,
				ExternalAmountMinor: ext.TotalMinor,
				InternalAmountMinor: ord.TotalMinor,
				BalanceDeltaMinor:   delta,
				Currency:            firstNonEmpty(ord.Currency, ext.Currency, summary.Currency),
				Detail:              fmt.Sprintf("order total delta for %s: external=%d internal=%d delta_minor=%d", ord.OrderID, ext.TotalMinor, ord.TotalMinor, delta),
			})
		}
	}

	for _, ord := range internal.Orders {
		if !matchedOrders[ord.OrderID] {
			summary.MissingExternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:        MismatchMissingExternal,
				ObjectType:          ObjectTypeOrder,
				InternalID:          ord.OrderID,
				InternalAmountMinor: ord.TotalMinor,
				Currency:            firstNonEmpty(ord.Currency, summary.Currency),
				Detail:              fmt.Sprintf("internal order %q missing in %s pull", ord.OrderID, summary.Provider),
			})
		}
	}

	// 6. Compare external client/invoice balances against Spanner ArInvoices.
	for _, ext := range extClients {
		var matchedInv *InternalInvoice
		if invID := extToInt[ObjectTypeInvoice+"|"+ext.ExternalID]; invID != "" {
			if inv, ok := invByID[invID]; ok {
				matchedInv = &inv
			}
		}
		if matchedInv == nil && ext.OrderID != "" {
			if inv, ok := invByOrderID[ext.OrderID]; ok {
				matchedInv = &inv
			} else if inv, ok := invByID[ext.OrderID]; ok {
				matchedInv = &inv
			}
		}
		if matchedInv != nil {
			matchedInvoices[matchedInv.InvoiceID] = true
			summary.MatchedCount++
			recordMap(ObjectTypeInvoice, ext.ExternalID, matchedInv.InvoiceID, ext.ContentHash)
			if ext.BalanceMinor != matchedInv.BalanceMinor {
				delta := ext.BalanceMinor - matchedInv.BalanceMinor
				summary.BalanceDeltaCount++
				summary.NetBalanceDeltaMinor += delta
				mismatches = append(mismatches, ShadowCompareMismatch{
					MismatchType:        MismatchBalanceDeltaMinor,
					ObjectType:          ObjectTypeInvoice,
					ExternalID:          ext.ExternalID,
					InternalID:          matchedInv.InvoiceID,
					Name:                ext.Name,
					ExternalAmountMinor: ext.BalanceMinor,
					InternalAmountMinor: matchedInv.BalanceMinor,
					BalanceDeltaMinor:   delta,
					Currency:            firstNonEmpty(matchedInv.Currency, ext.Currency, summary.Currency),
					Detail:              fmt.Sprintf("AR invoice balance delta for %s: external=%d internal=%d delta_minor=%d", matchedInv.InvoiceID, ext.BalanceMinor, matchedInv.BalanceMinor, delta),
				})
			}
			continue
		}

		// Match by RetailerID aggregate balance across open ArInvoices.
		retailerKey := firstNonEmpty(extToInt[ObjectTypeClient+"|"+ext.ExternalID], ext.RetailerID, ext.ExternalID)
		if intBal, hasRetailer := invTotalByRetailer[retailerKey]; hasRetailer {
			for _, inv := range internal.Invoices {
				if inv.RetailerID == retailerKey {
					matchedInvoices[inv.InvoiceID] = true
				}
			}
			summary.MatchedCount++
			recordMap(ObjectTypeClient, ext.ExternalID, retailerKey, ext.ContentHash)
			if ext.BalanceMinor != intBal {
				delta := ext.BalanceMinor - intBal
				summary.BalanceDeltaCount++
				summary.NetBalanceDeltaMinor += delta
				mismatches = append(mismatches, ShadowCompareMismatch{
					MismatchType:        MismatchBalanceDeltaMinor,
					ObjectType:          ObjectTypeInvoice,
					ExternalID:          ext.ExternalID,
					InternalID:          retailerKey,
					Name:                ext.Name,
					ExternalAmountMinor: ext.BalanceMinor,
					InternalAmountMinor: intBal,
					BalanceDeltaMinor:   delta,
					Currency:            firstNonEmpty(invCurrencyByRetailer[retailerKey], ext.Currency, summary.Currency),
					Detail:              fmt.Sprintf("client AR balance delta for %s: external=%d internal=%d delta_minor=%d", retailerKey, ext.BalanceMinor, intBal, delta),
				})
			}
			continue
		}

		if ext.BalanceMinor > 0 {
			summary.MissingInternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:        MismatchMissingInternal,
				ObjectType:          ObjectTypeInvoice,
				ExternalID:          ext.ExternalID,
				Name:                ext.Name,
				ExternalAmountMinor: ext.BalanceMinor,
				Currency:            firstNonEmpty(ext.Currency, summary.Currency),
				Detail:              fmt.Sprintf("external client balance %q (%d minor) has no matching Spanner AR invoice", ext.ExternalID, ext.BalanceMinor),
			})
		}
	}

	for _, inv := range internal.Invoices {
		if !matchedInvoices[inv.InvoiceID] && inv.BalanceMinor > 0 {
			summary.MissingExternal++
			mismatches = append(mismatches, ShadowCompareMismatch{
				MismatchType:        MismatchMissingExternal,
				ObjectType:          ObjectTypeInvoice,
				InternalID:          inv.InvoiceID,
				InternalAmountMinor: inv.BalanceMinor,
				Currency:            firstNonEmpty(inv.Currency, summary.Currency),
				Detail:              fmt.Sprintf("internal AR invoice %q (balance=%d minor) missing in %s pull", inv.InvoiceID, inv.BalanceMinor, summary.Provider),
			})
		}
	}

	// Deterministic sort of mismatches and entity maps.
	sort.Slice(mismatches, func(i, j int) bool {
		if mismatches[i].ObjectType != mismatches[j].ObjectType {
			return mismatches[i].ObjectType < mismatches[j].ObjectType
		}
		if mismatches[i].MismatchType != mismatches[j].MismatchType {
			return mismatches[i].MismatchType < mismatches[j].MismatchType
		}
		if mismatches[i].ExternalID != mismatches[j].ExternalID {
			return mismatches[i].ExternalID < mismatches[j].ExternalID
		}
		if mismatches[i].InternalID != mismatches[j].InternalID {
			return mismatches[i].InternalID < mismatches[j].InternalID
		}
		return mismatches[i].WarehouseID < mismatches[j].WarehouseID
	})

	sort.Slice(mapsToSave, func(i, j int) bool {
		if mapsToSave[i].ObjectType != mapsToSave[j].ObjectType {
			return mapsToSave[i].ObjectType < mapsToSave[j].ObjectType
		}
		return mapsToSave[i].ExternalID < mapsToSave[j].ExternalID
	})

	summary.TotalMismatches = int64(len(mismatches))
	return summary, mismatches, mapsToSave
}
