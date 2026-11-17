package multientity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository implements Repository against Cloud Spanner using ReadOnlyTransaction
// for consistent snapshot reads and ReadWriteTransaction for atomic GL + Outbox + Audit writes.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed multi-entity Repository.
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

func entityColumns() []string {
	return []string{
		"TenantId", "EntityId", "Name", "Tin", "CountryCode", "BaseCurrency",
		"IsParent", "ParentEntityId", "Status", "CreatedAt", "UpdatedAt",
	}
}

func scanEntityRow(row *spanner.Row) (LegalEntity, error) {
	var ent LegalEntity
	var tin, parentID spanner.NullString
	if err := row.Columns(
		&ent.TenantID, &ent.EntityID, &ent.Name, &tin, &ent.CountryCode, &ent.BaseCurrency,
		&ent.IsParent, &parentID, &ent.Status, &ent.CreatedAt, &ent.UpdatedAt,
	); err != nil {
		return LegalEntity{}, fmt.Errorf("scan LegalEntities: %w", err)
	}
	ent.TIN = nullStr(tin)
	ent.ParentEntityID = nullStr(parentID)
	ent.CreatedAt = ent.CreatedAt.UTC()
	ent.UpdatedAt = ent.UpdatedAt.UTC()
	return ent, nil
}

func (r *SpannerRepository) ListEntities(ctx context.Context, tenantID, statusFilter string) ([]LegalEntity, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	sf := strings.ToUpper(strings.TrimSpace(statusFilter))

	sql := `SELECT ` + strings.Join(entityColumns(), ", ") + ` FROM LegalEntities WHERE TenantId = @tenantId`
	params := map[string]any{"tenantId": tenantID}
	if sf != "" {
		sql += ` AND UPPER(Status) = @status`
		params["status"] = sf
	}
	sql += ` ORDER BY IsParent DESC, EntityId ASC`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]LegalEntity, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list LegalEntities: %w", err)
		}
		ent, err := scanEntityRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, ent)
	}
	return out, nil
}

func (r *SpannerRepository) GetEntity(ctx context.Context, tenantID, entityID string) (LegalEntity, bool, error) {
	if r == nil || r.client == nil {
		return LegalEntity{}, false, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	row, err := ro.ReadRow(ctx, "LegalEntities", spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(entityID)}, entityColumns())
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return LegalEntity{}, false, nil
		}
		return LegalEntity{}, false, fmt.Errorf("read LegalEntities(%s, %s): %w", tenantID, entityID, err)
	}
	ent, err := scanEntityRow(row)
	if err != nil {
		return LegalEntity{}, false, err
	}
	return ent, true, nil
}

