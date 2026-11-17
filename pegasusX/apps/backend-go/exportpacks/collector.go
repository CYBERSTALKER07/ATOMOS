package exportpacks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultCollectorBatchSize = 2000

// MemoryCollector returns pre-seeded PackDatasets (or empty datasets) for isolated unit tests.
type MemoryCollector struct {
	Datasets *PackDatasets
}

func (m *MemoryCollector) Collect(_ context.Context, _, _ string, _, _ time.Time) (*PackDatasets, error) {
	if m == nil || m.Datasets == nil {
		return &PackDatasets{}, nil
	}
	return m.Datasets, nil
}

// SpannerCollector collects unlimited rows across all 15 audit tables inside a single
// ReadOnlyTransaction snapshot using deterministic keyset pagination.
type SpannerCollector struct {
	client    *spanner.Client
	batchSize int
}

// NewSpannerCollector creates a keyset-paginated Spanner collector.
func NewSpannerCollector(client *spanner.Client, batchSize int) *SpannerCollector {
	if batchSize <= 0 {
		batchSize = defaultCollectorBatchSize
	}
	return &SpannerCollector{
		client:    client,
		batchSize: batchSize,
	}
}

func (c *SpannerCollector) Collect(ctx context.Context, entityID, periodID string, from, to time.Time) (*PackDatasets, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("%w: spanner collector unavailable", ErrRepoUnavailable)
	}
	eid := strings.TrimSpace(entityID)
	pid := strings.TrimSpace(periodID)
	if eid == "" {
		return nil, fmt.Errorf("%w: entity_id required for dataset collection", ErrInvalidInput)
	}

	batch := c.batchSize
	if batch <= 0 {
		batch = defaultCollectorBatchSize
	}

	roTxn := c.client.ReadOnlyTransaction()
	defer roTxn.Close()

	return collectStandardDatasetsInTxn(ctx, roTxn, eid, pid, from, to, batch)
}

