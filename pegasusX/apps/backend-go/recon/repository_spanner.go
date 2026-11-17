package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository implements Repository against Cloud Spanner using strong/consistent snapshots.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed reconciliation repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) LoadSnapshot(ctx context.Context, entityID, periodID, currency string, asOf time.Time) (Snapshot, error) {
	if r == nil || r.client == nil {
		return Snapshot{}, fmt.Errorf("recon spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	periodID = strings.TrimSpace(periodID)
	if entityID == "" {
		return Snapshot{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	hasCutoff, periodEnd := computePeriodCutoff(ctx, entityID, periodID)

	snap := Snapshot{
		EntityID: entityID,
		PeriodID: periodID,
		Currency: currency,
		AsOf:     asOf,
	}

	var skuUnitCost map[string]int64
	var warehouseIDs []string

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(txCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		glBal, err := queryGLBalancesMap(txCtx, ro, entityID, periodID)
		if err != nil {
			return err
		}
		snap.GLARRetailerNetMinor = glBal[gl.AccountAccountsReceivable]
		snap.GLInventoryNetMinor = glBal[gl.AccountInventory]
		snap.GLPSPGatewayNetMinor = glBal[gl.AccountCardPSPTransit]
		snap.GLCashDriverNetMinor = glBal[gl.AccountCashInTransit]
		snap.GLCashOfficeNetMinor = glBal[gl.AccountCashOffice]
		snap.GLBankOperatingNetMinor = glBal[gl.AccountBankOperating]
		// TAX:VAT:OUTPUT is credit-normal, so invert (Debit - Credit) to obtain (Credit - Debit).
		snap.GLVatOutputCreditMinor = -glBal[gl.AccountVatOutput]

		if err := loadOrdersARSnapshot(txCtx, ro, &snap, hasCutoff, periodEnd); err != nil {
			return err
		}
		warehouseIDs, skuUnitCost, err = loadStockBaseSnapshot(txCtx, ro, &snap, hasCutoff, periodEnd)
		if err != nil {
			return err
		}
		if err := loadCashPSPSnapshot(txCtx, ro, &snap, periodID, hasCutoff, periodEnd); err != nil {
			return err
		}
		if err := loadTaxFiscalSnapshot(txCtx, ro, &snap, hasCutoff, periodEnd); err != nil {
			return err
		}

		snap.WarehousesChecked = int64(len(warehouseIDs))
		missingCostSKUs := make([]string, 0)
		for _, wid := range warehouseIDs {
			report, recErr := stocklots.ReconcileInventoryV2WithReader(txCtx, ro, entityID, wid)
			if recErr != nil {
				return fmt.Errorf("reconcile inventory v2 (%s): %w", wid, recErr)
			}
			for _, row := range report.Mismatches {
				cost := skuUnitCost[row.ProductID]
				if cost == 0 {
					missingCostSKUs = append(missingCostSKUs, row.ProductID)
				}
				snap.V2Mismatches = append(snap.V2Mismatches, StockV2Mismatch{
					WarehouseID:    wid,
					ProductID:      row.ProductID,
					MissingV2:      row.MissingV2,
					LotAvailable:   row.LotsOnHand,
					LotReserved:    row.LotsReserved,
					V2Available:    row.V2OnHand,
					V2Reserved:     row.V2Reserved,
					DeltaAvailable: row.V2OnHand - row.LotsOnHand,
					DeltaReserved:  row.V2Reserved - row.LotsReserved,
					UnitCostMinor:  cost,
				})
			}
		}

		if len(missingCostSKUs) > 0 && len(snap.V2Mismatches) > 0 {
			extraCosts, cErr := lookupSKUUnitCostsWithReader(txCtx, ro, entityID, missingCostSKUs)
			if cErr != nil {
				return fmt.Errorf("lookup sku unit costs: %w", cErr)
			}
			for i := range snap.V2Mismatches {
				if snap.V2Mismatches[i].UnitCostMinor == 0 {
					snap.V2Mismatches[i].UnitCostMinor = extraCosts[snap.V2Mismatches[i].ProductID]
				}
			}
		}
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}

	return snap, nil
}

func (r *SpannerRepository) SaveRunAndBreaks(ctx context.Context, run ReconciliationRun, breaks []ReconciliationBreak) (ReconciliationRun, []ReconciliationBreak, error) {
	if r == nil || r.client == nil {
		return ReconciliationRun{}, nil, fmt.Errorf("recon spanner repository: nil client")
	}
	checksRaw, err := json.Marshal(run.Checks)
	if err != nil {
		return ReconciliationRun{}, nil, fmt.Errorf("marshal checks json: %w", err)
	}
	summaryRaw, err := json.Marshal(run.Summary)
	if err != nil {
		return ReconciliationRun{}, nil, fmt.Errorf("marshal summary json: %w", err)
	}

	savedBreaks := make([]ReconciliationBreak, len(breaks))
	copy(savedBreaks, breaks)
	for i := range savedBreaks {
		if strings.TrimSpace(savedBreaks[i].ControlTowerExceptionID) == "" {
			savedBreaks[i].ControlTowerExceptionID = uuid.NewString()
		}
	}

	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, 1+len(savedBreaks)*2)
		muts = append(muts, spanner.InsertMap("ReconciliationRuns", map[string]any{
			"EntityId":      run.EntityID,
			"RunId":         run.RunID,
			"PeriodId":      run.PeriodID,
			"Mode":          run.Mode,
			"Status":        run.Status,
			"ChecksJson":    string(checksRaw),
			"SummaryJson":   string(summaryRaw),
			"BreakCount":    run.BreakCount,
			"ExposureMinor": run.ExposureMinor,
			"TriggeredBy":   run.TriggeredBy,
			"StartedAt":     spanner.CommitTimestamp,
			"CompletedAt":   run.CompletedAt.UTC(),
		}))

		outboxInner := outbox.NewSpannerTxnBuffer(txn)
		outboxBuf := &entityStampedOutboxBuf{entityID: run.EntityID, inner: outboxInner}

		for _, b := range savedBreaks {
			muts = append(muts, spanner.InsertMap("ReconciliationBreaks", map[string]any{
				"EntityId":                b.EntityID,
				"BreakId":                 b.BreakID,
				"RunId":                   b.RunID,
				"Dimension":               b.Dimension,
				"ObjectType":              b.ObjectType,
				"ObjectId":                b.ObjectID,
				"ExpectedMinor":           b.ExpectedMinor,
				"ActualMinor":             b.ActualMinor,
				"DeltaMinor":              b.DeltaMinor,
				"Currency":                b.Currency,
				"Severity":                b.Severity,
				"Status":                  b.Status,
				"OwnerId":                 nullableStr(b.OwnerID),
				"SlaDueAt":                nullableTime(b.SlaDueAt),
				"ResolutionNote":          nullableStr(b.ResolutionNote),
				"ControlTowerExceptionId": nullableStr(b.ControlTowerExceptionID),
				"CreatedAt":               spanner.CommitTimestamp,
				"UpdatedAt":               spanner.CommitTimestamp,
			}))

			ticketOrderID := b.BreakID
			if b.ObjectType == ObjectOrder && len(b.ObjectID) <= 36 {
				ticketOrderID = b.ObjectID
			} else if len(ticketOrderID) > 36 {
				ticketOrderID = ticketOrderID[:36]
			}
			payloadJSON, mErr := json.Marshal(map[string]any{
				"break_id":       b.BreakID,
				"run_id":         b.RunID,
				"entity_id":      b.EntityID,
				"dimension":      b.Dimension,
				"object_type":    b.ObjectType,
				"object_id":      b.ObjectID,
				"expected_minor": b.ExpectedMinor,
				"actual_minor":   b.ActualMinor,
				"delta_minor":    b.DeltaMinor,
				"currency":       b.Currency,
				"sla_due_at":     formatNullableTime(b.SlaDueAt),
			})
			if mErr != nil {
				return fmt.Errorf("marshal exception payload: %w", mErr)
			}
			title := fmt.Sprintf("[RECON][%s] %s %s delta %d %s", b.Dimension, b.ObjectType, b.ObjectID, b.DeltaMinor, b.Currency)
			if len(title) > 256 {
				title = title[:256]
			}
			muts = append(muts, spanner.InsertMap("ExceptionTickets", map[string]any{
				"TicketId":     b.ControlTowerExceptionID,
				"Type":         "RECON_" + b.Dimension,
				"OrderId":      ticketOrderID,
				"Severity":     b.Severity,
				"Status":       BreakStatusOpen,
				"Title":        title,
				"Description":  nullableStr(b.ResolutionNote),
				"AssignedRole": nullableStr(b.OwnerID),
				"CreatedAt":    spanner.CommitTimestamp,
				"CreatedBy":    "recon-engine",
				"Payload":      spanner.NullJSON{Value: json.RawMessage(payloadJSON), Valid: true},
			}))

			if eErr := outbox.EmitJSON(txCtx, outboxBuf, events.AggregateReconBreak, b.BreakID, events.TopicLedger, events.ReconBreakOpenedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventReconBreakOpened,
					Timestamp: run.CompletedAt.UTC().Format(time.RFC3339Nano),
				},
				BreakID:                 b.BreakID,
				RunID:                   b.RunID,
				EntityID:                b.EntityID,
				Dimension:               b.Dimension,
				ObjectType:              b.ObjectType,
				ObjectID:                b.ObjectID,
				ExpectedMinor:           b.ExpectedMinor,
				ActualMinor:             b.ActualMinor,
				DeltaMinor:              b.DeltaMinor,
				Currency:                b.Currency,
				Severity:                b.Severity,
				Status:                  b.Status,
				OwnerID:                 b.OwnerID,
				SlaDueAt:                formatNullableTime(b.SlaDueAt),
				ControlTowerExceptionID: b.ControlTowerExceptionID,
			}); eErr != nil {
				return fmt.Errorf("emit recon.break.opened: %w", eErr)
			}
		}

		if eErr := outbox.EmitJSON(txCtx, outboxBuf, events.AggregateReconRun, run.RunID, events.TopicLedger, events.ReconRunCompletedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventReconRunCompleted,
				Timestamp: run.CompletedAt.UTC().Format(time.RFC3339Nano),
			},
			RunID:         run.RunID,
			EntityID:      run.EntityID,
			PeriodID:      run.PeriodID,
			Mode:          run.Mode,
			Status:        run.Status,
			BreakCount:    run.BreakCount,
			ExposureMinor: run.ExposureMinor,
			Currency:      run.Currency,
			TriggeredBy:   run.TriggeredBy,
			StartedAt:     run.StartedAt.UTC().Format(time.RFC3339Nano),
			CompletedAt:   run.CompletedAt.UTC().Format(time.RFC3339Nano),
		}); eErr != nil {
			return fmt.Errorf("emit recon.run.completed: %w", eErr)
		}

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		return outboxInner.Flush(txCtx)
	})
	if err != nil {
		return ReconciliationRun{}, nil, err
	}
	return run, savedBreaks, nil
}

