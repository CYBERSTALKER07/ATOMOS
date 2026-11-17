package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// LoadLiveOverlay queries read-only Spanner state for non-mutating virtual dry-run simulation.
func (r *SpannerRepository) LoadLiveOverlay(ctx context.Context, supplierID string) (LiveOverlaySnapshot, error) {
	out := NewEmptyLiveOverlay()
	if r == nil || r.client == nil {
		return out, nil
	}

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	if err := loadLiveSupplierMeta(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveWarehouses(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveProducts(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveIdentityMappings(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveCustomers(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveInventory(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveLots(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	if err := loadLiveOrdersAndInvoices(ctx, ro, supplierID, &out); err != nil {
		return out, err
	}
	return out, nil
}

func loadLiveSupplierMeta(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	row, err := ro.ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"Currency", "HomeCell"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil
		}
		return err
	}
	var cur string
	var cell spanner.NullString
	if err := row.Columns(&cur, &cell); err != nil {
		return err
	}
	if strings.TrimSpace(cur) != "" {
		out.SupplierCurrency = strings.ToUpper(strings.TrimSpace(cur))
	}
	out.HomeCell = cell.StringVal
	return nil
}

func loadLiveWarehouses(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId, Name FROM Warehouses WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var id, name string
		if err := row.Columns(&id, &name); err != nil {
			return err
		}
		w := LiveWarehouse{WarehouseID: id, Name: name}
		out.WarehousesByExt[strings.ToLower(id)] = w
		out.WarehousesByName[strings.ToLower(strings.TrimSpace(name))] = w
		if out.DefaultWarehouseID == "" {
			out.DefaultWarehouseID = id
		}
	}
}

func loadLiveProducts(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, Name, PriceMinor, Currency, Unit, Barcode, RequiresColdChain, IsPerishable
		      FROM Products WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var p LiveProduct
		var unit, barcode spanner.NullString
		var cold, perish spanner.NullBool
		if err := row.Columns(&p.ProductID, &p.Name, &p.PriceMinor, &p.Currency, &unit, &barcode, &cold, &perish); err != nil {
			return err
		}
		p.Unit = unit.StringVal
		p.Barcode = barcode.StringVal
		p.RequiresColdChain = cold.Bool
		p.IsPerishable = perish.Bool
		registerVirtualProduct(out, p)
	}
}

func loadLiveIdentityMappings(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT pack, external_id, internal_id FROM MigrationIdentityMap WHERE supplier_id = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var packStr, extID, internalID string
		if err := row.Columns(&packStr, &extID, &internalID); err != nil {
			return err
		}
		switch Pack(packStr) {
		case PackWarehouses:
			if w, ok := out.WarehousesByExt[strings.ToLower(internalID)]; ok {
				w.ExternalID = extID
				out.WarehousesByExt[strings.ToLower(extID)] = w
			}
		case PackProductsPrices:
			if p, ok := out.ProductsByID[internalID]; ok {
				p.SKU = extID
				registerVirtualProduct(out, p)
			}
		case PackCustomers:
			out.CustomersByExt[strings.ToLower(extID)] = LiveCustomer{RetailerID: internalID, ExternalID: extID}
		}
	}
}

func loadLiveCustomers(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, TaxIdTIN, NormalizedPhone, LegalName
		      FROM SupplierRetailerLinks WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var retID string
		var tin, phone, legal spanner.NullString
		if err := row.Columns(&retID, &tin, &phone, &legal); err != nil {
			return err
		}
		c := LiveCustomer{
			RetailerID:   retID,
			INN:          NormalizeINN(tin.StringVal),
			Phone:        NormalizePhoneUZ(phone.StringVal),
			CustomerName: legal.StringVal,
			Currency:     out.SupplierCurrency,
		}
		registerVirtualCustomer(out, c)
	}
}

func loadLiveInventory(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, ProductId, QuantityOnHand, QuantityReserved
		      FROM SupplierInventoryV2 WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var whID, prodID string
		var onHand, reserved int64
		if err := row.Columns(&whID, &prodID, &onHand, &reserved); err != nil {
			return err
		}
		unit := ""
		if p, ok := out.ProductsByID[prodID]; ok {
			unit = p.Unit
		}
		out.StockByWHProduct[whID+"|"+prodID] = LiveStock{
			WarehouseID: whID,
			ProductID:   prodID,
			OnHand:      onHand,
			Reserved:    reserved,
			Unit:        unit,
		}
		out.StockTotalByProd[prodID] += onHand
	}
}

