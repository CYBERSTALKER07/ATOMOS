package featureflags

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository persists FeatureFlagOverrides.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) Get(ctx context.Context, flagKey, tenantType, tenantID string) (Override, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "FeatureFlagOverrides",
		spanner.Key{flagKey, tenantType, tenantID},
		[]string{"FlagKey", "TenantType", "TenantId", "Enabled", "UpdatedBy", "UpdatedAt", "Reason", "Status", "ApprovedBy", "ApprovedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Override{}, false, nil
		}
		return Override{}, false, err
	}
	o, err := scanOverride(row)
	return o, err == nil, err
}

func (r *SpannerRepository) Upsert(ctx context.Context, o Override) error {
	status := o.Status
	if status == "" {
		status = StatusActive
	}
	m := map[string]any{
		"FlagKey":    o.FlagKey,
		"TenantType": o.TenantType,
		"TenantId":   o.TenantID,
		"Enabled":    o.Enabled,
		"UpdatedBy":  o.UpdatedBy,
		"UpdatedAt":  spanner.CommitTimestamp,
		"Reason":     o.Reason,
		"Status":     status,
	}
	if o.ApprovedBy != "" {
		m["ApprovedBy"] = spanner.NullString{StringVal: o.ApprovedBy, Valid: true}
	}
	if !o.ApprovedAt.IsZero() {
		m["ApprovedAt"] = spanner.NullTime{Time: o.ApprovedAt, Valid: true}
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("FeatureFlagOverrides", m)})
	return err
}

func (r *SpannerRepository) ListForTenant(ctx context.Context, tenantType, tenantID string) ([]Override, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FlagKey, TenantType, TenantId, Enabled, UpdatedBy, UpdatedAt, Reason, Status, ApprovedBy, ApprovedAt
			FROM FeatureFlagOverrides WHERE TenantType=@tt AND TenantId=@tid`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID},
	})
	defer iter.Stop()
	out := make([]Override, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		o, err := scanOverride(row)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func scanOverride(row *spanner.Row) (Override, error) {
	var o Override
	var reason, status, approvedBy spanner.NullString
	var approvedAt spanner.NullTime
	var updated time.Time
	if err := row.Columns(&o.FlagKey, &o.TenantType, &o.TenantID, &o.Enabled, &o.UpdatedBy, &updated, &reason, &status, &approvedBy, &approvedAt); err != nil {
		return Override{}, err
	}
	o.UpdatedAt = updated.UTC()
	o.Reason = reason.StringVal
	o.Status = status.StringVal
	if o.Status == "" {
		o.Status = StatusActive
	}
	o.ApprovedBy = approvedBy.StringVal
	if approvedAt.Valid {
		o.ApprovedAt = approvedAt.Time.UTC()
	}
	return o, nil
}
