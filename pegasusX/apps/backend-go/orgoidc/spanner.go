package orgoidc

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
)

// SpannerStore persists SupplierOIDC. Nil client → treat as empty store.
type SpannerStore struct {
	client *spanner.Client
}

func NewSpannerStore(client *spanner.Client) *SpannerStore {
	return &SpannerStore{client: client}
}

func (s *SpannerStore) Get(ctx context.Context, supplierID string) (Config, bool, error) {
	if s == nil || s.client == nil {
		return Config{}, false, nil
	}
	row, err := s.client.Single().ReadRow(ctx, "SupplierOIDC", spanner.Key{supplierID},
		[]string{"SupplierId", "Issuer", "ClientId", "Audience", "AuthorizationEndpoint", "RedirectURI", "AdminEmails", "Enabled"})
	if err != nil {
		if err == spanner.ErrRowNotFound {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}
	var c Config
	var aud, authz, redir spanner.NullString
	if err := row.Columns(&c.SupplierID, &c.Issuer, &c.ClientID, &aud, &authz, &redir, &c.AdminEmails, &c.Enabled); err != nil {
		return Config{}, false, err
	}
	c.Audience = aud.StringVal
	c.AuthorizationEndpoint = authz.StringVal
	c.RedirectURI = redir.StringVal
	if c.AdminEmails == nil {
		c.AdminEmails = []string{}
	}
	return c, true, nil
}

func (s *SpannerStore) Put(ctx context.Context, c Config) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierOIDC", map[string]any{
		"SupplierId":            c.SupplierID,
		"Issuer":                c.Issuer,
		"ClientId":              c.ClientID,
		"Audience":              nullableStr(c.Audience),
		"AuthorizationEndpoint": nullableStr(c.AuthorizationEndpoint),
		"RedirectURI":           nullableStr(c.RedirectURI),
		"Enabled":               c.Enabled,
		"UpdatedAt":             spanner.CommitTimestamp,
	})})
	return err
}

func (s *SpannerStore) Delete(ctx context.Context, supplierID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.Apply(ctx, []*spanner.Mutation{spanner.Delete("SupplierOIDC", spanner.Key{supplierID})})
	return err
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
