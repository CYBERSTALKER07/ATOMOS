package partner

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerAs2ConfigRepository persists PartnerAs2Configs.
type SpannerAs2ConfigRepository struct {
	client *spanner.Client
}

func NewSpannerAs2ConfigRepository(client *spanner.Client) *SpannerAs2ConfigRepository {
	return &SpannerAs2ConfigRepository{client: client}
}

func (r *SpannerAs2ConfigRepository) Upsert(ctx context.Context, c As2Config) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("PartnerAs2Configs", map[string]any{
			"TenantType":           c.TenantType,
			"TenantId":             c.TenantID,
			"As2Enabled":           c.As2Enabled,
			"OurAs2Id":             c.OurAs2Id,
			"PartnerAs2Id":         c.PartnerAs2Id,
			"PartnerUrl":           nullableStr(c.PartnerURL),
			"OurCertSecretRef":     nullableStr(c.OurCertSecretRef),
			"OurKeySecretRef":      nullableStr(c.OurKeySecretRef),
			"PartnerCertSecretRef": nullableStr(c.PartnerCertSecretRef),
			"SignRequired":         c.SignRequired,
			"EncryptRequired":      c.EncryptRequired,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
	})
	return err
}

func (r *SpannerAs2ConfigRepository) Get(ctx context.Context, tenantType, tenantID string) (As2Config, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PartnerAs2Configs", spanner.Key{tenantType, tenantID},
		[]string{
			"TenantType", "TenantId", "As2Enabled", "OurAs2Id", "PartnerAs2Id", "PartnerUrl",
			"OurCertSecretRef", "OurKeySecretRef", "PartnerCertSecretRef",
			"SignRequired", "EncryptRequired", "UpdatedAt",
		})
	if err != nil {
		if isSpannerNotFound(err) {
			return As2Config{}, false, nil
		}
		return As2Config{}, false, err
	}
	c, err := scanAs2Config(row)
	if err != nil {
		return As2Config{}, false, err
	}
	return c, true, nil
}

func (r *SpannerAs2ConfigRepository) GetByOurAs2Id(ctx context.Context, ourAs2Id string) (As2Config, bool, error) {
	ourAs2Id = strings.TrimSpace(ourAs2Id)
	if ourAs2Id == "" {
		return As2Config{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT TenantType, TenantId, As2Enabled, OurAs2Id, PartnerAs2Id, PartnerUrl,
		             OurCertSecretRef, OurKeySecretRef, PartnerCertSecretRef,
		             SignRequired, EncryptRequired, UpdatedAt
		      FROM PartnerAs2Configs
		      WHERE OurAs2Id = @id
		      LIMIT 1`,
		Params: map[string]any{"id": ourAs2Id},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return As2Config{}, false, nil
	}
	if err != nil {
		if isSpannerNotFound(err) {
			return As2Config{}, false, nil
		}
		return As2Config{}, false, err
	}
	c, err := scanAs2Config(row)
	if err != nil {
		return As2Config{}, false, err
	}
	return c, true, nil
}

func scanAs2Config(row *spanner.Row) (As2Config, error) {
	var c As2Config
	var partnerURL, ourCert, ourKey, partnerCert spanner.NullString
	if err := row.Columns(
		&c.TenantType, &c.TenantID, &c.As2Enabled, &c.OurAs2Id, &c.PartnerAs2Id, &partnerURL,
		&ourCert, &ourKey, &partnerCert,
		&c.SignRequired, &c.EncryptRequired, &c.UpdatedAt,
	); err != nil {
		return As2Config{}, err
	}
	c.PartnerURL = partnerURL.StringVal
	c.OurCertSecretRef = ourCert.StringVal
	c.OurKeySecretRef = ourKey.StringVal
	c.PartnerCertSecretRef = partnerCert.StringVal
	return c, nil
}
