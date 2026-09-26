package creditnote

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) SaveCreditNote(ctx context.Context, cn CreditNote, eventType string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("creditnote repository unavailable")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts, err := r.saveCreditNoteMutations(cn)
		if err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		if eventType != "" {
			payload := map[string]any{
				"type":              eventType,
				"credit_note_id":    cn.CreditNoteId,
				"order_id":          cn.OrderId,
				"status":            string(cn.Status),
				"total_gross_minor": cn.TotalGrossMinor,
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, cn.OrderId, events.TopicMain, payload); err != nil {
				return err
			}
		}
		muts = append(muts, outboxMutations(buf.events)...)
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) saveCreditNoteMutations(cn CreditNote) ([]*spanner.Mutation, error) {
	var muts []*spanner.Mutation

	cnRow := map[string]interface{}{
		"CreditNoteId":    cn.CreditNoteId,
		"OrderId":         cn.OrderId,
		"Type":            string(cn.Type),
		"Status":          string(cn.Status),
		"ReasonCode":      cn.ReasonCode,
		"ReasonText":      spanner.NullString{StringVal: derefStr(cn.ReasonText), Valid: cn.ReasonText != nil},
		"TotalNetMinor":   cn.TotalNetMinor,
		"TotalVatMinor":   cn.TotalVatMinor,
		"TotalGrossMinor": cn.TotalGrossMinor,
		"RegimeId":        spanner.NullString{StringVal: derefStr(cn.RegimeId), Valid: cn.RegimeId != nil},
		"OriginalEhfId":   spanner.NullString{StringVal: derefStr(cn.OriginalEhfId), Valid: cn.OriginalEhfId != nil},
		"CorrectiveEhfId": spanner.NullString{StringVal: derefStr(cn.CorrectiveEhfId), Valid: cn.CorrectiveEhfId != nil},
		"CreatedBy":       cn.CreatedBy,
		"CreatedAt":       cn.CreatedAt,
		"IssuedAt":        spanner.NullTime{Time: derefTime(cn.IssuedAt), Valid: cn.IssuedAt != nil},
		"CompletedAt":     spanner.NullTime{Time: derefTime(cn.CompletedAt), Valid: cn.CompletedAt != nil},
	}
	muts = append(muts, spanner.InsertOrUpdateMap("CreditNotes", cnRow))

	for _, line := range cn.Lines {
		m, err := spanner.InsertOrUpdateStruct("CreditNoteLines", line)
		if err != nil {
			return nil, err
		}
		muts = append(muts, m)
	}

	return muts, nil
}

func (r *SpannerRepository) GetCreditNote(ctx context.Context, id string) (*CreditNote, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("creditnote repository unavailable")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "CreditNotes", spanner.Key{id},
		[]string{"CreditNoteId", "OrderId", "Type", "Status", "ReasonCode", "ReasonText",
			"TotalNetMinor", "TotalVatMinor", "TotalGrossMinor", "CreatedBy", "CreatedAt", "IssuedAt", "CompletedAt"})
	if err != nil {
		if grpcstatus.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	cn, err := scanCreditNoteRow(row)
	if err != nil {
		return nil, err
	}
	lines, err := r.listLines(ctx, id)
	if err != nil {
		return nil, err
	}
	cn.Lines = lines
	return cn, nil
}

func (r *SpannerRepository) ListCreditNotesByOrder(ctx context.Context, orderId string) ([]CreditNote, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("creditnote repository unavailable")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT CreditNoteId, OrderId, Type, Status, ReasonCode, ReasonText,
		             TotalNetMinor, TotalVatMinor, TotalGrossMinor, CreatedBy, CreatedAt, IssuedAt, CompletedAt
		      FROM CreditNotes WHERE OrderId = @orderId ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"orderId": orderId},
	})
	defer iter.Stop()
	return collectCreditNotes(iter)
}

func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string, status CreditNoteStatus, limit int) ([]CreditNote, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("creditnote repository unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	params := map[string]interface{}{
		"supplierId": supplierID,
		"limit":      limit,
	}
	sql := `SELECT cn.CreditNoteId, cn.OrderId, cn.Type, cn.Status, cn.ReasonCode, cn.ReasonText,
	               cn.TotalNetMinor, cn.TotalVatMinor, cn.TotalGrossMinor, cn.CreatedBy, cn.CreatedAt, cn.IssuedAt, cn.CompletedAt
	        FROM CreditNotes cn
	        JOIN Orders o ON cn.OrderId = o.OrderId
	        WHERE o.SupplierId = @supplierId`
	if status != "" {
		sql += ` AND cn.Status = @status`
		params["status"] = string(status)
	}
	sql += ` ORDER BY cn.CreatedAt DESC LIMIT @limit`
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	return collectCreditNotes(iter)
}