func collectStandardDatasetsInTxn(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid, pid string, from, to time.Time, batch int) (*PackDatasets, error) {
	out := &PackDatasets{}
	var err error

	if out.GlAccounts, err = collectGlAccounts(ctx, roTxn, eid, batch); err != nil {
		return nil, err
	}
	if out.JournalEntries, err = collectJournalEntries(ctx, roTxn, eid, pid, from, to, batch); err != nil {
		return nil, err
	}
	if out.JournalPostings, err = collectJournalPostings(ctx, roTxn, eid, pid, from, to, batch); err != nil {
		return nil, err
	}
	if out.ArInvoices, err = collectArInvoices(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.InventoryCostLayers, err = collectInventoryCostLayers(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.InventoryMovements, err = collectInventoryMovements(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.EsfDocuments, err = collectEsfDocuments(ctx, roTxn, eid, pid, from, to, batch); err != nil {
		return nil, err
	}
	if out.CashierBlindCountSessions, err = collectCashierBlindCountSessions(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.DepotVaultLedgerEntries, err = collectDepotVaultLedgerEntries(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.PaymentLedgerEntries, err = collectPaymentLedgerEntries(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.ApprovalRequests, err = collectApprovalRequests(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.ApprovalDecisions, err = collectApprovalDecisions(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.AuditChainAnchors, err = collectAuditChainAnchors(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	if out.ReconciliationRuns, err = collectReconciliationRuns(ctx, roTxn, eid, pid, from, to, batch); err != nil {
		return nil, err
	}
	if out.ReconciliationBreaks, err = collectReconciliationBreaks(ctx, roTxn, eid, from, to, batch); err != nil {
		return nil, err
	}
	return out, nil
}

func runKeysetPages[K any](
	ctx context.Context,
	roTxn *spanner.ReadOnlyTransaction,
	batchSize int,
	tableName string,
	buildStmt func(hasCursor bool, cursor K) spanner.Statement,
	scan func(row *spanner.Row) (K, error),
) error {
	var cursor K
	hasCursor := false
	for {
		stmt := buildStmt(hasCursor, cursor)
		stmt.Params["lim"] = int64(batchSize)
		iter := roTxn.Query(ctx, stmt)
		pageRows := 0
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return fmt.Errorf("collect %s: %w", tableName, err)
			}
			nextCursor, serr := scan(row)
			if serr != nil {
				iter.Stop()
				return fmt.Errorf("scan %s: %w", tableName, serr)
			}
			cursor = nextCursor
			hasCursor = true
			pageRows++
		}
		iter.Stop()
		if pageRows < batchSize {
			break
		}
	}
	return nil
}

func optTimePtr(nt spanner.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time.UTC()
	return &t
}

func collectGlAccounts(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, batch int) ([]GlAccountRow, error) {
	var rows []GlAccountRow
	err := runKeysetPages(ctx, roTxn, batch, "GlAccounts", func(has bool, lastCode string) spanner.Statement {
		sql := `SELECT EntityId, AccountCode, Class, Name, COALESCE(ParentCode, ''), Currency, IsControl, CountryPack, ActiveFrom, ActiveTo, CreatedAt, UpdatedAt
		        FROM GlAccounts WHERE EntityId = @eid`
		params := map[string]any{"eid": eid}
		if has {
			sql += ` AND AccountCode > @k1`
			params["k1"] = lastCode
		}
		sql += ` ORDER BY AccountCode ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r GlAccountRow
		var af, at spanner.NullTime
		if err := row.Columns(&r.EntityID, &r.AccountCode, &r.Class, &r.Name, &r.ParentCode, &r.Currency, &r.IsControl, &r.CountryPack, &af, &at, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.ActiveFrom = optTimePtr(af)
		r.ActiveTo = optTimePtr(at)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.AccountCode, nil
	})
	return rows, err
}

func collectJournalEntries(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid, pid string, from, to time.Time, batch int) ([]JournalEntryRow, error) {
	var rows []JournalEntryRow
	err := runKeysetPages(ctx, roTxn, batch, "JournalEntries", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntityId, EntryId, PeriodId, COALESCE(OriginalPeriodId, ''), COALESCE(SourceEventId, ''), SourceType, SourceId,
		               COALESCE(OrderId, ''), COALESCE(Memo, ''), Currency, CountryPack, COALESCE(ReversesEntryId, ''), PrevHash, EntryHash, PostedBy, OccurredAt, PostedAt
		        FROM JournalEntries WHERE EntityId = @eid AND OccurredAt >= @from AND OccurredAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if pid != "" {
			sql += ` AND PeriodId = @pid`
			params["pid"] = pid
		}
		if has {
			sql += ` AND EntryId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY EntryId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r JournalEntryRow
		if err := row.Columns(&r.EntityID, &r.EntryID, &r.PeriodID, &r.OriginalPeriodID, &r.SourceEventID, &r.SourceType, &r.SourceID,
			&r.OrderID, &r.Memo, &r.Currency, &r.CountryPack, &r.ReversesEntryID, &r.PrevHash, &r.EntryHash, &r.PostedBy, &r.OccurredAt, &r.PostedAt); err != nil {
			return "", err
		}
		r.OccurredAt = r.OccurredAt.UTC()
		r.PostedAt = r.PostedAt.UTC()
		rows = append(rows, r)
		return r.EntryID, nil
	})
	return rows, err
}

type entryLineKey struct {
	entryID string
	line    int64
}

func collectJournalPostings(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid, pid string, from, to time.Time, batch int) ([]JournalPostingRow, error) {
	var rows []JournalPostingRow
	err := runKeysetPages(ctx, roTxn, batch, "JournalPostings", func(has bool, k entryLineKey) spanner.Statement {
		sql := `SELECT p.EntityId, p.EntryId, p.Line, p.AccountCode, p.Direction, p.AmountMinor, p.Currency, p.FunctionalAmountMinor,
		               COALESCE(p.FxRateId, ''), COALESCE(p.PartyType, ''), COALESCE(p.PartyId, ''), COALESCE(p.WarehouseId, ''),
		               COALESCE(p.ProductId, ''), COALESCE(p.LotId, ''), COALESCE(p.Description, '')
		        FROM JournalPostings p
		        JOIN JournalEntries e ON p.EntityId = e.EntityId AND p.EntryId = e.EntryId
		        WHERE p.EntityId = @eid AND e.OccurredAt >= @from AND e.OccurredAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if pid != "" {
			sql += ` AND e.PeriodId = @pid`
			params["pid"] = pid
		}
		if has {
			sql += ` AND (p.EntryId > @k1 OR (p.EntryId = @k1 AND p.Line > @k2))`
			params["k1"] = k.entryID
			params["k2"] = k.line
		}
		sql += ` ORDER BY p.EntryId ASC, p.Line ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (entryLineKey, error) {
		var r JournalPostingRow
		if err := row.Columns(&r.EntityID, &r.EntryID, &r.Line, &r.AccountCode, &r.Direction, &r.AmountMinor, &r.Currency, &r.FunctionalAmountMinor,
			&r.FxRateID, &r.PartyType, &r.PartyID, &r.WarehouseID, &r.ProductID, &r.LotID, &r.Description); err != nil {
			return entryLineKey{}, err
		}
		rows = append(rows, r)
		return entryLineKey{entryID: r.EntryID, line: r.Line}, nil
	})
	return rows, err
}

func collectArInvoices(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]ArInvoiceRow, error) {
	var rows []ArInvoiceRow
	err := runKeysetPages(ctx, roTxn, batch, "ArInvoices", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT InvoiceId, SupplierId, RetailerId, OrderId, Status, PrincipalMinor, BalanceMinor, Currency, CreditLeaveAt, DueAt,
		               TermsDays, GracePeriodDays, COALESCE(AgingBucket, ''), LastDunnedAt, DunningStep, CreatedAt, UpdatedAt
		        FROM ArInvoices WHERE SupplierId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND InvoiceId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY InvoiceId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ArInvoiceRow
		var lastDun spanner.NullTime
		if err := row.Columns(&r.InvoiceID, &r.SupplierID, &r.RetailerID, &r.OrderID, &r.Status, &r.PrincipalMinor, &r.BalanceMinor,
			&r.Currency, &r.CreditLeaveAt, &r.DueAt, &r.TermsDays, &r.GracePeriodDays, &r.AgingBucket, &lastDun, &r.DunningStep, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.CreditLeaveAt = r.CreditLeaveAt.UTC()
		r.DueAt = r.DueAt.UTC()
		r.LastDunnedAt = optTimePtr(lastDun)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.InvoiceID, nil
	})
	return rows, err
}

type costLayerKey struct {
	whID   string
	prodID string
	layer  string
}

func collectInventoryCostLayers(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]InventoryCostLayerRow, error) {
	var rows []InventoryCostLayerRow
	err := runKeysetPages(ctx, roTxn, batch, "InventoryCostLayers", func(has bool, k costLayerKey) spanner.Statement {
		sql := `SELECT EntityId, WarehouseId, ProductId, LayerId, COALESCE(LotId, ''), COALESCE(ReceiptRefType, ''), COALESCE(ReceiptRefId, ''),
		               QtyRemaining, OriginalQty, UnitCostMinor, Currency, COALESCE(FxRateId, ''), ReceivedAt, ExpiryDate, CreatedAt, UpdatedAt
		        FROM InventoryCostLayers WHERE EntityId = @eid AND ReceivedAt >= @from AND ReceivedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND (WarehouseId > @k1 OR (WarehouseId = @k1 AND (ProductId > @k2 OR (ProductId = @k2 AND LayerId > @k3))))`
			params["k1"] = k.whID
			params["k2"] = k.prodID
			params["k3"] = k.layer
		}
		sql += ` ORDER BY WarehouseId ASC, ProductId ASC, LayerId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (costLayerKey, error) {
		var r InventoryCostLayerRow
		var expDate spanner.NullDate
		if err := row.Columns(&r.EntityID, &r.WarehouseID, &r.ProductID, &r.LayerID, &r.LotID, &r.ReceiptRefType, &r.ReceiptRefID,
			&r.QtyRemaining, &r.OriginalQty, &r.UnitCostMinor, &r.Currency, &r.FxRateID, &r.ReceivedAt, &expDate, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return costLayerKey{}, err
		}
		if expDate.Valid {
			r.ExpiryDate = expDate.Date.String()
		}
		r.ReceivedAt = r.ReceivedAt.UTC()
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return costLayerKey{whID: r.WarehouseID, prodID: r.ProductID, layer: r.LayerID}, nil
	})
	return rows, err
}

func collectInventoryMovements(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]InventoryMovementRow, error) {
	var rows []InventoryMovementRow
	err := runKeysetPages(ctx, roTxn, batch, "InventoryMovements", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntityId, MovementId, MovementType, ReferenceType, ReferenceId, WarehouseId, ProductId,
		               COALESCE(LotId, ''), COALESCE(LayerId, ''), QtyDelta, UnitCostMinor, TotalCostMinor, Currency, COALESCE(GlEntryId, ''), CreatedAt
		        FROM InventoryMovements WHERE EntityId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND MovementId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY MovementId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r InventoryMovementRow
		if err := row.Columns(&r.EntityID, &r.MovementID, &r.MovementType, &r.ReferenceType, &r.ReferenceID, &r.WarehouseID, &r.ProductID,
			&r.LotID, &r.LayerID, &r.QtyDelta, &r.UnitCostMinor, &r.TotalCostMinor, &r.Currency, &r.GlEntryID, &r.CreatedAt); err != nil {
			return "", err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		rows = append(rows, r)
		return r.MovementID, nil
	})
	return rows, err
}

func collectEsfDocuments(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid, pid string, from, to time.Time, batch int) ([]EsfDocumentRow, error) {
	var rows []EsfDocumentRow
	err := runKeysetPages(ctx, roTxn, batch, "EsfDocuments", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntityId, EsfId, Direction, COALESCE(OrderId, ''), COALESCE(PoId, ''), CounterpartyTin, COALESCE(CounterpartyName, ''),
		               TotalMinor, VatMinor, VatRateBps, Status, Operator, COALESCE(OperatorRef, ''), COALESCE(SignedPayloadUri, ''),
		               COALESCE(PayloadHash, ''), COALESCE(FailureReason, ''), PeriodId, IssuedAt, AcceptedAt, CreatedAt, UpdatedAt
		        FROM EsfDocuments WHERE EntityId = @eid AND COALESCE(IssuedAt, CreatedAt) >= @from AND COALESCE(IssuedAt, CreatedAt) <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if pid != "" {
			sql += ` AND PeriodId = @pid`
			params["pid"] = pid
		}
		if has {
			sql += ` AND EsfId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY EsfId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r EsfDocumentRow
		var iss, acc spanner.NullTime
		if err := row.Columns(&r.EntityID, &r.EsfID, &r.Direction, &r.OrderID, &r.PoID, &r.CounterpartyTIN, &r.CounterpartyName,
			&r.TotalMinor, &r.VatMinor, &r.VatRateBps, &r.Status, &r.Operator, &r.OperatorRef, &r.SignedPayloadURI,
			&r.PayloadHash, &r.FailureReason, &r.PeriodID, &iss, &acc, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.IssuedAt = optTimePtr(iss)
		r.AcceptedAt = optTimePtr(acc)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.EsfID, nil
	})
	return rows, err
}

func collectCashierBlindCountSessions(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]CashierBlindCountRow, error) {
	var rows []CashierBlindCountRow
	err := runKeysetPages(ctx, roTxn, batch, "CashierBlindCountSessions", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT SessionId, SupplierId, DepotId, DriverId, RouteId, BagSealBarcode, AttemptNumber,
		               CountedCashMinor, ExpectedCashMinor, VarianceMinor, Status, CreatedAt, ResolvedAt
		        FROM CashierBlindCountSessions WHERE SupplierId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND SessionId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY SessionId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r CashierBlindCountRow
		var res spanner.NullTime
		if err := row.Columns(&r.SessionID, &r.SupplierID, &r.DepotID, &r.DriverID, &r.RouteID, &r.BagSealBarcode,
			&r.AttemptNumber, &r.CountedCashMinor, &r.ExpectedCashMinor, &r.VarianceMinor, &r.Status, &r.CreatedAt, &res); err != nil {
			return "", err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		r.ResolvedAt = optTimePtr(res)
		rows = append(rows, r)
		return r.SessionID, nil
	})
	return rows, err
}

func collectDepotVaultLedgerEntries(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]DepotVaultLedgerRow, error) {
	var rows []DepotVaultLedgerRow
	err := runKeysetPages(ctx, roTxn, batch, "DepotVaultLedgerEntries", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT VaultEntryId, DepotId, SupplierId, COALESCE(SessionId, ''), COALESCE(CITManifestId, ''),
		               EntryType, AmountMinor, RunningBalanceMinor, COALESCE(CourierBadgeId, ''), COALESCE(SealBarcode, ''), CreatedAt
		        FROM DepotVaultLedgerEntries WHERE SupplierId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND VaultEntryId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY VaultEntryId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r DepotVaultLedgerRow
		if err := row.Columns(&r.VaultEntryID, &r.DepotID, &r.SupplierID, &r.SessionID, &r.CITManifestID,
			&r.EntryType, &r.AmountMinor, &r.RunningBalanceMinor, &r.CourierBadgeID, &r.SealBarcode, &r.CreatedAt); err != nil {
			return "", err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		rows = append(rows, r)
		return r.VaultEntryID, nil
	})
	return rows, err
}

func collectPaymentLedgerEntries(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]PaymentLedgerRow, error) {
	var rows []PaymentLedgerRow
	err := runKeysetPages(ctx, roTxn, batch, "PaymentLedgerEntries", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT LedgerEntryId, COALESCE(SessionId, ''), COALESCE(OrderId, ''), COALESCE(SupplierId, ''), COALESCE(RetailerId, ''),
		               Gateway, EntryType, AmountMinor, Currency, COALESCE(ReferenceId, ''), Source, OccurredAt, CreatedAt
		        FROM PaymentLedgerEntries WHERE SupplierId = @eid AND OccurredAt >= @from AND OccurredAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND LedgerEntryId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY LedgerEntryId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r PaymentLedgerRow
		if err := row.Columns(&r.LedgerEntryID, &r.SessionID, &r.OrderID, &r.SupplierID, &r.RetailerID,
			&r.Gateway, &r.EntryType, &r.AmountMinor, &r.Currency, &r.ReferenceID, &r.Source, &r.OccurredAt, &r.CreatedAt); err != nil {
			return "", err
		}
		r.OccurredAt = r.OccurredAt.UTC()
		r.CreatedAt = r.CreatedAt.UTC()
		rows = append(rows, r)
		return r.LedgerEntryID, nil
	})
	return rows, err
}

func collectApprovalRequests(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]ApprovalRequestRow, error) {
	var rows []ApprovalRequestRow
	err := runKeysetPages(ctx, roTxn, batch, "ApprovalRequests", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT TenantId, RequestId, COALESCE(PolicyId, ''), PolicyVersion, ObjectType, ObjectId, Action, AmountMinor,
		               RequestedBy, ActorType, PayloadHash, Status, CurrentStep, RequiredApprovers, BreakGlass,
		               COALESCE(BreakGlassReason, ''), ExpiresAt, ResolvedAt, CreatedAt, UpdatedAt
		        FROM ApprovalRequests WHERE TenantId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND RequestId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY RequestId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ApprovalRequestRow
		var exp, res spanner.NullTime
		if err := row.Columns(&r.TenantID, &r.RequestID, &r.PolicyID, &r.PolicyVersion, &r.ObjectType, &r.ObjectID, &r.Action, &r.AmountMinor,
			&r.RequestedBy, &r.ActorType, &r.PayloadHash, &r.Status, &r.CurrentStep, &r.RequiredApprovers, &r.BreakGlass,
			&r.BreakGlassReason, &exp, &res, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.ExpiresAt = optTimePtr(exp)
		r.ResolvedAt = optTimePtr(res)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.RequestID, nil
	})
	return rows, err
}

type decisionKey struct {
	reqID string
	decID string
}

func collectApprovalDecisions(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]ApprovalDecisionRow, error) {
	var rows []ApprovalDecisionRow
	err := runKeysetPages(ctx, roTxn, batch, "ApprovalDecisions", func(has bool, k decisionKey) spanner.Statement {
		sql := `SELECT TenantId, RequestId, DecisionId, StepIndex, ApproverId, COALESCE(DelegatedFrom, ''), Decision, COALESCE(Comment, ''), MfaVerified, CreatedAt
		        FROM ApprovalDecisions WHERE TenantId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND (RequestId > @k1 OR (RequestId = @k1 AND DecisionId > @k2))`
			params["k1"] = k.reqID
			params["k2"] = k.decID
		}
		sql += ` ORDER BY RequestId ASC, DecisionId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (decisionKey, error) {
		var r ApprovalDecisionRow
		if err := row.Columns(&r.TenantID, &r.RequestID, &r.DecisionID, &r.StepIndex, &r.ApproverID, &r.DelegatedFrom,
			&r.Decision, &r.Comment, &r.MfaVerified, &r.CreatedAt); err != nil {
			return decisionKey{}, err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		rows = append(rows, r)
		return decisionKey{reqID: r.RequestID, decID: r.DecisionID}, nil
	})
	return rows, err
}

func collectAuditChainAnchors(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]AuditChainAnchorRow, error) {
	var rows []AuditChainAnchorRow
	err := runKeysetPages(ctx, roTxn, batch, "AuditChainAnchors", func(has bool, lastDay string) spanner.Statement {
		sql := `SELECT TenantId, DayBucket, HeadEventId, EventCount, RootHash, COALESCE(WormUri, ''), COALESCE(SignerKeyId, ''), AnchoredBy, AnchoredAt
		        FROM AuditChainAnchors WHERE TenantId = @eid AND AnchoredAt >= @from AND AnchoredAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND DayBucket > @k1`
			params["k1"] = lastDay
		}
		sql += ` ORDER BY DayBucket ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r AuditChainAnchorRow
		if err := row.Columns(&r.TenantID, &r.DayBucket, &r.HeadEventID, &r.EventCount, &r.RootHash, &r.WormURI, &r.SignerKeyID, &r.AnchoredBy, &r.AnchoredAt); err != nil {
			return "", err
		}
		r.AnchoredAt = r.AnchoredAt.UTC()
		rows = append(rows, r)
		return r.DayBucket, nil
	})
	return rows, err
}

