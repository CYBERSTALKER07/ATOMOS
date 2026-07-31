package allocation

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
)

func (s *AllocationService) allocateWithPolicy(
	ctx context.Context,
	req *AllocationRequest,
	query queryFunc,
) (*AllocationResult, error) {
	supplierID := strings.TrimSpace(req.SupplierId)
	retailerID := strings.TrimSpace(req.RetailerId)

	productIds, qtyReqs, err := normalizeAllocationItems(req.Items)
	if err != nil {
		return nil, err
	}
	if len(productIds) == 0 {
		return &AllocationResult{Fulfillments: map[string]string{}, Mode: segment.AllocationModePolicy}, nil
	}

	activeWarehouses, inventory, err := s.loadInventory(ctx, query, supplierID, productIds)
	if err != nil {
		return nil, err
	}

	lineContexts := make(map[string]segment.LineAllocationContext, len(qtyReqs))
	for pid := range qtyReqs {
		ctxLine, err := s.segment.ResolveLineContext(ctx, supplierID, retailerID, pid)
		if err != nil {
			return nil, fmt.Errorf("resolve policy for %s: %w", pid, err)
		}
		lineContexts[pid] = ctxLine
	}

	candidates := warehouseCandidates(activeWarehouses, inventory, qtyReqs, lineContexts)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("insufficient stock for constrained allocation")
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].slack != candidates[j].slack {
			return candidates[i].slack > candidates[j].slack
		}
		return candidates[i].warehouseID < candidates[j].warehouseID
	})
	chosen := candidates[0].warehouseID

	fulfillments := make(map[string]string, len(qtyReqs))
	decisions := make([]LineDecision, 0, len(qtyReqs))
	for pid := range qtyReqs {
		fulfillments[pid] = chosen
		lc := lineContexts[pid]
		decisions = append(decisions, LineDecision{
			Sku:              pid,
			WarehouseId:     chosen,
			AllocationMode:  segment.AllocationModePolicy,
			PriorityScore:   lc.PriorityScore,
			FairShareBps:    0,
			PolicyId:        lc.Policy.PolicyID,
			RetailerSegment: lc.RetailerSegment,
			SkuClass:        lc.SkuClass,
			RiskTier:        string(lc.RiskTier),
			ConstraintReason: ConstraintReasonOK,
			RequestedQty:    qtyReqs[pid],
			AllocatedQty:    qtyReqs[pid],
		})
	}

	return &AllocationResult{
		Fulfillments: fulfillments,
		Decisions:    decisions,
		Mode:         segment.AllocationModePolicy,
	}, nil
}

type warehouseCandidate struct {
	warehouseID string
	score       int64
	slack       int64
}

func warehouseCandidates(
	active map[string]bool,
	inventory map[string][]stockInfo,
	qtyReqs map[string]int64,
	lineContexts map[string]segment.LineAllocationContext,
) []warehouseCandidate {
	warehouseIDs := make([]string, 0, len(active))
	for wid := range active {
		warehouseIDs = append(warehouseIDs, wid)
	}
	sort.Strings(warehouseIDs)

	var candidates []warehouseCandidate
	for _, wid := range warehouseIDs {
		var score int64
		var slack int64
		ok := true
		for pid, reqQty := range qtyReqs {
			stocks, exists := inventory[pid]
			if !exists {
				ok = false
				break
			}
			var available int64
			for _, st := range stocks {
				if st.WarehouseId == wid {
					available = st.Available
					break
				}
			}
			if available < reqQty {
				ok = false
				break
			}
			score += lineContexts[pid].PriorityScore
			slack += available - reqQty
		}
		if !ok {
			continue
		}
		candidates = append(candidates, warehouseCandidate{
			warehouseID: wid,
			score:       score,
			slack:       slack,
		})
	}
	return candidates
}
