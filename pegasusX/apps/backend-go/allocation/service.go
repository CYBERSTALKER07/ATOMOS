package allocation

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
	"google.golang.org/api/iterator"
)

const (
	ConstraintReasonOK     = "OK"
	ConstraintReasonLegacy = "LEGACY"
)

type AllocationRequest struct {
	SupplierId string
	RetailerId string
	OrderId    string
	Items      []AllocationItem
}

type AllocationItem struct {
	ProductId        string
	QuantityRequired int64
}

type LineDecision struct {
	Sku             string `json:"sku"`
	WarehouseId     string `json:"warehouse_id"`
	AllocationMode  string `json:"allocation_mode"`
	PriorityScore   int64  `json:"priority_score"`
	FairShareBps    int64  `json:"fair_share_bps"`
	PolicyId        string `json:"policy_id,omitempty"`
	RetailerSegment string `json:"retailer_segment,omitempty"`
	SkuClass        string `json:"sku_class,omitempty"`
	RiskTier        string `json:"risk_tier,omitempty"`
	ConstraintReason string `json:"constraint_reason,omitempty"`
	RequestedQty    int64  `json:"requested_qty,omitempty"`
	AllocatedQty    int64  `json:"allocated_qty,omitempty"`
}

type AllocationResult struct {
	Fulfillments map[string]string
	Decisions    []LineDecision
	Mode         string
}

type stockInfo struct {
	WarehouseId string
	Available   int64
}

type AllocationService struct {
	client             *spanner.Client
	segment            *segment.Service
	constrainedEnabled bool
}

func NewAllocationService(client *spanner.Client) *AllocationService {
	return &AllocationService{client: client}
}

func (s *AllocationService) SetSegmentService(seg *segment.Service) {
	if s != nil {
		s.segment = seg
	}
}

func (s *AllocationService) SetConstrainedAllocationEnabled(enabled bool) {
	if s != nil {
		s.constrainedEnabled = enabled
	}
}

// AllocateOrder returns a mapping of which WarehouseId fulfills which products.
func (s *AllocationService) AllocateOrder(ctx context.Context, req *AllocationRequest) (*AllocationResult, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("allocation service unavailable")
	}
	if len(req.Items) == 0 {
		return &AllocationResult{Fulfillments: make(map[string]string), Mode: segment.AllocationModeFirstFit}, nil
	}
	return s.allocateFromInventory(ctx, req, s.client.Single().Query)
}

// AllocateOrderTxn runs allocation reads inside an existing Spanner RW transaction.
func (s *AllocationService) AllocateOrderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, req *AllocationRequest) (*AllocationResult, error) {
	if txn == nil {
		return nil, fmt.Errorf("allocation txn required")
	}
	if len(req.Items) == 0 {
		return &AllocationResult{Fulfillments: make(map[string]string), Mode: segment.AllocationModeFirstFit}, nil
	}
	return s.allocateFromInventory(ctx, req, txn.Query)
}

type queryFunc func(ctx context.Context, stmt spanner.Statement) *spanner.RowIterator

func (s *AllocationService) allocateFromInventory(ctx context.Context, req *AllocationRequest, query queryFunc) (*AllocationResult, error) {
	if s.constrainedEnabled && s.segment != nil {
		return s.allocateWithPolicy(ctx, req, query)
	}
	return s.allocateFirstFit(ctx, req, query)
}

func (s *AllocationService) allocateFirstFit(ctx context.Context, req *AllocationRequest, query queryFunc) (*AllocationResult, error) {
	supplierID := strings.TrimSpace(req.SupplierId)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required for allocation")
	}

	productIds, qtyReqs, err := normalizeAllocationItems(req.Items)
	if err != nil {
		return nil, err
	}
	if len(productIds) == 0 {
		return &AllocationResult{Fulfillments: make(map[string]string), Mode: segment.AllocationModeFirstFit}, nil
	}

	_, inventory, err := s.loadInventory(ctx, query, supplierID, productIds)
	if err != nil {
		return nil, err
	}

	fulfillments := make(map[string]string)
	decisions := make([]LineDecision, 0, len(qtyReqs))
	for pid, reqQty := range qtyReqs {
		stocks, ok := inventory[pid]
		if !ok {
			return nil, fmt.Errorf("product %s not found in any warehouse", pid)
		}

		allocated := false
		for _, st := range stocks {
			if st.Available >= reqQty {
				fulfillments[pid] = st.WarehouseId
				decisions = append(decisions, LineDecision{
					Sku:              pid,
					WarehouseId:      st.WarehouseId,
					AllocationMode:   segment.AllocationModeFirstFit,
					ConstraintReason: ConstraintReasonLegacy,
					RequestedQty:     reqQty,
					AllocatedQty:     reqQty,
				})
				allocated = true
				break
			}
		}
		if !allocated {
			return nil, fmt.Errorf("insufficient stock for product %s (required: %d)", pid, reqQty)
		}
	}

	return &AllocationResult{
		Fulfillments: fulfillments,
		Decisions:    decisions,
		Mode:         segment.AllocationModeFirstFit,
	}, nil
}

func normalizeAllocationItems(items []AllocationItem) ([]string, map[string]int64, error) {
	productIds := make([]string, 0, len(items))
	qtyReqs := make(map[string]int64)
	for _, item := range items {
		pid := strings.TrimSpace(item.ProductId)
		if pid == "" || item.QuantityRequired <= 0 {
			continue
		}
		productIds = append(productIds, pid)
		qtyReqs[pid] += item.QuantityRequired
	}
	return productIds, qtyReqs, nil
}

func (s *AllocationService) loadInventory(
	ctx context.Context,
	query queryFunc,
	supplierID string,
	productIds []string,
) (map[string]bool, map[string][]stockInfo, error) {
	activeWarehouses, err := s.loadActiveWarehouses(ctx, query, supplierID)
	if err != nil {
		return nil, nil, err
	}
	if len(activeWarehouses) == 0 {
		return nil, nil, fmt.Errorf("no active warehouses for supplier %s", supplierID)
	}

	stmt := spanner.Statement{
		SQL: `SELECT ProductId, WarehouseId, QuantityOnHand, QuantityReserved
			  FROM SupplierInventoryV2
			  WHERE SupplierId = @supplierId
			    AND ProductId IN UNNEST(@productIds)`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"productIds": productIds,
		},
	}

	iter := query(ctx, stmt)
	defer iter.Stop()

	inventory := make(map[string][]stockInfo)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query inventory: %w", err)
		}

		var pid, wid string
		var qoh, qr int64
		if err := row.Columns(&pid, &wid, &qoh, &qr); err != nil {
			return nil, nil, fmt.Errorf("failed to parse inventory row: %w", err)
		}
		if !activeWarehouses[wid] {
			continue
		}

		inventory[pid] = append(inventory[pid], stockInfo{
			WarehouseId: wid,
			Available:   qoh - qr,
		})
	}
	return activeWarehouses, inventory, nil
}

func (s *AllocationService) loadActiveWarehouses(ctx context.Context, query queryFunc, supplierID string) (map[string]bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId FROM Warehouses
		      WHERE SupplierId = @supplierId AND IsActive = true`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}
	iter := query(ctx, stmt)
	defer iter.Stop()

	active := make(map[string]bool)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query warehouses: %w", err)
		}
		var wid string
		if err := row.Column(0, &wid); err != nil {
			return nil, fmt.Errorf("failed to parse warehouse row: %w", err)
		}
		wid = strings.TrimSpace(wid)
		if wid != "" {
			active[wid] = true
		}
	}
	return active, nil
}