func (r *SpannerRepository) ListRuns(ctx context.Context, filter ListRunsFilter) ([]ReconciliationRun, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("recon spanner repository: nil client")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	sql := `SELECT EntityId, RunId, PeriodId, Mode, Status, ChecksJson, SummaryJson,
	               BreakCount, ExposureMinor, TriggeredBy, StartedAt, CompletedAt
	        FROM ReconciliationRuns
	        WHERE EntityId = @entityId`
	params := map[string]any{
		"entityId": strings.TrimSpace(filter.EntityID),
		"limit":    int64(limit),
	}
	if p := strings.TrimSpace(filter.PeriodID); p != "" {
		sql += " AND PeriodId = @periodId"
		params["periodId"] = p
	}
	if s := strings.ToUpper(strings.TrimSpace(filter.Status)); s != "" {
		sql += " AND Status = @status"
		params["status"] = s
	}
	sql += " ORDER BY StartedAt DESC LIMIT @limit"

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]ReconciliationRun, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list reconciliation runs: %w", err)
		}
		run, err := scanRunRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *SpannerRepository) ListBreaks(ctx context.Context, filter ListBreaksFilter) ([]ReconciliationBreak, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("recon spanner repository: nil client")
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	sql := `SELECT EntityId, BreakId, RunId, Dimension, ObjectType, ObjectId,
	               ExpectedMinor, ActualMinor, DeltaMinor, Currency, Severity, Status,
	               OwnerId, SlaDueAt, ResolutionNote, ControlTowerExceptionId, CreatedAt, UpdatedAt
	        FROM ReconciliationBreaks
	        WHERE EntityId = @entityId`
	params := map[string]any{
		"entityId": strings.TrimSpace(filter.EntityID),
		"limit":    int64(limit),
	}
	if rid := strings.TrimSpace(filter.RunID); rid != "" {
		sql += " AND RunId = @runId"
		params["runId"] = rid
	}
	if d := strings.ToUpper(strings.TrimSpace(filter.Dimension)); d != "" {
		sql += " AND Dimension = @dimension"
		params["dimension"] = d
	}
	if s := strings.ToUpper(strings.TrimSpace(filter.Status)); s != "" && s != "ALL" {
		sql += " AND Status = @status"
		params["status"] = s
	}
	if sev := strings.ToUpper(strings.TrimSpace(filter.Severity)); sev != "" {
		sql += " AND Severity = @severity"
		params["severity"] = sev
	}
	sql += " ORDER BY ABS(DeltaMinor) DESC, CreatedAt DESC LIMIT @limit"

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]ReconciliationBreak, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list reconciliation breaks: %w", err)
		}
		b, err := scanBreakRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *SpannerRepository) GetBreak(ctx context.Context, entityID, breakID string) (ReconciliationBreak, error) {
	if r == nil || r.client == nil {
		return ReconciliationBreak{}, fmt.Errorf("recon spanner repository: nil client")
	}
	row, err := r.client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(
		ctx,
		"ReconciliationBreaks",
		spanner.Key{strings.TrimSpace(entityID), strings.TrimSpace(breakID)},
		[]string{
			"EntityId", "BreakId", "RunId", "Dimension", "ObjectType", "ObjectId",
			"ExpectedMinor", "ActualMinor", "DeltaMinor", "Currency", "Severity", "Status",
			"OwnerId", "SlaDueAt", "ResolutionNote", "ControlTowerExceptionId", "CreatedAt", "UpdatedAt",
		},
	)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ReconciliationBreak{}, fmt.Errorf("%w: break %s", ErrNotFound, breakID)
		}
		return ReconciliationBreak{}, fmt.Errorf("get reconciliation break: %w", err)
	}
	return scanBreakRow(row)
}

