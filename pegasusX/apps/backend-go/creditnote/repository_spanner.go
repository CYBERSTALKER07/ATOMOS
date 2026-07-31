package creditnote

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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

func (r *SpannerRepository) GetDeliveredOrderLines(ctx context.Context, orderId string) ([]CreditNoteLine, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT ol.OrderLineId, ol.Sku, ol.Quantity, fs.NetMinor, fs.VatMinor, fs.GrossMinor, fs.VatRateBps
			FROM OrderLines ol
			JOIN OrderLineFiscalSnapshots fs ON ol.OrderId = fs.OrderId AND ol.OrderLineId = fs.OrderLineId
			WHERE ol.OrderId = @orderId
		`,
		Params: map[string]interface{}{
			"orderId": orderId,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
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
		if err := row.Columns(&l.OrderLineId, &l.Sku, &l.Qty, &l.LineNetMinor, &l.LineVatMinor, &l.LineGrossMinor, &l.VatRateBps); err != nil {
			return nil, err
		}
		if l.Qty > 0 {
			l.UnitNetMinor = l.LineNetMinor / l.Qty
		}
		lines = append(lines, l)
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

func (r *SpannerRepository) SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask, eventType string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("creditnote repository unavailable")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m, err := spanner.InsertOrUpdateStruct("ReverseLogisticsTasks", task)
		if err != nil {
			return err
		}
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
	if err := row.Columns(&t.TaskId, &t.CreditNoteId, &t.OrderId, &t.Status, &wh, &driver, &t.ExpectedQtyJson, &t.ReceivedQtyJson, &t.CreatedAt, &t.UpdatedAt); err != nil {
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
	return &t, nil
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
		if err := row.Columns(&task.TaskId, &task.CreditNoteId, &task.OrderId, &task.Status, &wh, &driver, &task.ExpectedQtyJson, &task.ReceivedQtyJson, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return err
		}
		now := time.Now().UTC()
		task.Status = ReverseTaskStatusReceived
		task.WarehouseId = &warehouseID
		task.ReceivedQtyJson = receivedJSON
		task.UpdatedAt = now
		m, err := spanner.InsertOrUpdateStruct("ReverseLogisticsTasks", task)
		if err != nil {
			return err
		}
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
