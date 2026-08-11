package platformadmin

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository persists PlatformTenants + PlatformAdminAudit.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) UpsertTenant(ctx context.Context, t Tenant) error {
	row := map[string]any{
		"TenantType":  t.TenantType,
		"TenantId":    t.TenantID,
		"Status":      t.Status,
		"DisplayName": t.DisplayName,
		"KybNotes":    t.KybNotes,
		"CreatedAt":   t.CreatedAt,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	if t.ApprovedAt != nil {
		row["ApprovedAt"] = *t.ApprovedAt
	}
	if t.SuspendedAt != nil {
		row["SuspendedAt"] = *t.SuspendedAt
	}
	if t.OffboardedAt != nil {
		row["OffboardedAt"] = *t.OffboardedAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("PlatformTenants", row)})
	return err
}

func (r *SpannerRepository) GetTenant(ctx context.Context, tenantType, tenantID string) (Tenant, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PlatformTenants", spanner.Key{tenantType, tenantID},
		[]string{"TenantType", "TenantId", "Status", "DisplayName", "KybNotes", "CreatedAt", "UpdatedAt",
			"ApprovedAt", "SuspendedAt", "OffboardedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Tenant{}, false, nil
		}
		return Tenant{}, false, err
	}
	t, err := scanTenant(row)
	return t, err == nil, err
}

func (r *SpannerRepository) ListTenants(ctx context.Context, status string, limit int) ([]Tenant, error) {
	if limit <= 0 {
		limit = 100
	}
	sql := `SELECT TenantType, TenantId, Status, DisplayName, KybNotes, CreatedAt, UpdatedAt,
		ApprovedAt, SuspendedAt, OffboardedAt
		FROM PlatformTenants`
	params := map[string]any{"lim": int64(limit)}
	if status != "" {
		sql += ` WHERE Status=@st`
		params["st"] = status
	}
	sql += ` ORDER BY UpdatedAt DESC LIMIT @lim`
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Tenant, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := scanTenant(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *SpannerRepository) InsertAudit(ctx context.Context, row AuditRow) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("PlatformAdminAudit", map[string]any{
		"AuditId":      row.AuditID,
		"ActorSubject": row.ActorSubject,
		"Action":       row.Action,
		"TenantType":   row.TenantType,
		"TenantId":     row.TenantID,
		"DetailJson":   row.DetailJSON,
		"CreatedAt":    spanner.CommitTimestamp,
	})})
	return err
}

func (r *SpannerRepository) ListAudit(ctx context.Context, limit int) ([]AuditRow, error) {
	if limit <= 0 {
		limit = 100
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT AuditId, ActorSubject, Action, TenantType, TenantId, DetailJson, CreatedAt
			FROM PlatformAdminAudit ORDER BY CreatedAt DESC LIMIT @lim`,
		Params: map[string]any{"lim": int64(limit)},
	})
	defer iter.Stop()
	out := make([]AuditRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var a AuditRow
		var tt, tid, detail spanner.NullString
		var created time.Time
		if err := row.Columns(&a.AuditID, &a.ActorSubject, &a.Action, &tt, &tid, &detail, &created); err != nil {
			return nil, err
		}
		a.TenantType = tt.StringVal
		a.TenantID = tid.StringVal
		a.DetailJSON = detail.StringVal
		a.CreatedAt = created.UTC()
		out = append(out, a)
	}
	return out, nil
}

func scanTenant(row *spanner.Row) (Tenant, error) {
	var t Tenant
	var name, notes spanner.NullString
	var approved, suspended, offboarded spanner.NullTime
	var created, updated time.Time
	if err := row.Columns(&t.TenantType, &t.TenantID, &t.Status, &name, &notes, &created, &updated,
		&approved, &suspended, &offboarded); err != nil {
		return Tenant{}, err
	}
	t.DisplayName = name.StringVal
	t.KybNotes = notes.StringVal
	t.CreatedAt = created.UTC()
	t.UpdatedAt = updated.UTC()
	if approved.Valid {
		v := approved.Time.UTC()
		t.ApprovedAt = &v
	}
	if suspended.Valid {
		v := suspended.Time.UTC()
		t.SuspendedAt = &v
	}
	if offboarded.Valid {
		v := offboarded.Time.UTC()
		t.OffboardedAt = &v
	}
	return t, nil
}
