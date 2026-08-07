package ar

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerContactResolver resolves dunning contacts from live master data:
// retailers are phone-authenticated (Retailers.Phone); supplier staff carry
// SupplierUsers.Email for the email channel.
type SpannerContactResolver struct {
	client *spanner.Client
}

func NewSpannerContactResolver(client *spanner.Client) *SpannerContactResolver {
	return &SpannerContactResolver{client: client}
}

func (r *SpannerContactResolver) ResolveRetailer(ctx context.Context, retailerID string) (Contact, error) {
	row, err := r.client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Phone", "Name"})
	if err != nil {
		return Contact{}, err
	}
	var phone string
	var name spanner.NullString
	if err := row.Columns(&phone, &name); err != nil {
		return Contact{}, err
	}
	return Contact{Phone: phone, Name: name.StringVal}, nil
}

func (r *SpannerContactResolver) ResolveSupplierStaff(ctx context.Context, supplierID string) ([]Contact, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Email, Name, Phone FROM SupplierUsers WHERE SupplierId = @sid AND IsActive = TRUE`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var out []Contact
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var email, phone spanner.NullString
		var name string
		if err := row.Columns(&email, &name, &phone); err != nil {
			return nil, err
		}
		c := Contact{Email: strings.TrimSpace(email.StringVal), Name: name, Phone: phone.StringVal}
		if c.Email == "" && c.Phone == "" {
			continue
		}
		out = append(out, c)
	}
}