func collectReconciliationRuns(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid, pid string, from, to time.Time, batch int) ([]ReconciliationRunRow, error) {
	var rows []ReconciliationRunRow
	err := runKeysetPages(ctx, roTxn, batch, "ReconciliationRuns", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntityId, RunId, PeriodId, Mode, Status, BreakCount, ExposureMinor, TriggeredBy, StartedAt, CompletedAt
		        FROM ReconciliationRuns WHERE EntityId = @eid AND StartedAt >= @from AND StartedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if pid != "" {
			sql += ` AND PeriodId = @pid`
			params["pid"] = pid
		}
		if has {
			sql += ` AND RunId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY RunId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ReconciliationRunRow
		var comp spanner.NullTime
		if err := row.Columns(&r.EntityID, &r.RunID, &r.PeriodID, &r.Mode, &r.Status, &r.BreakCount, &r.ExposureMinor, &r.TriggeredBy, &r.StartedAt, &comp); err != nil {
			return "", err
		}
		r.StartedAt = r.StartedAt.UTC()
		r.CompletedAt = optTimePtr(comp)
		rows = append(rows, r)
		return r.RunID, nil
	})
	return rows, err
}

func collectReconciliationBreaks(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, eid string, from, to time.Time, batch int) ([]ReconciliationBreakRow, error) {
	var rows []ReconciliationBreakRow
	err := runKeysetPages(ctx, roTxn, batch, "ReconciliationBreaks", func(has bool, lastID string) spanner.Statement {
		sql := `SELECT EntityId, BreakId, RunId, Dimension, ObjectType, ObjectId, ExpectedMinor, ActualMinor, DeltaMinor, Currency, Severity, Status,
		               COALESCE(OwnerId, ''), SlaDueAt, COALESCE(ResolutionNote, ''), COALESCE(ControlTowerExceptionId, ''), CreatedAt, UpdatedAt
		        FROM ReconciliationBreaks WHERE EntityId = @eid AND CreatedAt >= @from AND CreatedAt <= @to`
		params := map[string]any{"eid": eid, "from": from.UTC(), "to": to.UTC()}
		if has {
			sql += ` AND BreakId > @k1`
			params["k1"] = lastID
		}
		sql += ` ORDER BY BreakId ASC LIMIT @lim`
		return spanner.Statement{SQL: sql, Params: params}
	}, func(row *spanner.Row) (string, error) {
		var r ReconciliationBreakRow
		var sla spanner.NullTime
		if err := row.Columns(&r.EntityID, &r.BreakID, &r.RunID, &r.Dimension, &r.ObjectType, &r.ObjectID,
			&r.ExpectedMinor, &r.ActualMinor, &r.DeltaMinor, &r.Currency, &r.Severity, &r.Status,
			&r.OwnerID, &sla, &r.ResolutionNote, &r.ControlTowerExceptionID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return "", err
		}
		r.SlaDueAt = optTimePtr(sla)
		r.CreatedAt = r.CreatedAt.UTC()
		r.UpdatedAt = r.UpdatedAt.UTC()
		rows = append(rows, r)
		return r.BreakID, nil
	})
	return rows, err
}
