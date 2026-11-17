package periods

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository stores M1.1 accounting periods in Cloud Spanner and emits outbox/audit events.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed period repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Client returns the underlying *spanner.Client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

type entityStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) CreatePeriod(ctx context.Context, p Period, auditRec AuditRecorder, auditIn AuditRecordInput) (Period, error) {
	if r == nil || r.client == nil {
		return Period{}, ErrRepoUnavailable
	}
	var out Period
	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, found, err := readPeriodRowInTxn(txCtx, txn, p.EntityID, p.PeriodID)
		if err != nil {
			return err
		}
		if found {
			return fmt.Errorf("%w: %s for %s", ErrPeriodAlreadyExists, p.PeriodID, p.EntityID)
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("Periods", map[string]any{
				"EntityId":      p.EntityID,
				"PeriodId":      p.PeriodID,
				"Status":        p.Status,
				"Timezone":      p.Timezone,
				"SoftClosedBy":  nil,
				"SoftClosedAt":  nil,
				"ClosedBy":      nil,
				"ClosedAt":      nil,
				"ReopenedBy":    nil,
				"ReopenedAt":    nil,
				"ReopenReason":  nil,
				"SnapshotUri":   nullIfEmpty(p.SnapshotURI),
				"SnapshotHash":  nullIfEmpty(p.SnapshotHash),
				"ChecklistJson": nullIfEmpty(p.ChecklistJSON),
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert Periods: %w", err)
		}
		if auditRec != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn); err != nil {
				return fmt.Errorf("audit period.open: %w", err)
			}
		}
		out = p
		now := time.Now().UTC()
		out.CreatedAt = now
		out.UpdatedAt = now
		EnrichPeriodWindow(&out)
		return nil
	})
	if err != nil {
		return Period{}, err
	}
	if loaded, ok, err := r.GetPeriod(ctx, p.EntityID, p.PeriodID); err == nil && ok {
		return loaded, nil
	}
	return out, nil
}

func (r *SpannerRepository) GetPeriod(ctx context.Context, entityID, periodID string) (Period, bool, error) {
	if r == nil || r.client == nil {
		return Period{}, false, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	row, err := ro.ReadRow(ctx, "Periods", spanner.Key{strings.TrimSpace(entityID), strings.TrimSpace(periodID)}, periodColumns())
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return Period{}, false, nil
		}
		return Period{}, false, fmt.Errorf("read Periods(%s, %s): %w", entityID, periodID, err)
	}
	p, err := scanPeriodRow(row)
	if err != nil {
		return Period{}, false, err
	}
	EnrichPeriodWindow(&p)
	return p, true, nil
}

func (r *SpannerRepository) ListPeriods(ctx context.Context, entityID, statusFilter string) ([]Period, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	statusFilter = strings.ToUpper(strings.TrimSpace(statusFilter))

	sql := `SELECT ` + strings.Join(periodColumns(), ", ") + `
	        FROM Periods
	        WHERE EntityId = @entityId`
	params := map[string]any{"entityId": entityID}
	if statusFilter != "" {
		sql += ` AND UPPER(Status) = @status`
		params["status"] = statusFilter
	}
	sql += ` ORDER BY PeriodId DESC`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]Period, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list Periods: %w", err)
		}
		p, err := scanPeriodRow(row)
		if err != nil {
			return nil, err
		}
		EnrichPeriodWindow(&p)
		out = append(out, p)
	}
	return out, nil
}