func (r *SpannerRepository) IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error) {
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	if tenantID == "" || entityID == "" {
		return false, nil
	}
	if tenantID == entityID {
		return true, nil
	}
	_, ok, err := r.GetEntity(ctx, tenantID, entityID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (r *SpannerRepository) CreateEntity(
	ctx context.Context,
	ent LegalEntity,
	seedGL bool,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (LegalEntity, error) {
	if r == nil || r.client == nil {
		return LegalEntity{}, ErrRepoUnavailable
	}

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(txCtx, "LegalEntities", spanner.Key{ent.TenantID, ent.EntityID}, []string{"EntityId"})
		if readErr == nil {
			return fmt.Errorf("%w: %s", ErrEntityExists, ent.EntityID)
		}
		if status.Code(readErr) != codes.NotFound && spanner.ErrCode(readErr) != codes.NotFound {
			return fmt.Errorf("check LegalEntities PK: %w", readErr)
		}
		if ent.ParentEntityID != "" {
			if _, pErr := txn.ReadRow(txCtx, "LegalEntities", spanner.Key{ent.TenantID, ent.ParentEntityID}, []string{"EntityId"}); pErr != nil {
				if status.Code(pErr) == codes.NotFound || spanner.ErrCode(pErr) == codes.NotFound {
					return fmt.Errorf("%w: parent_entity_id %s", ErrEntityNotFound, ent.ParentEntityID)
				}
				return fmt.Errorf("check parent LegalEntities PK: %w", pErr)
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("LegalEntities", map[string]any{
				"TenantId":       ent.TenantID,
				"EntityId":       ent.EntityID,
				"Name":           ent.Name,
				"Tin":            nullIfEmpty(ent.TIN),
				"CountryCode":    ent.CountryCode,
				"BaseCurrency":   ent.BaseCurrency,
				"IsParent":       ent.IsParent,
				"ParentEntityId": nullIfEmpty(ent.ParentEntityID),
				"Status":         ent.Status,
				"CreatedAt":      spanner.CommitTimestamp,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert LegalEntities: %w", err)
		}

		if seedGL {
			if _, err := gl.SeedDefaultAccountsInTxn(txCtx, txn, ent.TenantID, ent.CountryCode, ent.BaseCurrency); err != nil {
				return fmt.Errorf("seed default GL accounts: %w", err)
			}
		}

		now := time.Now().UTC()
		outboxBuf := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &tenantStampedOutboxBuf{inner: outboxBuf, tenantID: ent.TenantID}
		ev := events.OntologyObjectChangedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventOntologyObjectChanged,
				Version:   1,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			CellID:          "cell-local",
			ObjectType:      "LegalEntity",
			ObjectID:        ent.EntityID,
			SupplierID:      ent.TenantID,
			Version:         1,
			ChangeKind:      "CREATED",
			SourceEventType: "org.entity.created",
			UpdatedAt:       now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateLegalEntity, ent.EntityID, events.TopicIntel, ev); err != nil {
			return fmt.Errorf("emit legal entity outbox: %w", err)
		}
		if err := outboxBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush legal entity outbox: %w", err)
		}

		if auditRec != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn); err != nil {
				return fmt.Errorf("audit org.entity.create: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return LegalEntity{}, err
	}
	loaded, ok, err := r.GetEntity(ctx, ent.TenantID, ent.EntityID)
	if err == nil && ok {
		return loaded, nil
	}
	now := time.Now().UTC()
	ent.CreatedAt = now
	ent.UpdatedAt = now
	return ent, nil
}

func (r *SpannerRepository) PatchEntity(
	ctx context.Context,
	tenantID, entityID string,
	mutate func(cur LegalEntity) (LegalEntity, error),
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (LegalEntity, LegalEntity, error) {
	if r == nil || r.client == nil {
		return LegalEntity{}, LegalEntity{}, ErrRepoUnavailable
	}
	var before, after LegalEntity
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "LegalEntities", spanner.Key{tenantID, entityID}, entityColumns())
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return fmt.Errorf("%w: %s", ErrEntityNotFound, entityID)
			}
			return fmt.Errorf("read LegalEntities for patch: %w", err)
		}
		cur, err := scanEntityRow(row)
		if err != nil {
			return err
		}
		before = cur
		next, err := mutate(cur)
		if err != nil {
			return err
		}
		if next.ParentEntityID != "" {
			if next.ParentEntityID == entityID {
				return fmt.Errorf("%w: entity cannot be its own parent", ErrInvalidInput)
			}
			if _, pErr := txn.ReadRow(txCtx, "LegalEntities", spanner.Key{tenantID, next.ParentEntityID}, []string{"EntityId"}); pErr != nil {
				if status.Code(pErr) == codes.NotFound || spanner.ErrCode(pErr) == codes.NotFound {
					return fmt.Errorf("%w: parent_entity_id %s", ErrEntityNotFound, next.ParentEntityID)
				}
				return fmt.Errorf("check parent LegalEntities PK: %w", pErr)
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("LegalEntities", map[string]any{
				"TenantId":       tenantID,
				"EntityId":       entityID,
				"Name":           next.Name,
				"Tin":            nullIfEmpty(next.TIN),
				"CountryCode":    next.CountryCode,
				"BaseCurrency":   next.BaseCurrency,
				"IsParent":       next.IsParent,
				"ParentEntityId": nullIfEmpty(next.ParentEntityID),
				"Status":         next.Status,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("update LegalEntities: %w", err)
		}

		now := time.Now().UTC()
		ver := now.UnixMicro()
		if ver < 2 {
			ver = 2
		}
		outboxBuf := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &tenantStampedOutboxBuf{inner: outboxBuf, tenantID: tenantID}
		ev := events.OntologyObjectChangedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventOntologyObjectChanged,
				Version:   ver,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			CellID:          "cell-local",
			ObjectType:      "LegalEntity",
			ObjectID:        entityID,
			SupplierID:      tenantID,
			Version:         ver,
			ChangeKind:      "UPDATED",
			SourceEventType: "org.entity.updated",
			UpdatedAt:       now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateLegalEntity, entityID, events.TopicIntel, ev); err != nil {
			return fmt.Errorf("emit legal entity patch outbox: %w", err)
		}
		if err := outboxBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush legal entity patch outbox: %w", err)
		}

		if auditRec != nil {
			aIn := auditIn
			if aIn.BeforeJSON == "" {
				if bb, bErr := json.Marshal(cur); bErr == nil {
					aIn.BeforeJSON = string(bb)
				}
			}
			if ab, aErr := json.Marshal(next); aErr == nil {
				aIn.AfterJSON = string(ab)
			}
			if err := auditRec.RecordEventInTxn(txCtx, txn, aIn); err != nil {
				return fmt.Errorf("audit org.entity.patch: %w", err)
			}
		}
		after = next
		return nil
	})
	if err != nil {
		return LegalEntity{}, LegalEntity{}, err
	}
	if loaded, ok, err := r.GetEntity(ctx, tenantID, entityID); err == nil && ok {
		after = loaded
	}
	return before, after, nil
}

func transferColumns() []string {
	return []string{
		"TenantId", "TransferId", "FromEntityId", "ToEntityId", "TransferType",
		"AmountMinor", "Currency", "PeriodId", "ReferenceType", "ReferenceId",
		"DebitJournalId", "CreditJournalId", "Status", "IdempotencyKey", "CreatedBy", "CreatedAt",
	}
}

func scanTransferRow(row *spanner.Row) (IntercompanyTransfer, error) {
	var tr IntercompanyTransfer
	var refType, refID, debJ, credJ, idem spanner.NullString
	if err := row.Columns(
		&tr.TenantID, &tr.TransferID, &tr.FromEntityID, &tr.ToEntityID, &tr.TransferType,
		&tr.AmountMinor, &tr.Currency, &tr.PeriodID, &refType, &refID,
		&debJ, &credJ, &tr.Status, &idem, &tr.CreatedBy, &tr.CreatedAt,
	); err != nil {
		return IntercompanyTransfer{}, fmt.Errorf("scan IntercompanyTransfers: %w", err)
	}
	tr.ReferenceType = nullStr(refType)
	tr.ReferenceID = nullStr(refID)
	tr.DebitJournalID = nullStr(debJ)
	tr.CreditJournalID = nullStr(credJ)
	tr.IdempotencyKey = nullStr(idem)
	tr.CreatedAt = tr.CreatedAt.UTC()
	return tr, nil
}

func (r *SpannerRepository) ListTransfers(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]IntercompanyTransfer, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	periodID = strings.TrimSpace(periodID)
	entityID = strings.TrimSpace(entityID)
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	params := map[string]any{
		"tenantId": tenantID,
		"limit":    int64(limit),
	}
	sql := `SELECT ` + strings.Join(transferColumns(), ", ") + ` FROM IntercompanyTransfers WHERE TenantId = @tenantId`
	if periodID != "" {
		sql += ` AND PeriodId = @periodId`
		params["periodId"] = periodID
	}
	if entityID != "" {
		sql += ` AND (FromEntityId = @entityId OR ToEntityId = @entityId)`
		params["entityId"] = entityID
	}
	sql += ` ORDER BY CreatedAt DESC, TransferId DESC LIMIT @limit`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]IntercompanyTransfer, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list IntercompanyTransfers: %w", err)
		}
		tr, err := scanTransferRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, nil
}

