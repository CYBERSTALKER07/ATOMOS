package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func parseOrderLineFromFields(ctx context.Context, tx ApplyTx, fields map[string]string, requireProduct bool) (order.LineItem, bool, error) {
	sku := strings.TrimSpace(fields["sku"])
	if sku == "" && strings.TrimSpace(fields["quantity"]) == "" && strings.TrimSpace(fields["unit_price"]) == "" {
		return order.LineItem{}, false, nil
	}

	var prod StoredProduct
	var found bool
	if sku != "" {
		mappedProdID, _, err := tx.LookupIdentity(ctx, PackProductsPrices, sku)
		if err != nil {
			return order.LineItem{}, false, err
		}
		p, ok, err := tx.FindProduct(ctx, mappedProdID, sku, sku, sku)
		if err != nil {
			return order.LineItem{}, false, err
		}
		prod, found = p, ok
	}
	if requireProduct && !found {
		return order.LineItem{}, false, fmt.Errorf("%w: open order SKU %q not found", ErrInvalidInput, sku)
	}

	qty := int64(1)
	if rawQty := strings.TrimSpace(fields["quantity"]); rawQty != "" {
		parsedQty, err := ParseQuantityInt(rawQty)
		if err != nil || parsedQty <= 0 {
			return order.LineItem{}, false, fmt.Errorf("%w: invalid order quantity %q", ErrInvalidInput, rawQty)
		}
		qty = parsedQty
	}

	unitPriceMinor := prod.PriceMinor
	if rawPrice := strings.TrimSpace(fields["unit_price"]); rawPrice != "" {
		parsedPrice, err := ParseMoneyMinor(rawPrice)
		if err != nil || parsedPrice < 0 {
			return order.LineItem{}, false, fmt.Errorf("%w: invalid unit_price %q", ErrInvalidInput, rawPrice)
		}
		unitPriceMinor = parsedPrice
	}

	if unitPriceMinor == 0 {
		if rawTotal := strings.TrimSpace(fields["total_amount"]); rawTotal != "" {
			if parsedTotal, err := ParseMoneyMinor(rawTotal); err == nil && parsedTotal >= 0 && qty > 0 {
				unitPriceMinor = parsedTotal / qty
			}
		}
	}

	lineSKU := firstNonEmpty(prod.ProductID, sku)
	prodName := firstNonEmpty(prod.Name, sku, "Migrated Item")
	return order.LineItem{
		SKU:           lineSKU,
		Name:          prodName,
		Quantity:      qty,
		UnitPrice:     unitPriceMinor,
		HandlingClass: prod.HandlingClass,
		IsPerishable:  prod.IsPerishable,
	}, true, nil
}

func applyHistoricalOrderRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, _ int64, fields map[string]string) error {
	extNo := strings.TrimSpace(fields["external_order_no"])
	if extNo == "" {
		return fmt.Errorf("%w: external_order_no is required", ErrInvalidInput)
	}
	orderDate, err := order.ParseMigrationTimestamp(fields["order_date"])
	if err != nil {
		return fmt.Errorf("%w: invalid order_date %q", ErrInvalidInput, fields["order_date"])
	}
	if orderDate.IsZero() {
		return fmt.Errorf("%w: order_date is required for historical order %s", ErrInvalidInput, extNo)
	}

	rawStatus := firstNonEmpty(fields["status"], "COMPLETED")
	if _, _, err := order.NormalizeHistoricalOrderStatus(rawStatus); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	lineItem, hasLine, err := parseOrderLineFromFields(ctx, tx, fields, false)
	if err != nil {
		return err
	}

	totalMinor := int64(0)
	if rawTotal := strings.TrimSpace(fields["total_amount"]); rawTotal != "" {
		parsedTotal, parseErr := ParseMoneyMinor(rawTotal)
		if parseErr != nil || parsedTotal < 0 {
			return fmt.Errorf("%w: invalid total_amount %q", ErrInvalidInput, rawTotal)
		}
		totalMinor = parsedTotal
	} else if hasLine {
		totalMinor = lineItem.Quantity * lineItem.UnitPrice
	} else {
		return fmt.Errorf("%w: total_amount is required for historical order %s", ErrInvalidInput, extNo)
	}

	_, supCur, err := tx.ReadSupplierMeta(ctx)
	if err != nil {
		return err
	}
	cur := strings.ToUpper(strings.TrimSpace(fields["currency"]))
	if cur == "" {
		cur = firstNonEmpty(supCur, "UZS")
	}

	inn := NormalizeINN(fields["customer_inn"])
	phone := NormalizePhoneUZ(fields["customer_phone"])
	custName := strings.TrimSpace(fields["customer_name"])
	retailerID := "retailer_anonymous_pos"
	if inn != "" || phone != "" || custName != "" {
		cust, ok, cErr := tx.ResolveCustomerForFinancials(ctx, rctx.HomeCell, "", inn, phone, custName)
		if cErr != nil {
			return cErr
		}
		if !ok {
			return fmt.Errorf("%w: customer not found for historical order %s (name=%q inn=%q phone=%q)", ErrInvalidInput, extNo, custName, inn, phone)
		}
		retailerID = cust.RetailerID
	}

	var warehouseID string
	if whHint := strings.TrimSpace(fields["warehouse_id"]); whHint != "" || rctx.DefaultWarehouseID != "" {
		if wh, ok, wErr := tx.ResolveWarehouseForOrder(ctx, whHint, rctx.DefaultWarehouseID); wErr != nil {
			return wErr
		} else if ok {
			warehouseID = wh.WarehouseID
		}
	}

	var lines []order.LineItem
	if hasLine {
		lines = []order.LineItem{lineItem}
	}
	created, updated, orderID, err := tx.ImportHistoricalOrder(ctx, order.HistoricalOrderRequest{
		SessionID:         rctx.Session.SessionID,
		SupplierID:        rctx.Session.SupplierID,
		RetailerID:        retailerID,
		WarehouseID:       warehouseID,
		ExternalOrderNo:   extNo,
		ExternalFiscalRef: strings.TrimSpace(fields["external_fiscal_ref"]),
		RawStatus:         rawStatus,
		Currency:          cur,
		TotalMinor:        totalMinor,
		LineItems:         lines,
		CreatedAt:         orderDate,
		Now:               tx.Now(),
	})
	if err != nil {
		return err
	}
	if err := tx.PutIdentity(ctx, PackHistoricalOrders, extNo, orderID); err != nil {
		return err
	}
	switch {
	case created:
		rctx.Result.Created++
	case updated:
		rctx.Result.Updated++
	default:
		rctx.Result.Skipped++
	}
	return nil
}

func applyOpenOrderRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, _ int64, fields map[string]string) error {
	extNo := strings.TrimSpace(fields["external_order_no"])
	if extNo == "" {
		return fmt.Errorf("%w: external_order_no is required", ErrInvalidInput)
	}
	deliveryDate, err := order.ParseMigrationTimestamp(fields["delivery_date"])
	if err != nil || deliveryDate.IsZero() {
		return fmt.Errorf("%w: invalid delivery_date %q", ErrInvalidInput, fields["delivery_date"])
	}

	wh, ok, err := tx.ResolveWarehouseForOrder(ctx, fields["warehouse_id"], rctx.DefaultWarehouseID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: warehouse %q could not be resolved for open order %s", ErrInvalidInput, fields["warehouse_id"], extNo)
	}

	inn := NormalizeINN(fields["customer_inn"])
	phone := NormalizePhoneUZ(fields["customer_phone"])
	custName := strings.TrimSpace(fields["customer_name"])
	cust, ok, err := tx.ResolveCustomerForFinancials(ctx, rctx.HomeCell, "", inn, phone, custName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: customer not found for open order %s (name=%q inn=%q phone=%q)", ErrInvalidInput, extNo, custName, inn, phone)
	}

	lineItem, hasLine, err := parseOrderLineFromFields(ctx, tx, fields, true)
	if err != nil {
		return err
	}
	if !hasLine {
		return fmt.Errorf("%w: sku and quantity are required for open order %s", ErrInvalidInput, extNo)
	}
	if rawQty := strings.TrimSpace(fields["quantity"]); rawQty == "" {
		return fmt.Errorf("%w: quantity is required for open order %s", ErrInvalidInput, extNo)
	}

	_, supCur, err := tx.ReadSupplierMeta(ctx)
	if err != nil {
		return err
	}
	cur := strings.ToUpper(strings.TrimSpace(fields["currency"]))
	if cur == "" {
		cur = firstNonEmpty(supCur, "UZS")
	}

	totalMinor := lineItem.Quantity * lineItem.UnitPrice
	if rawTotal := strings.TrimSpace(fields["total_amount"]); rawTotal != "" {
		parsedTotal, parseErr := ParseMoneyMinor(rawTotal)
		if parseErr != nil || parsedTotal < 0 {
			return fmt.Errorf("%w: invalid total_amount %q", ErrInvalidInput, rawTotal)
		}
		if parsedTotal > 0 {
			totalMinor = parsedTotal
		}
	}

	intendedStatus := order.StatusPending
	if strings.EqualFold(strings.TrimSpace(fields["status"]), string(order.StatusScheduled)) {
		intendedStatus = order.StatusScheduled
	}

	reqDel := deliveryDate.UTC()
	var lines []order.LineItem = []order.LineItem{lineItem}
	created, updated, orderID, err := tx.ImportMigratedOpenOrder(ctx, order.MigratedOpenOrderRequest{
		SessionID:             rctx.Session.SessionID,
		SupplierID:            rctx.Session.SupplierID,
		RetailerID:            cust.RetailerID,
		WarehouseID:           wh.WarehouseID,
		ExternalOrderNo:       extNo,
		ExternalFiscalRef:     strings.TrimSpace(fields["external_fiscal_ref"]),
		IntendedLiveStatus:    intendedStatus,
		Currency:              cur,
		TotalMinor:            totalMinor,
		LineItems:             lines,
		RequestedDeliveryDate: &reqDel,
		CreatedAt:             minTime(tx.Now(), reqDel),
		Now:                   tx.Now(),
		Notes:                 strings.TrimSpace(fields["delivery_address"]),
	})
	if err != nil {
		return err
	}
	if err := tx.PutIdentity(ctx, PackOpenOrders, extNo, orderID); err != nil {
		return err
	}
	switch {
	case created:
		rctx.Result.Created++
	case updated:
		rctx.Result.Updated++
	default:
		rctx.Result.Skipped++
	}
	return nil
}

func minTime(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.Before(b) {
		return a
	}
	return b
}
