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
			        AND CreatedAt < @end AND (CreatedAt >= @start OR NetPayoutMinor < 0 OR GrossMinor < 0)`,
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

// GenerateBatchesInTxn computes sums and updates slices atomically in a single transaction.
func (r *Repository) GenerateBatchesInTxn(ctx context.Context, supplierID string, start, end time.Time, buildBatch func(SumLegsResult) (Batch, error)) ([]Batch, error) {
	var batches []Batch
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		batches = nil // Reset on retry

		sums, err := readUnsettledSettlementSlicesInTxn(ctx, txn, supplierID, start, end)
		if err != nil {
			return err
		}

		// Fallback to OrderPaymentLegs JOIN Orders when no settlement slices
		// were recorded for this supplier/period (e.g. legacy legs or direct leg fixtures).
		if len(sums) == 0 {
			sums, err = readFallbackCapturedLegsInTxn(ctx, txn, supplierID, start, end)
			if err != nil {
				return err
			}
		}

		buf := outbox.NewSpannerTxnBuffer(txn)

		for _, s := range sums {
			if s.NetPayout <= 0 {
				continue
			}
			b, err := buildBatch(*s)
			if err != nil {
				if err == ErrNothingPayable {
					continue
				}
				return err
			}

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

			if len(s.SliceIDs) > 0 {
				stmt := spanner.Statement{
					SQL: `UPDATE InvoiceSettlementSlices
					      SET Status = 'BATCHED', PayoutBatchId = @b
					      WHERE SliceId IN UNNEST(@slice_ids)`,
					Params: map[string]any{
						"b":         b.BatchID,
						"slice_ids": s.SliceIDs,
					},
				}
				if _, err := txn.Update(ctx, stmt); err != nil {
					return err
				}
			}

			batches = append(batches, b)
		}

		return buf.Flush(ctx)
	})

	if spanner.ErrCode(err) == codes.AlreadyExists {
		if existingList, err2 := r.ListBySupplierPeriod(ctx, supplierID, start, end); err2 == nil && len(existingList) > 0 {
			return existingList, nil
		}
	}

	return batches, err
}

func readUnsettledSettlementSlicesInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID string,
	start, end time.Time,
) (map[string]*SumLegsResult, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT Currency, GrossMinor, CommissionMinor, NetPayoutMinor, SliceId
		      FROM InvoiceSettlementSlices
		      WHERE SupplierId = @sid AND Status = 'UNSETTLED'
		        AND CreatedAt < @end AND (CreatedAt >= @start OR NetPayoutMinor < 0 OR GrossMinor < 0)`,
		Params: map[string]any{"sid": supplierID, "start": start, "end": end},
	})
	defer iter.Stop()

	sums := make(map[string]*SumLegsResult)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var cur, sliceID string
		var gross, comm, net int64
		if err := row.Columns(&cur, &gross, &comm, &net, &sliceID); err != nil {
			return nil, err
		}
		if sums[cur] == nil {
			sums[cur] = &SumLegsResult{Currency: cur}
		}
		if gross >= 0 {
			sums[cur].Captured += gross
		} else {
			sums[cur].Refunded += -gross
		}
		sums[cur].Commission += comm
		sums[cur].NetPayout += net
		sums[cur].SliceIDs = append(sums[cur].SliceIDs, sliceID)
	}
	return sums, nil
}

func readFallbackCapturedLegsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID string,
	start, end time.Time,
) (map[string]*SumLegsResult, error) {
	legIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT l.Method, l.AmountMinor, o.Currency
		      FROM OrderPaymentLegs l
		      JOIN Orders o ON o.OrderId = l.OrderId
		      WHERE o.SupplierId = @sid AND l.Status = 'CAPTURED'
		        AND l.CapturedAt >= @start AND l.CapturedAt < @end`,
		Params: map[string]any{"sid": supplierID, "start": start, "end": end},
	})
	defer legIter.Stop()

	sums := make(map[string]*SumLegsResult)
	for {
		row, err := legIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var method, cur string
		var amount int64
		if err := row.Columns(&method, &amount, &cur); err != nil {
			return nil, err
		}
		if sums[cur] == nil {
			sums[cur] = &SumLegsResult{Currency: cur}
		}
		if method == "REFUND" {
			sums[cur].Refunded += amount
			sums[cur].NetPayout -= amount
		} else {
			sums[cur].Captured += amount
			sums[cur].NetPayout += amount
		}
	}
	return sums, nil
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
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
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

// ClaimDispatchInTxn atomically transitions an eligible batch from DRAFT/EXPORTED to
// SUBMITTING before external rail submission, returning the previous status for rollback on failure.
func (r *Repository) ClaimDispatchInTxn(ctx context.Context, supplierID, batchID string) (prevStatus string, b Batch, err error) {
	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(ctx, "PayoutBatches", spanner.Key{batchID}, []string{
			"SupplierId", "Status", "GrossCapturedMinor", "RefundedMinor", "CommissionMinor", "NetPayoutMinor", "Currency",
		})
		if rerr != nil {
			if spanner.ErrCode(rerr) == codes.NotFound {
				return ErrBatchNotFound
			}
			return rerr
		}
		if cerr := row.Columns(&b.SupplierID, &prevStatus, &b.GrossCapturedMinor, &b.RefundedMinor, &b.CommissionMinor, &b.NetPayoutMinor, &b.Currency); cerr != nil {
			return cerr
		}
		if b.SupplierID != supplierID {
			return ErrBatchNotFound
		}
		if prevStatus != StatusDraft && prevStatus != StatusExported {
			return fmt.Errorf("%w: batch %s not dispatchable from status %s", ErrBatchConflict, batchID, prevStatus)
		}
		b.BatchID = batchID
		b.Status = StatusSubmitting
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PayoutBatches", map[string]any{
				"BatchId":   batchID,
				"Status":    StatusSubmitting,
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return prevStatus, b, err
}

// RevertDispatchInTxn restores prevStatus when a claimed external live rail call fails.
func (r *Repository) RevertDispatchInTxn(ctx context.Context, batchID, prevStatus string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rerr := txn.ReadRow(ctx, "PayoutBatches", spanner.Key{batchID}, []string{"Status"})
		if rerr != nil {
			return rerr
		}
		var cur string
		if err := row.Columns(&cur); err != nil {
			return err
		}
		if cur != StatusSubmitting {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PayoutBatches", map[string]any{
				"BatchId":   batchID,
				"Status":    prevStatus,
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
}

// UpdateStatusRef updates status, optional export URI, and optional rail
// reference (set when a live rail dispatches or confirms settlement). Enforces valid
// monotonic status transitions atomically inside Spanner and emits matching outbox events.
func (r *Repository) UpdateStatusRef(ctx context.Context, batchID, status, exportURI, railRef string) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PayoutBatches", spanner.Key{batchID}, []string{
			"SupplierId", "Status", "GrossCapturedMinor", "RefundedMinor", "CommissionMinor", "NetPayoutMinor", "Currency",
		})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrBatchNotFound
			}
			return err
		}
		var b Batch
		var curStatus string
		if err := row.Columns(&b.SupplierID, &curStatus, &b.GrossCapturedMinor, &b.RefundedMinor, &b.CommissionMinor, &b.NetPayoutMinor, &b.Currency); err != nil {
			return err
		}
		if curStatus == StatusPaid && status == StatusPaid {
			return nil // idempotent settlement confirmation replay
		}
		if !isValidBatchTransition(curStatus, status) {
			return fmt.Errorf("%w: invalid batch status transition %s -> %s", ErrBatchConflict, curStatus, status)
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
		if err := applySettlementJournalInTxn(txn, batchID, status, b.SupplierID, b.Currency, b.GrossCapturedMinor, b.RefundedMinor, b.CommissionMinor); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("PayoutBatches", m)}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

func isValidBatchTransition(from, to string) bool {
	switch to {
	case StatusExported:
		return from == StatusDraft || from == StatusExported || from == StatusSubmitting
	case StatusSubmitted:
		return from == StatusSubmitting || from == StatusDraft || from == StatusExported
	case StatusPaid:
		return from == StatusExported || from == StatusSubmitted
	default:
		return false
	}
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

func applySettlementJournalInTxn(txn *spanner.ReadWriteTransaction, batchID, status, supplierID, currency string, grossCapturedMinor, refundedMinor, commissionMinor int64) error {
	if status != StatusPaid {
		return nil
	}
	gross := grossCapturedMinor - refundedMinor
	if gross <= 0 {
		return nil
	}
	if SettlementJournal == nil {
		return fmt.Errorf("payout: settlement journal hook required for paid batch %s", batchID)
	}
	return SettlementJournal(txn, batchID, supplierID, currency, gross, commissionMinor)
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