func (r *SpannerRepository) CreateTransferAtomic(
	ctx context.Context,
	tr IntercompanyTransfer,
	debitJE, creditJE gl.JournalEntryInput,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (IntercompanyTransfer, bool, error) {
	if r == nil || r.client == nil {
		return IntercompanyTransfer{}, false, ErrRepoUnavailable
	}
	var out IntercompanyTransfer
	var alreadyPosted bool

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if tr.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT ` + strings.Join(transferColumns(), ", ") + `
				      FROM IntercompanyTransfers@{FORCE_INDEX=UQ_IntercompanyTransfers_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tenantId AND IdempotencyKey = @idemKey AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{
					"tenantId": tr.TenantID,
					"idemKey":  tr.IdempotencyKey,
				},
			})
			row, rErr := iter.Next()
			iter.Stop()
			if rErr == nil {
				existing, sErr := scanTransferRow(row)
				if sErr != nil {
					return sErr
				}
				out = existing
				alreadyPosted = true
				return nil
			}
			if rErr != iterator.Done {
				return fmt.Errorf("check IntercompanyTransfers idempotency: %w", rErr)
			}
		}

		if err := ensureEntityActiveInTxn(txCtx, txn, tr.TenantID, tr.FromEntityID); err != nil {
			return err
		}
		if err := ensureEntityActiveInTxn(txCtx, txn, tr.TenantID, tr.ToEntityID); err != nil {
			return err
		}

		outboxBuf := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &tenantStampedOutboxBuf{inner: outboxBuf, tenantID: tr.TenantID}

		jeResults, err := gl.BufferJournalEntriesBatchInTxn(txCtx, txn, stampedBuf, []gl.JournalEntryInput{debitJE, creditJE})
		if err != nil {
			return err
		}
		tr.DebitJournalID = jeResults[0].EntryID
		tr.CreditJournalID = jeResults[1].EntryID

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("IntercompanyTransfers", map[string]any{
				"TenantId":        tr.TenantID,
				"TransferId":      tr.TransferID,
				"FromEntityId":    tr.FromEntityID,
				"ToEntityId":      tr.ToEntityID,
				"TransferType":    tr.TransferType,
				"AmountMinor":     tr.AmountMinor,
				"Currency":        tr.Currency,
				"PeriodId":        tr.PeriodID,
				"ReferenceType":   nullIfEmpty(tr.ReferenceType),
				"ReferenceId":     nullIfEmpty(tr.ReferenceID),
				"DebitJournalId":  tr.DebitJournalID,
				"CreditJournalId": tr.CreditJournalID,
				"Status":          tr.Status,
				"IdempotencyKey":  nullIfEmpty(tr.IdempotencyKey),
				"CreatedBy":       tr.CreatedBy,
				"CreatedAt":       spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert IntercompanyTransfers: %w", err)
		}

		ts := debitJE.OccurredAt.UTC().Format(time.RFC3339Nano)
		ev := events.EntityIntercompanyPostedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventEntityIntercompanyPosted,
				Version:   1,
				Timestamp: ts,
			},
			TenantID:        tr.TenantID,
			TransferID:      tr.TransferID,
			FromEntityID:    tr.FromEntityID,
			ToEntityID:      tr.ToEntityID,
			TransferType:    tr.TransferType,
			AmountMinor:     tr.AmountMinor,
			Currency:        tr.Currency,
			PeriodID:        tr.PeriodID,
			ReferenceType:   tr.ReferenceType,
			ReferenceID:     tr.ReferenceID,
			DebitJournalID:  tr.DebitJournalID,
			CreditJournalID: tr.CreditJournalID,
			Status:          tr.Status,
			CreatedBy:       tr.CreatedBy,
			OccurredAt:      ts,
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateIntercompanyTransfer, tr.TransferID, events.TopicLedger, ev); err != nil {
			return fmt.Errorf("emit entity.intercompany.posted: %w", err)
		}
		if err := outboxBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush intercompany outbox: %w", err)
		}

		if auditRec != nil {
			aIn := auditIn
			aIn.ObjectID = tr.TransferID
			if err := auditRec.RecordEventInTxn(txCtx, txn, aIn); err != nil {
				return fmt.Errorf("audit intercompany transfer: %w", err)
			}
		}

		out = tr
		out.CreatedAt = time.Now().UTC()
		alreadyPosted = false
		return nil
	})
	if err != nil {
		return IntercompanyTransfer{}, false, err
	}
	if !alreadyPosted {
		ro := r.client.ReadOnlyTransaction()
		defer ro.Close()
		if row, rErr := ro.ReadRow(ctx, "IntercompanyTransfers", spanner.Key{out.TenantID, out.TransferID}, transferColumns()); rErr == nil {
			if loaded, sErr := scanTransferRow(row); sErr == nil {
				return loaded, false, nil
			}
		}
	}
	return out, alreadyPosted, nil
}

func ensureEntityActiveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, entityID string) error {
	row, err := txn.ReadRow(ctx, "LegalEntities", spanner.Key{tenantID, entityID}, []string{"Status"})
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return fmt.Errorf("%w: %s", ErrEntityNotFound, entityID)
		}
		return fmt.Errorf("read LegalEntities(%s, %s): %w", tenantID, entityID, err)
	}
	var st string
	if err := row.Columns(&st); err != nil {
		return fmt.Errorf("scan LegalEntities status: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(st)) != EntityStatusActive {
		return fmt.Errorf("%w: %s status=%s", ErrEntityInactive, entityID, st)
	}
	return nil
}

func revalColumns() []string {
	return []string{
		"TenantId", "RunId", "EntityId", "PeriodId", "BaseCurrency",
		"UnrealizedGainLossMinor", "RoundingDeltaMinor", "RatesJson",
		"JournalEntryId", "IdempotencyKey", "CreatedBy", "CreatedAt",
	}
}

func scanRevalRow(row *spanner.Row) (FxRevaluationRun, error) {
	var run FxRevaluationRun
	var entID, ratesJSON, jeID, idem spanner.NullString
	if err := row.Columns(
		&run.TenantID, &run.RunID, &entID, &run.PeriodID, &run.BaseCurrency,
		&run.UnrealizedGainLossMinor, &run.RoundingDeltaMinor, &ratesJSON,
		&jeID, &idem, &run.CreatedBy, &run.CreatedAt,
	); err != nil {
		return FxRevaluationRun{}, fmt.Errorf("scan FxRevaluationRuns: %w", err)
	}
	run.EntityID = nullStr(entID)
	run.RatesJSON = nullStr(ratesJSON)
	run.JournalEntryID = nullStr(jeID)
	run.IdempotencyKey = nullStr(idem)
	run.CreatedAt = run.CreatedAt.UTC()
	return run, nil
}

