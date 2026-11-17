package valuation

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository is a thread-safe in-memory Repository for unit tests and non-Spanner harnesses.
type MemoryRepository struct {
	mu            sync.Mutex
	layers        map[string][]InventoryCostLayer
	movements     map[string][]InventoryMovement
	idemMovements map[string][]InventoryMovement
	now           func() time.Time
}

// NewMemoryRepository constructs an initialized in-memory valuation repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		layers:        make(map[string][]InventoryCostLayer),
		movements:     make(map[string][]InventoryMovement),
		idemMovements: make(map[string][]InventoryMovement),
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (m *MemoryRepository) RecordReceipt(ctx context.Context, in ReceiptInput) (ReceiptResult, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return ReceiptResult{}, err
		}
	}
	norm, totalCost, creditAcct, expiryDateStr, expiryCivil, err := validateAndNormalizeReceipt(in, m.now())
	if err != nil {
		return ReceiptResult{}, err
	}
	_ = creditAcct

	m.mu.Lock()
	defer m.mu.Unlock()

	idemKey := norm.EntityID + "\x00" + norm.IdempotencyKey
	if prev, ok := m.idemMovements[idemKey]; ok && len(prev) > 0 {
		mv := prev[0]
		var foundLayer InventoryCostLayer
		for _, l := range m.layers[norm.EntityID] {
			if l.LayerID == mv.LayerID {
				foundLayer = l
				break
			}
		}
		return ReceiptResult{
			Layer:            foundLayer,
			Movement:         mv,
			GlEntryID:        mv.GlEntryID,
			TotalCostMinor:   mv.TotalCostMinor,
			VatMinor:         norm.VatMinor,
			AlreadyProcessed: true,
		}, nil
	}

	layerID := "vcl_" + uuid.NewString()
	mvID := "vmv_" + uuid.NewString()
	glEntryID := "mem_gl_" + uuid.NewString()[:12]
	ts := norm.ReceivedAt

	daysToExp := ComputeExpiryDaysRemaining(expiryCivil, ts)
	winDays := norm.ExpiryRiskWindowDays
	if winDays <= 0 {
		winDays = DefaultExpiryRiskWindowDays
	}
	nearExpiry := daysToExp != nil && *daysToExp <= winDays

	layer := InventoryCostLayer{
		EntityID:        norm.EntityID,
		WarehouseID:     norm.WarehouseID,
		ProductID:       norm.ProductID,
		LayerID:         layerID,
		LotID:           norm.LotID,
		ReceiptRefType:  norm.ReceiptRefType,
		ReceiptRefID:    norm.ReceiptRefID,
		QtyRemaining:    norm.Quantity,
		OriginalQty:     norm.Quantity,
		UnitCostMinor:   norm.UnitCostMinor,
		TotalValueMinor: totalCost,
		Currency:        norm.Currency,
		FxRateID:        norm.FxRateID,
		ReceivedAt:      ts,
		ExpiryDate:      expiryDateStr,
		NearExpiry:      nearExpiry,
		DaysToExpiry:    daysToExp,
		CreatedAt:       ts,
		UpdatedAt:       ts,
	}
	mv := InventoryMovement{
		EntityID:       norm.EntityID,
		MovementID:     mvID,
		MovementType:   MovementTypeReceipt,
		ReferenceType:  norm.ReceiptRefType,
		ReferenceID:    norm.ReceiptRefID,
		WarehouseID:    norm.WarehouseID,
		ProductID:      norm.ProductID,
		LotID:          norm.LotID,
		LayerID:        layerID,
		QtyDelta:       norm.Quantity,
		UnitCostMinor:  norm.UnitCostMinor,
		TotalCostMinor: totalCost,
		Currency:       norm.Currency,
		GlEntryID:      glEntryID,
		IdempotencyKey: norm.IdempotencyKey,
		CreatedAt:      ts,
	}

	m.layers[norm.EntityID] = append(m.layers[norm.EntityID], layer)
	m.movements[norm.EntityID] = append(m.movements[norm.EntityID], mv)
	m.idemMovements[idemKey] = []InventoryMovement{mv}

	return ReceiptResult{
		Layer:             layer,
		Movement:          mv,
		GlEntryID:         glEntryID,
		TotalCostMinor:    totalCost,
		VatMinor:          norm.VatMinor,
		ExpiryRiskEmitted: nearExpiry,
	}, nil
}