func (r *SpannerRepository) UpdateBreak(ctx context.Context, entityID, breakID string, req PatchBreakRequest, now time.Time) (ReconciliationBreak, error) {
	if r == nil || r.client == nil {
		return ReconciliationBreak{}, fmt.Errorf("recon spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	breakID = strings.TrimSpace(breakID)

	var updated ReconciliationBreak
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(
			txCtx,
			"ReconciliationBreaks",
			spanner.Key{entityID, breakID},
			[]string{
				"EntityId", "BreakId", "RunId", "Dimension", "ObjectType", "ObjectId",
				"ExpectedMinor", "ActualMinor", "DeltaMinor", "Currency", "Severity", "Status",
				"OwnerId", "SlaDueAt", "ResolutionNote", "ControlTowerExceptionId", "CreatedAt", "UpdatedAt",
			},
		)
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return fmt.Errorf("%w: break %s", ErrNotFound, breakID)
			}
			return fmt.Errorf("read reconciliation break: %w", err)
		}
		cur, err := scanBreakRow(row)
		if err != nil {
			return err
		}

		nextStatus, err := resolveBreakTransition(cur.Status, req)
		if err != nil {
			return err
		}

		if owner := strings.TrimSpace(req.OwnerID); owner != "" {
			cur.OwnerID = owner
		}
		if note := strings.TrimSpace(coalesce(req.Note, req.ResolutionNote)); note != "" {
			cur.ResolutionNote = note
		}
		cur.Status = nextStatus
		cur.UpdatedAt = now.UTC()

		muts := []*spanner.Mutation{
			spanner.UpdateMap("ReconciliationBreaks", map[string]any{
				"EntityId":       cur.EntityID,
				"BreakId":        cur.BreakID,
				"Status":         cur.Status,
				"OwnerId":        nullableStr(cur.OwnerID),
				"ResolutionNote": nullableStr(cur.ResolutionNote),
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}

		if cur.ControlTowerExceptionID != "" {
			// Check existence first so direct UpdateMap never fails if ticket was pruned externally.
			if _, tErr := txn.ReadRow(txCtx, "ExceptionTickets", spanner.Key{cur.ControlTowerExceptionID}, []string{"TicketId"}); tErr == nil {
				ticketUpdate := map[string]any{
					"TicketId": cur.ControlTowerExceptionID,
					"Status":   cur.Status,
				}
				if cur.OwnerID != "" {
					ticketUpdate["AssignedRole"] = cur.OwnerID
				}
				muts = append(muts, spanner.UpdateMap("ExceptionTickets", ticketUpdate))
			}
		}

		if nextStatus == BreakStatusResolved || nextStatus == BreakStatusWaived {
			outboxInner := outbox.NewSpannerTxnBuffer(txn)
			outboxBuf := &entityStampedOutboxBuf{entityID: cur.EntityID, inner: outboxInner}
			if eErr := outbox.EmitJSON(txCtx, outboxBuf, events.AggregateReconBreak, cur.BreakID, events.TopicLedger, events.ReconBreakResolvedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventReconBreakResolved,
					Timestamp: now.UTC().Format(time.RFC3339Nano),
				},
				BreakID:                 cur.BreakID,
				RunID:                   cur.RunID,
				EntityID:                cur.EntityID,
				Dimension:               cur.Dimension,
				ObjectType:              cur.ObjectType,
				ObjectID:                cur.ObjectID,
				DeltaMinor:              cur.DeltaMinor,
				Currency:                cur.Currency,
				Status:                  cur.Status,
				ResolvedBy:              coalesce(strings.TrimSpace(req.ActorID), "system"),
				ResolutionNote:          cur.ResolutionNote,
				ControlTowerExceptionID: cur.ControlTowerExceptionID,
			}); eErr != nil {
				return fmt.Errorf("emit recon.break.resolved: %w", eErr)
			}
			if err := txn.BufferWrite(muts); err != nil {
				return err
			}
			if err := outboxInner.Flush(txCtx); err != nil {
				return err
			}
		} else {
			if err := txn.BufferWrite(muts); err != nil {
				return err
			}
		}

		updated = cur
		return nil
	})
	if err != nil {
		return ReconciliationBreak{}, err
	}
	return updated, nil
}

func (r *SpannerRepository) ListActiveSupplierIDs(ctx context.Context, limit int) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId
		      FROM Suppliers
		      ORDER BY CreatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{"limit": int64(limit)},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()

	out := make([]string, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list active suppliers: %w", err)
		}
		var sid string
		if err := row.Columns(&sid); err != nil {
			return nil, fmt.Errorf("scan supplier id: %w", err)
		}
		if strings.TrimSpace(sid) != "" {
			out = append(out, sid)
		}
	}
	return out, nil
}

