package exportpacks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

type tenantOutboxBuffer struct {
	inner    *outbox.SpannerTxnBuffer
	tenantID string
}

func (b *tenantOutboxBuffer) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" && (e.SupplierID == "" || e.SupplierID == outbox.PlatformSupplierID) {
		e.SupplierID = b.tenantID
	}
	return b.inner.BufferOutbox(ctx, e)
}

// SpannerRepository persists ExportPacks rows with atomic OutboxEvents and AuditEvents writes.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed ExportPacks repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) CreatePack(
	ctx context.Context,
	pack *ExportPack,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	if pack == nil || strings.TrimSpace(pack.TenantID) == "" || strings.TrimSpace(pack.PackID) == "" {
		return fmt.Errorf("%w: tenant_id and pack_id required", ErrInvalidInput)
	}

	createdTS := pack.CreatedAt.UTC()
	if createdTS.IsZero() {
		createdTS = time.Now().UTC()
	}
	mut := spanner.InsertMap("ExportPacks", map[string]any{
		"TenantId":      pack.TenantID,
		"PackId":        pack.PackID,
		"PackType":      pack.PackType,
		"EntityId":      nullStr(pack.EntityID),
		"PeriodId":      nullStr(pack.PeriodID),
		"FromDate":      pack.FromDate.UTC(),
		"ToDate":        pack.ToDate.UTC(),
		"Format":        pack.Format,
		"Status":        pack.Status,
		"ManifestJson":  nullStr(pack.ManifestJSON),
		"ArtifactsJson": nullStr(pack.ArtifactsJSON),
		"DownloadToken": nullStr(pack.DownloadToken),
		"WatermarkText": nullStr(pack.WatermarkText),
		"RowCount":      pack.RowCount,
		"Sha256Hex":     nullStr(pack.Sha256Hex),
		"CreatedBy":     pack.CreatedBy,
		"ExpiresAt":     pack.ExpiresAt.UTC(),
		"CreatedAt":     createdTS,
	})

	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return fmt.Errorf("insert ExportPacks: %w", err)
		}

		buf := &tenantOutboxBuffer{inner: outbox.NewSpannerTxnBuffer(txn), tenantID: pack.TenantID}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateExportPack, pack.PackID, events.TopicLedger, events.ExportPackCreatedEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventExportPackCreated, Timestamp: createdTS.Format(time.RFC3339Nano)},
			TenantID:   pack.TenantID,
			PackID:     pack.PackID,
			PackType:   pack.PackType,
			EntityID:   pack.EntityID,
			PeriodID:   pack.PeriodID,
			FromDate:   pack.FromDate.UTC().Format(time.RFC3339),
			ToDate:     pack.ToDate.UTC().Format(time.RFC3339),
			Format:     pack.Format,
			Status:     pack.Status,
			RowCount:   pack.RowCount,
			Sha256Hex:  pack.Sha256Hex,
			CreatedBy:  pack.CreatedBy,
			ExpiresAt:  pack.ExpiresAt.UTC().Format(time.RFC3339),
			CreatedAt:  createdTS.Format(time.RFC3339Nano),
			OccurredAt: createdTS.Format(time.RFC3339Nano),
		}); err != nil {
			return fmt.Errorf("emit export.pack.created: %w", err)
		}
		if err := buf.inner.Flush(txCtx); err != nil {
			return fmt.Errorf("flush export.pack.created outbox: %w", err)
		}

		if auditFn != nil {
			if err := auditFn(txCtx, txn); err != nil {
				return fmt.Errorf("audit export.pack.created: %w", err)
			}
		}
		return nil
	})
}

const exportPackSelectCols = `
	TenantId, PackId, PackType, COALESCE(EntityId, ''), COALESCE(PeriodId, ''),
	FromDate, ToDate, Format, Status, COALESCE(ManifestJson, ''), COALESCE(ArtifactsJson, ''),
	COALESCE(DownloadToken, ''), COALESCE(WatermarkText, ''), RowCount, COALESCE(Sha256Hex, ''),
	CreatedBy, ExpiresAt, CreatedAt`

func (r *SpannerRepository) GetPack(ctx context.Context, tenantID, packID string) (*ExportPack, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ` + exportPackSelectCols + ` FROM ExportPacks WHERE TenantId = @tid AND PackId = @pid`,
		Params: map[string]any{
			"tid": strings.TrimSpace(tenantID),
			"pid": strings.TrimSpace(packID),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, packID)
	}
	if err != nil {
		return nil, fmt.Errorf("get ExportPack: %w", err)
	}
	return scanExportPack(row)
}

func (r *SpannerRepository) GetPackByID(ctx context.Context, packID string) (*ExportPack, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT ` + exportPackSelectCols + ` FROM ExportPacks WHERE PackId = @pid LIMIT 1`,
		Params: map[string]any{
			"pid": strings.TrimSpace(packID),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, packID)
	}
	if err != nil {
		return nil, fmt.Errorf("get ExportPack by id: %w", err)
	}
	return scanExportPack(row)
}

func (r *SpannerRepository) ListPacks(ctx context.Context, tenantID, entityID string, limit int) ([]*ExportPack, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	tid := strings.TrimSpace(tenantID)
	eid := strings.TrimSpace(entityID)

	sql := `SELECT ` + exportPackSelectCols + ` FROM ExportPacks`
	params := map[string]any{"lim": int64(limit)}
	var conds []string
	if tid != "" {
		conds = append(conds, `TenantId = @tid`)
		params["tid"] = tid
	}
	if eid != "" {
		conds = append(conds, `EntityId = @eid`)
		params["eid"] = eid
	}
	if len(conds) > 0 {
		sql += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	sql += ` ORDER BY CreatedAt DESC, PackId DESC LIMIT @lim`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []*ExportPack
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list ExportPacks: %w", err)
		}
		p, err := scanExportPack(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []*ExportPack{}
	}
	return out, nil
}

func scanExportPack(row *spanner.Row) (*ExportPack, error) {
	var p ExportPack
	if err := row.Columns(
		&p.TenantID, &p.PackID, &p.PackType, &p.EntityID, &p.PeriodID,
		&p.FromDate, &p.ToDate, &p.Format, &p.Status, &p.ManifestJSON, &p.ArtifactsJSON,
		&p.DownloadToken, &p.WatermarkText, &p.RowCount, &p.Sha256Hex,
		&p.CreatedBy, &p.ExpiresAt, &p.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan ExportPack: %w", err)
	}
	p.FromDate = p.FromDate.UTC()
	p.ToDate = p.ToDate.UTC()
	p.ExpiresAt = p.ExpiresAt.UTC()
	p.CreatedAt = p.CreatedAt.UTC()
	if strings.TrimSpace(p.ManifestJSON) != "" {
		var m PackManifest
		if err := json.Unmarshal([]byte(p.ManifestJSON), &m); err == nil {
			p.Manifest = &m
		}
	}
	if strings.TrimSpace(p.ArtifactsJSON) != "" {
		var arts []ArtifactMeta
		if err := json.Unmarshal([]byte(p.ArtifactsJSON), &arts); err == nil {
			p.Artifacts = arts
		}
	}
	return &p, nil
}

func nullStr(s string) spanner.NullString {
	v := strings.TrimSpace(s)
	if v == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}