func (m *MemoryRepository) IssueStockFEFO(ctx context.Context, in IssueInput) (IssueResult, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return IssueResult{}, err
		}
	}
	norm, err := validateAndNormalizeIssue(in, m.now())
	if err != nil {
		return IssueResult{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idemKey := norm.EntityID + "\x00" + norm.IdempotencyKey
	if prev, ok := m.idemMovements[idemKey]; ok && len(prev) > 0 {
		return buildIssueResultFromMovements(prev, true), nil
	}

	layers := m.layers[norm.EntityID]
	idxs := selectActiveLayerIndicesFEFO(layers, norm.WarehouseID, norm.ProductID, norm.LotID)
	var totalAvail int64
	var totalVal int64
	for _, idx := range idxs {
		totalAvail += layers[idx].QtyRemaining
		totalVal += layers[idx].QtyRemaining * layers[idx].UnitCostMinor
	}
	if totalAvail < norm.Quantity {
		return IssueResult{}, fmt.Errorf("%w: have %d need %d for (%s/%s)", ErrInsufficientCostLayers, totalAvail, norm.Quantity, norm.WarehouseID, norm.ProductID)
	}

	glEntryID := "mem_gl_" + uuid.NewString()[:12]
	remaining := norm.Quantity
	created := make([]InventoryMovement, 0, len(idxs))

	if strings.EqualFold(norm.CostingMethod, CostingMethodMovingAverage) {
		targetTotalCost := computeProportionalCost(totalVal, totalAvail, norm.Quantity)
		avgUnit := ComputeMovingAverageUnitCostMinor(targetTotalCost, norm.Quantity)
		remCost := targetTotalCost
		for partIdx, idx := range idxs {
			if remaining == 0 {
				break
			}
			take := layers[idx].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[idx].QtyRemaining -= take
			layers[idx].TotalValueMinor = layers[idx].QtyRemaining * layers[idx].UnitCostMinor
			layers[idx].UpdatedAt = norm.OccurredAt

			var layerCost int64
			if remaining == 0 {
				layerCost = remCost
			} else {
				layerCost = take * avgUnit
				if layerCost > remCost {
					layerCost = remCost
				}
			}
			remCost -= layerCost

			mvKey := norm.IdempotencyKey
			if partIdx > 0 {
				mvKey = norm.IdempotencyKey + "#layer:" + strconv.Itoa(partIdx)
			}
			created = append(created, InventoryMovement{
				EntityID:       norm.EntityID,
				MovementID:     "vmv_" + uuid.NewString(),
				MovementType:   MovementTypeIssueCOGS,
				ReferenceType:  norm.ReferenceType,
				ReferenceID:    norm.ReferenceID,
				WarehouseID:    norm.WarehouseID,
				ProductID:      norm.ProductID,
				LotID:          layers[idx].LotID,
				LayerID:        layers[idx].LayerID,
				QtyDelta:       -take,
				UnitCostMinor:  ComputeMovingAverageUnitCostMinor(layerCost, take),
				TotalCostMinor: layerCost,
				Currency:       norm.Currency,
				GlEntryID:      glEntryID,
				IdempotencyKey: mvKey,
				CreatedAt:      norm.OccurredAt,
			})
		}
		// Keep active layers' remaining value tied cleanly to GL INVENTORY (totalVal - targetTotalCost).
		if split := rebalanceRemainingLayerCosts(&layers, idxs, totalAvail-norm.Quantity, totalVal-targetTotalCost, norm.OccurredAt); split != nil {
			layers = append(layers, *split)
		}
	} else {
		for partIdx, idx := range idxs {
			if remaining == 0 {
				break
			}
			take := layers[idx].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[idx].QtyRemaining -= take
			layers[idx].TotalValueMinor = layers[idx].QtyRemaining * layers[idx].UnitCostMinor
			layers[idx].UpdatedAt = norm.OccurredAt

			mvKey := norm.IdempotencyKey
			if partIdx > 0 {
				mvKey = norm.IdempotencyKey + "#layer:" + strconv.Itoa(partIdx)
			}
			created = append(created, InventoryMovement{
				EntityID:       norm.EntityID,
				MovementID:     "vmv_" + uuid.NewString(),
				MovementType:   MovementTypeIssueCOGS,
				ReferenceType:  norm.ReferenceType,
				ReferenceID:    norm.ReferenceID,
				WarehouseID:    norm.WarehouseID,
				ProductID:      norm.ProductID,
				LotID:          layers[idx].LotID,
				LayerID:        layers[idx].LayerID,
				QtyDelta:       -take,
				UnitCostMinor:  layers[idx].UnitCostMinor,
				TotalCostMinor: take * layers[idx].UnitCostMinor,
				Currency:       norm.Currency,
				GlEntryID:      glEntryID,
				IdempotencyKey: mvKey,
				CreatedAt:      norm.OccurredAt,
			})
		}
	}

	m.layers[norm.EntityID] = layers
	m.movements[norm.EntityID] = append(m.movements[norm.EntityID], created...)
	m.idemMovements[idemKey] = append([]InventoryMovement(nil), created...)
	return buildIssueResultFromMovements(created, false), nil
}

func (m *MemoryRepository) RecordAdjustmentCost(ctx context.Context, in AdjustmentCostInput) (AdjustmentCostResult, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return AdjustmentCostResult{}, err
		}
	}
	norm, err := validateAndNormalizeAdjustment(in, m.now())
	if err != nil {
		return AdjustmentCostResult{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	idemKey := norm.EntityID + "\x00" + norm.IdempotencyKey
	if prev, ok := m.idemMovements[idemKey]; ok && len(prev) > 0 {
		var total int64
		var glID string
		for _, mv := range prev {
			total += mv.TotalCostMinor
			if glID == "" {
				glID = mv.GlEntryID
			}
		}
		return AdjustmentCostResult{
			Movements:        append([]InventoryMovement(nil), prev...),
			TotalCostMinor:   total,
			GlEntryID:        glID,
			AlreadyProcessed: true,
		}, nil
	}

	layers := m.layers[norm.EntityID]
	idxs := selectActiveLayerIndicesFEFO(layers, norm.WarehouseID, norm.ProductID, norm.LotID)
	glEntryID := "mem_gl_" + uuid.NewString()[:12]

	if norm.DeltaQty < 0 {
		need := -norm.DeltaQty
		var avail int64
		for _, idx := range idxs {
			avail += layers[idx].QtyRemaining
		}
		if avail < need {
			return AdjustmentCostResult{}, fmt.Errorf("%w: shrink %d exceeds active qty %d", ErrInsufficientCostLayers, need, avail)
		}
		remaining := need
		var totalCost int64
		created := make([]InventoryMovement, 0, len(idxs))
		for partIdx, idx := range idxs {
			if remaining == 0 {
				break
			}
			take := layers[idx].QtyRemaining
			if take > remaining {
				take = remaining
			}
			remaining -= take
			layers[idx].QtyRemaining -= take
			layers[idx].TotalValueMinor = layers[idx].QtyRemaining * layers[idx].UnitCostMinor
			layers[idx].UpdatedAt = norm.OccurredAt
			cost := take * layers[idx].UnitCostMinor
			totalCost += cost

			mvKey := norm.IdempotencyKey
			if partIdx > 0 {
				mvKey = norm.IdempotencyKey + "#layer:" + strconv.Itoa(partIdx)
			}
			created = append(created, InventoryMovement{
				EntityID:       norm.EntityID,
				MovementID:     "vmv_" + uuid.NewString(),
				MovementType:   MovementTypeAdjustmentShrink,
				ReferenceType:  "INVENTORY_ADJUSTMENT",
				ReferenceID:    norm.AdjustmentID,
				WarehouseID:    norm.WarehouseID,
				ProductID:      norm.ProductID,
				LotID:          layers[idx].LotID,
				LayerID:        layers[idx].LayerID,
				QtyDelta:       -take,
				UnitCostMinor:  layers[idx].UnitCostMinor,
				TotalCostMinor: cost,
				Currency:       norm.Currency,
				GlEntryID:      glEntryID,
				IdempotencyKey: mvKey,
				CreatedAt:      norm.OccurredAt,
			})
		}
		m.layers[norm.EntityID] = layers
		m.movements[norm.EntityID] = append(m.movements[norm.EntityID], created...)
		m.idemMovements[idemKey] = append([]InventoryMovement(nil), created...)
		return AdjustmentCostResult{
			Movements:      created,
			TotalCostMinor: totalCost,
			GlEntryID:      glEntryID,
		}, nil
	}

	unitCost := norm.UnitCostMinor
	if unitCost <= 0 {
		var totalQ, totalV int64
		for _, idx := range idxs {
			totalQ += layers[idx].QtyRemaining
			totalV += layers[idx].QtyRemaining * layers[idx].UnitCostMinor
		}
		unitCost = ComputeMovingAverageUnitCostMinor(totalV, totalQ)
	}
	if unitCost <= 0 {
		return AdjustmentCostResult{}, fmt.Errorf("%w: unit_cost_minor required for positive adjustment when no active layers exist", ErrInvalidInput)
	}

	totalCost := norm.DeltaQty * unitCost
	layerID := "vcl_" + uuid.NewString()
	layer := InventoryCostLayer{
		EntityID:        norm.EntityID,
		WarehouseID:     norm.WarehouseID,
		ProductID:       norm.ProductID,
		LayerID:         layerID,
		LotID:           norm.LotID,
		ReceiptRefType:  "INVENTORY_ADJUSTMENT",
		ReceiptRefID:    norm.AdjustmentID,
		QtyRemaining:    norm.DeltaQty,
		OriginalQty:     norm.DeltaQty,
		UnitCostMinor:   unitCost,
		TotalValueMinor: totalCost,
		Currency:        norm.Currency,
		ReceivedAt:      norm.OccurredAt,
		CreatedAt:       norm.OccurredAt,
		UpdatedAt:       norm.OccurredAt,
	}
	mv := InventoryMovement{
		EntityID:       norm.EntityID,
		MovementID:     "vmv_" + uuid.NewString(),
		MovementType:   MovementTypeAdjustmentGain,
		ReferenceType:  "INVENTORY_ADJUSTMENT",
		ReferenceID:    norm.AdjustmentID,
		WarehouseID:    norm.WarehouseID,
		ProductID:      norm.ProductID,
		LotID:          norm.LotID,
		LayerID:        layerID,
		QtyDelta:       norm.DeltaQty,
		UnitCostMinor:  unitCost,
		TotalCostMinor: totalCost,
		Currency:       norm.Currency,
		GlEntryID:      glEntryID,
		IdempotencyKey: norm.IdempotencyKey,
		CreatedAt:      norm.OccurredAt,
	}
	m.layers[norm.EntityID] = append(layers, layer)
	m.movements[norm.EntityID] = append(m.movements[norm.EntityID], mv)
	m.idemMovements[idemKey] = []InventoryMovement{mv}
	return AdjustmentCostResult{
		Movements:      []InventoryMovement{mv},
		Layer:          &layer,
		TotalCostMinor: totalCost,
		GlEntryID:      glEntryID,
	}, nil
}

func (m *MemoryRepository) GetValuationSummary(ctx context.Context, filter ValuationFilter) (ValuationSummaryResponse, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return ValuationSummaryResponse{}, err
		}
	}
	entityID := strings.TrimSpace(filter.EntityID)
	if entityID == "" {
		return ValuationSummaryResponse{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	winDays := filter.ExpiryRiskDays
	if winDays <= 0 {
		winDays = DefaultExpiryRiskWindowDays
	}

	m.mu.Lock()
	all := append([]InventoryCostLayer(nil), m.layers[entityID]...)
	now := m.now()
	m.mu.Unlock()

	var filtered []InventoryCostLayer
	for _, l := range all {
		if filter.WarehouseID != "" && l.WarehouseID != strings.TrimSpace(filter.WarehouseID) {
			continue
		}
		if filter.ProductID != "" && l.ProductID != strings.TrimSpace(filter.ProductID) {
			continue
		}
		if !filter.IncludeZeroQty && l.QtyRemaining <= 0 {
			continue
		}
		if l.ExpiryDate != "" {
			if d, err := ParseExpiryDate(l.ExpiryDate); err == nil && d != nil {
				days := ComputeExpiryDaysRemaining(d, now)
				l.DaysToExpiry = days
				l.NearExpiry = days != nil && *days <= winDays
			}
		}
		l.TotalValueMinor = l.QtyRemaining * l.UnitCostMinor
		filtered = append(filtered, l)
	}
	sortLayersFEFO(filtered)
	return buildValuationResponse(entityID, strings.TrimSpace(filter.WarehouseID), winDays, filtered), nil
}

func (m *MemoryRepository) ListMovements(ctx context.Context, filter MovementsFilter) ([]InventoryMovement, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	entityID := strings.TrimSpace(filter.EntityID)
	if entityID == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	m.mu.Lock()
	all := append([]InventoryMovement(nil), m.movements[entityID]...)
	m.mu.Unlock()

	out := make([]InventoryMovement, 0, len(all))
	for _, mv := range all {
		if filter.WarehouseID != "" && mv.WarehouseID != strings.TrimSpace(filter.WarehouseID) {
			continue
		}
		if filter.ProductID != "" && mv.ProductID != strings.TrimSpace(filter.ProductID) {
			continue
		}
		if filter.MovementType != "" && !strings.EqualFold(mv.MovementType, strings.TrimSpace(filter.MovementType)) {
			continue
		}
		out = append(out, mv)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].MovementID > out[j].MovementID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryRepository) ScanExpiryRisks(ctx context.Context, entityID, warehouseID string, withinDays int64, _ bool) ([]ExpiryRiskLayer, error) {
	sum, err := m.GetValuationSummary(ctx, ValuationFilter{
		EntityID:       entityID,
		WarehouseID:    warehouseID,
		ExpiryRiskDays: withinDays,
	})
	if err != nil {
		return nil, err
	}
	var out []ExpiryRiskLayer
	for _, l := range sum.Layers {
		if l.NearExpiry && l.DaysToExpiry != nil && l.QtyRemaining > 0 {
			out = append(out, ExpiryRiskLayer{
				Layer:            l,
				ValueAtRiskMinor: l.TotalValueMinor,
				DaysRemaining:    *l.DaysToExpiry,
			})
		}
	}
	return out, nil
}

func selectActiveLayerIndicesFEFO(layers []InventoryCostLayer, warehouseID, productID, preferredLotID string) []int {
	var preferred []int
	var rest []int
	for i := range layers {
		l := &layers[i]
		if l.WarehouseID != warehouseID || l.ProductID != productID || l.QtyRemaining <= 0 {
			continue
		}
		if preferredLotID != "" && l.LotID == preferredLotID {
			preferred = append(preferred, i)
		} else {
			rest = append(rest, i)
		}
	}
	cmp := func(a, b int) bool {
		la, lb := layers[a], layers[b]
		if la.ExpiryDate != lb.ExpiryDate {
			if la.ExpiryDate == "" {
				return false // NULLS LAST
			}
			if lb.ExpiryDate == "" {
				return true
			}
			return la.ExpiryDate < lb.ExpiryDate
		}
		if !la.ReceivedAt.Equal(lb.ReceivedAt) {
			return la.ReceivedAt.Before(lb.ReceivedAt)
		}
		return la.LayerID < lb.LayerID
	}
	sort.Slice(preferred, func(i, j int) bool { return cmp(preferred[i], preferred[j]) })
	sort.Slice(rest, func(i, j int) bool { return cmp(rest[i], rest[j]) })
	return append(preferred, rest...)
}

func rebalanceRemainingLayerCosts(layers *[]InventoryCostLayer, idxs []int, remQty, remValue int64, now time.Time) *InventoryCostLayer {
	if remQty <= 0 {
		return nil
	}
	avg := remValue / remQty
	rem := remValue % remQty
	lastActive := -1
	for _, idx := range idxs {
		if (*layers)[idx].QtyRemaining > 0 {
			lastActive = idx
		}
	}
	var split *InventoryCostLayer
	for _, idx := range idxs {
		l := &(*layers)[idx]
		if l.QtyRemaining <= 0 {
			continue
		}
		if idx == lastActive && rem != 0 {
			if l.QtyRemaining == 1 {
				l.UnitCostMinor = avg + rem
			} else {
				l.QtyRemaining -= 1
				l.UnitCostMinor = avg
				l.TotalValueMinor = l.QtyRemaining * l.UnitCostMinor
				cp := *l
				cp.LayerID = l.LayerID + "-rem"
				cp.QtyRemaining = 1
				cp.OriginalQty = 1
				cp.UnitCostMinor = avg + rem
				cp.TotalValueMinor = avg + rem
				cp.UpdatedAt = now
				split = &cp
				continue
			}
		} else {
			l.UnitCostMinor = avg
		}
		l.TotalValueMinor = l.QtyRemaining * l.UnitCostMinor
	}
	return split
}