func loadLiveLots(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT LotId, WarehouseId, ProductId, LotCode, ExpiryDate, QuantityOnHand, QuantityReserved
		      FROM StockLots WHERE SupplierId = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var lot LiveLot
		if err := row.Columns(&lot.LotID, &lot.WarehouseID, &lot.ProductID, &lot.LotCode, &lot.ExpiryDate, &lot.OnHand, &lot.Reserved); err != nil {
			return err
		}
		key := lot.WarehouseID + "|" + lot.ProductID + "|" + strings.ToLower(strings.TrimSpace(lot.LotCode))
		out.LotsByKey[key] = lot
	}
}

func loadLiveOrdersAndInvoices(ctx context.Context, ro *spanner.ReadOnlyTransaction, supplierID string, out *LiveOverlaySnapshot) error {
	oIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT OrderId, ExternalOrderNo, OrderSource, Status FROM Orders WHERE SupplierId = @supplierId AND ExternalOrderNo IS NOT NULL`,
		Params: map[string]any{"supplierId": supplierID},
	})
	for {
		row, err := oIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			oIter.Stop()
			return err
		}
		var o LiveOrder
		var src spanner.NullString
		if err := row.Columns(&o.OrderID, &o.ExternalOrderNo, &src, &o.Status); err != nil {
			oIter.Stop()
			return err
		}
		o.OrderSource = src.StringVal
		out.OrdersByExtNo[strings.ToLower(strings.TrimSpace(o.ExternalOrderNo))] = o
	}
	oIter.Stop()

	iIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT InvoiceId, ExternalId, BalanceMinor, Currency, Status FROM ArInvoices WHERE SupplierId = @supplierId AND ExternalId IS NOT NULL`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer iIter.Stop()
	for {
		row, err := iIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var inv LiveARInvoice
		if err := row.Columns(&inv.InvoiceID, &inv.ExternalID, &inv.BalanceMinor, &inv.Currency, &inv.Status); err != nil {
			return err
		}
		out.ArInvoicesByExt[strings.ToLower(strings.TrimSpace(inv.ExternalID))] = inv
	}

	lIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT supplier_id, warehouse_id, session_id, acquired_at, expires_at FROM MigrationWarehouseLocks WHERE supplier_id = @supplierId`,
		Params: map[string]any{"supplierId": supplierID},
	})
	defer lIter.Stop()
	for {
		row, err := lIter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var lock WarehouseLock
		if err := row.Columns(&lock.SupplierID, &lock.WarehouseID, &lock.SessionID, &lock.AcquiredAt, &lock.ExpiresAt); err != nil {
			return err
		}
		out.WarehouseLocks = append(out.WarehouseLocks, lock)
	}
}

// SaveMigrationReport writes an immutable dry-run or apply report row into MigrationReports.
func (r *SpannerRepository) SaveMigrationReport(ctx context.Context, rep MigrationReport) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	summaryJSON, err := toNullJSON(rep.SummaryJSON)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("MigrationReports", map[string]any{
			"supplier_id":         rep.SupplierID,
			"report_id":           rep.ReportID,
			"project_id":          toNullString(rep.ProjectID),
			"session_id":          toNullString(rep.SessionID),
			"report_kind":         rep.ReportKind,
			"content_fingerprint": rep.ContentFingerprint,
			"rows_total":          rep.RowsTotal,
			"rows_valid":          rep.RowsValid,
			"rows_invalid":        rep.RowsInvalid,
			"would_create":        rep.WouldCreate,
			"would_update":        rep.WouldUpdate,
			"would_skip":          rep.WouldSkip,
			"money_impact_minor":  rep.MoneyImpactMinor,
			"currency":            toNullString(rep.Currency),
			"blocking_count":      rep.BlockingCount,
			"warning_count":       rep.WarningCount,
			"summary_json":        summaryJSON,
			"expires_at":          rep.ExpiresAt.UTC(),
			"created_at":          spanner.CommitTimestamp,
		}),
	})
	return err
}

// GetMigrationReport reads a MigrationReports row by tenant and reportId.
func (r *SpannerRepository) GetMigrationReport(ctx context.Context, supplierID, reportID string) (MigrationReport, error) {
	if err := r.requireClient(); err != nil {
		return MigrationReport{}, err
	}
	row, err := r.client.Single().ReadRow(ctx, "MigrationReports", spanner.Key{supplierID, reportID}, []string{
		"supplier_id", "report_id", "project_id", "session_id", "report_kind", "content_fingerprint",
		"rows_total", "rows_valid", "rows_invalid", "would_create", "would_update", "would_skip",
		"money_impact_minor", "currency", "blocking_count", "warning_count", "summary_json", "expires_at", "created_at",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return MigrationReport{}, ErrNotFound
		}
		return MigrationReport{}, err
	}
	var rep MigrationReport
	var projID, sessID, currency spanner.NullString
	var summary spanner.NullJSON
	if err := row.Columns(
		&rep.SupplierID, &rep.ReportID, &projID, &sessID, &rep.ReportKind, &rep.ContentFingerprint,
		&rep.RowsTotal, &rep.RowsValid, &rep.RowsInvalid, &rep.WouldCreate, &rep.WouldUpdate, &rep.WouldSkip,
		&rep.MoneyImpactMinor, &currency, &rep.BlockingCount, &rep.WarningCount, &summary, &rep.ExpiresAt, &rep.CreatedAt,
	); err != nil {
		return MigrationReport{}, err
	}
	rep.ProjectID = projID.StringVal
	rep.SessionID = sessID.StringVal
	rep.Currency = currency.StringVal
	rep.SummaryJSON = fromNullJSON(summary)
	rep.ExpiresAt = rep.ExpiresAt.UTC()
	rep.CreatedAt = rep.CreatedAt.UTC()
	return rep, nil
}

// MarkSessionDryRunReady records dry_run_report_id + dry_run_fingerprint and advances session to DRY_RUN_READY.
func (r *SpannerRepository) MarkSessionDryRunReady(ctx context.Context, supplierID, sessionID, reportID, fingerprint, applyMode string) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var statusStr string
		if err := row.Columns(&statusStr); err != nil {
			return err
		}
		cur := NormalizeSessionStatus(statusStr)
		if cur != SessionStatusDryRunReady {
			if err := ValidateSessionTransition(cur, SessionStatusDryRunReady); err != nil {
				return err
			}
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SupplierImportSessions", map[string]any{
				"supplier_id":           supplierID,
				"session_id":            sessionID,
				"status":                string(SessionStatusDryRunReady),
				"dry_run_report_id":     reportID,
				"dry_run_fingerprint":   fingerprint,
				"apply_mode":            NormalizeApplyMode(applyMode),
				"set_on_hand_confirmed": false,
				"updated_at":            spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// ApproveSession transitions a DRY_RUN_READY session to APPROVED after confirmation flags are verified.
func (r *SpannerRepository) ApproveSession(ctx context.Context, supplierID, sessionID, applyMode string, setOnHandConfirmed bool) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var statusStr string
		if err := row.Columns(&statusStr); err != nil {
			return err
		}
		cur := NormalizeSessionStatus(statusStr)
		if cur != SessionStatusApproved {
			if err := ValidateSessionTransition(cur, SessionStatusApproved); err != nil {
				return err
			}
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SupplierImportSessions", map[string]any{
				"supplier_id":           supplierID,
				"session_id":            sessionID,
				"status":                string(SessionStatusApproved),
				"apply_mode":            NormalizeApplyMode(applyMode),
				"set_on_hand_confirmed": setOnHandConfirmed,
				"updated_at":            spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// MarkSessionApplied transitions an APPLYING (or APPROVED) session to APPLIED with 72h rollback_until.
func (r *SpannerRepository) MarkSessionApplied(ctx context.Context, supplierID, sessionID string, rollbackUntil time.Time) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var statusStr string
		if err := row.Columns(&statusStr); err != nil {
			return err
		}
		cur := NormalizeSessionStatus(statusStr)
		if cur != SessionStatusApplied {
			if err := ValidateSessionTransition(cur, SessionStatusApplied); err != nil {
				return err
			}
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SupplierImportSessions", map[string]any{
				"supplier_id":    supplierID,
				"session_id":     sessionID,
				"status":         string(SessionStatusApplied),
				"rollback_until": rollbackUntil.UTC(),
				"updated_at":     spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// SaveSnapshotsChunk persists a batch of pre-apply row before-images into MigrationSnapshots.
func (r *SpannerRepository) SaveSnapshotsChunk(ctx context.Context, supplierID, sessionID string, snapshots []MigrationSnapshotRecord) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return nil
	}
	const maxBatch = 500
	for start := 0; start < len(snapshots); start += maxBatch {
		end := start + maxBatch
		if end > len(snapshots) {
			end = len(snapshots)
		}
		muts := make([]*spanner.Mutation, 0, end-start)
		for _, snap := range snapshots[start:end] {
			beforeJSON, err := toNullJSON(snap.BeforeJSON)
			if err != nil {
				return err
			}
			muts = append(muts, spanner.InsertOrUpdateMap("MigrationSnapshots", map[string]any{
				"supplier_id":  supplierID,
				"session_id":   sessionID,
				"snapshot_seq": snap.SnapshotSeq,
				"target_table": snap.TargetTable,
				"row_key_json": snap.RowKeyJSON,
				"op_kind":      snap.OpKind,
				"before_json":  beforeJSON,
				"expires_at":   snap.ExpiresAt.UTC(),
				"created_at":   spanner.CommitTimestamp,
			}))
		}
		if _, err := r.client.Apply(ctx, muts); err != nil {
			return err
		}
	}
	return nil
}

// ListSnapshotsBySession reads all MigrationSnapshots for a session in reverse snapshot_seq order.
func (r *SpannerRepository) ListSnapshotsBySession(ctx context.Context, supplierID, sessionID string) ([]MigrationSnapshotRecord, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT supplier_id, session_id, snapshot_seq, target_table, row_key_json, op_kind, before_json, expires_at, created_at
		      FROM MigrationSnapshots
		      WHERE supplier_id = @supplierId AND session_id = @sessionId
		      ORDER BY snapshot_seq DESC`,
		Params: map[string]any{"supplierId": supplierID, "sessionId": sessionID},
	})
	defer iter.Stop()

	out := make([]MigrationSnapshotRecord, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var snap MigrationSnapshotRecord
		var before spanner.NullJSON
		if err := row.Columns(
			&snap.SupplierID, &snap.SessionID, &snap.SnapshotSeq, &snap.TargetTable,
			&snap.RowKeyJSON, &snap.OpKind, &before, &snap.ExpiresAt, &snap.CreatedAt,
		); err != nil {
			return nil, err
		}
		snap.BeforeJSON = fromNullJSON(before)
		snap.ExpiresAt = snap.ExpiresAt.UTC()
		snap.CreatedAt = snap.CreatedAt.UTC()
		out = append(out, snap)
	}
}