func (r *SpannerRepository) ListFxRevaluationRuns(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]FxRevaluationRun, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	periodID = strings.TrimSpace(periodID)
	entityID = strings.TrimSpace(entityID)
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	params := map[string]any{"tenantId": tenantID, "limit": int64(limit)}
	sql := `SELECT ` + strings.Join(revalColumns(), ", ") + ` FROM FxRevaluationRuns WHERE TenantId = @tenantId`
	if periodID != "" {
		sql += ` AND PeriodId = @periodId`
		params["periodId"] = periodID
	}
	if entityID != "" {
		sql += ` AND EntityId = @entityId`
		params["entityId"] = entityID
	}
	sql += ` ORDER BY CreatedAt DESC, RunId DESC LIMIT @limit`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]FxRevaluationRun, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list FxRevaluationRuns: %w", err)
		}
		run, err := scanRevalRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *SpannerRepository) CollectOpenForeignCurrencyPositions(
	ctx context.Context,
	tenantID, entityID, baseCurrency string,
	asOf time.Time,
) ([]FxRevalPosition, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	includeTenantSubledger := entityID == ""
	if entityID != "" {
		row, err := ro.ReadRow(ctx, "LegalEntities", spanner.Key{tenantID, entityID}, []string{"IsParent", "Status"})
		if err != nil {
			if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
				return nil, fmt.Errorf("%w: %s", ErrEntityNotFound, entityID)
			}
			return nil, fmt.Errorf("read LegalEntities for revaluation: %w", err)
		}
		var isParent bool
		var st string
		if err := row.Columns(&isParent, &st); err == nil && isParent {
			includeTenantSubledger = true
		}
	}

	positions := make([]FxRevalPosition, 0)
	glOrderIDs := make(map[string]struct{})
	glSourceIDs := make(map[string]struct{})

	// 1. Open foreign-currency GL monetary positions (AR, AP, GRNI, Cash/Bank) plus prior FX revaluation functional deltas.
	glParams := map[string]any{
		"tenantId": tenantID,
		"asOf":     asOf,
	}
	glSQL := `SELECT p.AccountCode, p.Direction, p.AmountMinor, p.Currency, p.FunctionalAmountMinor,
	                 COALESCE(p.FxRateId, ''), COALESCE(p.PartyType, ''), COALESCE(p.PartyId, ''),
	                 e.SourceType, e.SourceId, COALESCE(e.OrderId, ''), e.OccurredAt
	          FROM JournalPostings p
	          JOIN JournalEntries e ON p.EntityId = e.EntityId AND p.EntryId = e.EntryId
	          WHERE e.EntityId = @tenantId
	            AND e.OccurredAt <= @asOf
	            AND p.AccountCode IN ('AR:RETAILER', 'AP:VENDOR', 'AP:SUPPLIER', 'GRNI', 'CASH:OFFICE', 'CASH:DRIVER', 'BANK:OPERATING', 'PSP:GATEWAY')`
	if entityID != "" {
		if includeTenantSubledger {
			glSQL += ` AND (e.LegalEntityId = @entityId OR e.LegalEntityId IS NULL OR e.LegalEntityId = '')`
		} else {
			glSQL += ` AND e.LegalEntityId = @entityId`
		}
		glParams["entityId"] = entityID
	}
	glSQL += ` ORDER BY e.OccurredAt ASC, e.EntryId ASC, p.Line ASC`

	type glKey struct {
		acct string
		curr string
	}
	glBuckets := make(map[glKey]*FxRevalPosition)
	glOrder := make([]glKey, 0)
	priorRevalByAcctCurr := make(map[glKey]int64)

	glIter := ro.Query(ctx, spanner.Statement{SQL: glSQL, Params: glParams})
	for {
		row, err := glIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			glIter.Stop()
			return nil, fmt.Errorf("query GL open foreign currency positions: %w", err)
		}
		var acct, dir, curr, fxRateID, partyType, partyID, srcType, srcID, orderID string
		var amtMinor, funcAmtMinor int64
		var occAt time.Time
		if err := row.Columns(&acct, &dir, &amtMinor, &curr, &funcAmtMinor, &fxRateID, &partyType, &partyID, &srcType, &srcID, &orderID, &occAt); err != nil {
			glIter.Stop()
			return nil, fmt.Errorf("scan GL foreign currency position: %w", err)
		}
		acct = strings.ToUpper(strings.TrimSpace(acct))
		curr = strings.ToUpper(strings.TrimSpace(curr))
		occAt = occAt.UTC()

		// Prior FX revaluation functional adjustments carry PartyType="FX_CURRENCY", PartyID=<foreignCurrency>.
		if strings.EqualFold(partyType, "FX_CURRENCY") && strings.TrimSpace(partyID) != "" && curr == baseCurrency {
			fc := strings.ToUpper(strings.TrimSpace(partyID))
			k := glKey{acct: acct, curr: fc}
			priorRevalByAcctCurr[k] += signedAccountAmount(acct, gl.Direction(dir), funcAmtMinor)
			continue
		}

		if curr == "" || curr == baseCurrency {
			continue
		}
		if orderID != "" {
			glOrderIDs[orderID] = struct{}{}
		}
		if srcID != "" {
			glSourceIDs[srcID] = struct{}{}
		}

		k := glKey{acct: acct, curr: curr}
		pos, exists := glBuckets[k]
		if !exists {
			posDir := "ASSET"
			if !gl.IsDebitNormalClass(accountClassFor(acct)) {
				posDir = "LIABILITY"
			}
			pos = &FxRevalPosition{
				SourceType:  "GL_BALANCE",
				SourceID:    acct + ":" + curr,
				AccountCode: acct,
				Direction:   posDir,
				Currency:    curr,
				OccurredAt:  occAt,
			}
			glBuckets[k] = pos
			glOrder = append(glOrder, k)
		}
		pos.ForeignAmountMinor += signedAccountAmount(acct, gl.Direction(dir), amtMinor)
		if fxRateID != "" || funcAmtMinor != amtMinor {
			pos.CarryingFunctionalMinor += signedAccountAmount(acct, gl.Direction(dir), funcAmtMinor)
			pos.HasHistoricalFunctional = true
		}
		if occAt.After(pos.OccurredAt) {
			pos.OccurredAt = occAt
		}
	}
	glIter.Stop()

	for _, k := range glOrder {
		pos := glBuckets[k]
		if pos.ForeignAmountMinor != 0 {
			if prior, ok := priorRevalByAcctCurr[k]; ok && prior != 0 {
				pos.PriorRevalFunctionalMinor += prior
				delete(priorRevalByAcctCurr, k)
			}
			positions = append(positions, *pos)
		}
	}

	// 2. Open foreign-currency ArInvoices (AR) not already represented in GL.
	if includeTenantSubledger {
		arIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT InvoiceId, OrderId, Currency, BalanceMinor, CreatedAt
			      FROM ArInvoices
			      WHERE SupplierId = @tenantId
			        AND UPPER(Currency) != @baseCurrency
			        AND BalanceMinor > 0
			        AND CreatedAt <= @asOf
			        AND UPPER(Status) NOT IN ('PAID', 'WRITTEN_OFF', 'CANCELLED')
			      ORDER BY CreatedAt ASC, InvoiceId ASC`,
			Params: map[string]any{
				"tenantId":     tenantID,
				"baseCurrency": baseCurrency,
				"asOf":         asOf,
			},
		})
		for {
			row, err := arIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				arIter.Stop()
				return nil, fmt.Errorf("query ArInvoices for FX revaluation: %w", err)
			}
			var invID, orderID, curr string
			var balMinor int64
			var createdAt time.Time
			if err := row.Columns(&invID, &orderID, &curr, &balMinor, &createdAt); err != nil {
				arIter.Stop()
				return nil, fmt.Errorf("scan ArInvoices for FX revaluation: %w", err)
			}
			if _, covered := glOrderIDs[orderID]; covered {
				continue
			}
			if _, covered := glSourceIDs[invID]; covered {
				continue
			}
			positions = append(positions, FxRevalPosition{
				SourceType:         "AR_INVOICE",
				SourceID:           invID,
				AccountCode:        gl.AccountAccountsReceivable,
				Direction:          "ASSET",
				Currency:           strings.ToUpper(strings.TrimSpace(curr)),
				ForeignAmountMinor: balMinor,
				OccurredAt:         createdAt.UTC(),
			})
		}
		arIter.Stop()
	}

	// 3. Open foreign-currency VendorInvoices (AP) not already represented in GL.
	viParams := map[string]any{
		"tenantId":     tenantID,
		"baseCurrency": baseCurrency,
		"asOf":         asOf,
	}
	viSQL := `SELECT VendorInvoiceId, PoId, Currency, TotalMinor, COALESCE(JournalEntryId, ''), CreatedAt
	          FROM VendorInvoices
	          WHERE TenantId = @tenantId
	            AND UPPER(Currency) != @baseCurrency
	            AND TotalMinor > 0
	            AND CreatedAt <= @asOf
	            AND UPPER(MatchStatus) NOT IN ('PAID', 'CANCELLED', 'VOID')`
	if entityID != "" {
		viSQL += ` AND EntityId = @entityId`
		viParams["entityId"] = entityID
	}
	viSQL += ` ORDER BY CreatedAt ASC, VendorInvoiceId ASC`

	viIter := ro.Query(ctx, spanner.Statement{SQL: viSQL, Params: viParams})
	for {
		row, err := viIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			viIter.Stop()
			return nil, fmt.Errorf("query VendorInvoices for FX revaluation: %w", err)
		}
		var viID, poID, curr, jeID string
		var totalMinor int64
		var createdAt time.Time
		if err := row.Columns(&viID, &poID, &curr, &totalMinor, &jeID, &createdAt); err != nil {
			viIter.Stop()
			return nil, fmt.Errorf("scan VendorInvoices for FX revaluation: %w", err)
		}
		if _, covered := glSourceIDs[viID]; covered {
			continue
		}
		if _, covered := glSourceIDs[poID]; covered {
			continue
		}
		positions = append(positions, FxRevalPosition{
			SourceType:         "VENDOR_INVOICE",
			SourceID:           viID,
			AccountCode:        gl.AccountVendorPayable,
			Direction:          "LIABILITY",
			Currency:           strings.ToUpper(strings.TrimSpace(curr)),
			ForeignAmountMinor: totalMinor,
			OccurredAt:         createdAt.UTC(),
		})
	}
	viIter.Stop()

	// 4. Open foreign-currency PayoutBatches (AP:SUPPLIER) not yet paid or in GL.
	if includeTenantSubledger {
		pbIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT BatchId, Currency, NetPayoutMinor, CreatedAt
			      FROM PayoutBatches
			      WHERE SupplierId = @tenantId
			        AND UPPER(Currency) != @baseCurrency
			        AND NetPayoutMinor > 0
			        AND CreatedAt <= @asOf
			        AND UPPER(Status) != 'PAID'
			      ORDER BY CreatedAt ASC, BatchId ASC`,
			Params: map[string]any{
				"tenantId":     tenantID,
				"baseCurrency": baseCurrency,
				"asOf":         asOf,
			},
		})
		for {
			row, err := pbIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				pbIter.Stop()
				return nil, fmt.Errorf("query PayoutBatches for FX revaluation: %w", err)
			}
			var batchID, curr string
			var netMinor int64
			var createdAt time.Time
			if err := row.Columns(&batchID, &curr, &netMinor, &createdAt); err != nil {
				pbIter.Stop()
				return nil, fmt.Errorf("scan PayoutBatches for FX revaluation: %w", err)
			}
			if _, covered := glSourceIDs[batchID]; covered {
				continue
			}
			positions = append(positions, FxRevalPosition{
				SourceType:         "PAYOUT_BATCH",
				SourceID:           batchID,
				AccountCode:        gl.AccountSupplierPayable,
				Direction:          "LIABILITY",
				Currency:           strings.ToUpper(strings.TrimSpace(curr)),
				ForeignAmountMinor: netMinor,
				OccurredAt:         createdAt.UTC(),
			})
		}
		pbIter.Stop()
	}

	// Attach any remaining prior FX revaluation deltas (e.g., from previous runs on subledger items)
	// to the first matching subledger position for (AccountCode, Currency).
	for i := range positions {
		k := glKey{acct: positions[i].AccountCode, curr: positions[i].Currency}
		if prior, ok := priorRevalByAcctCurr[k]; ok && prior != 0 {
			positions[i].PriorRevalFunctionalMinor += prior
			delete(priorRevalByAcctCurr, k)
		}
	}

	return positions, nil
}

