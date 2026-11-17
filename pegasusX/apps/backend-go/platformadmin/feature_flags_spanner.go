package platformadmin

import (
	"context"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (r *SpannerRepository) GetFeatureFlag(ctx context.Context, tenantType, tenantID, flagKey string) (FeatureFlag, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FlagKey, TenantType, TenantId, Enabled, Status, Reason, UpdatedBy, UpdatedAt, ApprovedBy 
		      FROM FeatureFlagOverrides 
		      WHERE TenantType = @type AND TenantId = @id AND FlagKey = @flagKey`,
		Params: map[string]interface{}{
			"type":    tenantType,
			"id":      tenantID,
			"flagKey": flagKey,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return FeatureFlag{}, false, nil
	}
	if err != nil {
		return FeatureFlag{}, false, err
	}
	var f FeatureFlag
	var approvedBy spanner.NullString
	var reason spanner.NullString
	if err := row.Columns(&f.FlagKey, &f.TenantType, &f.TenantID, &f.Enabled, &f.Status, &reason, &f.UpdatedBy, &f.UpdatedAt, &approvedBy); err != nil {
		return FeatureFlag{}, false, err
	}
	if approvedBy.Valid {
		f.ApprovedBy = approvedBy.StringVal
	}
	if reason.Valid {
		f.Reason = reason.StringVal
	}
	return f, true, nil
}

func (r *SpannerRepository) ListFeatureFlags(ctx context.Context, tenantType, tenantID string) ([]FeatureFlag, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FlagKey, TenantType, TenantId, Enabled, Status, Reason, UpdatedBy, UpdatedAt, ApprovedBy 
		      FROM FeatureFlagOverrides 
		      WHERE TenantType = @type AND TenantId = @id`,
		Params: map[string]interface{}{
			"type": tenantType,
			"id":   tenantID,
		},
	}
	var flags []FeatureFlag
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var f FeatureFlag
		var approvedBy spanner.NullString
		var reason spanner.NullString
		if err := row.Columns(&f.FlagKey, &f.TenantType, &f.TenantID, &f.Enabled, &f.Status, &reason, &f.UpdatedBy, &f.UpdatedAt, &approvedBy); err != nil {
			return nil, err
		}
		if approvedBy.Valid {
			f.ApprovedBy = approvedBy.StringVal
		}
		if reason.Valid {
			f.Reason = reason.StringVal
		}
		flags = append(flags, f)
	}
	return flags, nil
}

func (r *SpannerRepository) SetFeatureFlag(ctx context.Context, f FeatureFlag) error {
	m := spanner.InsertOrUpdateMap("FeatureFlagOverrides", map[string]interface{}{
		"FlagKey":    f.FlagKey,
		"TenantType": f.TenantType,
		"TenantId":   f.TenantID,
		"Enabled":    f.Enabled,
		"Status":     f.Status,
		"Reason":     spanner.NullString{StringVal: f.Reason, Valid: f.Reason != ""},
		"UpdatedBy":  f.UpdatedBy,
		"UpdatedAt":  f.UpdatedAt,
	})
	if f.ApprovedBy != "" {
		m = spanner.InsertOrUpdateMap("FeatureFlagOverrides", map[string]interface{}{
			"FlagKey":    f.FlagKey,
			"TenantType": f.TenantType,
			"TenantId":   f.TenantID,
			"Enabled":    f.Enabled,
			"Status":     f.Status,
			"Reason":     spanner.NullString{StringVal: f.Reason, Valid: f.Reason != ""},
			"UpdatedBy":  f.UpdatedBy,
			"UpdatedAt":  f.UpdatedAt,
			"ApprovedBy": f.ApprovedBy,
			"ApprovedAt": f.UpdatedAt,
		})
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}