func queryGLBalancesMap(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, periodID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AccountCode, COALESCE(SUM(DebitMinor - CreditMinor), 0)
		      FROM GlBalances
		      WHERE EntityId = @entityId
		        AND PeriodId <= @periodId
		        AND AccountCode IN ('AR:RETAILER', 'INVENTORY', 'PSP:GATEWAY', 'CASH:DRIVER', 'CASH:OFFICE', 'BANK:OPERATING', 'TAX:VAT:OUTPUT')
		      GROUP BY AccountCode`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	res := make(map[string]int64, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query GlBalances: %w", err)
		}
		var acct string
		var net int64
		if err := row.Columns(&acct, &net); err != nil {
			return nil, fmt.Errorf("scan GlBalances: %w", err)
		}
		res[acct] = net
	}
	return res, nil
}

func loadOrdersARSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, snap *Snapshot, hasCutoff bool, periodEnd time.Time) error {
	// 1. Delivered credit orders summary & anomalies where ArInvoice is missing or PrincipalMinor != TotalMinor.
	sumIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*), COALESCE(SUM(TotalMinor), 0)
		      FROM Orders
		      WHERE SupplierId = @entityId
		        AND Status = 'DELIVERED_ON_CREDIT'
		        AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer sumIter.Stop()
	if row, err := sumIter.Next(); err == nil {
		if err := row.Columns(&snap.DeliveredCreditOrderCount, &snap.DeliveredCreditOrderMinor); err != nil {
			return fmt.Errorf("scan delivered credit order summary: %w", err)
		}
	} else if err != iterator.Done {
		return fmt.Errorf("query delivered credit orders: %w", err)
	}

	anomIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT o.OrderId, o.TotalMinor, COALESCE(ai.PrincipalMinor, 0), COALESCE(ai.Status, '')
		      FROM Orders o
		      LEFT JOIN ArInvoices ai
		        ON ai.OrderId = o.OrderId
		       AND ai.SupplierId = o.SupplierId
		       AND ai.Status != 'VOID'
		      WHERE o.SupplierId = @entityId
		        AND o.Status = 'DELIVERED_ON_CREDIT'
		        AND o.TotalMinor > 0
		        AND (@hasCutoff = FALSE OR o.CreatedAt < @periodEnd)
		        AND (ai.InvoiceId IS NULL OR ai.PrincipalMinor != o.TotalMinor)`,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer anomIter.Stop()
	for {
		row, err := anomIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query delivered credit order AR anomalies: %w", err)
		}
		var a OrderARAnomaly
		if err := row.Columns(&a.OrderID, &a.OrderTotalMinor, &a.InvoicePrincipalMinor, &a.InvoiceStatus); err != nil {
			return fmt.Errorf("scan order AR anomaly: %w", err)
		}
		snap.DeliveredCreditAnomalies = append(snap.DeliveredCreditAnomalies, a)
	}

	// 2. Open AR invoices balance (OPEN / PARTIAL).
	arIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*), COALESCE(SUM(BalanceMinor), 0)
		      FROM ArInvoices
		      WHERE SupplierId = @entityId
		        AND Status IN ('OPEN', 'PARTIAL')
		        AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer arIter.Stop()
	if row, err := arIter.Next(); err == nil {
		if err := row.Columns(&snap.OpenARInvoiceCount, &snap.OpenARBalanceMinor); err != nil {
			return fmt.Errorf("scan open ArInvoices summary: %w", err)
		}
	} else if err != iterator.Done {
		return fmt.Errorf("query open ArInvoices: %w", err)
	}
	return nil
}

func loadStockBaseSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, snap *Snapshot, hasCutoff bool, periodEnd time.Time) ([]string, map[string]int64, error) {
	// Check optional cost columns and InventoryCostLayers table safely via INFORMATION_SCHEMA.COLUMNS.
	var hasLotUnitCost, hasProductCost, hasCostLayersTable bool
	colIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_SCHEMA = ''
		        AND (
		          (TABLE_NAME = 'StockLots' AND COLUMN_NAME = 'UnitCostMinor')
		          OR (TABLE_NAME = 'Products' AND COLUMN_NAME = 'CostMinor')
		          OR (TABLE_NAME = 'InventoryCostLayers' AND COLUMN_NAME = 'UnitCostMinor')
		        )`,
	})
	defer colIter.Stop()
	for {
		row, err := colIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("query schema columns: %w", err)
		}
		var tbl, col string
		if err := row.Columns(&tbl, &col); err != nil {
			return nil, nil, fmt.Errorf("scan schema columns: %w", err)
		}
		if tbl == "StockLots" && col == "UnitCostMinor" {
			hasLotUnitCost = true
		}
		if tbl == "Products" && col == "CostMinor" {
			hasProductCost = true
		}
		if tbl == "InventoryCostLayers" && col == "UnitCostMinor" {
			hasCostLayersTable = true
		}
	}

	skuUnitCost := make(map[string]int64)
	layerCostedLots := make(map[string]bool)
	layerCostedWHSKU := make(map[string]bool)
	if hasCostLayersTable {
		layerIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT WarehouseId, ProductId, COALESCE(LotId, ''), QtyRemaining, UnitCostMinor
			      FROM InventoryCostLayers
			      WHERE EntityId = @entityId
			        AND (@hasCutoff = FALSE OR ReceivedAt < @periodEnd)`,
			Params: map[string]any{
				"entityId":  snap.EntityID,
				"hasCutoff": hasCutoff,
				"periodEnd": periodEnd,
			},
		})
		defer layerIter.Stop()
		for {
			lrow, lerr := layerIter.Next()
			if lerr == iterator.Done {
				break
			}
			if lerr != nil {
				return nil, nil, fmt.Errorf("query InventoryCostLayers: %w", lerr)
			}
			var wid, pid, lid string
			var qtyRem, unitCost int64
			if cErr := lrow.Columns(&wid, &pid, &lid, &qtyRem, &unitCost); cErr != nil {
				return nil, nil, fmt.Errorf("scan InventoryCostLayer: %w", cErr)
			}
			snap.HasCostLayers = true
			snap.CostLayersCount++
			if qtyRem > 0 {
				snap.CostLayersQtyRemaining += qtyRem
				snap.CostLayersValuedMinor += qtyRem * unitCost
				if lid != "" {
					layerCostedLots[lid] = true
				}
				layerCostedWHSKU[wid+"\x00"+pid] = true
			}
			if unitCost > 0 && skuUnitCost[pid] == 0 {
				skuUnitCost[pid] = unitCost
			}
		}
		if snap.HasCostLayers {
			snap.CostedStockValuedMinor = snap.CostLayersValuedMinor
		}
	}

	whSQL := `SELECT DISTINCT WarehouseId FROM (
		        SELECT WarehouseId FROM Warehouses WHERE SupplierId = @entityId
		        UNION DISTINCT
		        SELECT WarehouseId FROM StockLots WHERE SupplierId = @entityId AND WarehouseId IS NOT NULL AND WarehouseId != ''
		        UNION DISTINCT
		        SELECT WarehouseId FROM SupplierInventoryV2 WHERE SupplierId = @entityId AND WarehouseId IS NOT NULL AND WarehouseId != ''`
	if hasCostLayersTable {
		whSQL += `
		        UNION DISTINCT
		        SELECT WarehouseId FROM InventoryCostLayers WHERE EntityId = @entityId AND WarehouseId IS NOT NULL AND WarehouseId != ''`
	}
	whSQL += `
		      )`
	whIter := ro.Query(ctx, spanner.Statement{
		SQL:    whSQL,
		Params: map[string]any{"entityId": snap.EntityID},
	})
	defer whIter.Stop()
	var warehouseIDs []string
	for {
		row, err := whIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("query supplier warehouses: %w", err)
		}
		var wid string
		if err := row.Columns(&wid); err != nil {
			return nil, nil, fmt.Errorf("scan warehouse id: %w", err)
		}
		if wid = strings.TrimSpace(wid); wid != "" {
			warehouseIDs = append(warehouseIDs, wid)
		}
	}

	lotCostExpr := "CAST(0 AS INT64)"
	if hasLotUnitCost {
		lotCostExpr = "COALESCE(l.UnitCostMinor, 0)"
	}
	prodCostExpr := "CAST(0 AS INT64)"
	if hasProductCost {
		prodCostExpr = "COALESCE(p.CostMinor, 0)"
	}
	lotSQL := fmt.Sprintf(`SELECT l.LotId, COALESCE(l.WarehouseId, ''), l.ProductId, l.QuantityOnHand,
	                              %s AS LotCostMinor,
	                              %s AS ProdCostMinor,
	                              COALESCE(spo.PriceMinor, 0) AS OfferPriceMinor,
	                              COALESCE(p.PriceMinor, 0) AS ProdPriceMinor
	                       FROM StockLots l
	                       LEFT JOIN Products p ON p.ProductId = l.ProductId
	                       LEFT JOIN SupplierProductOffers spo ON spo.SupplierId = l.SupplierId AND spo.ProductId = l.ProductId
	                       WHERE l.SupplierId = @entityId
	                         AND l.QuantityOnHand > 0
	                         AND UPPER(COALESCE(l.Status, 'AVAILABLE')) NOT IN ('DEPLETED', 'CONSUMED', 'EXPIRED', 'WRITTEN_OFF')
	                         AND (@hasCutoff = FALSE OR l.ReceivedAt < @periodEnd)`, lotCostExpr, prodCostExpr)

	lotIter := ro.Query(ctx, spanner.Statement{
		SQL: lotSQL,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer lotIter.Stop()

	for {
		row, err := lotIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("query active stock lots: %w", err)
		}
		var lotID, wid, pid string
		var qty, lotCost, prodCost, offerPrice, prodPrice int64
		if err := row.Columns(&lotID, &wid, &pid, &qty, &lotCost, &prodCost, &offerPrice, &prodPrice); err != nil {
			return nil, nil, fmt.Errorf("scan active stock lot: %w", err)
		}
		unitCost := firstPositiveInt64(lotCost, prodCost, offerPrice, prodPrice)
		if unitCost > 0 && skuUnitCost[pid] == 0 {
			skuUnitCost[pid] = unitCost
		}
		if strings.TrimSpace(wid) == "" {
			snap.MissingWarehouseLots = append(snap.MissingWarehouseLots, MissingWarehouseLot{
				LotID:         lotID,
				ProductID:     pid,
				QuantityTotal: qty,
				UnitCostMinor: unitCost,
			})
		}
		if snap.HasCostLayers {
			if unitCost <= 0 && !layerCostedLots[lotID] && !layerCostedWHSKU[wid+"\x00"+pid] {
				snap.UncostedLots = append(snap.UncostedLots, UncostedLot{
					LotID:       lotID,
					WarehouseID: wid,
					ProductID:   pid,
					Quantity:    qty,
				})
			}
			continue
		}
		if unitCost <= 0 {
			snap.UncostedLots = append(snap.UncostedLots, UncostedLot{
				LotID:       lotID,
				WarehouseID: wid,
				ProductID:   pid,
				Quantity:    qty,
			})
			continue
		}
		snap.CostedStockValuedMinor += qty * unitCost
	}

	return warehouseIDs, skuUnitCost, nil
}

func lookupSKUUnitCostsWithReader(
	ctx context.Context,
	rdr interface {
		Query(context.Context, spanner.Statement) *spanner.RowIterator
	},
	entityID string,
	productIDs []string,
) (map[string]int64, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	out := make(map[string]int64, len(productIDs))
	iter := rdr.Query(ctx, spanner.Statement{
		SQL: `SELECT p.ProductId, COALESCE(spo.PriceMinor, 0), COALESCE(p.PriceMinor, 0)
		      FROM Products p
		      LEFT JOIN SupplierProductOffers spo ON spo.SupplierId = @entityId AND spo.ProductId = p.ProductId
		      WHERE p.ProductId IN UNNEST(@pids)`,
		Params: map[string]any{
			"entityId": entityID,
			"pids":     productIDs,
		},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, err
		}
		var pid string
		var offerPrice, prodPrice int64
		if err := row.Columns(&pid, &offerPrice, &prodPrice); err == nil {
			out[pid] = firstPositiveInt64(offerPrice, prodPrice)
		}
	}
	return out, nil
}

func loadCashPSPSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, snap *Snapshot, periodID string, hasCutoff bool, periodEnd time.Time) error {
	// 1. Settled PaymentSessions
	if val, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(AmountMinor), 0)
		FROM PaymentSessions
		WHERE SupplierId = @entityId
		  AND UPPER(Status) IN ('PAID', 'CAPTURED', 'SETTLED', 'SUCCESS')
		  AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd}); err != nil {
		return fmt.Errorf("query settled payment sessions: %w", err)
	} else {
		snap.SettledPSPSessionsMinor = val
	}

	// 2. Captured card OrderPaymentLegs without settled PaymentSessions
	if val, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(l.AmountMinor), 0)
		FROM OrderPaymentLegs l
		INNER JOIN Orders o ON l.OrderId = o.OrderId
		WHERE o.SupplierId = @entityId
		  AND UPPER(COALESCE(o.Status, '')) != 'CANCELLED'
		  AND UPPER(l.Method) IN ('CARD', 'PAYME', 'CLICK', 'UZUM', 'ADYEN', 'STRIPE', 'GLOBAL_PAY', 'PSP')
		  AND UPPER(l.Status) IN ('CAPTURED', 'PAID', 'SETTLED', 'SUCCESS')
		  AND (@hasCutoff = FALSE OR l.CreatedAt < @periodEnd)
		  AND NOT EXISTS (
		    SELECT 1 FROM PaymentSessions ps
		    WHERE ps.OrderId = l.OrderId
		      AND UPPER(ps.Status) IN ('PAID', 'CAPTURED', 'SETTLED', 'SUCCESS')
		  )`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd}); err != nil {
		return fmt.Errorf("query fallback card legs: %w", err)
	} else {
		snap.FallbackCapturedCardLegsMinor = val
	}

	// 3. PaymentChargebacks that hit PSP:GATEWAY (exclude LEDGER_ONLY chargebacks against AP/AR).
	if val, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(pc.AmountMinor), 0)
		FROM PaymentChargebacks pc
		INNER JOIN Orders o ON pc.OrderId = o.OrderId
		WHERE o.SupplierId = @entityId
		  AND UPPER(COALESCE(pc.Gateway, '')) != 'LEDGER'
		  AND (@hasCutoff = FALSE OR pc.CreatedAt < @periodEnd)
		  AND (
		    UPPER(COALESCE(pc.Gateway, '')) = 'GATEWAY_REFUND'
		    OR EXISTS (
		      SELECT 1 FROM JournalPostings jp
		      INNER JOIN JournalEntries je ON je.EntityId = jp.EntityId AND je.EntryId = jp.EntryId
		      WHERE je.EntityId = @entityId
		        AND UPPER(je.SourceType) = 'CLAIM_DISPUTE'
		        AND je.SourceId = pc.ChargebackId
		        AND jp.AccountCode = 'PSP:GATEWAY'
		        AND jp.Direction = 'CREDIT'
		    )
		  )`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd}); err != nil {
		return fmt.Errorf("query payment chargebacks: %w", err)
	} else {
		snap.PaymentChargebacksMinor = val
	}

	// 4. Driver cash collections: captured CASH OrderPaymentLegs + completed fiscal cash orders without CASH legs
	cashLegs, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(l.AmountMinor), 0)
		FROM OrderPaymentLegs l
		INNER JOIN Orders o ON l.OrderId = o.OrderId
		WHERE o.SupplierId = @entityId
		  AND UPPER(COALESCE(o.Status, '')) != 'CANCELLED'
		  AND UPPER(l.Method) IN ('CASH', 'CASH_ON_DELIVERY', 'CASH_DRIVER')
		  AND UPPER(l.Status) IN ('CAPTURED', 'PAID', 'SETTLED', 'SUCCESS')
		  AND (@hasCutoff = FALSE OR l.CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd})
	if err != nil {
		return fmt.Errorf("query driver cash legs: %w", err)
	}
	cashFiscalOrders, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(o.TotalMinor), 0)
		FROM Orders o
		WHERE o.SupplierId = @entityId
		  AND UPPER(o.Status) = 'COMPLETED'
		  AND (@hasCutoff = FALSE OR o.CreatedAt < @periodEnd)
		  AND EXISTS (
		    SELECT 1 FROM OrderFiscalReceipts ofr
		    WHERE ofr.OrderId = o.OrderId
		      AND ofr.SupplierId = @entityId
		      AND UPPER(COALESCE(ofr.PaymentMethod, '')) IN ('CASH', 'CASH_ON_DELIVERY', 'CASH_DRIVER')
		      AND UPPER(ofr.Status) IN ('SUCCESS', 'ISSUED', 'REGISTERED', 'COMPLETED')
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM OrderPaymentLegs l
		    WHERE l.OrderId = o.OrderId
		      AND UPPER(l.Method) IN ('CASH', 'CASH_ON_DELIVERY', 'CASH_DRIVER')
		      AND UPPER(l.Status) IN ('CAPTURED', 'PAID', 'SETTLED', 'SUCCESS')
		  )`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd})
	if err != nil {
		return fmt.Errorf("query completed cash fiscal orders: %w", err)
	}
	snap.DriverCashCollectedMinor = cashLegs + cashFiscalOrders

	// 5. CashReconciliations (accepted/write-off + disputed)
	crIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ReconciliationId, DriverId, ExpectedCashMinor, DeclaredCashMinor, DifferenceMinor, UPPER(Status)
		      FROM CashReconciliations
		      WHERE SupplierId = @entityId
		        AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		Params: map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd},
	})
	defer crIter.Stop()
	for {
		row, err := crIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query CashReconciliations: %w", err)
		}
		var cr DisputedCashRecon
		var status string
		if err := row.Columns(&cr.ReconciliationID, &cr.DriverID, &cr.ExpectedCashMinor, &cr.DeclaredCashMinor, &cr.DifferenceMinor, &status); err != nil {
			return fmt.Errorf("scan CashReconciliations: %w", err)
		}
		if cr.DifferenceMinor < 0 {
			shortfall := -cr.DifferenceMinor
			switch status {
			case "ACCEPTED", "VERIFIED":
				snap.CashReconAcceptedShortfallRepaidMinor += shortfall
			case "WRITE_OFF":
				snap.CashReconWrittenOffShortfallMinor += shortfall
			case "PENDING", "DISPUTED":
				snap.CashReconPendingOrDisputedShortfallMinor += shortfall
			}
		} else if cr.DifferenceMinor > 0 {
			snap.CashReconOverageMinor += cr.DifferenceMinor
		}
		if status == "DISPUTED" {
			snap.DisputedCashRecons = append(snap.DisputedCashRecons, cr)
		}
	}

	// 6. Verified AR payment & advance receipts and bank payouts (capped cleanly by ArLedgerEntries & PayoutBatches)
	arPayCap, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(-AmountMinor), 0)
		FROM ArLedgerEntries
		WHERE SupplierId = @entityId
		  AND UPPER(EntryType) = 'PAYMENT'
		  AND AmountMinor < 0
		  AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd})
	if err != nil {
		return fmt.Errorf("query ArLedgerEntries payments cap: %w", err)
	}

	if arPayCap > 0 {
		var arCardGL, arCashOfficeGL, arCashDriverGL, arBankGL int64
		arGLIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT jp.AccountCode, COALESCE(SUM(jp.AmountMinor), 0)
			      FROM JournalPostings jp
			      INNER JOIN JournalEntries je ON je.EntityId = jp.EntityId AND je.EntryId = jp.EntryId
			      WHERE je.EntityId = @entityId
			        AND je.PeriodId <= @periodId
			        AND jp.Direction = 'DEBIT'
			        AND UPPER(je.SourceType) IN ('AR_PAYMENT', 'AR_PAYMENT_RECEIVED')
			        AND jp.AccountCode IN ('PSP:GATEWAY', 'CASH:OFFICE', 'CASH:DRIVER', 'BANK:OPERATING')
			      GROUP BY jp.AccountCode`,
			Params: map[string]any{"entityId": snap.EntityID, "periodId": periodID},
		})
		defer arGLIter.Stop()
		for {
			row, err := arGLIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query AR payment GL splits: %w", err)
			}
			var acct string
			var amt int64
			if err := row.Columns(&acct, &amt); err != nil {
				return fmt.Errorf("scan AR payment GL split: %w", err)
			}
			switch acct {
			case gl.AccountCardPSPTransit:
				arCardGL += amt
			case gl.AccountCashOffice:
				arCashOfficeGL += amt
			case gl.AccountCashInTransit:
				arCashDriverGL += amt
			case gl.AccountBankOperating:
				arBankGL += amt
			}
		}
		if arCardGL+arCashOfficeGL+arCashDriverGL+arBankGL <= arPayCap {
			snap.VerifiedARCardGLMinor = arCardGL
			snap.VerifiedARCashOfficeGLMinor = arCashOfficeGL
			snap.VerifiedARCashDriverGLMinor = arCashDriverGL
			snap.VerifiedARBankGLMinor = arBankGL
		}
	}

	arAdvanceCap, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(AmountMinor), 0)
		FROM ArLedgerEntries
		WHERE SupplierId = @entityId
		  AND UPPER(EntryType) = 'ADVANCE_CREDIT'
		  AND AmountMinor > 0
		  AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd})
	if err != nil {
		return fmt.Errorf("query ArLedgerEntries advance cap: %w", err)
	}
	if arAdvanceCap > 0 {
		var advBankGL, advCardGL, advCashOfficeGL, advCashDriverGL int64
		advIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT jp.AccountCode, COALESCE(SUM(jp.AmountMinor), 0)
			      FROM JournalPostings jp
			      INNER JOIN JournalEntries je ON je.EntityId = jp.EntityId AND je.EntryId = jp.EntryId
			      WHERE je.EntityId = @entityId
			        AND je.PeriodId <= @periodId
			        AND jp.Direction = 'DEBIT'
			        AND UPPER(je.SourceType) IN ('AR_ADVANCE', 'AR_ADVANCE_CREDITED', 'RETAILER_CREDIT_TOPPED_UP')
			        AND jp.AccountCode IN ('PSP:GATEWAY', 'CASH:OFFICE', 'CASH:DRIVER', 'BANK:OPERATING')
			      GROUP BY jp.AccountCode`,
			Params: map[string]any{"entityId": snap.EntityID, "periodId": periodID},
		})
		defer advIter.Stop()
		for {
			row, err := advIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query AR advance GL splits: %w", err)
			}
			var acct string
			var amt int64
			if err := row.Columns(&acct, &amt); err != nil {
				return fmt.Errorf("scan AR advance GL split: %w", err)
			}
			switch acct {
			case gl.AccountBankOperating:
				advBankGL += amt
			case gl.AccountCardPSPTransit:
				advCardGL += amt
			case gl.AccountCashOffice:
				advCashOfficeGL += amt
			case gl.AccountCashInTransit:
				advCashDriverGL += amt
			}
		}
		if advBankGL+advCardGL+advCashOfficeGL+advCashDriverGL <= arAdvanceCap {
			snap.VerifiedCreditTopupBankMinor += advBankGL
			snap.VerifiedARCardGLMinor += advCardGL
			snap.VerifiedARCashOfficeGLMinor += advCashOfficeGL
			snap.VerifiedARCashDriverGLMinor += advCashDriverGL
		}
	}

	paidPayoutCap, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(NetPayoutMinor), 0)
		FROM PayoutBatches
		WHERE SupplierId = @entityId
		  AND UPPER(Status) = 'PAID'
		  AND (@hasCutoff = FALSE OR CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd})
	if err != nil {
		return fmt.Errorf("query PayoutBatches paid cap: %w", err)
	}
	if paidPayoutCap > 0 {
		bankOut, err := querySingleInt64(ctx, ro, `
			SELECT COALESCE(SUM(jp.AmountMinor), 0)
			FROM JournalPostings jp
			INNER JOIN JournalEntries je ON je.EntityId = jp.EntityId AND je.EntryId = jp.EntryId
			WHERE je.EntityId = @entityId
			  AND je.PeriodId <= @periodId
			  AND jp.Direction = 'CREDIT'
			  AND jp.AccountCode = 'BANK:OPERATING'
			  AND UPPER(je.SourceType) IN ('PAYOUT_PAID', 'SUPPLIER_PAYOUT_RELEASED')`,
			map[string]any{"entityId": snap.EntityID, "periodId": periodID})
		if err != nil {
			return fmt.Errorf("query bank payout GL: %w", err)
		}
		if bankOut <= paidPayoutCap {
			snap.BankPaidPayoutsMinor = bankOut
		}
	}

	return nil
}

func loadTaxFiscalSnapshot(ctx context.Context, ro *spanner.ReadOnlyTransaction, snap *Snapshot, hasCutoff bool, periodEnd time.Time) error {
	// 1. Registered VAT across OrderLineFiscalSnapshots for fiscalized orders.
	snapIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(DISTINCT o.OrderId), COALESCE(SUM(ol.VatMinor), 0)
		      FROM OrderLineFiscalSnapshots ol
		      INNER JOIN Orders o ON ol.OrderId = o.OrderId
		      WHERE o.SupplierId = @entityId
		        AND UPPER(COALESCE(o.Status, '')) != 'CANCELLED'
		        AND (@hasCutoff = FALSE OR o.CreatedAt < @periodEnd)
		        AND (
		          UPPER(COALESCE(o.FiscalStatus, '')) IN ('SUCCESS', 'ISSUED', 'REGISTERED', 'COMPLETED', 'FISCALIZED')
		          OR UPPER(COALESCE(o.Status, '')) = 'FISCALIZED'
		          OR EXISTS (
		            SELECT 1 FROM OrderFiscalReceipts ofr
		            WHERE ofr.OrderId = o.OrderId
		              AND ofr.SupplierId = @entityId
		              AND UPPER(ofr.Status) IN ('SUCCESS', 'ISSUED', 'REGISTERED', 'COMPLETED')
		          )
		        )`,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer snapIter.Stop()
	if row, err := snapIter.Next(); err == nil {
		if err := row.Columns(&snap.FiscalizedOrderCount, &snap.SnapshotVatMinor); err != nil {
			return fmt.Errorf("scan fiscal VAT snapshot sum: %w", err)
		}
	} else if err != iterator.Done {
		return fmt.Errorf("query fiscal VAT snapshot sum: %w", err)
	}

	// 2. Credit note VAT reversals.
	if val, err := querySingleInt64(ctx, ro, `
		SELECT COALESCE(SUM(cn.TotalVatMinor), 0)
		FROM CreditNotes cn
		INNER JOIN Orders o ON cn.OrderId = o.OrderId
		WHERE o.SupplierId = @entityId
		  AND UPPER(cn.Status) IN ('ISSUED', 'FISCAL_PENDING', 'COMPLETED')
		  AND (@hasCutoff = FALSE OR cn.CreatedAt < @periodEnd)`,
		map[string]any{"entityId": snap.EntityID, "hasCutoff": hasCutoff, "periodEnd": periodEnd}); err != nil {
		return fmt.Errorf("query credit notes VAT: %w", err)
	} else {
		snap.CreditNoteVatMinor = val
	}

	// 3. Detect fiscalized non-zero orders missing OrderLineFiscalSnapshots when GL TAX:VAT:OUTPUT > 0 or receipt issued.
	missingIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT o.OrderId, o.TotalMinor
		      FROM Orders o
		      WHERE o.SupplierId = @entityId
		        AND o.TotalMinor > 0
		        AND UPPER(COALESCE(o.Status, '')) != 'CANCELLED'
		        AND (@hasCutoff = FALSE OR o.CreatedAt < @periodEnd)
		        AND EXISTS (
		          SELECT 1 FROM OrderFiscalReceipts ofr
		          WHERE ofr.OrderId = o.OrderId
		            AND ofr.SupplierId = @entityId
		            AND UPPER(ofr.Status) IN ('SUCCESS', 'ISSUED', 'REGISTERED', 'COMPLETED')
		        )
		        AND NOT EXISTS (
		          SELECT 1 FROM OrderLineFiscalSnapshots ol
		          WHERE ol.OrderId = o.OrderId
		        )`,
		Params: map[string]any{
			"entityId":  snap.EntityID,
			"hasCutoff": hasCutoff,
			"periodEnd": periodEnd,
		},
	})
	defer missingIter.Stop()
	for {
		row, err := missingIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query missing fiscal snapshot orders: %w", err)
		}
		var m MissingFiscalSnapshotOrder
		if err := row.Columns(&m.OrderID, &m.TotalMinor); err != nil {
			return fmt.Errorf("scan missing fiscal snapshot order: %w", err)
		}
		snap.MissingSnapshotOrders = append(snap.MissingSnapshotOrders, m)
	}
	return nil
}

func querySingleInt64(ctx context.Context, ro *spanner.ReadOnlyTransaction, sql string, params map[string]any) (int64, error) {
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var val int64
	if err := row.Columns(&val); err != nil {
		return 0, err
	}
	return val, nil
}

func scanRunRow(row *spanner.Row) (ReconciliationRun, error) {
	var run ReconciliationRun
	var checksStr, summaryStr spanner.NullString
	var completedAt spanner.NullTime
	if err := row.Columns(
		&run.EntityID, &run.RunID, &run.PeriodID, &run.Mode, &run.Status,
		&checksStr, &summaryStr, &run.BreakCount, &run.ExposureMinor,
		&run.TriggeredBy, &run.StartedAt, &completedAt,
	); err != nil {
		return ReconciliationRun{}, fmt.Errorf("scan ReconciliationRun: %w", err)
	}
	if completedAt.Valid {
		run.CompletedAt = completedAt.Time
	}
	if checksStr.Valid && strings.TrimSpace(checksStr.StringVal) != "" {
		if err := json.Unmarshal([]byte(checksStr.StringVal), &run.Checks); err != nil {
			return ReconciliationRun{}, fmt.Errorf("unmarshal ChecksJson: %w", err)
		}
		for _, chk := range run.Checks {
			if chk.Currency != "" {
				run.Currency = chk.Currency
				break
			}
		}
	}
	if run.Checks == nil {
		run.Checks = make(map[string]DimensionCheckSummary)
	}
	if summaryStr.Valid && strings.TrimSpace(summaryStr.StringVal) != "" {
		if err := json.Unmarshal([]byte(summaryStr.StringVal), &run.Summary); err != nil {
			return ReconciliationRun{}, fmt.Errorf("unmarshal SummaryJson: %w", err)
		}
	}
	return run, nil
}

func scanBreakRow(row *spanner.Row) (ReconciliationBreak, error) {
	var b ReconciliationBreak
	var owner, note, ctID spanner.NullString
	var slaDue spanner.NullTime
	if err := row.Columns(
		&b.EntityID, &b.BreakID, &b.RunID, &b.Dimension, &b.ObjectType, &b.ObjectID,
		&b.ExpectedMinor, &b.ActualMinor, &b.DeltaMinor, &b.Currency, &b.Severity, &b.Status,
		&owner, &slaDue, &note, &ctID, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return ReconciliationBreak{}, fmt.Errorf("scan ReconciliationBreak: %w", err)
	}
	if owner.Valid {
		b.OwnerID = owner.StringVal
	}
	if slaDue.Valid {
		t := slaDue.Time
		b.SlaDueAt = &t
	}
	if note.Valid {
		b.ResolutionNote = note.StringVal
	}
	if ctID.Valid {
		b.ControlTowerExceptionID = ctID.StringVal
	}
	return b, nil
}

func computePeriodCutoff(ctx context.Context, entityID, periodID string) (bool, time.Time) {
	p := strings.TrimSpace(periodID)
	if len(p) != 7 || p[4] != '-' {
		return false, time.Time{}
	}
	loc := gl.ResolveEntityLocation(ctx, entityID, "")
	start, err := time.ParseInLocation("2006-01", p, loc)
	if err != nil {
		return false, time.Time{}
	}
	return true, start.AddDate(0, 1, 0).UTC()
}

type entityStampedOutboxBuf struct {
	entityID string
	inner    *outbox.SpannerTxnBuffer
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(e.SupplierID) == "" || e.SupplierID == outbox.PlatformSupplierID {
		e.SupplierID = b.entityID
	}
	return b.inner.BufferOutbox(ctx, e)
}

func firstPositiveInt64(vals ...int64) int64 {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
}

func nullableStr(v string) spanner.NullString {
	v = strings.TrimSpace(v)
	if v == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}

func nullableTime(t *time.Time) spanner.NullTime {
	if t == nil || t.IsZero() {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: t.UTC(), Valid: true}
}

func formatNullableTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