func (r *SpannerRepository) CreateFxRevaluationRunAtomic(
	ctx context.Context,
	run FxRevaluationRun,
	journalIn *gl.JournalEntryInput,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (FxRevaluationRun, bool, error) {
	if r == nil || r.client == nil {
		return FxRevaluationRun{}, false, ErrRepoUnavailable
	}
	var out FxRevaluationRun
	var alreadyPosted bool

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if run.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT ` + strings.Join(revalColumns(), ", ") + `
				      FROM FxRevaluationRuns@{FORCE_INDEX=UQ_FxRevaluationRuns_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tenantId AND IdempotencyKey = @idemKey AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{
					"tenantId": run.TenantID,
					"idemKey":  run.IdempotencyKey,
				},
			})
			row, rErr := iter.Next()
			iter.Stop()
			if rErr == nil {
				existing, sErr := scanRevalRow(row)
				if sErr != nil {
					return sErr
				}
				out = existing
				alreadyPosted = true
				return nil
			}
			if rErr != iterator.Done {
				return fmt.Errorf("check FxRevaluationRuns idempotency: %w", rErr)
			}
		}

		if run.EntityID != "" {
			if err := ensureEntityActiveInTxn(txCtx, txn, run.TenantID, run.EntityID); err != nil {
				return err
			}
		}

		outboxBuf := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &tenantStampedOutboxBuf{inner: outboxBuf, tenantID: run.TenantID}

		if journalIn != nil {
			jeRes, err := gl.BufferJournalEntryInTxn(txCtx, txn, stampedBuf, *journalIn)
			if err != nil {
				return err
			}
			run.JournalEntryID = jeRes.EntryID
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("FxRevaluationRuns", map[string]any{
				"TenantId":                run.TenantID,
				"RunId":                   run.RunID,
				"EntityId":                nullIfEmpty(run.EntityID),
				"PeriodId":                run.PeriodID,
				"BaseCurrency":            run.BaseCurrency,
				"UnrealizedGainLossMinor": run.UnrealizedGainLossMinor,
				"RoundingDeltaMinor":      run.RoundingDeltaMinor,
				"RatesJson":               nullIfEmpty(run.RatesJSON),
				"JournalEntryId":          nullIfEmpty(run.JournalEntryID),
				"IdempotencyKey":          nullIfEmpty(run.IdempotencyKey),
				"CreatedBy":               run.CreatedBy,
				"CreatedAt":               spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert FxRevaluationRuns: %w", err)
		}

		ts := time.Now().UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
		if journalIn != nil && !journalIn.OccurredAt.IsZero() {
			ts = journalIn.OccurredAt.UTC().Format(time.RFC3339Nano)
		}
		ev := events.FxRevaluationPostedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventFxRevaluationPosted,
				Version:   1,
				Timestamp: ts,
			},
			TenantID:                run.TenantID,
			RunID:                   run.RunID,
			EntityID:                run.EntityID,
			PeriodID:                run.PeriodID,
			BaseCurrency:            run.BaseCurrency,
			UnrealizedGainLossMinor: run.UnrealizedGainLossMinor,
			RoundingDeltaMinor:      run.RoundingDeltaMinor,
			JournalEntryID:          run.JournalEntryID,
			RatesJSON:               run.RatesJSON,
			CreatedBy:               run.CreatedBy,
			OccurredAt:              ts,
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateFxRevaluationRun, run.RunID, events.TopicLedger, ev); err != nil {
			return fmt.Errorf("emit fx.revaluation.posted: %w", err)
		}
		if err := outboxBuf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush fx revaluation outbox: %w", err)
		}

		if auditRec != nil {
			aIn := auditIn
			aIn.ObjectID = run.RunID
			if err := auditRec.RecordEventInTxn(txCtx, txn, aIn); err != nil {
				return fmt.Errorf("audit fx revaluation: %w", err)
			}
		}

		out = run
		out.CreatedAt = time.Now().UTC()
		alreadyPosted = false
		return nil
	})
	if err != nil {
		return FxRevaluationRun{}, false, err
	}
	if !alreadyPosted {
		ro := r.client.ReadOnlyTransaction()
		defer ro.Close()
		if row, rErr := ro.ReadRow(ctx, "FxRevaluationRuns", spanner.Key{out.TenantID, out.RunID}, revalColumns()); rErr == nil {
			if loaded, sErr := scanRevalRow(row); sErr == nil {
				return loaded, false, nil
			}
		}
	}
	return out, alreadyPosted, nil
}

