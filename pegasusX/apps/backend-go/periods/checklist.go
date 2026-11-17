package periods

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func defaultChecklistZeroItems() []ChecklistItem {
	return []ChecklistItem{
		{Key: "deliveries_fiscal_receipts", Label: "Delivered orders & fiscal receipts finalized", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 pending receipts or unfiscalized deliveries"},
		{Key: "esf_documents", Label: "Electronic invoices (ESF) accepted", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 unaccepted ESF documents"},
		{Key: "nightly_reconciliation", Label: "Nightly financial reconciliation & zero open breaks", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 open reconciliation breaks"},
		{Key: "cash_reconciliations", Label: "Driver cash reconciliations settled", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 unresolved cash reconciliations"},
		{Key: "payout_slices", Label: "Invoice settlement payout slices batched", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 unsettled payout slices"},
		{Key: "inventory_adjustments", Label: "Inventory count adjustments approved/posted", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 pending inventory count adjustments"},
		{Key: "fee_accruals", Label: "Platform fee accruals posted to GL", Passed: true, Blocking: true, OpenCount: 0, Detail: "0 unposted platform fee accruals"},
	}
}

func evaluateChecklistSpanner(
	ctx context.Context,
	client *spanner.Client,
	entityID, periodID, tzName string,
) (ChecklistReport, error) {
	if client == nil {
		return ChecklistReport{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return ChecklistReport{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	yr, mo, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return ChecklistReport{}, err
	}
	periodID = normPeriod

	if strings.TrimSpace(tzName) == "" {
		resolvedTZ, _, tzErr := ResolveEntityTimezone(ctx, entityID, "")
		if tzErr != nil {
			return ChecklistReport{}, tzErr
		}
		tzName = resolvedTZ
	}
	startUTC, endUTC, _, resolvedTZ, err := ComputePeriodWindow(periodID, tzName)
	if err != nil {
		return ChecklistReport{}, err
	}

	loc, _ := time.LoadLocation(resolvedTZ)
	if loc == nil {
		loc = time.UTC
	}
	localStartDate := civil.Date{Year: yr, Month: time.Month(mo), Day: 1}
	localNextMonth := time.Date(yr, time.Month(mo), 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0)
	localEndDate := civil.Date{Year: localNextMonth.Year(), Month: localNextMonth.Month(), Day: localNextMonth.Day()}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	var items []ChecklistItem

	// 1. Delivered orders missing finalized fiscal receipts & non-terminal fiscal attempts.
	unfiscalizedOrders, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM Orders o
		      WHERE o.SupplierId = @entityId
		        AND o.UpdatedAt >= @startUTC AND o.UpdatedAt < @endUTC
		        AND UPPER(o.Status) IN ('DELIVERED', 'COMPLETED')
		        AND COALESCE(UPPER(o.FiscalStatus), '') NOT IN ('FISCALIZED', 'ISSUED', 'PAID_RECEIPT', 'NOT_REQUIRED')
		        AND NOT EXISTS (
		          SELECT 1 FROM OrderFiscalReceipts r
		          WHERE r.OrderId = o.OrderId AND UPPER(r.Status) IN ('ISSUED', 'COMPLETED', 'FINALIZED')
		        )`,
		Params: map[string]any{
			"entityId": entityID,
			"startUTC": startUTC,
			"endUTC":   endUTC,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist deliveries_fiscal_receipts orders: %w", err)
	}
	pendingReceipts, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM OrderFiscalReceipts
		      WHERE SupplierId = @entityId
		        AND CreatedAt >= @startUTC AND CreatedAt < @endUTC
		        AND UPPER(Status) IN ('PENDING', 'FAILED', 'RETRYING', 'SUBMITTED')`,
		Params: map[string]any{
			"entityId": entityID,
			"startUTC": startUTC,
			"endUTC":   endUTC,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist deliveries_fiscal_receipts attempts: %w", err)
	}
	fiscalOpen := unfiscalizedOrders + pendingReceipts
	items = append(items, ChecklistItem{
		Key:       "deliveries_fiscal_receipts",
		Label:     "Delivered orders & fiscal receipts finalized",
		Passed:    fiscalOpen == 0,
		Blocking:  true,
		OpenCount: fiscalOpen,
		Detail:    fmt.Sprintf("%d unfiscalized deliveries, %d pending/failed receipts", unfiscalizedOrders, pendingReceipts),
	})

	// 2. Unaccepted ESF documents in period.
	esfOpen, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM EsfDocuments
		      WHERE EntityId = @entityId
		        AND PeriodId = @periodId
		        AND UPPER(Status) NOT IN ('ACCEPTED', 'CANCELLED')`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist esf_documents: %w", err)
	}
	items = append(items, ChecklistItem{
		Key:       "esf_documents",
		Label:     "Electronic invoices (ESF) accepted",
		Passed:    esfOpen == 0,
		Blocking:  true,
		OpenCount: esfOpen,
		Detail:    fmt.Sprintf("%d non-accepted ESF document(s) in %s", esfOpen, periodID),
	})

	// 3. Nightly reconciliation runs & open breaks.
	openBreaks, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM ReconciliationBreaks b
		      JOIN ReconciliationRuns r ON r.EntityId = b.EntityId AND r.RunId = b.RunId
		      WHERE b.EntityId = @entityId
		        AND r.PeriodId = @periodId
		        AND UPPER(b.Status) IN ('OPEN', 'ACKNOWLEDGED', 'IN_PROGRESS', 'ESCALATED')`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist nightly_reconciliation breaks: %w", err)
	}
	latestRunStatus, latestRunFound, err := queryLatestReconRunStatusRO(ctx, ro, entityID, periodID)
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist nightly_reconciliation run: %w", err)
	}
	runOpen := int64(0)
	if latestRunFound && (latestRunStatus == "RUNNING" || latestRunStatus == "FAILED") {
		runOpen = 1
	}
	reconOpen := openBreaks + runOpen
	reconDetail := fmt.Sprintf("%d open reconciliation break(s)", openBreaks)
	if latestRunFound {
		reconDetail = fmt.Sprintf("latest run=%s, %d open break(s)", latestRunStatus, openBreaks)
	}
	items = append(items, ChecklistItem{
		Key:       "nightly_reconciliation",
		Label:     "Nightly financial reconciliation & zero open breaks",
		Passed:    reconOpen == 0,
		Blocking:  true,
		OpenCount: reconOpen,
		Detail:    reconDetail,
	})

	// 4. Unresolved driver cash reconciliations in local period date range.
	cashOpen, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM CashReconciliations
		      WHERE SupplierId = @entityId
		        AND ShiftDate >= @startDate AND ShiftDate < @endDate
		        AND UPPER(Status) NOT IN ('ACCEPTED', 'WRITE_OFF')`,
		Params: map[string]any{
			"entityId":  entityID,
			"startDate": localStartDate,
			"endDate":   localEndDate,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist cash_reconciliations: %w", err)
	}
	items = append(items, ChecklistItem{
		Key:       "cash_reconciliations",
		Label:     "Driver cash reconciliations settled",
		Passed:    cashOpen == 0,
		Blocking:  true,
		OpenCount: cashOpen,
		Detail:    fmt.Sprintf("%d pending/disputed driver cash reconciliation(s)", cashOpen),
	})

	// 5. Unsettled payout settlement slices in period window.
	unsettledSlices, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM InvoiceSettlementSlices
		      WHERE SupplierId = @entityId
		        AND CreatedAt >= @startUTC AND CreatedAt < @endUTC
		        AND UPPER(Status) = 'UNSETTLED'`,
		Params: map[string]any{
			"entityId": entityID,
			"startUTC": startUTC,
			"endUTC":   endUTC,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist payout_slices: %w", err)
	}
	items = append(items, ChecklistItem{
		Key:       "payout_slices",
		Label:     "Invoice settlement payout slices batched",
		Passed:    unsettledSlices == 0,
		Blocking:  true,
		OpenCount: unsettledSlices,
		Detail:    fmt.Sprintf("%d unsettled payout slice(s)", unsettledSlices),
	})

	// 6. Pending inventory count adjustments on entity warehouses.
	pendingAdj, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM InventoryAdjustments ia
		      JOIN Warehouses w ON w.WarehouseId = ia.WarehouseId
		      WHERE w.SupplierId = @entityId
		        AND ia.CreatedAt >= @startUTC AND ia.CreatedAt < @endUTC
		        AND UPPER(ia.Status) IN ('PENDING', 'DRAFT', 'OPEN', 'SUBMITTED')`,
		Params: map[string]any{
			"entityId": entityID,
			"startUTC": startUTC,
			"endUTC":   endUTC,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist inventory_adjustments: %w", err)
	}
	items = append(items, ChecklistItem{
		Key:       "inventory_adjustments",
		Label:     "Inventory count adjustments approved/posted",
		Passed:    pendingAdj == 0,
		Blocking:  true,
		OpenCount: pendingAdj,
		Detail:    fmt.Sprintf("%d pending inventory count adjustment(s)", pendingAdj),
	})

	// 7. Unposted billing fee meter accruals in period window.
	unpostedFees, err := queryCountRO(ctx, ro, spanner.Statement{
		SQL: `SELECT COUNT(*)
		      FROM BillingMeterEvents b
		      WHERE b.SupplierId = @entityId
		        AND b.ProcessedAt >= @startUTC AND b.ProcessedAt < @endUTC
		        AND b.Amount > 0
		        AND NOT EXISTS (
		          SELECT 1 FROM JournalEntries j
		          WHERE j.EntityId = @entityId
		            AND (j.SourceId = b.EventId OR (b.OrderId <> '' AND j.OrderId = b.OrderId))
		            AND UPPER(j.SourceType) IN ('FEE', 'PLATFORM_FEE', 'FEE_ACCRUAL', 'BILLING_FEE', 'ORDER_FEE', 'FEE_PER_ORDER')
		        )`,
		Params: map[string]any{
			"entityId": entityID,
			"startUTC": startUTC,
			"endUTC":   endUTC,
		},
	})
	if err != nil {
		return ChecklistReport{}, fmt.Errorf("checklist fee_accruals: %w", err)
	}
	items = append(items, ChecklistItem{
		Key:       "fee_accruals",
		Label:     "Platform fee accruals posted to GL",
		Passed:    unpostedFees == 0,
		Blocking:  true,
		OpenCount: unpostedFees,
		Detail:    fmt.Sprintf("%d unposted billing fee accrual event(s)", unpostedFees),
	})

	blockingCount := 0
	for _, it := range items {
		if it.Blocking && !it.Passed {
			blockingCount++
		}
	}

	return ChecklistReport{
		EntityID:       entityID,
		PeriodID:       periodID,
		Timezone:       resolvedTZ,
		PeriodStartUTC: startUTC,
		PeriodEndUTC:   endUTC,
		ReadyToClose:   blockingCount == 0,
		BlockingCount:  blockingCount,
		EvaluatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		Items:          items,
	}, nil
}

func queryCountRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, stmt spanner.Statement) (int64, error) {
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var cnt int64
	if err := row.Columns(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

func queryLatestReconRunStatusRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, periodID string) (string, bool, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT Status
		      FROM ReconciliationRuns
		      WHERE EntityId = @entityId AND PeriodId = @periodId
		      ORDER BY StartedAt DESC
		      LIMIT 1`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var st string
	if err := row.Columns(&st); err != nil {
		return "", false, err
	}
	return strings.ToUpper(strings.TrimSpace(st)), true, nil
}