func (r *SpannerRepository) TransitionPeriod(
	ctx context.Context,
	entityID, periodID string,
	mutateFn func(existing *Period, found bool) (PeriodTransition, error),
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (Period, error) {
	if r == nil || r.client == nil {
		return Period{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	periodID = strings.TrimSpace(periodID)

	var out Period
	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur, found, err := readPeriodRowInTxn(txCtx, txn, entityID, periodID)
		if err != nil {
			return err
		}
		var curPtr *Period
		if found {
			cp := cur
			curPtr = &cp
		}

		tr, err := mutateFn(curPtr, found)
		if err != nil {
			return err
		}
		next := tr.NextPeriod
		next.EntityID = entityID
		next.PeriodID = periodID

		if !found {
			insertCols := map[string]any{
				"EntityId":      entityID,
				"PeriodId":      periodID,
				"Status":        next.Status,
				"Timezone":      next.Timezone,
				"SoftClosedBy":  nullIfEmpty(next.SoftClosedBy),
				"SoftClosedAt":  commitTSIfSet(next.SoftClosedAt),
				"ClosedBy":      nullIfEmpty(next.ClosedBy),
				"ClosedAt":      commitTSIfSet(next.ClosedAt),
				"ReopenedBy":    nullIfEmpty(next.ReopenedBy),
				"ReopenedAt":    commitTSIfSet(next.ReopenedAt),
				"ReopenReason":  nullIfEmpty(next.ReopenReason),
				"SnapshotUri":   nullIfEmpty(next.SnapshotURI),
				"SnapshotHash":  nullIfEmpty(next.SnapshotHash),
				"ChecklistJson": nullIfEmpty(next.ChecklistJSON),
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("Periods", insertCols)}); err != nil {
				return fmt.Errorf("insert transitioned Period: %w", err)
			}
		} else {
			updateCols := map[string]any{
				"EntityId":      entityID,
				"PeriodId":      periodID,
				"Status":        next.Status,
				"Timezone":      next.Timezone,
				"SoftClosedBy":  nullIfEmpty(next.SoftClosedBy),
				"ClosedBy":      nullIfEmpty(next.ClosedBy),
				"ReopenedBy":    nullIfEmpty(next.ReopenedBy),
				"ReopenReason":  nullIfEmpty(next.ReopenReason),
				"SnapshotUri":   nullIfEmpty(next.SnapshotURI),
				"SnapshotHash":  nullIfEmpty(next.SnapshotHash),
				"ChecklistJson": nullIfEmpty(next.ChecklistJSON),
				"UpdatedAt":     spanner.CommitTimestamp,
			}
			// Only mutate allow_commit_timestamp=true nullable columns when transitioning that stage.
			if next.Status == StatusSoftClosed {
				updateCols["SoftClosedAt"] = spanner.CommitTimestamp
			}
			if next.Status == StatusClosed {
				updateCols["ClosedAt"] = spanner.CommitTimestamp
			}
			if next.Status == StatusReopened {
				updateCols["ReopenedAt"] = spanner.CommitTimestamp
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("Periods", updateCols)}); err != nil {
				return fmt.Errorf("update Period(%s, %s): %w", entityID, periodID, err)
			}
		}

		if tr.EventType != "" {
			now := time.Now().UTC()
			outboxInner := outbox.NewSpannerTxnBuffer(txn)
			stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: entityID}
			actorID := next.ClosedBy
			if next.Status == StatusSoftClosed {
				actorID = next.SoftClosedBy
			} else if next.Status == StatusReopened {
				actorID = next.ReopenedBy
			}
			ev := events.PeriodEvent{
				BaseEvent: events.BaseEvent{
					Type:      tr.EventType,
					Version:   1,
					Timestamp: now.Format(time.RFC3339Nano),
				},
				EntityID:     entityID,
				PeriodID:     periodID,
				Status:       next.Status,
				Timezone:     next.Timezone,
				ActorID:      actorID,
				Reason:       tr.EventReason,
				SnapshotURI:  next.SnapshotURI,
				SnapshotHash: next.SnapshotHash,
				ApprovalID:   tr.ApprovalID,
				OccurredAt:   now.Format(time.RFC3339Nano),
			}
			aggID := entityID + ":" + periodID
			if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregatePeriod, aggID, events.TopicLedger, ev); err != nil {
				return fmt.Errorf("emit period outbox event: %w", err)
			}
			if err := outboxInner.Flush(txCtx); err != nil {
				return fmt.Errorf("flush period outbox event: %w", err)
			}
		}

		if auditRec != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn); err != nil {
				return fmt.Errorf("audit period transition: %w", err)
			}
		}

		out = next
		return nil
	})
	if err != nil {
		return Period{}, err
	}
	if loaded, ok, err := r.GetPeriod(ctx, entityID, periodID); err == nil && ok {
		return loaded, nil
	}
	EnrichPeriodWindow(&out)
	return out, nil
}

