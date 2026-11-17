package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type perOrderFeeInfo struct {
	FeeMinor int64
	Currency string
}

// SetPerOrderFeeResolver sets an optional pre-txn resolver for per-order platform fees (testing / custom hooks).
func (s *Service) SetPerOrderFeeResolver(fn func(ctx context.Context, supplierID string) (feeMinor int64, currency string, err error)) {
	if s != nil {
		s.perOrderFeeFn = fn
	}
}

func (s *Service) resolvePerOrderFeeBeforeTxn(ctx context.Context, supplierID string) (*perOrderFeeInfo, error) {
	if s == nil || s.perOrderFeeFn == nil || strings.TrimSpace(supplierID) == "" {
		return nil, nil
	}
	feeMinor, cur, err := s.perOrderFeeFn(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if feeMinor <= 0 {
		return nil, nil
	}
	return &perOrderFeeInfo{FeeMinor: feeMinor, Currency: strings.ToUpper(strings.TrimSpace(cur))}, nil
}

func resolveActivePerOrderFeeInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, now time.Time) (*perOrderFeeInfo, error) {
	if txn == nil || strings.TrimSpace(supplierID) == "" {
		return nil, nil
	}
	find := func(where string, params map[string]any) (*perOrderFeeInfo, bool, error) {
		if params == nil {
			params = map[string]any{}
		}
		params["now"] = now.UTC()
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT PerOrderMinor, Currency
			      FROM BillingFeeSchedules
			      WHERE ` + where + ` AND EffectiveFrom <= @now AND (EffectiveTo IS NULL OR EffectiveTo > @now)
			      ORDER BY EffectiveFrom DESC LIMIT 1`,
			Params: params,
		})
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		var perOrder int64
		var cur string
		if err := row.Columns(&perOrder, &cur); err != nil {
			return nil, false, err
		}
		if perOrder <= 0 {
			return nil, true, nil
		}
		return &perOrderFeeInfo{FeeMinor: perOrder, Currency: strings.ToUpper(strings.TrimSpace(cur))}, true, nil
	}

	if info, found, err := find(`SupplierId = @id`, map[string]any{"id": supplierID}); err != nil {
		return nil, err
	} else if found {
		return info, nil
	}

	tier := "STANDARD"
	if row, err := txn.ReadRow(ctx, "SupplierProfiles", spanner.Key{supplierID}, []string{"Tier"}); err == nil {
		var t spanner.NullString
		if cErr := row.Column(0, &t); cErr == nil && strings.TrimSpace(t.StringVal) != "" {
			tier = strings.ToUpper(strings.TrimSpace(t.StringVal))
		}
	}
	if info, found, err := find(`SupplierId = '' AND Tier = @tier`, map[string]any{"tier": tier}); err != nil {
		return nil, err
	} else if found {
		return info, nil
	}
	if tier != "STANDARD" {
		if info, found, err := find(`SupplierId = '' AND Tier = 'STANDARD'`, nil); err != nil {
			return nil, err
		} else if found {
			return info, nil
		}
	}
	return nil, nil
}

func hasOrderDeliveredGLInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID, orderID string) (bool, error) {
	if txn == nil || strings.TrimSpace(entityID) == "" || strings.TrimSpace(orderID) == "" {
		return false, nil
	}
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntryId FROM JournalEntries
		      WHERE EntityId = @eid AND IdempotencyKey = @k
		      LIMIT 1`,
		Params: map[string]any{
			"eid": strings.TrimSpace(entityID),
			"k":   "gl:order_delivered:" + strings.TrimSpace(orderID),
		},
	})
	_, err := iter.Next()
	iter.Stop()
	if err == nil {
		return true, nil
	}
	if err == iterator.Done {
		return false, nil
	}
	return false, fmt.Errorf("check order delivered GL entry: %w", err)
}

func (s *Service) resolveOrderPaymentModeInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order) (paymentMode string, escrowPrePosted bool, err error) {
	escrowIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntryId FROM JournalEntries
		      WHERE EntityId = @eid AND IdempotencyKey IN UNNEST(@ks)
		      LIMIT 1`,
		Params: map[string]any{
			"eid": o.SupplierID,
			"ks": []string{
				"gl:split_tender:" + o.OrderID,
				"gl:shortfall:" + o.OrderID,
			},
		},
	})
	_, eErr := escrowIter.Next()
	escrowIter.Stop()
	if eErr == nil {
		return "", true, nil
	}
	if eErr != iterator.Done {
		return "", false, fmt.Errorf("check pre-escrow GL entry: %w", eErr)
	}

	legIter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT Method, Status, AmountMinor
		      FROM OrderPaymentLegs
		      WHERE OrderId = @oid`,
		Params: map[string]any{"oid": o.OrderID},
	})
	defer legIter.Stop()

	var capturedMode string
	var pendingMode string
	for {
		row, lErr := legIter.Next()
		if lErr == iterator.Done {
			break
		}
		if lErr != nil {
			return "", false, fmt.Errorf("query order payment legs for GL: %w", lErr)
		}
		var method, st string
		var amt int64
		if cErr := row.Columns(&method, &st, &amt); cErr != nil {
			return "", false, cErr
		}
		if amt <= 0 {
			continue
		}
		stUpper := strings.ToUpper(strings.TrimSpace(st))
		if stUpper == "FAILED" || stUpper == "CANCELLED" || stUpper == "REFUNDED" {
			continue
		}
		norm := normalizeOrderGLPaymentMode(method)
		if norm == "" {
			continue
		}
		if stUpper == string(PaymentStatusCaptured) {
			capturedMode = norm
		} else if pendingMode == "" {
			pendingMode = norm
		}
	}

	if capturedMode != "" {
		return capturedMode, false, nil
	}
	if pendingMode != "" {
		return pendingMode, false, nil
	}
	if o.FiscalReceiptUpdate != nil {
		if m := normalizeOrderGLPaymentMode(o.FiscalReceiptUpdate.PaymentMethod); m != "" {
			return m, false, nil
		}
	}
	if o.Status == StatusDeliveredOnCredit {
		return "CREDIT", false, nil
	}
	return "CASH", false, nil
}

func normalizeOrderGLPaymentMode(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "CASH", "FORCE":
		return "CASH"
	case "CARD", "PAYME", "CLICK", "ADYEN", "GLOBAL_PAY", "GLOBALPAY", "GP":
		return "CARD"
	case "WALLET":
		return "WALLET"
	case "CREDIT", "NASIYA":
		return "CREDIT"
	default:
		return ""
	}
}

func (s *Service) completeOrderTaxAndGLInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	orderRecord *Order,
	preFee *perOrderFeeInfo,
	explicitPaymentMode string,
) error {
	var netMinor, vatMinor, grossMinor int64
	if s.taxSvc != nil {
		var err error
		netMinor, vatMinor, grossMinor, err = s.stampTaxRegimeBreakdownTxn(ctx, txn, orderRecord)
		if err != nil {
			return err
		}
	}
	occurredAt := s.now().UTC()
	if orderRecord != nil && !orderRecord.UpdatedAt.IsZero() {
		occurredAt = orderRecord.UpdatedAt.UTC()
	}
	return s.recordOrderDeliveredGLInTxn(ctx, txn, outboxBuf, orderRecord, netMinor, vatMinor, grossMinor, preFee, explicitPaymentMode, occurredAt)
}

func (s *Service) recordOrderDeliveredGLInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	orderRecord *Order,
	netMinor int64,
	vatMinor int64,
	grossMinor int64,
	preFee *perOrderFeeInfo,
	explicitPaymentMode string,
	occurredAt time.Time,
) error {
	if txn == nil || orderRecord == nil || strings.TrimSpace(orderRecord.SupplierID) == "" {
		return nil
	}
	if grossMinor <= 0 {
		grossMinor = deliveredGrossFromOrder(*orderRecord)
	}
	if grossMinor <= 0 {
		return nil
	}
	if netMinor == 0 && vatMinor == 0 {
		vatMinor = orderRecord.VatTotalMinor()
		if vatMinor < 0 || vatMinor > grossMinor {
			vatMinor = 0
		}
		netMinor = grossMinor - vatMinor
	} else if netMinor+vatMinor != grossMinor {
		if vatMinor < 0 || vatMinor > grossMinor {
			vatMinor = 0
		}
		netMinor = grossMinor - vatMinor
	}

	cur := strings.ToUpper(strings.TrimSpace(orderRecord.Currency))
	if cur == "" {
		cur = "UZS"
	}
	if occurredAt.IsZero() {
		occurredAt = s.now().UTC()
	}

	feeInfo := preFee
	if feeInfo == nil && s.perOrderFeeFn == nil {
		var err error
		feeInfo, err = resolveActivePerOrderFeeInTxn(ctx, txn, orderRecord.SupplierID, occurredAt)
		if err != nil {
			return fmt.Errorf("resolve per-order fee for GL: %w", err)
		}
	}

	paymentMode := normalizeOrderGLPaymentMode(explicitPaymentMode)
	var splitEscrowPosted bool
	if paymentMode == "" {
		var err error
		paymentMode, splitEscrowPosted, err = s.resolveOrderPaymentModeInTxn(ctx, txn, orderRecord)
		if err != nil {
			return err
		}
	} else {
		var sErr error
		_, splitEscrowPosted, sErr = s.resolveOrderPaymentModeInTxn(ctx, txn, orderRecord)
		if sErr != nil {
			return sErr
		}
		if splitEscrowPosted {
			paymentMode = ""
		}
	}

	orderIdemKey := "gl:order_delivered:" + orderRecord.OrderID

	// Fast single-entry PostingRules path when no per-order fee and no prior split-tender escrow.
	if !splitEscrowPosted && (feeInfo == nil || feeInfo.FeeMinor <= 0) {
		_, err := gl.PostEventInTxn(ctx, txn, outboxBuf, gl.RuleEventOrderDelivered, gl.RuleEvalContext{
			EntityID:       orderRecord.SupplierID,
			SupplierID:     orderRecord.SupplierID,
			RetailerID:     orderRecord.RetailerID,
			DriverID:       orderRecord.DriverID,
			WarehouseID:    orderRecord.WarehouseID,
			OrderID:        orderRecord.OrderID,
			SourceID:       orderRecord.OrderID,
			Currency:       cur,
			PaymentMode:    paymentMode,
			NetMinor:       netMinor,
			VatMinor:       vatMinor,
			GrossMinor:     grossMinor,
			IdempotencyKey: orderIdemKey,
			Memo:           "Order delivered " + orderRecord.OrderID,
			OccurredAt:     occurredAt,
		})
		return err
	}

	inputs := make([]gl.JournalEntryInput, 0, 2)
	if splitEscrowPosted {
		postings := []gl.PostingInput{
			{
				AccountCode: gl.AccountPlatformEscrow,
				Direction:   gl.DirectionDebit,
				AmountMinor: grossMinor,
				Currency:    cur,
				PartyType:   "SUPPLIER",
				PartyID:     orderRecord.SupplierID,
				Description: "Release split-tender escrow on delivery " + orderRecord.OrderID,
			},
		}
		if netMinor > 0 {
			postings = append(postings, gl.PostingInput{
				AccountCode: gl.AccountSalesRevenue,
				Direction:   gl.DirectionCredit,
				AmountMinor: netMinor,
				Currency:    cur,
				Description: "Order delivered (split-tender) net sales revenue",
			})
		}
		if vatMinor > 0 {
			postings = append(postings, gl.PostingInput{
				AccountCode: gl.AccountVatOutput,
				Direction:   gl.DirectionCredit,
				AmountMinor: vatMinor,
				Currency:    cur,
				Description: "Order delivered (split-tender) output VAT",
			})
		}
		inputs = append(inputs, gl.JournalEntryInput{
			EntityID:       orderRecord.SupplierID,
			SourceType:     "ORDER_DELIVERED",
			SourceID:       orderRecord.OrderID,
			OrderID:        orderRecord.OrderID,
			Memo:           "Order delivered (split-tender escrow release) " + orderRecord.OrderID,
			Currency:       cur,
			IdempotencyKey: orderIdemKey,
			OccurredAt:     occurredAt,
			Postings:       postings,
		})
	} else {
		delivIn, err := gl.BuildOrderDeliveredInput(gl.RuleEvalContext{
			EntityID:       orderRecord.SupplierID,
			SupplierID:     orderRecord.SupplierID,
			RetailerID:     orderRecord.RetailerID,
			DriverID:       orderRecord.DriverID,
			WarehouseID:    orderRecord.WarehouseID,
			OrderID:        orderRecord.OrderID,
			SourceID:       orderRecord.OrderID,
			Currency:       cur,
			PaymentMode:    paymentMode,
			NetMinor:       netMinor,
			VatMinor:       vatMinor,
			GrossMinor:     grossMinor,
			IdempotencyKey: orderIdemKey,
			Memo:           "Order delivered " + orderRecord.OrderID,
			OccurredAt:     occurredAt,
		})
		if err != nil {
			return err
		}
		inputs = append(inputs, delivIn)
	}

	if feeInfo != nil && feeInfo.FeeMinor > 0 {
		feeCur := feeInfo.Currency
		if feeCur == "" {
			feeCur = cur
		}
		feeIn, err := gl.BuildPerOrderFeeInput(gl.RuleEvalContext{
			EntityID:       orderRecord.SupplierID,
			SupplierID:     orderRecord.SupplierID,
			OrderID:        orderRecord.OrderID,
			SourceID:       orderRecord.OrderID,
			Currency:       feeCur,
			FeeMinor:       feeInfo.FeeMinor,
			IdempotencyKey: "gl:fee_per_order:" + orderRecord.OrderID,
			Memo:           "Per-order platform billing fee " + orderRecord.OrderID,
			OccurredAt:     occurredAt,
		})
		if err != nil {
			return err
		}
		inputs = append(inputs, feeIn)
	}

	_, err := gl.BufferJournalEntriesBatchInTxn(ctx, txn, outboxBuf, inputs)
	return err
}

func recordSplitTenderGLInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	orderRecord Order,
	tenders []gl.TenderComponent,
	occurredAt time.Time,
) error {
	if txn == nil || strings.TrimSpace(orderRecord.SupplierID) == "" || len(tenders) == 0 {
		return nil
	}
	var grossMinor int64
	for _, t := range tenders {
		if t.AmountMinor > 0 {
			grossMinor += t.AmountMinor
		}
	}
	if grossMinor <= 0 {
		return nil
	}
	cur := strings.ToUpper(strings.TrimSpace(orderRecord.Currency))
	if cur == "" {
		cur = "UZS"
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	_, err := gl.PostEventInTxn(ctx, txn, outboxBuf, gl.RuleEventSplitTender, gl.RuleEvalContext{
		EntityID:       orderRecord.SupplierID,
		SupplierID:     orderRecord.SupplierID,
		RetailerID:     orderRecord.RetailerID,
		DriverID:       orderRecord.DriverID,
		OrderID:        orderRecord.OrderID,
		SourceID:       orderRecord.OrderID,
		Currency:       cur,
		GrossMinor:     grossMinor,
		Tenders:        tenders,
		IdempotencyKey: "gl:split_tender:" + orderRecord.OrderID,
		Memo:           "Split tender payment " + orderRecord.OrderID,
		OccurredAt:     occurredAt,
	})
	return err
}