// ExecuteSessionRollback atomically restores non-financial pre-images/deletes and creates compensating financial reversals.
func (r *SpannerRepository) ExecuteSessionRollback(
	ctx context.Context,
	supplierID, sessionID string,
	snapshots []MigrationSnapshotRecord,
	now time.Time,
) (RollbackResult, error) {
	if err := r.requireClient(); err != nil {
		return RollbackResult{}, err
	}
	res := RollbackResult{}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, len(snapshots)*2+4)
		var glInputs []gl.JournalEntryInput
		for _, snap := range snapshots {
			res.RevertedSnapshots++
			switch snap.TargetTable {
			case "Orders":
				orderID, err := extractPKField(snap.RowKeyJSON, "OrderId")
				if err != nil {
					return err
				}
				oRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status"})
				if err != nil && spanner.ErrCode(err) != codes.NotFound {
					return err
				}
				if err == nil {
					var statusStr string
					if err := oRow.Columns(&statusStr); err != nil {
						return err
					}
					upd := map[string]any{
						"OrderId":              orderID,
						"SupersededByRollback": true,
						"UpdatedAt":            spanner.CommitTimestamp,
					}
					if statusStr == "MIGRATION_OPEN" {
						upd["Status"] = "CANCELLED"
					}
					muts = append(muts, spanner.UpdateMap("Orders", upd))
					res.SupersededOrders++
				}
			case "ArInvoices":
				invID, err := extractPKField(snap.RowKeyJSON, "InvoiceId")
				if err != nil {
					return err
				}
				iRow, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invID}, []string{"RetailerId", "OrderId", "BalanceMinor", "Currency", "Status"})
				if err != nil && spanner.ErrCode(err) != codes.NotFound {
					return err
				}
				if err == nil {
					var retailerID, orderID, currency, statusStr string
					var balMinor int64
					if err := iRow.Columns(&retailerID, &orderID, &balMinor, &currency, &statusStr); err != nil {
						return err
					}
					if statusStr != "ROLLED_BACK" {
						muts = append(muts, spanner.UpdateMap("ArInvoices", map[string]any{
							"InvoiceId":    invID,
							"Status":       "ROLLED_BACK",
							"BalanceMinor": int64(0),
							"UpdatedAt":    spanner.CommitTimestamp,
						}))
						res.ReversedInvoices++
						if balMinor > 0 {
							rollbackKey := fmt.Sprintf("rollback:%s:%s", sessionID, invID)
							lIter := txn.Query(ctx, spanner.Statement{
								SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
								Params: map[string]any{"k": rollbackKey},
							})
							_, lErr := lIter.Next()
							lIter.Stop()
							if lErr != nil && lErr != iterator.Done {
								return lErr
							}
							if lErr == iterator.Done {
								muts = append(muts, spanner.InsertMap("ArLedgerEntries", map[string]any{
									"EntryId":        deterministicEntityID("arrev", supplierID, "ar_rollback", sessionID+"|"+invID),
									"InvoiceId":      invID,
									"SupplierId":     supplierID,
									"RetailerId":     retailerID,
									"EntryType":      "MIGRATION_ROLLBACK",
									"AmountMinor":    -balMinor,
									"IdempotencyKey": rollbackKey,
									"RefOrderId":     orderID,
									"CreatedAt":      spanner.CommitTimestamp,
								}))
							}
							je, err := BuildReversalJournalEntry(supplierID, sessionID, invID, orderID, retailerID, currency, balMinor, now)
							if err != nil {
								return err
							}
							jeMuts, err := payment.JournalEntryMutations(je)
							if err != nil {
								return err
							}
							muts = append(muts, jeMuts...)
							cur := strings.ToUpper(strings.TrimSpace(currency))
							if cur == "" {
								cur = "UZS"
							}
							occurred := now
							if occurred.IsZero() {
								occurred = time.Now().UTC()
							}
							glIn, err := gl.BuildMigrationOpeningBalanceInput(gl.RuleEvalContext{
								EntityID:       supplierID,
								SupplierID:     supplierID,
								RetailerID:     retailerID,
								SourceID:       invID,
								OrderID:        orderID,
								Currency:       cur,
								GrossMinor:     balMinor,
								Action:         "ROLLBACK",
								IdempotencyKey: "gl:mig_rollback:" + sessionID + ":" + invID,
								Memo:           fmt.Sprintf("Migration session %s rollback AR %s", sessionID, invID),
								OccurredAt:     occurred,
							})
							if err != nil {
								return err
							}
							glInputs = append(glInputs, glIn)
							res.ReversedAmountMinor += balMinor
							res.ReversalEntryIDs = append(res.ReversalEntryIDs, je.EntryID)
						}
					}
				}
			default:
				m, err := buildGenericRollbackMutation(snap)
				if err != nil {
					return err
				}
				if m != nil {
					muts = append(muts, m)
				}
			}
		}

		// Clean up any identity mappings tagged to this rolled-back session and finalize session state.
		idIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT supplier_id, pack, external_id FROM MigrationIdentityMap
			      WHERE supplier_id = @supplierId AND created_by_session = @sessionId`,
			Params: map[string]any{"supplierId": supplierID, "sessionId": sessionID},
		})
		for {
			row, err := idIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				idIter.Stop()
				return err
			}
			var sid, pack, extID string
			if err := row.Columns(&sid, &pack, &extID); err != nil {
				idIter.Stop()
				return err
			}
			muts = append(muts, spanner.Delete("MigrationIdentityMap", spanner.Key{sid, pack, extID}))
		}
		idIter.Stop()

		muts = append(muts, spanner.UpdateMap("SupplierImportSessions", map[string]any{
			"supplier_id": supplierID,
			"session_id":  sessionID,
			"status":      string(SessionStatusRolledBack),
			"updated_at":  spanner.CommitTimestamp,
		}))
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		if len(glInputs) > 0 {
			if _, err := gl.BufferJournalEntriesBatchInTxn(ctx, txn, nil, glInputs); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return RollbackResult{}, err
	}
	sess, err := r.GetSession(ctx, supplierID, sessionID)
	if err == nil {
		res.Session = sess
	}
	return res, nil
}

func buildGenericRollbackMutation(snap MigrationSnapshotRecord) (*spanner.Mutation, error) {
	if snap.OpKind == SnapshotOpInsert {
		key, err := spannerKeyForSnapshot(snap.TargetTable, snap.RowKeyJSON)
		if err != nil {
			return nil, err
		}
		return spanner.Delete(snap.TargetTable, key), nil
	}
	if snap.OpKind == SnapshotOpUpdate && len(snap.BeforeJSON) > 0 {
		dec := json.NewDecoder(bytes.NewReader(snap.BeforeJSON))
		dec.UseNumber()
		var cols map[string]any
		if err := dec.Decode(&cols); err != nil {
			return nil, fmt.Errorf("decode before_json for %s seq=%d: %w", snap.TargetTable, snap.SnapshotSeq, err)
		}
		for k, v := range cols {
			if num, ok := v.(json.Number); ok {
				if iv, err := num.Int64(); err == nil {
					cols[k] = iv
				} else if fv, err := num.Float64(); err == nil {
					cols[k] = fv
				}
			}
			if snap.TargetTable == "StockLots" && (k == "ExpiryDate" || k == "ManufacturedDate") {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					datePart := s
					if len(datePart) >= 10 {
						datePart = datePart[:10]
					}
					if cd, err := civil.ParseDate(datePart); err == nil {
						cols[k] = cd
					}
				}
			}
		}
		return spanner.InsertOrUpdateMap(snap.TargetTable, cols), nil
	}
	return nil, nil
}

func spannerKeyForSnapshot(table, keyJSON string) (spanner.Key, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(keyJSON), &m); err != nil {
		return nil, fmt.Errorf("parse snapshot key_json for %s: %w", table, err)
	}
	lookup := func(cols ...string) (spanner.Key, error) {
		vals := make(spanner.Key, 0, len(cols))
		for _, c := range cols {
			v, ok := m[c]
			if !ok {
				return nil, fmt.Errorf("snapshot key %s missing column %q", table, c)
			}
			vals = append(vals, fmt.Sprint(v))
		}
		return vals, nil
	}
	switch table {
	case "Warehouses":
		return lookup("WarehouseId")
	case "Products":
		return lookup("ProductId")
	case "SupplierRetailerLinks":
		return lookup("SupplierId", "RetailerId")
	case "SupplierInventoryV2":
		return lookup("SupplierId", "WarehouseId", "ProductId")
	case "StockLots":
		return lookup("LotId")
	case "PartnerPlantMaps":
		return lookup("TenantType", "TenantId", "ExternalPlantId")
	case "MigrationIdentityMap":
		return lookup("supplier_id", "pack", "external_id")
	default:
		return nil, fmt.Errorf("unsupported snapshot table %q", table)
	}
}

func extractPKField(keyJSON, field string) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(keyJSON), &m); err != nil {
		return "", fmt.Errorf("parse pk field %s: %w", field, err)
	}
	v, ok := m[field]
	if !ok || strings.TrimSpace(fmt.Sprint(v)) == "" {
		return "", fmt.Errorf("missing pk field %q in %s", field, keyJSON)
	}
	return fmt.Sprint(v), nil
}
