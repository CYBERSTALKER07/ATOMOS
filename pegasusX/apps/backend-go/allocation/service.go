package allocation

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type AllocationRequest struct {
	Items []AllocationItem
}

type AllocationItem struct {
	ProductId        string
	QuantityRequired int64
}

type AllocationResult struct {
	// Map of ProductId to WarehouseId
	Fulfillments map[string]string
}

type AllocationService struct {
	client *spanner.Client
}

func NewAllocationService(client *spanner.Client) *AllocationService {
	return &AllocationService{client: client}
}

// AllocateOrder returns a mapping of which WarehouseId fulfills which products.
// It uses a naive algorithm: pick the first warehouse that has sufficient (QuantityOnHand - QuantityReserved).
func (s *AllocationService) AllocateOrder(ctx context.Context, req *AllocationRequest) (*AllocationResult, error) {
	if len(req.Items) == 0 {
		return &AllocationResult{Fulfillments: make(map[string]string)}, nil
	}

	productIds := make([]string, 0, len(req.Items))
	qtyReqs := make(map[string]int64)
	for _, item := range req.Items {
		productIds = append(productIds, item.ProductId)
		qtyReqs[item.ProductId] = item.QuantityRequired
	}

	stmt := spanner.Statement{
		SQL: `SELECT ProductId, WarehouseId, QuantityOnHand, QuantityReserved
			  FROM InventoryLevels
			  WHERE ProductId IN UNNEST(@productIds)`,
		Params: map[string]interface{}{
			"productIds": productIds,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type stockInfo struct {
		WarehouseId string
		Available   int64
	}
	inventory := make(map[string][]stockInfo)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query inventory: %w", err)
		}

		var pid, wid string
		var qoh, qr int64
		if err := row.Columns(&pid, &wid, &qoh, &qr); err != nil {
			return nil, fmt.Errorf("failed to parse inventory row: %w", err)
		}

		inventory[pid] = append(inventory[pid], stockInfo{
			WarehouseId: wid,
			Available:   qoh - qr,
		})
	}

	fulfillments := make(map[string]string)
	for pid, reqQty := range qtyReqs {
		stocks, ok := inventory[pid]
		if !ok {
			return nil, fmt.Errorf("product %s not found in any warehouse", pid)
		}

		allocated := false
		for _, st := range stocks {
			if st.Available >= reqQty {
				fulfillments[pid] = st.WarehouseId
				allocated = true
				break
			}
		}

		if !allocated {
			return nil, fmt.Errorf("insufficient stock for product %s (required: %d)", pid, reqQty)
		}
	}

	return &AllocationResult{Fulfillments: fulfillments}, nil
}
