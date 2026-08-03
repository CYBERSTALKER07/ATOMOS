package bootstrap

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
)

// retailerClaimStockBridge adapts retailer store stock to claims quarantine hold.
type retailerClaimStockBridge struct {
	svc *retailer.Service
}

func (b *retailerClaimStockBridge) HoldForClaim(ctx context.Context, retailerID, claimID, orderID string, lines []claims.ClaimLine, actor string) error {
	if b == nil || b.svc == nil {
		return nil
	}
	hl := make([]retailer.ClaimHoldLine, 0, len(lines))
	for _, l := range lines {
		hl = append(hl, retailer.ClaimHoldLine{SKU: l.SKU, Qty: l.Quantity})
	}
	return b.svc.HoldForClaim(ctx, retailerID, claimID, orderID, hl, actor)
}

func (b *retailerClaimStockBridge) ResolveClaimStock(ctx context.Context, retailerID, claimID string, lines []claims.ClaimLine, disposition, actor string) error {
	if b == nil || b.svc == nil {
		return nil
	}
	hl := make([]retailer.ClaimHoldLine, 0, len(lines))
	for _, l := range lines {
		hl = append(hl, retailer.ClaimHoldLine{SKU: l.SKU, Qty: l.Quantity})
	}
	return b.svc.ResolveClaimStock(ctx, retailerID, claimID, hl, disposition, actor)
}

// orderClaimsLookup adapts order.Service to claims.OrderLookup without cycles.
type orderClaimsLookup struct {
	svc *order.Service
}

func (l orderClaimsLookup) GetOrder(ctx context.Context, orderID string) (claims.OrderSnapshot, bool, error) {
	if l.svc == nil {
		return claims.OrderSnapshot{}, false, nil
	}
	o, ok, err := l.svc.GetOrder(ctx, orderID)
	if err != nil || !ok {
		return claims.OrderSnapshot{}, ok, err
	}
	lines := make([]claims.OrderLine, 0, len(o.LineItems))
	for _, li := range o.LineItems {
		lines = append(lines, claims.OrderLine{
			SKU:            li.SKU,
			Quantity:       li.Quantity,
			UnitPriceMinor: li.UnitPrice,
		})
	}
	return claims.OrderSnapshot{
		OrderID:            o.OrderID,
		SupplierID:         o.SupplierID,
		RetailerID:         o.RetailerID,
		WarehouseID:        o.WarehouseID,
		Currency:           o.Currency,
		Status:             string(o.Status),
		TotalMinor:         o.TotalMinor,
		OriginalTotalMinor: o.OriginalTotalMinor,
		LineItems:          lines,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}, true, nil
}

// driverClaimsBridge opens Claims rows from driver OS&D exception reports.
type driverClaimsBridge struct {
	svc *claims.Service
}

func (b *driverClaimsBridge) OnDriverException(ctx context.Context, o order.Order, driverID string, items []order.ExceptionReportItem, photos []string, note string) error {
	if b == nil || b.svc == nil {
		return nil
	}
	lines := make([]claims.ClaimLine, 0, len(items))
	claimType := claims.ClaimTypeMissing
	for _, it := range items {
		reason := strings.ToUpper(strings.TrimSpace(it.Reason))
		if reason == "" {
			reason = "MISSING"
		}
		if reason == "DAMAGED" || reason == "WRONG_ITEM" {
			claimType = claims.ClaimTypeDamaged
		}
		lines = append(lines, claims.ClaimLine{
			SKU:      strings.TrimSpace(it.SKU),
			Quantity: it.Quantity,
			Reason:   reason,
		})
	}
	orderLines := make([]claims.OrderLine, 0, len(o.LineItems))
	for _, li := range o.LineItems {
		orderLines = append(orderLines, claims.OrderLine{
			SKU:            li.SKU,
			Quantity:       li.Quantity,
			UnitPriceMinor: li.UnitPrice,
		})
	}
	snap := claims.OrderSnapshot{
		OrderID:            o.OrderID,
		SupplierID:         o.SupplierID,
		RetailerID:         o.RetailerID,
		WarehouseID:        o.WarehouseID,
		Currency:           o.Currency,
		Status:             string(o.Status),
		TotalMinor:         o.TotalMinor,
		OriginalTotalMinor: o.OriginalTotalMinor,
		LineItems:          orderLines,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}
	_, err := b.svc.CreateFromDriverException(ctx, snap, driverID, claimType, lines, photos, note)
	return err
}

func (b *driverClaimsBridge) GetRemainingClaimable(ctx context.Context, orderID string) (int64, int64, error) {
	if b == nil || b.svc == nil {
		return 0, 0, nil
	}
	res, err := b.svc.GetRemainingClaimable(ctx, orderID)
	if err != nil {
		return 0, 0, err
	}
	return res.RemainingClaimableMinor, res.DeliveredGrossMinor, nil
}

// returnsClaimsBridge adapts returns.Service to claims.ReverseLogisticsOpener.
type returnsClaimsBridge struct {
	svc *returns.Service
}

func (b *returnsClaimsBridge) OpenFromClaim(ctx context.Context, in claims.ReverseLogisticsInput) error {
	if b == nil || b.svc == nil {
		return nil
	}
	lines := make([]returns.TicketLine, 0, len(in.Lines))
	for _, li := range in.Lines {
		reason := strings.TrimSpace(li.Reason)
		if reason == "" {
			// Fall back to claim source type when line reason empty.
			reason = strings.TrimSpace(in.Source)
			if strings.EqualFold(reason, "RETAILER_CLAIM") || strings.EqualFold(reason, "CLAIM") {
				reason = "DAMAGED"
			}
		}
		lines = append(lines, returns.TicketLine{
			SKU:      strings.TrimSpace(li.SKU),
			Quantity: li.Quantity,
			Reason:   reason,
		})
	}
	_, err := b.svc.OpenTickets(ctx, returns.OpenTicketsInput{
		OrderID:     in.OrderID,
		WarehouseID: in.WarehouseID,
		SupplierID:  in.SupplierID,
		DriverID:    in.DriverID,
		ClaimID:     in.ClaimID,
		Source:      in.Source,
		Note:        in.Note,
		Lines:       lines,
	})
	return err
}