func (r *SpannerRepository) EvaluateChecklist(ctx context.Context, entityID, periodID, tzName string) (ChecklistReport, error) {
	if r == nil || r.client == nil {
		return ChecklistReport{}, ErrRepoUnavailable
	}
	return evaluateChecklistSpanner(ctx, r.client, entityID, periodID, tzName)
}

func (r *SpannerRepository) BuildCloseSnapshotData(ctx context.Context, entityID, periodID, tzName, closedBy string) (CloseSnapshot, error) {
	if r == nil || r.client == nil {
		return CloseSnapshot{}, ErrRepoUnavailable
	}
	return buildCloseSnapshotSpanner(ctx, r.client, entityID, periodID, tzName, closedBy)
}

func periodColumns() []string {
	return []string{
		"EntityId", "PeriodId", "Status", "Timezone",
		"SoftClosedBy", "SoftClosedAt",
		"ClosedBy", "ClosedAt",
		"ReopenedBy", "ReopenedAt", "ReopenReason",
		"SnapshotUri", "SnapshotHash", "ChecklistJson",
		"CreatedAt", "UpdatedAt",
	}
}

func readPeriodRowInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID, periodID string) (Period, bool, error) {
	row, err := txn.ReadRow(ctx, "Periods", spanner.Key{entityID, periodID}, periodColumns())
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return Period{}, false, nil
		}
		return Period{}, false, fmt.Errorf("read Periods(%s, %s): %w", entityID, periodID, err)
	}
	p, err := scanPeriodRow(row)
	if err != nil {
		return Period{}, false, err
	}
	EnrichPeriodWindow(&p)
	return p, true, nil
}

func scanPeriodRow(row *spanner.Row) (Period, error) {
	var (
		p                                                             Period
		softBy, closedBy, reopenedBy, reopenReason, snapURI, snapHash spanner.NullString
		checklistJSON                                                 spanner.NullString
		softAt, closedAt, reopenedAt                                  spanner.NullTime
	)
	if err := row.Columns(
		&p.EntityID, &p.PeriodID, &p.Status, &p.Timezone,
		&softBy, &softAt,
		&closedBy, &closedAt,
		&reopenedBy, &reopenedAt, &reopenReason,
		&snapURI, &snapHash, &checklistJSON,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Period{}, fmt.Errorf("scan Period row: %w", err)
	}
	p.Status = strings.ToUpper(strings.TrimSpace(p.Status))
	p.SoftClosedBy = nullStr(softBy)
	p.SoftClosedAt = nullTimePtr(softAt)
	p.ClosedBy = nullStr(closedBy)
	p.ClosedAt = nullTimePtr(closedAt)
	p.ReopenedBy = nullStr(reopenedBy)
	p.ReopenedAt = nullTimePtr(reopenedAt)
	p.ReopenReason = nullStr(reopenReason)
	p.SnapshotURI = nullStr(snapURI)
	p.SnapshotHash = nullStr(snapHash)
	p.ChecklistJSON = nullStr(checklistJSON)
	return p, nil
}

func commitTSIfSet(t *time.Time) any {
	if t != nil {
		return spanner.CommitTimestamp
	}
	return nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func nullStr(ns spanner.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.StringVal
}

func nullTimePtr(nt spanner.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time.UTC()
	return &t
}