func (r *SpannerRepository) ReadConsolidationSnapshot(ctx context.Context, tenantID, periodID string, entityIDs []string) (ConsolidationSnapshot, error) {
	if r == nil || r.client == nil {
		return ConsolidationSnapshot{}, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	periodID = strings.TrimSpace(periodID)

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	// 1. Load all LegalEntities for tenant.
	entIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT ` + strings.Join(entityColumns(), ", ") + ` FROM LegalEntities WHERE TenantId = @tenantId ORDER BY IsParent DESC, EntityId ASC`,
		Params: map[string]any{"tenantId": tenantID},
	})
	allEntities := make(map[string]LegalEntity)
	orderedAll := make([]LegalEntity, 0)
	for {
		row, err := entIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			entIter.Stop()
			return ConsolidationSnapshot{}, fmt.Errorf("read LegalEntities for consolidation: %w", err)
		}
		ent, err := scanEntityRow(row)
		if err != nil {
			entIter.Stop()
			return ConsolidationSnapshot{}, err
		}
		allEntities[ent.EntityID] = ent
		orderedAll = append(orderedAll, ent)
	}
	entIter.Stop()

	selected := make([]LegalEntity, 0, len(orderedAll))
	if len(entityIDs) > 0 {
		seen := make(map[string]struct{}, len(entityIDs))
		for _, id := range entityIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			ent, ok := allEntities[id]
			if !ok {
				return ConsolidationSnapshot{}, fmt.Errorf("%w: %s", ErrEntityNotFound, id)
			}
			selected = append(selected, ent)
		}
	} else {
		for _, ent := range orderedAll {
			if strings.ToUpper(strings.TrimSpace(ent.Status)) == EntityStatusActive {
				selected = append(selected, ent)
			}
		}
	}

	selectedIDs := make([]string, 0, len(selected))
	selectedMap := make(map[string]LegalEntity, len(selected))
	for _, ent := range selected {
		selectedIDs = append(selectedIDs, ent.EntityID)
		selectedMap[ent.EntityID] = ent
	}

	// 2. Read per-entity trial balances in the same ReadOnlyTransaction snapshot.
	tbs := make(map[string]gl.TrialBalanceReport, len(selected))
	for _, ent := range selected {
		tbs[ent.EntityID] = gl.TrialBalanceReport{
			EntityID:      ent.EntityID,
			TenantID:      tenantID,
			LegalEntityID: ent.EntityID,
			PeriodID:      periodID,
			Currency:      ent.BaseCurrency,
			Balanced:      true,
			Accounts:      []gl.TrialBalanceLine{},
		}
	}

	if len(selectedIDs) > 0 {
		tbIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT e.LegalEntityId, p.AccountCode, p.Currency, COALESCE(p.FxRateId, ''),
			             p.Direction, p.AmountMinor, p.FunctionalAmountMinor,
			             a.Class, a.Name
			      FROM JournalPostings p
			      JOIN JournalEntries e ON p.EntityId = e.EntityId AND p.EntryId = e.EntryId
			      LEFT JOIN GlAccounts a ON a.EntityId = p.EntityId AND a.AccountCode = p.AccountCode
			      WHERE e.EntityId = @tenantId
			        AND e.PeriodId = @periodId
			        AND e.LegalEntityId IN UNNEST(@entityIds)
			      ORDER BY e.LegalEntityId ASC, p.AccountCode ASC, p.EntryId ASC, p.Line ASC`,
			Params: map[string]any{
				"tenantId":  tenantID,
				"periodId":  periodID,
				"entityIds": selectedIDs,
			},
		})
		type tbKey struct {
			entityID string
			code     string
			curr     string
		}
		grouped := make(map[tbKey]*gl.TrialBalanceLine)
		orderByEntity := make(map[string][]tbKey)
		for {
			row, err := tbIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				tbIter.Stop()
				return ConsolidationSnapshot{}, fmt.Errorf("query consolidation postings: %w", err)
			}
			var leID, code, curr, fxRateID, dir string
			var amtMinor, funcAmtMinor int64
			var classNull, nameNull spanner.NullString
			if err := row.Columns(&leID, &code, &curr, &fxRateID, &dir, &amtMinor, &funcAmtMinor, &classNull, &nameNull); err != nil {
				tbIter.Stop()
				return ConsolidationSnapshot{}, fmt.Errorf("scan consolidation posting: %w", err)
			}
			ent := selectedMap[leID]
			uc := strings.ToUpper(strings.TrimSpace(code))
			effCurr := strings.ToUpper(strings.TrimSpace(curr))
			useAmt := amtMinor
			if strings.TrimSpace(fxRateID) != "" || (ent.BaseCurrency != "" && effCurr != ent.BaseCurrency && funcAmtMinor != 0 && funcAmtMinor != amtMinor) {
				useAmt = funcAmtMinor
				if ent.BaseCurrency != "" {
					effCurr = ent.BaseCurrency
				}
			} else if funcAmtMinor != 0 && funcAmtMinor == amtMinor {
				useAmt = funcAmtMinor
			}

			k := tbKey{entityID: leID, code: uc, curr: effCurr}
			line, ok := grouped[k]
			if !ok {
				cls := strings.ToUpper(strings.TrimSpace(nullStr(classNull)))
				nm := nullStr(nameNull)
				if cls == "" {
					if def, found := gl.LookupSystemAccount(uc); found {
						cls = def.Class
						if nm == "" {
							nm = def.Name
						}
					}
				}
				line = &gl.TrialBalanceLine{
					AccountCode: uc,
					AccountName: nm,
					Class:       cls,
					Currency:    effCurr,
				}
				grouped[k] = line
				orderByEntity[leID] = append(orderByEntity[leID], k)
			}
			rep := tbs[leID]
			if strings.EqualFold(dir, string(gl.DirectionDebit)) {
				line.DebitMinor += useAmt
				rep.TotalDebitsMinor += useAmt
			} else {
				line.CreditMinor += useAmt
				rep.TotalCreditsMinor += useAmt
			}
			tbs[leID] = rep
		}
		tbIter.Stop()

		for eid, keys := range orderByEntity {
			rep := tbs[eid]
			lines := make([]gl.TrialBalanceLine, 0, len(keys))
			for _, k := range keys {
				l := grouped[k]
				if gl.IsDebitNormalClass(l.Class) {
					l.NetMinor = l.DebitMinor - l.CreditMinor
				} else {
					l.NetMinor = l.CreditMinor - l.DebitMinor
				}
				lines = append(lines, *l)
			}
			sort.Slice(lines, func(i, j int) bool {
				if lines[i].AccountCode != lines[j].AccountCode {
					return lines[i].AccountCode < lines[j].AccountCode
				}
				return lines[i].Currency < lines[j].Currency
			})
			rep.Accounts = lines
			rep.Balanced = rep.TotalDebitsMinor == rep.TotalCreditsMinor
			tbs[eid] = rep
		}
	}

	// 3. Read posted IntercompanyTransfers for (tenantID, periodID).
	trIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ` + strings.Join(transferColumns(), ", ") + `
		      FROM IntercompanyTransfers
		      WHERE TenantId = @tenantId AND PeriodId = @periodId AND UPPER(Status) = 'POSTED'
		      ORDER BY CreatedAt ASC, TransferId ASC`,
		Params: map[string]any{
			"tenantId": tenantID,
			"periodId": periodID,
		},
	})
	transfers := make([]IntercompanyTransfer, 0)
	for {
		row, err := trIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			trIter.Stop()
			return ConsolidationSnapshot{}, fmt.Errorf("query consolidation IntercompanyTransfers: %w", err)
		}
		tr, err := scanTransferRow(row)
		if err != nil {
			trIter.Stop()
			return ConsolidationSnapshot{}, err
		}
		transfers = append(transfers, tr)
	}
	trIter.Stop()

	return ConsolidationSnapshot{
		Entities:      selected,
		TrialBalances: tbs,
		Transfers:     transfers,
	}, nil
}

func isRevalMonetaryAccount(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case gl.AccountAccountsReceivable,
		gl.AccountVendorPayable,
		gl.AccountSupplierPayable,
		gl.AccountGRNI,
		gl.AccountCashOffice,
		gl.AccountCashInTransit,
		gl.AccountBankOperating,
		gl.AccountCardPSPTransit:
		return true
	default:
		return false
	}
}

func accountClassFor(code string) string {
	if def, ok := gl.LookupSystemAccount(code); ok {
		return def.Class
	}
	return gl.ClassAsset
}

func signedAccountAmount(code string, dir gl.Direction, amt int64) int64 {
	if gl.IsDebitNormalClass(accountClassFor(code)) {
		if dir == gl.DirectionDebit {
			return amt
		}
		return -amt
	}
	if dir == gl.DirectionCredit {
		return amt
	}
	return -amt
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func nullStr(ns spanner.NullString) string {
	if ns.Valid {
		return ns.StringVal
	}
	return ""
}