func collectCreditNotes(iter *spanner.RowIterator) ([]CreditNote, error) {
	var out []CreditNote
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		cn, err := scanCreditNoteRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *cn)
	}
	return out, nil
}

func scanCreditNoteRow(row *spanner.Row) (*CreditNote, error) {
	var cn CreditNote
	var reasonText spanner.NullString
	var typ, status string
	var issuedAt, completedAt spanner.NullTime
	if err := row.Columns(
		&cn.CreditNoteId, &cn.OrderId, &typ, &status, &cn.ReasonCode, &reasonText,
		&cn.TotalNetMinor, &cn.TotalVatMinor, &cn.TotalGrossMinor, &cn.CreatedBy, &cn.CreatedAt, &issuedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	cn.Type = CreditNoteType(typ)
	cn.Status = CreditNoteStatus(status)
	if reasonText.Valid {
		v := reasonText.StringVal
		cn.ReasonText = &v
	}
	if issuedAt.Valid {
		t := issuedAt.Time
		cn.IssuedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		cn.CompletedAt = &t
	}
	return &cn, nil
}

func (r *SpannerRepository) listLines(ctx context.Context, creditNoteID string) ([]CreditNoteLine, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT CreditNoteId, LineId, OrderLineId, Sku, Qty, UnitNetMinor, VatRateBps,
		             LineNetMinor, LineVatMinor, LineGrossMinor
		      FROM CreditNoteLines WHERE CreditNoteId = @id`,
		Params: map[string]interface{}{"id": creditNoteID},
	})
	defer iter.Stop()
	var lines []CreditNoteLine
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var l CreditNoteLine
		if err := row.Columns(&l.CreditNoteId, &l.LineId, &l.OrderLineId, &l.Sku, &l.Qty,
			&l.UnitNetMinor, &l.VatRateBps, &l.LineNetMinor, &l.LineVatMinor, &l.LineGrossMinor); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, nil
}

func (r *SpannerRepository) OrderOwnedBySupplier(ctx context.Context, orderID, supplierID string) (bool, error) {
	orderID = strings.TrimSpace(orderID)
	supplierID = strings.TrimSpace(supplierID)
	if orderID == "" || supplierID == "" {
		return false, nil
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT 1 FROM Orders WHERE OrderId = @orderId AND SupplierId = @supplierId LIMIT 1`,
		Params: map[string]interface{}{"orderId": orderID, "supplierId": supplierID},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *SpannerRepository) GetDeliveredOrderLines(ctx context.Context, orderId string) ([]CreditNoteLine, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT LineItemsJson FROM Orders WHERE OrderId = @orderId`,
		Params: map[string]interface{}{"orderId": orderId},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lineItemsRaw []byte
	if err := row.ColumnByName("LineItemsJson", &lineItemsRaw); err != nil {
		return nil, err
	}
	var items []struct {
		SKU          string `json:"sku"`
		Quantity     int64  `json:"quantity"`
		UnitPrice    int64  `json:"unit_price_minor"`
		DeliveredQty int64  `json:"delivered_qty"`
	}
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &items); err != nil {
			return nil, fmt.Errorf("decode line items: %w", err)
		}
	}

	fiscalBySku := map[string]struct {
		TaxableMinor      int64
		VatMinor          int64
		TotalMinor        int64
		AppliedVatRateBps int64
	}{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				fiscalBySku = nil
			}
		}()
		fiscalIter := r.client.Single().Query(ctx, spanner.Statement{
			SQL: `
				SELECT OrderLineId, NetMinor, VatMinor, GrossMinor, VatRateBps
				FROM OrderLineFiscalSnapshots
				WHERE OrderId = @orderId
			`,
			Params: map[string]interface{}{"orderId": orderId},
		})
		defer fiscalIter.Stop()
		for {
			frow, err := fiscalIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fiscalBySku = nil
				return
			}
			var sku string
			var taxable, vat, total, rateBps int64
			if err := frow.Columns(&sku, &taxable, &vat, &total, &rateBps); err != nil {
				fiscalBySku = nil
				return
			}
			fiscalBySku[sku] = struct {
				TaxableMinor      int64
				VatMinor          int64
				TotalMinor        int64
				AppliedVatRateBps int64
			}{taxable, vat, total, rateBps}
		}
	}()

	var lines []CreditNoteLine
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" {
			continue
		}
		qty := item.Quantity
		if item.DeliveredQty > 0 {
			qty = item.DeliveredQty
		}
		if qty <= 0 {
			continue
		}
		line := CreditNoteLine{
			OrderLineId: sku,
			Sku:         sku,
			Qty:         qty,
		}
		if fiscalBySku != nil {
			if fs, ok := fiscalBySku[sku]; ok {
				line.LineNetMinor = fs.TaxableMinor
				line.LineVatMinor = fs.VatMinor
				line.LineGrossMinor = fs.TotalMinor
				line.VatRateBps = fs.AppliedVatRateBps
			} else {
				gross := item.UnitPrice * qty
				line.LineGrossMinor = gross
				line.LineNetMinor = gross
				line.LineVatMinor = 0
			}
		} else {
			gross := item.UnitPrice * qty
			line.LineGrossMinor = gross
			line.LineNetMinor = gross
			line.LineVatMinor = 0
		}
		if line.Qty > 0 {
			line.UnitNetMinor = line.LineNetMinor / line.Qty
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (r *SpannerRepository) GetClaimOrder(ctx context.Context, claimID string) (string, int64, bool, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId, AmountMinor FROM Claims WHERE ClaimId = @claimId`,
		Params: map[string]interface{}{"claimId": claimID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	var orderID string
	var amount int64
	if err := row.Columns(&orderID, &amount); err != nil {
		return "", 0, false, err
	}
	return orderID, amount, true, nil
}

func reverseLogisticsTaskMutation(task ReverseLogisticsTask) *spanner.Mutation {
	expectedNJ := spanner.NullJSON{Valid: false}
	if len(task.ExpectedQtyJson) > 0 {
		expectedNJ = spanner.NullJSON{Value: string(task.ExpectedQtyJson), Valid: true}
	}
	receivedNJ := spanner.NullJSON{Valid: false}
	if len(task.ReceivedQtyJson) > 0 {
		receivedNJ = spanner.NullJSON{Value: string(task.ReceivedQtyJson), Valid: true}
	}
	rowMap := map[string]interface{}{
		"TaskId":          task.TaskId,
		"CreditNoteId":    task.CreditNoteId,
		"OrderId":         task.OrderId,
		"Status":          string(task.Status),
		"WarehouseId":     task.WarehouseId,
		"DriverId":        task.DriverId,
		"ExpectedQtyJson": expectedNJ,
		"ReceivedQtyJson": receivedNJ,
		"CreatedAt":       task.CreatedAt,
		"UpdatedAt":       task.UpdatedAt,
	}
	return spanner.InsertOrUpdateMap("ReverseLogisticsTasks", rowMap)
}

func scanNullJSONBytes(nj spanner.NullJSON) []byte {
	if !nj.Valid || nj.Value == nil {
		return nil
	}
	switch v := nj.Value.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		b, _ := json.Marshal(v)
		return b
	}
}

func (r *SpannerRepository) SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask, eventType string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("creditnote repository unavailable")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := reverseLogisticsTaskMutation(task)
		buf := &spannerTxnBuffer{}
		if eventType != "" {
			payload := map[string]any{
				"type":           eventType,
				"task_id":        task.TaskId,
				"credit_note_id": task.CreditNoteId,
				"order_id":       task.OrderId,
				"status":         string(task.Status),
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, task.OrderId, events.TopicMain, payload); err != nil {
				return err
			}
		}
		muts := []*spanner.Mutation{m}
		muts = append(muts, outboxMutations(buf.events)...)
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) GetReverseLogisticsTask(ctx context.Context, taskID string) (*ReverseLogisticsTask, error) {
	row, err := r.client.Single().ReadRow(ctx, "ReverseLogisticsTasks", spanner.Key{taskID},
		[]string{"TaskId", "CreditNoteId", "OrderId", "Status", "WarehouseId", "DriverId", "ExpectedQtyJson", "ReceivedQtyJson", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if grpcstatus.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var t ReverseLogisticsTask
	var wh, driver spanner.NullString
	var expNJ, recNJ spanner.NullJSON
	if err := row.Columns(&t.TaskId, &t.CreditNoteId, &t.OrderId, &t.Status, &wh, &driver, &expNJ, &recNJ, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	if wh.Valid {
		v := wh.StringVal
		t.WarehouseId = &v
	}
	if driver.Valid {
		v := driver.StringVal
		t.DriverId = &v
	}
	t.ExpectedQtyJson = scanNullJSONBytes(expNJ)
	t.ReceivedQtyJson = scanNullJSONBytes(recNJ)
	return &t, nil
}

func (r *SpannerRepository) ListReverseLogisticsTasks(ctx context.Context, warehouseID, status string, limit int) ([]ReverseLogisticsTask, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("creditnote repository unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	params := map[string]interface{}{"limit": limit}
	sql := `SELECT TaskId, CreditNoteId, OrderId, Status, WarehouseId, DriverId, ExpectedQtyJson, ReceivedQtyJson, CreatedAt, UpdatedAt
	        FROM ReverseLogisticsTasks WHERE Status = @status`
	params["status"] = status
	if strings.TrimSpace(warehouseID) != "" {
		sql += ` AND (WarehouseId = @warehouseId OR WarehouseId IS NULL)`
		params["warehouseId"] = warehouseID
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @limit`
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []ReverseLogisticsTask
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var t ReverseLogisticsTask
		var wh, driver spanner.NullString
		var expNJ, recNJ spanner.NullJSON
		if err := row.Columns(&t.TaskId, &t.CreditNoteId, &t.OrderId, &t.Status, &wh, &driver, &expNJ, &recNJ, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		if wh.Valid {
			v := wh.StringVal
			t.WarehouseId = &v
		}
		if driver.Valid {
			v := driver.StringVal
			t.DriverId = &v
		}
		t.ExpectedQtyJson = scanNullJSONBytes(expNJ)
		t.ReceivedQtyJson = scanNullJSONBytes(recNJ)
		out = append(out, t)
	}
	return out, nil
}

func (r *SpannerRepository) ReceiveReverseLogisticsTask(ctx context.Context, taskID string, warehouseID string, receivedJSON []byte, actor string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ReverseLogisticsTasks", spanner.Key{taskID},
			[]string{"TaskId", "CreditNoteId", "OrderId", "Status", "WarehouseId", "DriverId", "ExpectedQtyJson", "ReceivedQtyJson", "CreatedAt", "UpdatedAt"})
		if err != nil {
			return fmt.Errorf("reverse logistics task not found")
		}
		var task ReverseLogisticsTask
		var wh, driver spanner.NullString
		var expNJ, recNJ spanner.NullJSON
		if err := row.Columns(&task.TaskId, &task.CreditNoteId, &task.OrderId, &task.Status, &wh, &driver, &expNJ, &recNJ, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return err
		}
		now := time.Now().UTC()
		task.Status = ReverseTaskStatusReceived
		task.WarehouseId = &warehouseID
		task.ReceivedQtyJson = receivedJSON
		task.UpdatedAt = now
		m := reverseLogisticsTaskMutation(task)
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":           EventReverseLogisticsReceived,
			"task_id":        task.TaskId,
			"credit_note_id": task.CreditNoteId,
			"order_id":       task.OrderId,
			"warehouse_id":   warehouseID,
			"actor":          actor,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, task.OrderId, events.TopicMain, payload); err != nil {
			return err
		}
		muts := []*spanner.Mutation{m}
		muts = append(muts, outboxMutations(buf.events)...)

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{task.OrderId}, []string{"SupplierId"})
		if err != nil {
			return fmt.Errorf("order lookup for restock: %w", err)
		}
		var supplierID string
		if err := orderRow.Columns(&supplierID); err != nil {
			return err
		}

		received := map[string]int64{}
		if len(receivedJSON) > 0 {
			_ = json.Unmarshal(receivedJSON, &received)
		}
		for sku, qty := range received {
			sku = strings.TrimSpace(sku)
			if sku == "" || qty <= 0 {
				continue
			}
			if stocklots.LotsEnabled() {
				if _, err := stocklots.CreditViaDefaultPutawayInTxn(
					ctx, txn, supplierID, warehouseID, sku,
					stocklots.DefaultReturnsLocationID, "RETURNS", qty,
				); err != nil {
					return err
				}
			} else if err := inventory.CreditSupplierInventoryV2InTxn(ctx, txn, supplierID, warehouseID, sku, qty); err != nil {
				return err
			}
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
