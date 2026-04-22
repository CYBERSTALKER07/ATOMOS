package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// correctionStore tracks per-retailer, per-warehouse, per-SKU adjustment factors
// learned from retailer RLHF feedback. Thread-safe in-memory cache backed by
// Spanner persistence — corrections survive AI Worker restarts.
type correctionStore struct {
	mu      sync.RWMutex
	weights map[string]map[string]correctionEntry // "retailerID:warehouseID" → skuID → entry
	client  *spanner.Client
}

type correctionEntry struct {
	Factor            float64
	TriggerDateShiftH float64
	CorrectionCount   int64
}

func newCorrectionStore(client *spanner.Client) *correctionStore {
	return &correctionStore{
		weights: make(map[string]map[string]correctionEntry),
		client:  client,
	}
}

func correctionKey(retailerID, warehouseID string) string {
	return retailerID + ":" + warehouseID
}

// loadFromSpanner hydrates the in-memory cache from the CorrectionWeights table on boot.
func (cs *correctionStore) loadFromSpanner(ctx context.Context) error {
	if cs.client == nil {
		logger.Warn("no Spanner client — corrections will be volatile (in-memory only)")
		return nil
	}

	stmt := spanner.Statement{SQL: `SELECT RetailerId, WarehouseId, SkuId, Factor, TriggerDateShiftH, CorrectionCount FROM CorrectionWeights`}
	iter := cs.client.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	count := 0
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("load corrections: %w", err)
		}

		var retailerID, warehouseID, skuID string
		var factor, triggerShift float64
		var corrCount int64
		if err := row.Columns(&retailerID, &warehouseID, &skuID, &factor, &triggerShift, &corrCount); err != nil {
			return fmt.Errorf("parse correction row: %w", err)
		}

		key := correctionKey(retailerID, warehouseID)
		if _, ok := cs.weights[key]; !ok {
			cs.weights[key] = make(map[string]correctionEntry)
		}
		cs.weights[key][skuID] = correctionEntry{
			Factor:            factor,
			TriggerDateShiftH: triggerShift,
			CorrectionCount:   corrCount,
		}
		count++
	}

	logger.Info("loaded RLHF corrections from Spanner", "count", count)
	return nil
}

func (cs *correctionStore) applyCorrection(retailerID, warehouseID, skuID string, rawQty int64) int64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	key := correctionKey(retailerID, warehouseID)
	if retail, ok := cs.weights[key]; ok {
		if entry, ok := retail[skuID]; ok {
			adjusted := int64(math.Round(float64(rawQty) * entry.Factor))
			if adjusted < 1 {
				adjusted = 1
			}
			return adjusted
		}
	}
	return rawQty
}

// getTriggerDateShift returns the accumulated trigger date shift in hours for a retailer+warehouse.
func (cs *correctionStore) getTriggerDateShift(retailerID, warehouseID string) float64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	key := correctionKey(retailerID, warehouseID)
	if retail, ok := cs.weights[key]; ok {
		var totalShift float64
		for _, entry := range retail {
			totalShift += entry.TriggerDateShiftH
		}
		// Return the average shift across all SKUs for this retailer+warehouse
		if len(retail) > 0 {
			return totalShift / float64(len(retail))
		}
	}
	return 0
}

func (cs *correctionStore) recordCorrection(retailerID, warehouseID, skuID, field, oldVal, newVal string) {
	cs.mu.Lock()
	key := correctionKey(retailerID, warehouseID)
	if _, ok := cs.weights[key]; !ok {
		cs.weights[key] = make(map[string]correctionEntry)
	}

	entry := cs.weights[key][skuID]
	if entry.Factor == 0 {
		entry.Factor = 1.0
	}
	entry.CorrectionCount++

	switch field {
	case "rejected":
		entry.Factor = 0.5
	case "amount":
		var oldAmt, newAmt float64
		fmt.Sscanf(oldVal, "%f", &oldAmt)
		fmt.Sscanf(newVal, "%f", &newAmt)
		if oldAmt > 0 && newAmt > 0 {
			ratio := newAmt / oldAmt
			entry.Factor = entry.Factor*0.7 + ratio*0.3
		}
	}

	cs.weights[key][skuID] = entry
	cs.mu.Unlock()

	// Write-through to Spanner (async, non-blocking)
	if cs.client != nil {
		go cs.persistCorrection(retailerID, warehouseID, skuID, entry)
	}
}

func (cs *correctionStore) recordDateShift(retailerID, warehouseID, oldDate, newDate string) {
	oldT, err1 := time.Parse(time.RFC3339, oldDate)
	newT, err2 := time.Parse(time.RFC3339, newDate)
	if err1 != nil || err2 != nil {
		// Try date-only format
		oldT, err1 = time.Parse("2006-01-02", oldDate)
		newT, err2 = time.Parse("2006-01-02", newDate)
		if err1 != nil || err2 != nil {
			logger.Warn("failed to parse date-shift dates", "old", oldDate, "new", newDate)
			return
		}
	}

	shiftHours := newT.Sub(oldT).Hours()

	cs.mu.Lock()
	key := correctionKey(retailerID, warehouseID)
	if _, ok := cs.weights[key]; !ok {
		cs.weights[key] = make(map[string]correctionEntry)
	}

	// Apply shift as an average across all known SKUs for this retailer+warehouse,
	// or create a synthetic "_date_shift" entry if no SKUs are known yet.
	syntheticKey := "_date_shift"
	entry := cs.weights[key][syntheticKey]
	if entry.Factor == 0 {
		entry.Factor = 1.0
	}
	// Exponential moving average for shift
	entry.TriggerDateShiftH = entry.TriggerDateShiftH*0.7 + shiftHours*0.3
	entry.CorrectionCount++
	cs.weights[key][syntheticKey] = entry
	cs.mu.Unlock()

	if cs.client != nil {
		go cs.persistCorrection(retailerID, warehouseID, syntheticKey, entry)
	}

	logger.Info("RLHF date shift recorded",
		"retailer", retailerID,
		"warehouse", warehouseID,
		"shift_hours", shiftHours,
		"ema_shift", entry.TriggerDateShiftH,
	)
}

func (cs *correctionStore) persistCorrection(retailerID, warehouseID, skuID string, entry correctionEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cols := []string{"RetailerId", "WarehouseId", "SkuId", "Factor", "TriggerDateShiftH", "CorrectionCount", "LastCorrectedAt"}
	m := spanner.InsertOrUpdate("CorrectionWeights", cols, []interface{}{
		retailerID, warehouseID, skuID, entry.Factor, entry.TriggerDateShiftH, entry.CorrectionCount, spanner.CommitTimestamp,
	})

	if _, err := cs.client.Apply(ctx, []*spanner.Mutation{m}); err != nil {
		logger.Error("failed to persist correction weight", "err", err,
			"retailer", retailerID, "warehouse", warehouseID, "sku", skuID)
	}
}

// allWeights returns all correction entries for the visibility endpoint.
func (cs *correctionStore) allWeights(retailerID string) map[string]map[string]correctionEntry {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[string]map[string]correctionEntry)
	for key, skus := range cs.weights {
		// Filter by retailer if provided
		if retailerID != "" {
			if len(key) < len(retailerID) || key[:len(retailerID)] != retailerID {
				continue
			}
		}
		clone := make(map[string]correctionEntry)
		for k, v := range skus {
			clone[k] = v
		}
		result[key] = clone
	}
	return result
}
