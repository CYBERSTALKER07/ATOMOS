package connectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists ExternalCredentials, ConnectorSyncRuns, ConnectorEntityMaps,
// and transactional OutboxEvents in Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed connector repository.
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

func (r *SpannerRepository) UpsertCredential(ctx context.Context, cred ExternalCredential) (ExternalCredential, error) {
	if r == nil || r.client == nil {
		return ExternalCredential{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(cred.TenantID)
	prov := normalizeProvider(cred.Provider)
	label := normalizeAccountLabel(cred.AccountLabel)

	var persistedID string
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		targetID := strings.TrimSpace(cred.CredentialID)
		var createdAt any = spanner.CommitTimestamp
		createdBy := strings.TrimSpace(cred.CreatedBy)

		if targetID == "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT CredentialId, CreatedBy, CreatedAt
				      FROM ExternalCredentials
				      WHERE TenantId = @tid AND Provider = @prov AND AccountLabel = @label
				      LIMIT 1`,
				Params: map[string]any{"tid": tid, "prov": prov, "label": label},
			})
			row, qErr := iter.Next()
			iter.Stop()
			if qErr == nil {
				var existingID, existingBy string
				var existingCreated time.Time
				if sErr := row.Columns(&existingID, &existingBy, &existingCreated); sErr == nil {
					targetID = existingID
					createdAt = existingCreated
					if createdBy == "" {
						createdBy = existingBy
					}
				}
			} else if qErr != iterator.Done {
				return fmt.Errorf("lookup existing credential: %w", qErr)
			}
		} else {
			if row, rErr := txn.ReadRow(txCtx, "ExternalCredentials", spanner.Key{tid, targetID}, []string{"CreatedBy", "CreatedAt"}); rErr == nil {
				var existingBy string
				var existingCreated time.Time
				if sErr := row.Columns(&existingBy, &existingCreated); sErr == nil {
					createdAt = existingCreated
					if createdBy == "" {
						createdBy = existingBy
					}
				}
			}
		}
		if targetID == "" {
			targetID = "cred_" + prov + "_" + sanitizeIDToken(label)
		}
		persistedID = targetID
		if createdBy == "" {
			createdBy = "system"
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ExternalCredentials", map[string]any{
				"TenantId":     tid,
				"CredentialId": targetID,
				"Provider":     prov,
				"AccountLabel": label,
				"AuthType":     normalizeAuthType(cred.AuthType),
				"SecretRef":    strings.TrimSpace(cred.SecretRef),
				"BaseUrl":      nullStr(cred.BaseURL),
				"MetadataJson": nullStr(cred.MetadataJSON),
				"Status":       firstNonEmpty(strings.ToUpper(strings.TrimSpace(cred.Status)), CredStatusActive),
				"CreatedBy":    createdBy,
				"CreatedAt":    createdAt,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return ExternalCredential{}, err
	}
	loaded, found, err := r.GetCredential(ctx, tid, prov, label, persistedID)
	if err != nil {
		return ExternalCredential{}, err
	}
	if !found {
		cred.CredentialID = persistedID
		return cred, nil
	}
	return loaded, nil
}

func (r *SpannerRepository) GetCredential(ctx context.Context, tenantID, provider, accountLabel, credentialID string) (ExternalCredential, bool, error) {
	if r == nil || r.client == nil {
		return ExternalCredential{}, false, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	if cid := strings.TrimSpace(credentialID); cid != "" {
		iter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT TenantId, CredentialId, Provider, AccountLabel, AuthType, SecretRef, BaseUrl, MetadataJson, Status, CreatedBy, CreatedAt, UpdatedAt
			      FROM ExternalCredentials
			      WHERE TenantId = @tid AND CredentialId = @cid
			      LIMIT 1`,
			Params: map[string]any{"tid": tid, "cid": cid},
		})
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return ExternalCredential{}, false, nil
		}
		if err != nil {
			return ExternalCredential{}, false, fmt.Errorf("read credential by id: %w", err)
		}
		c, err := scanCredentialRow(row)
		if err != nil {
			return ExternalCredential{}, false, err
		}
		if prov != "" && normalizeProvider(c.Provider) != prov {
			return ExternalCredential{}, false, nil
		}
		return c, true, nil
	}

	label := strings.TrimSpace(accountLabel)
	stmt := spanner.Statement{
		SQL: `SELECT TenantId, CredentialId, Provider, AccountLabel, AuthType, SecretRef, BaseUrl, MetadataJson, Status, CreatedBy, CreatedAt, UpdatedAt
		      FROM ExternalCredentials@{FORCE_INDEX=Idx_ExternalCredentials_ByProvider}
		      WHERE TenantId = @tid AND Provider = @prov AND Status = @status`,
		Params: map[string]any{"tid": tid, "prov": prov, "status": CredStatusActive},
	}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()

	var fallback *ExternalCredential
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return ExternalCredential{}, false, fmt.Errorf("query credentials by provider: %w", err)
		}
		c, sErr := scanCredentialRow(row)
		if sErr != nil {
			return ExternalCredential{}, false, sErr
		}
		if label != "" && strings.EqualFold(c.AccountLabel, label) {
			return c, true, nil
		}
		if label == "" && strings.EqualFold(c.AccountLabel, "default") {
			return c, true, nil
		}
		if label == "" && fallback == nil {
			cp := c
			fallback = &cp
		}
	}
	if fallback != nil {
		return *fallback, true, nil
	}
	return ExternalCredential{}, false, nil
}

