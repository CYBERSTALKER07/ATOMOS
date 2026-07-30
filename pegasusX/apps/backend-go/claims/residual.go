package claims

import (
	"context"
	"strings"
)

type ResidualResult struct {
	OrderID                 string
	RemainingClaimableMinor int64
	DeliveredGrossMinor     int64
	AlreadyClaimedMinor     int64
	Lines                   []LineResidual
}

type LineResidual struct {
	OrderLineID             string
	SKU                     string
	DeliveredQty            int64
	ClaimedQty              int64
	RemainingQty            int64
	RemainingClaimableMinor int64
}

// GetRemainingClaimable calculates the remaining residual value and quantity claimable on an order.
// This is the canonical, integer-safe function required by compliance dashboards, claim creation, and settlement.
func (s *Service) GetRemainingClaimable(ctx context.Context, orderID string) (*ResidualResult, error) {
	if s == nil || s.orders == nil || s.repo == nil {
		return nil, ErrOrderNotFound
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, ErrOrderNotFound
	}

	// 1. Load the order snapshot (which contains delivered quantities and TotalMinor)
	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOrderNotFound
	}

	// 2. Load all claims for the order
	claimsList, err := s.repo.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 3. Sum claimed values and quantities from APPROVED/SETTLED/RESOLVED claims only
	var claimedMinor int64
	claimedQtyBySKU := make(map[string]int64)

	for _, c := range claimsList {
		if c.Status == StatusApproved || c.Status == StatusResolved {
			claimedMinor += c.AmountMinor
			for _, li := range c.LineItems {
				sku := strings.TrimSpace(li.SKU)
				if sku != "" && li.Quantity > 0 {
					claimedQtyBySKU[sku] += li.Quantity
				}
			}
		}
	}

	// 4. Calculate total residuals using integer math
	deliveredGross := o.TotalMinor
	if deliveredGross < 0 {
		deliveredGross = 0
	}
	remainingGross := deliveredGross - claimedMinor
	if remainingGross < 0 {
		remainingGross = 0
	}

	res := &ResidualResult{
		OrderID:                 orderID,
		DeliveredGrossMinor:     deliveredGross,
		AlreadyClaimedMinor:     claimedMinor,
		RemainingClaimableMinor: remainingGross,
		Lines:                   make([]LineResidual, 0, len(o.LineItems)),
	}

	// 5. Calculate per-line residuals
	for _, li := range o.LineItems {
		sku := strings.TrimSpace(li.SKU)
		if sku == "" || li.Quantity <= 0 {
			continue
		}
		
		cQty := claimedQtyBySKU[sku]
		rQty := li.Quantity - cQty
		if rQty < 0 {
			rQty = 0
		}

		// Calculate remaining minor proportionally strictly by remaining qty * unit price
		remMinor := rQty * li.UnitPriceMinor
		
		res.Lines = append(res.Lines, LineResidual{
			SKU:                     sku,
			DeliveredQty:            li.Quantity,
			ClaimedQty:              cQty,
			RemainingQty:            rQty,
			RemainingClaimableMinor: remMinor,
		})
	}

	return res, nil
}
