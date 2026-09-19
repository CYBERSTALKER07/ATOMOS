package partner

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// SpannerCoaRepository persists PartnerCoaMaps.
type SpannerCoaRepository struct {
	client *spanner.Client
}

func NewSpannerCoaRepository(client *spanner.Client) *SpannerCoaRepository {
	return &SpannerCoaRepository{client: client}
}

func (r *SpannerCoaRepository) Get(ctx context.Context, tenantType, tenantID string) (CoaMap, bool, error) {
	if r == nil || r.client == nil {
		return CoaMap{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "PartnerCoaMaps", spanner.Key{tenantType, tenantID},
		[]string{"TenantType", "TenantId", "AccountAR", "AccountRevenue", "AccountBankCash", "UpdatedAt", "UpdatedBy"})
	if err != nil {
		if isSpannerNotFound(err) {
			return CoaMap{}, false, nil
		}
		// Table missing in older envs → treat as defaults.
		if strings.Contains(err.Error(), "PartnerCoaMaps") {
			return CoaMap{}, false, nil
		}
		return CoaMap{}, false, err
	}
	var m CoaMap
	var updatedBy spanner.NullString
	var updatedAt time.Time
	if err := row.Columns(&m.TenantType, &m.TenantID, &m.AccountAR, &m.AccountRevenue, &m.AccountBankCash,
		&updatedAt, &updatedBy); err != nil {
		return CoaMap{}, false, err
	}
	m.UpdatedAt = updatedAt.UTC()
	m.UpdatedBy = updatedBy.StringVal
	return m, true, nil
}

func (r *SpannerCoaRepository) Upsert(ctx context.Context, m CoaMap) error {
	NormalizeCoa(&m)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("PartnerCoaMaps", map[string]any{
			"TenantType":       m.TenantType,
			"TenantId":         m.TenantID,
			"AccountAR":        m.AccountAR,
			"AccountRevenue":   m.AccountRevenue,
			"AccountBankCash":  m.AccountBankCash,
			"UpdatedAt":        spanner.CommitTimestamp,
			"UpdatedBy":        nullableStr(m.UpdatedBy),
		}),
	})
	return err
}
