package payout

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

func (r *Repository) Insert(ctx context.Context, b Batch) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := emitPayoutEvent(ctx, buf, events.EventPayoutBatchGenerated, b, ""); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("PayoutBatches", map[string]any{
				"BatchId":            b.BatchID,
				"SupplierId":         b.SupplierID,
				"PeriodStart":        civilDateOf(b.PeriodStart),
				"PeriodEnd":          civilDateOf(b.PeriodEnd),
				"GrossCapturedMinor": b.GrossCapturedMinor,
				"RefundedMinor":      b.RefundedMinor,
				"CommissionMinor":    b.CommissionMinor,
				"NetPayoutMinor":     b.NetPayoutMinor,
				"Currency":           b.Currency,
				"Status":             b.Status,
				"IdempotencyKey":     b.IdempotencyKey,
				"CreatedBy":          b.CreatedBy,
				"CreatedAt":          spanner.CommitTimestamp,
				"UpdatedAt":          spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}

		// Phase 2.2: Link and lock the immutable settlement slices for this batch
		stmt := spanner.Statement{
			SQL: `UPDATE InvoiceSettlementSlices
			      SET Status = 'BATCHED', PayoutBatchId = @b
			      WHERE SupplierId = @s AND Currency = @c AND Status = 'UNSETTLED'
			        AND CreatedAt >= @start AND CreatedAt < @end`,
			Params: map[string]any{
				"b":     b.BatchID,
				"s":     b.SupplierID,
				"c":     b.Currency,
				"start": b.PeriodStart,
				"end":   b.PeriodEnd,
			},
		}
		_, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}

		return buf.Flush(ctx)
	})
}

// emitPayoutEvent buffers a payout lifecycle event. The event type is chosen by
// the caller so the same status-transition code path can label generate /
// export / dispatch / paid distinctly.
func emitPayoutEvent(ctx context.Context, buf *outbox.SpannerTxnBuffer, eventType string, b Batch, railRef string) error {
	return outbox.EmitJSON(ctx, buf, events.AggregatePayoutBatch, b.BatchID, events.TopicMain, events.PayoutBatchEvent{
		BaseEvent:      events.BaseEvent{Type: eventType, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
		BatchID:        b.BatchID,
		SupplierID:     b.SupplierID,
		Status:         b.Status,
		NetPayoutMinor: b.NetPayoutMinor,
		Currency:       b.Currency,
		RailReference:  railRef,
	})
}

func (r *Repository) Get(ctx context.Context, batchID string) (Batch, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PayoutBatches", spanner.Key{batchID}, batchColumns())
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Batch{}, false, nil
		}
		return Batch{}, false, err
	}
	b, err := scanBatch(row)
	return b, err == nil, err
}

