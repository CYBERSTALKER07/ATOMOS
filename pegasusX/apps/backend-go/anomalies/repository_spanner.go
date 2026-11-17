package anomalies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// NewSpannerService constructs an anomalies Service backed by Cloud Spanner.
func NewSpannerService(client *spanner.Client, cfg ServiceConfig) *Service {
	if client == nil {
		return NewService(nil, cfg)
	}
	return NewService(NewSpannerRepository(client), cfg)
}

// SpannerRepository implements Repository over Cloud Spanner using
// ReadOnlyTransaction queries and atomic ReadWriteTransaction mutations.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed anomaly repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type tenantStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	tenantID string
}

func (b *tenantStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.tenantID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) LoadDetectorDataset(ctx context.Context, tenantID, entityID string) (DetectorDataset, error) {
	if r == nil || r.client == nil {
		return DetectorDataset{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	eid := strings.TrimSpace(entityID)
	ds := DetectorDataset{TenantID: tid, EntityID: eid}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	var err error
	if ds.CashierSessions, err = loadCashierSessions(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.DepotVaultEntries, err = loadDepotVaultEntries(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.ShopClosedLogs, err = loadShopClosedLogs(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.Orders, err = loadOrdersWithRetailerTIN(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.ReconBreaks, err = loadStockGLBreaks(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.CycleCounts, err = loadCycleCounts(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.InventoryAdjustments, err = loadInventoryAdjustments(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.CostLayers, err = loadCostLayers(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.ProductPrices, err = loadProductPrices(ctx, roTxn, tid); err != nil {
		return DetectorDataset{}, err
	}
	if ds.ManualJournals, err = loadManualJournals(ctx, roTxn, tid, eid); err != nil {
		return DetectorDataset{}, err
	}
	return ds, nil
}

func loadCashierSessions(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]CashierBlindCountRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT SessionId, SupplierId, DepotId, DriverId, RouteId,
		             CountedCashMinor, ExpectedCashMinor, VarianceMinor, Status, CreatedAt
		      FROM CashierBlindCountSessions
		      WHERE SupplierId = @tid
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []CashierBlindCountRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query CashierBlindCountSessions: %w", err)
		}
		var rec CashierBlindCountRecord
		if err := row.Columns(
			&rec.SessionID, &rec.SupplierID, &rec.DepotID, &rec.DriverID, &rec.RouteID,
			&rec.CountedCashMinor, &rec.ExpectedCashMinor, &rec.VarianceMinor, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan CashierBlindCountSessions: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadDepotVaultEntries(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]DepotVaultLedgerRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT VaultEntryId, DepotId, SupplierId, COALESCE(SessionId, ''), EntryType,
		             AmountMinor, RunningBalanceMinor, CreatedAt
		      FROM DepotVaultLedgerEntries
		      WHERE SupplierId = @tid
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []DepotVaultLedgerRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query DepotVaultLedgerEntries: %w", err)
		}
		var rec DepotVaultLedgerRecord
		if err := row.Columns(
			&rec.VaultEntryID, &rec.DepotID, &rec.SupplierID, &rec.SessionID, &rec.EntryType,
			&rec.AmountMinor, &rec.RunningBalanceMinor, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan DepotVaultLedgerEntries: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadShopClosedLogs(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ShopClosedLogRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT l.OrderId, l.EventId, l.Actor, l.Action, l.Payload,
		             o.TotalMinor, COALESCE(o.H3Cell, ''), o.RetailerId, COALESCE(r.H3Cell, ''),
		             COALESCE(o.ProximityMethod, ''), COALESCE(o.ShopClosedResolution, ''), l.CreatedAt
		      FROM OrderShopClosedLog l
		      JOIN Orders o ON o.OrderId = l.OrderId
		      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
		      WHERE o.SupplierId = @tid
		      ORDER BY l.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []ShopClosedLogRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query OrderShopClosedLog: %w", err)
		}
		var (
			rec     ShopClosedLogRecord
			payload spanner.NullJSON
		)
		if err := row.Columns(
			&rec.OrderID, &rec.EventID, &rec.Actor, &rec.Action, &payload,
			&rec.OrderTotalMinor, &rec.OrderH3Cell, &rec.RetailerID, &rec.RetailerH3Cell,
			&rec.ProximityMethod, &rec.ShopClosedRes, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan OrderShopClosedLog: %w", err)
		}
		if payload.Valid {
			if raw, mErr := json.Marshal(payload.Value); mErr == nil {
				rec.PayloadJSON = string(raw)
			}
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadOrdersWithRetailerTIN(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]OrderRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, o.RetailerId, COALESCE(r.Name, ''),
		             COALESCE(r.NormalizedTaxId, r.TaxId, ''), o.Status,
		             o.TotalMinor, o.OriginalTotalMinor, o.Currency, o.LineItemsJson, o.CreatedAt
		      FROM Orders o
		      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
		      WHERE o.SupplierId = @tid
		      ORDER BY o.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []OrderRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Orders: %w", err)
		}
		var (
			rec       OrderRecord
			lineBytes []byte
		)
		if err := row.Columns(
			&rec.OrderID, &rec.SupplierID, &rec.RetailerID, &rec.RetailerName,
			&rec.NormalizedTaxID, &rec.Status, &rec.TotalMinor, &rec.OriginalTotalMinor,
			&rec.Currency, &lineBytes, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan Orders: %w", err)
		}
		rec.LineItemsJSON = string(lineBytes)
		out = append(out, rec)
	}
	return out, nil
}

func loadStockGLBreaks(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ReconBreakRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT BreakId, RunId, Dimension, ObjectType, ObjectId,
		             ExpectedMinor, ActualMinor, DeltaMinor, Currency, Severity, Status, CreatedAt
		      FROM ReconciliationBreaks
		      WHERE (EntityId = @tid OR EntityId IN (SELECT EntityId FROM LegalEntities WHERE TenantId = @tid))
		        AND Dimension = 'STOCK_GL'
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []ReconBreakRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ReconciliationBreaks: %w", err)
		}
		var rec ReconBreakRecord
		if err := row.Columns(
			&rec.BreakID, &rec.RunID, &rec.Dimension, &rec.ObjectType, &rec.ObjectID,
			&rec.ExpectedMinor, &rec.ActualMinor, &rec.DeltaMinor, &rec.Currency,
			&rec.Severity, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ReconciliationBreaks: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadCycleCounts(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]CycleCountRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT c.CountId, c.WarehouseId, c.LocationId, c.ProductId, COALESCE(c.LotId, ''),
		             c.ExpectedQty, c.CountedQty, c.VarianceQty, COALESCE(c.ReasonCode, ''), c.Status, c.CreatedAt
		      FROM CycleCounts c
		      JOIN Warehouses w ON w.WarehouseId = c.WarehouseId
		      WHERE w.SupplierId = @tid
		      ORDER BY c.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []CycleCountRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query CycleCounts: %w", err)
		}
		var (
			rec        CycleCountRecord
			countedQty spanner.NullInt64
			varQty     spanner.NullInt64
		)
		if err := row.Columns(
			&rec.CountID, &rec.WarehouseID, &rec.LocationID, &rec.ProductID, &rec.LotID,
			&rec.ExpectedQty, &countedQty, &varQty, &rec.ReasonCode, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan CycleCounts: %w", err)
		}
		if countedQty.Valid {
			rec.HasCounted = true
			rec.CountedQty = countedQty.Int64
		}
		if varQty.Valid {
			rec.VarianceQty = varQty.Int64
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadInventoryAdjustments(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]InventoryAdjustmentRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ia.AdjustmentId, ia.WarehouseId, ia.ProductId, COALESCE(ia.LotId, ''),
		             ia.DeltaQty, COALESCE(ia.ReasonCode, ''), ia.Status, ia.CreatedAt
		      FROM InventoryAdjustments ia
		      JOIN Warehouses w ON w.WarehouseId = ia.WarehouseId
		      WHERE w.SupplierId = @tid
		      ORDER BY ia.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []InventoryAdjustmentRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query InventoryAdjustments: %w", err)
		}
		var rec InventoryAdjustmentRecord
		if err := row.Columns(
			&rec.AdjustmentID, &rec.WarehouseID, &rec.ProductID, &rec.LotID,
			&rec.DeltaQty, &rec.ReasonCode, &rec.Status, &rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan InventoryAdjustments: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadCostLayers(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]CostLayerSummary, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, UnitCostMinor
		      FROM InventoryCostLayers
		      WHERE (EntityId = @tid OR EntityId IN (SELECT EntityId FROM LegalEntities WHERE TenantId = @tid))
		      ORDER BY ReceivedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer iter.Stop()
	var out []CostLayerSummary
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query InventoryCostLayers: %w", err)
		}
		var rec CostLayerSummary
		if err := row.Columns(&rec.ProductID, &rec.UnitCostMinor); err != nil {
			return nil, fmt.Errorf("scan InventoryCostLayers: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadProductPrices(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ProductPriceRecord, error) {
	var out []ProductPriceRecord

	pIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, PriceMinor
		      FROM Products
		      WHERE SupplierId = @tid
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer pIter.Stop()
	for {
		row, err := pIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Products: %w", err)
		}
		var pid string
		var price int64
		if err := row.Columns(&pid, &price); err != nil {
			return nil, fmt.Errorf("scan Products: %w", err)
		}
		out = append(out, ProductPriceRecord{ProductID: pid, SKU: pid, UnitPriceMinor: price})
	}

	plIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT pli.Sku, pli.UnitPriceMinor
		      FROM PriceListItems pli
		      JOIN PriceLists pl ON pl.PriceListId = pli.PriceListId
		      WHERE pl.SupplierId = @tid
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(ScanQueryCap)},
	})
	defer plIter.Stop()
	for {
		row, err := plIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PriceListItems: %w", err)
		}
		var sku string
		var price int64
		if err := row.Columns(&sku, &price); err != nil {
			return nil, fmt.Errorf("scan PriceListItems: %w", err)
		}
		out = append(out, ProductPriceRecord{ProductID: sku, SKU: sku, UnitPriceMinor: price})
	}
	return out, nil
}

func loadManualJournals(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid, eid string) ([]ManualJournalRecord, error) {
	sql := `SELECT je.EntityId, je.EntryId, je.PeriodId, je.SourceType, je.SourceId,
	               COALESCE(je.Memo, ''), je.Currency, je.PostedBy, je.OccurredAt, je.PostedAt,
	               COALESCE((SELECT SUM(jp.AmountMinor)
	                         FROM JournalPostings jp
	                         WHERE jp.EntityId = je.EntityId
	                           AND jp.EntryId = je.EntryId
	                           AND jp.Direction = 'DEBIT'), 0)
	        FROM JournalEntries je
	        WHERE (je.EntityId = @tid OR je.EntityId IN (SELECT EntityId FROM LegalEntities WHERE TenantId = @tid))
	          AND je.SourceType IN ('MANUAL', 'MANUAL_JOURNAL')`
	params := map[string]any{"tid": tid, "lim": int64(ScanQueryCap)}
	if eid != "" {
		sql += ` AND (je.EntityId = @eid OR je.LegalEntityId = @eid)`
		params["eid"] = eid
	}
	sql += ` ORDER BY je.PostedAt DESC LIMIT @lim`

	iter := roTxn.Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()
	var out []ManualJournalRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query JournalEntries: %w", err)
		}
		var rec ManualJournalRecord
		if err := row.Columns(
			&rec.EntityID, &rec.EntryID, &rec.PeriodID, &rec.SourceType, &rec.SourceID,
			&rec.Memo, &rec.Currency, &rec.PostedBy, &rec.OccurredAt, &rec.PostedAt,
			&rec.DebitTotalMinor,
		); err != nil {
			return nil, fmt.Errorf("scan JournalEntries: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *SpannerRepository) SaveDetectedCases(
	ctx context.Context,
	tenantID string,
	candidates []AnomalyCase,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) ([]AnomalyCase, int, error) {
	if r == nil || r.client == nil {
		return nil, 0, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	var created []AnomalyCase
	var deduped int

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		created = nil
		deduped = 0

		existingKeys := make(map[string]struct{})
		eIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT RuleCode, ObjectType, ObjectId FROM AnomalyCases WHERE TenantId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, qErr := eIter.Next()
			if errors.Is(qErr, iterator.Done) {
				break
			}
			if qErr != nil {
				eIter.Stop()
				return fmt.Errorf("query existing AnomalyCases: %w", qErr)
			}
			var rc, ot, oid string
			if sErr := row.Columns(&rc, &ot, &oid); sErr != nil {
				eIter.Stop()
				return fmt.Errorf("scan existing AnomalyCases: %w", sErr)
			}
			existingKeys[FindingKey(rc, ot, oid)] = struct{}{}
		}
		eIter.Stop()

		var muts []*spanner.Mutation
		buf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &tenantStampedOutboxBuf{inner: buf, tenantID: tid}

		for _, cand := range candidates {
			fk := FindingKey(cand.RuleCode, cand.ObjectType, cand.ObjectID)
			if _, exists := existingKeys[fk]; exists {
				deduped++
				continue
			}
			existingKeys[fk] = struct{}{}

			muts = append(muts, spanner.InsertOrUpdateMap("AnomalyCases", map[string]any{
				"TenantId":       cand.TenantID,
				"CaseId":         cand.CaseID,
				"RuleCode":       cand.RuleCode,
				"Category":       cand.Category,
				"Severity":       cand.Severity,
				"Status":         cand.Status,
				"ObjectType":     cand.ObjectType,
				"ObjectId":       cand.ObjectID,
				"ExposureMinor":  cand.ExposureMinor,
				"ScoreBps":       cand.ScoreBps,
				"EvidenceJson":   cand.EvidenceJSON,
				"PlaybookAction": nullableStr(cand.PlaybookAction),
				"ProposalId":     nullableStr(cand.ProposalID),
				"AssignedTo":     nullableStr(cand.AssignedTo),
				"ResolutionNote": nullableStr(cand.ResolutionNote),
				"ResolvedBy":     nullableStr(cand.ResolvedBy),
				"ResolvedAt":     nullableTime(cand.ResolvedAt),
				"DetectedAt":     cand.DetectedAt.UTC(),
				"UpdatedAt":      spanner.CommitTimestamp,
			}))

			if eErr := outbox.EmitJSON(txCtx, stamped, events.AggregateAnomalyCase, cand.CaseID, events.TopicExceptions, events.AnomalyCaseEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventAnomalyDetected,
					Version:   1,
					Timestamp: cand.UpdatedAt.UTC().Format(time.RFC3339Nano),
				},
				TenantID:       cand.TenantID,
				CaseID:         cand.CaseID,
				RuleCode:       cand.RuleCode,
				Category:       cand.Category,
				Severity:       cand.Severity,
				Status:         cand.Status,
				ObjectType:     cand.ObjectType,
				ObjectID:       cand.ObjectID,
				ExposureMinor:  cand.ExposureMinor,
				ScoreBps:       cand.ScoreBps,
				PlaybookAction: cand.PlaybookAction,
				DetectedAt:     cand.DetectedAt.UTC().Format(time.RFC3339Nano),
				ActorID:        auditIn.ActorID,
				OccurredAt:     cand.UpdatedAt.UTC().Format(time.RFC3339Nano),
			}); eErr != nil {
				return fmt.Errorf("emit anomaly.case.detected: %w", eErr)
			}
			created = append(created, cand)
		}

		if len(muts) > 0 {
			if err := txn.BufferWrite(muts); err != nil {
				return fmt.Errorf("buffer AnomalyCases: %w", err)
			}
			if err := buf.Flush(txCtx); err != nil {
				return fmt.Errorf("flush anomaly outbox: %w", err)
			}
			if auditRec != nil {
				if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn); err != nil {
					return fmt.Errorf("audit anomalies.scan: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return created, deduped, nil
}

func (r *SpannerRepository) ListCases(ctx context.Context, f ListFilter) ([]AnomalyCase, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	var clauses []string
	params := map[string]any{
		"tid": strings.TrimSpace(f.TenantID),
		"lim": int64(clampLimit(f.Limit)),
	}
	clauses = append(clauses, "TenantId = @tid")
	if st := strings.ToUpper(strings.TrimSpace(f.Status)); st != "" {
		clauses = append(clauses, "Status = @status")
		params["status"] = st
	}
	if sev := strings.ToUpper(strings.TrimSpace(f.Severity)); sev != "" {
		clauses = append(clauses, "Severity = @severity")
		params["severity"] = sev
	}
	if cat := strings.ToUpper(strings.TrimSpace(f.Category)); cat != "" {
		clauses = append(clauses, "Category = @category")
		params["category"] = cat
	}
	if rc := strings.ToUpper(strings.TrimSpace(f.RuleCode)); rc != "" {
		clauses = append(clauses, "RuleCode = @ruleCode")
		params["ruleCode"] = rc
	}

	sql := fmt.Sprintf(`SELECT TenantId, CaseId, RuleCode, Category, Severity, Status, ObjectType, ObjectId,
	                           ExposureMinor, ScoreBps, COALESCE(EvidenceJson, ''), COALESCE(PlaybookAction, ''),
	                           COALESCE(ProposalId, ''), COALESCE(AssignedTo, ''), COALESCE(ResolutionNote, ''),
	                           COALESCE(ResolvedBy, ''), ResolvedAt, DetectedAt, UpdatedAt
	                    FROM AnomalyCases
	                    WHERE %s
	                    ORDER BY DetectedAt DESC, CaseId ASC
	                    LIMIT @lim`, strings.Join(clauses, " AND "))

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []AnomalyCase
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query AnomalyCases: %w", err)
		}
		c, err := scanAnomalyCaseRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *SpannerRepository) GetCase(ctx context.Context, tenantID, caseID string) (AnomalyCase, bool, error) {
	if r == nil || r.client == nil {
		return AnomalyCase{}, false, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	row, err := roTxn.ReadRow(ctx, "AnomalyCases", spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(caseID)}, anomalyCaseColumns())
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound || errors.Is(err, spanner.ErrRowNotFound) {
			return AnomalyCase{}, false, nil
		}
		return AnomalyCase{}, false, fmt.Errorf("read AnomalyCases: %w", err)
	}
	c, err := scanAnomalyCaseRow(row)
	if err != nil {
		return AnomalyCase{}, false, err
	}
	return c, true, nil
}

func (r *SpannerRepository) ResolveCase(
	ctx context.Context,
	tenantID, caseID string,
	mutate func(cur AnomalyCase) (AnomalyCase, error),
	auditRec AuditRecorder,
	auditIn func(updated AnomalyCase) AuditRecordInput,
	resolvedAt time.Time,
) (AnomalyCase, error) {
	if r == nil || r.client == nil {
		return AnomalyCase{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	cid := strings.TrimSpace(caseID)
	var updated AnomalyCase

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rErr := txn.ReadRow(txCtx, "AnomalyCases", spanner.Key{tid, cid}, anomalyCaseColumns())
		if rErr != nil {
			if spanner.ErrCode(rErr) == codes.NotFound || errors.Is(rErr, spanner.ErrRowNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("read AnomalyCases for resolve: %w", rErr)
		}
		cur, sErr := scanAnomalyCaseRow(row)
		if sErr != nil {
			return sErr
		}
		next, mErr := mutate(cur)
		if mErr != nil {
			return mErr
		}
		updated = next

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("AnomalyCases", map[string]any{
				"TenantId":       tid,
				"CaseId":         cid,
				"Status":         next.Status,
				"PlaybookAction": nullableStr(next.PlaybookAction),
				"ProposalId":     nullableStr(next.ProposalID),
				"AssignedTo":     nullableStr(next.AssignedTo),
				"ResolutionNote": nullableStr(next.ResolutionNote),
				"ResolvedBy":     nullableStr(next.ResolvedBy),
				"ResolvedAt":     nullableTime(next.ResolvedAt),
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("buffer AnomalyCases update: %w", err)
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &tenantStampedOutboxBuf{inner: buf, tenantID: tid}
		resAtStr := resolvedAt.UTC().Format(time.RFC3339Nano)
		if next.ResolvedAt != nil {
			resAtStr = next.ResolvedAt.UTC().Format(time.RFC3339Nano)
		}
		if eErr := outbox.EmitJSON(txCtx, stamped, events.AggregateAnomalyCase, cid, events.TopicExceptions, events.AnomalyCaseEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventAnomalyResolved,
				Version:   1,
				Timestamp: resolvedAt.UTC().Format(time.RFC3339Nano),
			},
			TenantID:       tid,
			CaseID:         cid,
			RuleCode:       next.RuleCode,
			Category:       next.Category,
			Severity:       next.Severity,
			Status:         next.Status,
			ObjectType:     next.ObjectType,
			ObjectID:       next.ObjectID,
			ExposureMinor:  next.ExposureMinor,
			ScoreBps:       next.ScoreBps,
			PlaybookAction: next.PlaybookAction,
			ProposalID:     next.ProposalID,
			AssignedTo:     next.AssignedTo,
			ResolutionNote: next.ResolutionNote,
			ResolvedBy:     next.ResolvedBy,
			ResolvedAt:     resAtStr,
			DetectedAt:     next.DetectedAt.UTC().Format(time.RFC3339Nano),
			ActorID:        next.ResolvedBy,
			OccurredAt:     resolvedAt.UTC().Format(time.RFC3339Nano),
		}); eErr != nil {
			return fmt.Errorf("emit anomaly.case.resolved: %w", eErr)
		}
		if err := buf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush anomaly resolve outbox: %w", err)
		}
		if auditRec != nil && auditIn != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn(next)); err != nil {
				return fmt.Errorf("audit anomaly.resolve: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return AnomalyCase{}, err
	}
	return updated, nil
}

func anomalyCaseColumns() []string {
	return []string{
		"TenantId", "CaseId", "RuleCode", "Category", "Severity", "Status",
		"ObjectType", "ObjectId", "ExposureMinor", "ScoreBps", "EvidenceJson",
		"PlaybookAction", "ProposalId", "AssignedTo", "ResolutionNote", "ResolvedBy",
		"ResolvedAt", "DetectedAt", "UpdatedAt",
	}
}

func scanAnomalyCaseRow(row *spanner.Row) (AnomalyCase, error) {
	var (
		c                             AnomalyCase
		ev, pb, pid, ass, note, resBy spanner.NullString
		resAt                         spanner.NullTime
	)
	if err := row.Columns(
		&c.TenantID, &c.CaseID, &c.RuleCode, &c.Category, &c.Severity, &c.Status,
		&c.ObjectType, &c.ObjectID, &c.ExposureMinor, &c.ScoreBps, &ev,
		&pb, &pid, &ass, &note, &resBy, &resAt, &c.DetectedAt, &c.UpdatedAt,
	); err != nil {
		return AnomalyCase{}, fmt.Errorf("scan AnomalyCases row: %w", err)
	}
	c.EvidenceJSON = ev.StringVal
	c.Evidence = decodeEvidenceMap(c.EvidenceJSON)
	c.PlaybookAction = pb.StringVal
	c.ProposalID = pid.StringVal
	c.AssignedTo = ass.StringVal
	c.ResolutionNote = note.StringVal
	c.ResolvedBy = resBy.StringVal
	if resAt.Valid {
		t := resAt.Time.UTC()
		c.ResolvedAt = &t
	}
	c.DetectedAt = c.DetectedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()
	return c, nil
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullableTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC()
}