func (r *SpannerRepository) ListCredentials(ctx context.Context, tenantID, provider string) ([]ExternalCredential, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	sql := `SELECT TenantId, CredentialId, Provider, AccountLabel, AuthType, SecretRef, BaseUrl, MetadataJson, Status, CreatedBy, CreatedAt, UpdatedAt
	        FROM ExternalCredentials
	        WHERE TenantId = @tid`
	params := map[string]any{"tid": tid}
	if prov != "" {
		sql += ` AND Provider = @prov`
		params["prov"] = prov
	}
	sql += ` ORDER BY Provider ASC, AccountLabel ASC, CredentialId ASC`

	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]ExternalCredential, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list credentials: %w", err)
		}
		c, sErr := scanCredentialRow(row)
		if sErr != nil {
			return nil, sErr
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *SpannerRepository) FindRunByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (ConnectorSyncRun, bool, error) {
	if r == nil || r.client == nil {
		return ConnectorSyncRun{}, false, ErrRepoUnavailable
	}
	idem := strings.TrimSpace(idempotencyKey)
	if idem == "" {
		return ConnectorSyncRun{}, false, nil
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, RunId, Provider, AccountLabel, Direction, StreamName, Status,
		             RecordsProcessed, MismatchCount, MismatchSummaryJson, ErrorMessage,
		             IdempotencyKey, StartedAt, CompletedAt, CreatedAt
		      FROM ConnectorSyncRuns@{FORCE_INDEX=UQ_ConnectorSyncRuns_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
		      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{"tid": strings.TrimSpace(tenantID), "idem": idem},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ConnectorSyncRun{}, false, nil
	}
	if err != nil {
		return ConnectorSyncRun{}, false, fmt.Errorf("lookup sync run idempotency: %w", err)
	}
	run, err := scanSyncRunRow(row)
	if err != nil {
		return ConnectorSyncRun{}, false, err
	}
	return run, true, nil
}

func (r *SpannerRepository) SaveSyncRunAndMaps(ctx context.Context, run ConnectorSyncRun, maps []ConnectorEntityMap, emissions []OutboxEmission) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	tid := strings.TrimSpace(run.TenantID)

	const entityBatchLimit = 500
	primaryMaps := maps
	var overflowMaps []ConnectorEntityMap
	if len(maps) > entityBatchLimit {
		primaryMaps = maps[:entityBatchLimit]
		overflowMaps = maps[entityBatchLimit:]
	}

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if run.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT RunId
				      FROM ConnectorSyncRuns@{FORCE_INDEX=UQ_ConnectorSyncRuns_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{"tid": tid, "idem": run.IdempotencyKey},
			})
			_, qErr := iter.Next()
			iter.Stop()
			if qErr == nil {
				return nil
			}
			if qErr != iterator.Done {
				return fmt.Errorf("check sync run idempotency in txn: %w", qErr)
			}
		}

		var completedAt any
		if run.CompletedAt != nil {
			completedAt = run.CompletedAt.UTC()
		}
		muts := make([]*spanner.Mutation, 0, len(primaryMaps)+1)
		muts = append(muts, spanner.InsertMap("ConnectorSyncRuns", map[string]any{
			"TenantId":            tid,
			"RunId":               run.RunID,
			"Provider":            normalizeProvider(run.Provider),
			"AccountLabel":        normalizeAccountLabel(run.AccountLabel),
			"Direction":           run.Direction,
			"StreamName":          run.StreamName,
			"Status":              run.Status,
			"RecordsProcessed":    run.RecordsProcessed,
			"MismatchCount":       run.MismatchCount,
			"MismatchSummaryJson": nullStr(run.MismatchSummaryJSON),
			"ErrorMessage":        nullStr(run.ErrorMessage),
			"IdempotencyKey":      nullStr(run.IdempotencyKey),
			"StartedAt":           run.StartedAt.UTC(),
			"CompletedAt":         completedAt,
			"CreatedAt":           spanner.CommitTimestamp,
		}))
		for _, em := range primaryMaps {
			muts = append(muts, spanner.InsertOrUpdateMap("ConnectorEntityMaps", map[string]any{
				"TenantId":     tid,
				"Provider":     normalizeProvider(em.Provider),
				"ObjectType":   em.ObjectType,
				"ExternalId":   em.ExternalID,
				"InternalId":   em.InternalID,
				"ContentHash":  nullStr(em.ContentHash),
				"LastRunId":    nullStr(em.LastRunID),
				"LastSyncedAt": spanner.CommitTimestamp,
			}))
		}
		if err := txn.BufferWrite(muts); err != nil {
			return fmt.Errorf("buffer sync run & entity maps: %w", err)
		}

		if len(emissions) > 0 {
			rawBuf := outbox.NewSpannerTxnBuffer(txn)
			stamped := &tenantStampedOutboxBuf{
				inner:    rawBuf,
				tenantID: tid,
			}
			for _, em := range emissions {
				if err := outbox.EmitJSON(txCtx, stamped, em.AggregateType, em.AggregateID, em.Topic, em.Event); err != nil {
					return fmt.Errorf("emit connector outbox event: %w", err)
				}
			}
			if err := rawBuf.Flush(txCtx); err != nil {
				return fmt.Errorf("flush connector outbox events: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Overflow entity map chunks (for huge catalogs >500 items).
	for start := 0; start < len(overflowMaps); start += entityBatchLimit {
		end := start + entityBatchLimit
		if end > len(overflowMaps) {
			end = len(overflowMaps)
		}
		chunk := overflowMaps[start:end]
		if err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
			muts := make([]*spanner.Mutation, 0, len(chunk))
			for _, em := range chunk {
				muts = append(muts, spanner.InsertOrUpdateMap("ConnectorEntityMaps", map[string]any{
					"TenantId":     tid,
					"Provider":     normalizeProvider(em.Provider),
					"ObjectType":   em.ObjectType,
					"ExternalId":   em.ExternalID,
					"InternalId":   em.InternalID,
					"ContentHash":  nullStr(em.ContentHash),
					"LastRunId":    nullStr(em.LastRunID),
					"LastSyncedAt": spanner.CommitTimestamp,
				}))
			}
			return txn.BufferWrite(muts)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *SpannerRepository) GetSyncRun(ctx context.Context, tenantID, runID string) (ConnectorSyncRun, bool, error) {
	if r == nil || r.client == nil {
		return ConnectorSyncRun{}, false, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, RunId, Provider, AccountLabel, Direction, StreamName, Status,
		             RecordsProcessed, MismatchCount, MismatchSummaryJson, ErrorMessage,
		             IdempotencyKey, StartedAt, CompletedAt, CreatedAt
		      FROM ConnectorSyncRuns
		      WHERE TenantId = @tid AND RunId = @runId
		      LIMIT 1`,
		Params: map[string]any{"tid": strings.TrimSpace(tenantID), "runId": strings.TrimSpace(runID)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ConnectorSyncRun{}, false, nil
	}
	if err != nil {
		return ConnectorSyncRun{}, false, fmt.Errorf("get sync run: %w", err)
	}
	run, err := scanSyncRunRow(row)
	if err != nil {
		return ConnectorSyncRun{}, false, err
	}
	return run, true, nil
}

func (r *SpannerRepository) ListSyncRuns(ctx context.Context, tenantID, provider string, limit int) ([]ConnectorSyncRun, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	sql := `SELECT TenantId, RunId, Provider, AccountLabel, Direction, StreamName, Status,
	               RecordsProcessed, MismatchCount, MismatchSummaryJson, ErrorMessage,
	               IdempotencyKey, StartedAt, CompletedAt, CreatedAt
	        FROM ConnectorSyncRuns
	        WHERE TenantId = @tid`
	params := map[string]any{"tid": strings.TrimSpace(tenantID), "lim": int64(limit)}
	if prov := normalizeProvider(provider); prov != "" {
		sql += ` AND Provider = @prov`
		params["prov"] = prov
	}
	sql += ` ORDER BY StartedAt DESC LIMIT @lim`

	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]ConnectorSyncRun, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list sync runs: %w", err)
		}
		run, sErr := scanSyncRunRow(row)
		if sErr != nil {
			return nil, sErr
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *SpannerRepository) ListLatestRunsByProvider(ctx context.Context, tenantID string) ([]ConnectorSyncRun, error) {
	runs, err := r.ListSyncRuns(ctx, tenantID, "", 100)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	latest := make([]ConnectorSyncRun, 0)
	for _, run := range runs {
		k := run.Provider + "|" + run.AccountLabel
		if seen[k] {
			continue
		}
		seen[k] = true
		latest = append(latest, run)
	}
	return latest, nil
}

func (r *SpannerRepository) LoadInternalSnapshot(ctx context.Context, tenantID, provider string) (InternalSnapshot, error) {
	if r == nil || r.client == nil {
		return InternalSnapshot{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	prov := normalizeProvider(provider)
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	var snap InternalSnapshot

	// 1. Products
	pIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, Name, Barcode, PriceMinor, StockQuantity, Currency, IsActive
		      FROM Products
		      WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := pIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			pIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("snapshot Products: %w", err)
		}
		var p InternalProduct
		var barcode spanner.NullString
		if err := row.Columns(&p.ProductID, &p.Name, &barcode, &p.PriceMinor, &p.StockQuantity, &p.Currency, &p.IsActive); err != nil {
			pIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("scan snapshot Products: %w", err)
		}
		p.Barcode = barcode.StringVal
		snap.Products = append(snap.Products, p)
	}
	pIter.Stop()

	// 2. SupplierInventoryV2
	sIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, ProductId, QuantityOnHand
		      FROM SupplierInventoryV2
		      WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := sIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			sIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("snapshot SupplierInventoryV2: %w", err)
		}
		var st InternalStock
		if err := row.Columns(&st.WarehouseID, &st.ProductID, &st.QuantityOnHand); err != nil {
			sIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("scan snapshot SupplierInventoryV2: %w", err)
		}
		snap.Stocks = append(snap.Stocks, st)
	}
	sIter.Stop()

	// 3. Orders
	oIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, TotalMinor, Currency, Status
		      FROM Orders
		      WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := oIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			oIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("snapshot Orders: %w", err)
		}
		var ord InternalOrder
		if err := row.Columns(&ord.OrderID, &ord.RetailerID, &ord.TotalMinor, &ord.Currency, &ord.Status); err != nil {
			oIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("scan snapshot Orders: %w", err)
		}
		snap.Orders = append(snap.Orders, ord)
	}
	oIter.Stop()

	// 4. ArInvoices
	iIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT InvoiceId, OrderId, RetailerId, PrincipalMinor, BalanceMinor, Currency, Status
		      FROM ArInvoices
		      WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	for {
		row, err := iIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("snapshot ArInvoices: %w", err)
		}
		var inv InternalInvoice
		if err := row.Columns(&inv.InvoiceID, &inv.OrderID, &inv.RetailerID, &inv.PrincipalMinor, &inv.BalanceMinor, &inv.Currency, &inv.Status); err != nil {
			iIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("scan snapshot ArInvoices: %w", err)
		}
		snap.Invoices = append(snap.Invoices, inv)
	}
	iIter.Stop()

	// 5. ConnectorEntityMaps
	mIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, Provider, ObjectType, ExternalId, InternalId, ContentHash, LastRunId, LastSyncedAt
		      FROM ConnectorEntityMaps
		      WHERE TenantId = @tid AND Provider = @prov`,
		Params: map[string]any{"tid": tid, "prov": prov},
	})
	for {
		row, err := mIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			mIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("snapshot ConnectorEntityMaps: %w", err)
		}
		var em ConnectorEntityMap
		var hash, runID spanner.NullString
		if err := row.Columns(&em.TenantID, &em.Provider, &em.ObjectType, &em.ExternalID, &em.InternalID, &hash, &runID, &em.LastSyncedAt); err != nil {
			mIter.Stop()
			return InternalSnapshot{}, fmt.Errorf("scan snapshot ConnectorEntityMaps: %w", err)
		}
		em.ContentHash = hash.StringVal
		em.LastRunID = runID.StringVal
		em.LastSyncedAt = em.LastSyncedAt.UTC()
		snap.EntityMaps = append(snap.EntityMaps, em)
	}
	mIter.Stop()

	return snap, nil
}

func scanCredentialRow(row *spanner.Row) (ExternalCredential, error) {
	var c ExternalCredential
	var baseURL, metaJSON spanner.NullString
	if err := row.Columns(
		&c.TenantID,
		&c.CredentialID,
		&c.Provider,
		&c.AccountLabel,
		&c.AuthType,
		&c.SecretRef,
		&baseURL,
		&metaJSON,
		&c.Status,
		&c.CreatedBy,
		&c.CreatedAt,
		&c.UpdatedAt,
	); err != nil {
		return ExternalCredential{}, fmt.Errorf("scan ExternalCredentials: %w", err)
	}
	c.BaseURL = baseURL.StringVal
	c.MetadataJSON = metaJSON.StringVal
	if strings.TrimSpace(c.MetadataJSON) != "" {
		_ = decodeJSONWithNumbers([]byte(c.MetadataJSON), &c.Metadata)
	}
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()
	return c, nil
}

func scanSyncRunRow(row *spanner.Row) (ConnectorSyncRun, error) {
	var run ConnectorSyncRun
	var summaryJSON, errMsg, idem spanner.NullString
	var completed spanner.NullTime
	if err := row.Columns(
		&run.TenantID,
		&run.RunID,
		&run.Provider,
		&run.AccountLabel,
		&run.Direction,
		&run.StreamName,
		&run.Status,
		&run.RecordsProcessed,
		&run.MismatchCount,
		&summaryJSON,
		&errMsg,
		&idem,
		&run.StartedAt,
		&completed,
		&run.CreatedAt,
	); err != nil {
		return ConnectorSyncRun{}, fmt.Errorf("scan ConnectorSyncRuns: %w", err)
	}
	run.MismatchSummaryJSON = summaryJSON.StringVal
	run.ErrorMessage = errMsg.StringVal
	run.IdempotencyKey = idem.StringVal
	run.StartedAt = run.StartedAt.UTC()
	run.CreatedAt = run.CreatedAt.UTC()
	if completed.Valid {
		t := completed.Time.UTC()
		run.CompletedAt = &t
	}
	if strings.TrimSpace(run.MismatchSummaryJSON) != "" {
		var env ShadowCompareEnvelope
		if err := decodeJSONWithNumbers([]byte(run.MismatchSummaryJSON), &env); err == nil {
			run.Summary = &env.Summary
			run.Mismatches = env.Mismatches
		}
	}
	return run, nil
}

func nullStr(s string) spanner.NullString {
	t := strings.TrimSpace(s)
	if t == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: t, Valid: true}
}

func sanitizeIDToken(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '-' || r == '_' {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "default"
	}
	if len(out) > 32 {
		return out[:32]
	}
	return out
}