func (r *Repository) ListBySupplierPeriod(ctx context.Context, supplierID string, start, end time.Time) ([]Batch, error) {
	rows := []Batch{}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ` + batchColumnList() + ` FROM PayoutBatches WHERE SupplierId = @sid AND PeriodStart = @s AND PeriodEnd = @e`,
		Params: map[string]any{"sid": supplierID, "s": civilDateOf(start), "e": civilDateOf(end)},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		b, err := scanBatch(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, b)
	}
	return rows, nil
}

func civilDateOf(t time.Time) civil.Date {
	return civil.DateOf(t.UTC())
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (Batch, bool, error) {
	return r.findOne(ctx, `SELECT `+batchColumnList()+` FROM PayoutBatches WHERE IdempotencyKey = @k`,
		map[string]any{"k": key})
}

func (r *Repository) ListBySupplier(ctx context.Context, supplierID string) ([]Batch, error) {
	rows := []Batch{}
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("payout repository unavailable")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ` + batchColumnList() + ` FROM PayoutBatches
		      WHERE SupplierId = @sid
		      ORDER BY CreatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		b, err := scanBatch(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, b)
	}
}

func (r *Repository) findOne(ctx context.Context, sql string, params map[string]any) (Batch, bool, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Batch{}, false, nil
	}
	if err != nil {
		return Batch{}, false, err
	}
	b, err := scanBatch(row)
	return b, err == nil, err
}

func (r *Repository) UpdateStatus(ctx context.Context, batchID, status, exportURI string) error {
	return r.UpdateStatusRef(ctx, batchID, status, exportURI, "")
}

// UpdateStatusRef updates status, optional export URI, and optional rail
// reference (set when a live rail dispatches or confirms settlement). Emits the
// matching payout lifecycle outbox event in the same transaction so partner
// webhooks / search / notifications see the state change atomically.
func (r *Repository) UpdateStatusRef(ctx context.Context, batchID, status, exportURI, railRef string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PayoutBatches", spanner.Key{batchID}, []string{"SupplierId", "NetPayoutMinor", "Currency"})
		if err != nil {
			return err
		}
		var b Batch
		if err := row.Columns(&b.SupplierID, &b.NetPayoutMinor, &b.Currency); err != nil {
			return err
		}
		b.BatchID = batchID
		b.Status = status

		eventType := ""
		switch status {
		case StatusExported:
			eventType = events.EventPayoutBatchExported
		case StatusSubmitted:
			eventType = events.EventPayoutBatchDispatched
		case StatusPaid:
			eventType = events.EventPayoutBatchPaid
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if eventType != "" {
			if err := emitPayoutEvent(ctx, buf, eventType, b, railRef); err != nil {
				return err
			}
		}

		m := map[string]any{
			"BatchId":   batchID,
			"Status":    status,
			"UpdatedAt": spanner.CommitTimestamp,
		}
		if strings.TrimSpace(exportURI) != "" {
			m["ExportFileUri"] = spanner.NullString{StringVal: exportURI, Valid: true}
		}
		if strings.TrimSpace(railRef) != "" {
			m["RailReference"] = spanner.NullString{StringVal: railRef, Valid: true}
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PayoutBatches", m)}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

// SupplierBankDetails holds the beneficiary account for the bank file.
type SupplierBankDetails struct {
	SupplierID    string
	LegalName     string
	BankName      string
	AccountHolder string
	AccountNumber string
	SwiftBic      string
	IBAN          string
}

func (r *Repository) SupplierBankDetails(ctx context.Context, supplierID string) (SupplierBankDetails, error) {
	row, err := r.client.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{supplierID},
		[]string{"SupplierId", "ContactName", "BankName", "AccountHolder", "AccountNumber", "SwiftBic", "IBAN"})
	if err != nil {
		return SupplierBankDetails{}, err
	}
	var d SupplierBankDetails
	var name, bank, holder, account, swift, iban spanner.NullString
	if err := row.Columns(&d.SupplierID, &name, &bank, &holder, &account, &swift, &iban); err != nil {
		return SupplierBankDetails{}, err
	}
	d.LegalName, d.BankName, d.AccountHolder = name.StringVal, bank.StringVal, holder.StringVal
	d.AccountNumber, d.SwiftBic, d.IBAN = account.StringVal, swift.StringVal, iban.StringVal
	return d, nil
}

func batchColumns() []string {
	return []string{"BatchId", "SupplierId", "PeriodStart", "PeriodEnd", "GrossCapturedMinor",
		"RefundedMinor", "CommissionMinor", "NetPayoutMinor", "Currency", "Status",
		"ExportFileUri", "RailReference", "IdempotencyKey", "CreatedBy", "CreatedAt", "UpdatedAt"}
}

func batchColumnList() string { return strings.Join(batchColumns(), ", ") }

func scanBatch(row *spanner.Row) (Batch, error) {
	var b Batch
	var exportURI, railRef spanner.NullString
	var start, end civil.Date
	err := row.Columns(&b.BatchID, &b.SupplierID, &start, &end, &b.GrossCapturedMinor,
		&b.RefundedMinor, &b.CommissionMinor, &b.NetPayoutMinor, &b.Currency, &b.Status,
		&exportURI, &railRef, &b.IdempotencyKey, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return Batch{}, err
	}
	b.PeriodStart = time.Date(start.Year, start.Month, start.Day, 0, 0, 0, 0, time.UTC)
	b.PeriodEnd = time.Date(end.Year, end.Month, end.Day, 0, 0, 0, 0, time.UTC)
	b.ExportFileURI = exportURI.StringVal
	b.RailReference = railRef.StringVal
	return b, nil
}

// RenderBankFile renders the CSV payment instruction for the batch. Fails
// closed when beneficiary details are incomplete — a payout file with a
// missing account number must never reach the bank.
func RenderBankFile(b Batch, d SupplierBankDetails) ([]byte, error) {
	var missing []string
	if d.AccountNumber == "" && d.IBAN == "" {
		missing = append(missing, "account_number/iban")
	}
	if d.BankName == "" {
		missing = append(missing, "bank_name")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrBankDetailsMissing, strings.Join(missing, ", "))
	}
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	if err := w.Write([]string{"batch_id", "supplier_id", "period_start", "period_end", "beneficiary", "account_number", "iban", "bank_name", "swift_bic", "amount_minor", "currency", "narrative"}); err != nil {
		return nil, err
	}
	narrative := fmt.Sprintf("PegasusX payout %s %s..%s (gross=%d refunds=%d commission=%d)",
		b.BatchID, b.PeriodStart.Format("2006-01-02"), b.PeriodEnd.Format("2006-01-02"),
		b.GrossCapturedMinor, b.RefundedMinor, b.CommissionMinor)
	if err := w.Write([]string{
		b.BatchID, b.SupplierID,
		b.PeriodStart.Format("2006-01-02"), b.PeriodEnd.Format("2006-01-02"),
		d.AccountHolder, d.AccountNumber, d.IBAN, d.BankName, d.SwiftBic,
		strconv.FormatInt(b.NetPayoutMinor, 10), b.Currency, narrative,
	}); err != nil {
		return nil, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}
