package globalproducts

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository persists GlobalProducts tables.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) EnsureStandardUoM(ctx context.Context) error {
	now := spanner.CommitTimestamp
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("UnitsOfMeasure", map[string]any{
			"UomId": UomEachID, "Code": "EACH", "Name": "Each", "FactorToBase": int64(1), "CreatedAt": now,
		}),
		spanner.InsertOrUpdateMap("UnitsOfMeasure", map[string]any{
			"UomId": UomInnerID, "Code": "INNER", "Name": "Inner pack", "FactorToBase": int64(6), "ParentUomId": UomEachID, "CreatedAt": now,
		}),
		spanner.InsertOrUpdateMap("UnitsOfMeasure", map[string]any{
			"UomId": UomCaseID, "Code": "CASE", "Name": "Case", "FactorToBase": int64(24), "ParentUomId": UomInnerID, "CreatedAt": now,
		}),
		spanner.InsertOrUpdateMap("UnitsOfMeasure", map[string]any{
			"UomId": UomPalletID, "Code": "PALLET", "Name": "Pallet", "FactorToBase": int64(96), "ParentUomId": UomCaseID, "CreatedAt": now,
		}),
	})
	return err
}

func (r *SpannerRepository) GetBrandByNormalizedName(ctx context.Context, normName string) (*GlobalBrand, error) {
	stmt := spanner.Statement{
		SQL: `SELECT BrandId, Name, NormalizedName, OwnerSupplierId, Status, CreatedAt, UpdatedAt
		      FROM GlobalBrands WHERE NormalizedName = @n LIMIT 1`,
		Params: map[string]any{"n": normName},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var b GlobalBrand
	var owner spanner.NullString
	if err := row.Columns(&b.BrandID, &b.Name, &b.NormalizedName, &owner, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	b.OwnerSupplierID = owner.StringVal
	return &b, nil
}

func (r *SpannerRepository) UpsertBrand(ctx context.Context, b GlobalBrand) error {
	m := spanner.InsertOrUpdateMap("GlobalBrands", map[string]any{
		"BrandId":         b.BrandID,
		"Name":            b.Name,
		"NormalizedName":  b.NormalizedName,
		"OwnerSupplierId": nullable(b.OwnerSupplierID),
		"Status":          b.Status,
		"CreatedAt":       spanner.CommitTimestamp,
		"UpdatedAt":       spanner.CommitTimestamp,
	})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) GetByGtin(ctx context.Context, gtin string) (*GlobalProduct, error) {
	stmt := spanner.Statement{
		SQL: `SELECT GlobalProductId, Gtin, BrandId, Manufacturer, Name, PackQty, BaseUomId, NormalizedKey, Version, CreatedAt, UpdatedAt
		      FROM GlobalProducts WHERE Gtin = @gtin LIMIT 1`,
		Params: map[string]any{"gtin": gtin},
	}
	return r.queryOne(ctx, stmt)
}

func (r *SpannerRepository) GetByID(ctx context.Context, id string) (*GlobalProduct, error) {
	row, err := r.client.Single().ReadRow(ctx, "GlobalProducts", spanner.Key{id},
		[]string{"GlobalProductId", "Gtin", "BrandId", "Manufacturer", "Name", "PackQty", "BaseUomId", "NormalizedKey", "Version", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return scanGlobal(row)
}

func isNotFound(err error) bool {
	return spanner.ErrCode(err) == codes.NotFound
}

func (r *SpannerRepository) queryOne(ctx context.Context, stmt spanner.Statement) (*GlobalProduct, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return scanGlobal(row)
}

func scanGlobal(row *spanner.Row) (*GlobalProduct, error) {
	var gp GlobalProduct
	var gtin, mfr, key spanner.NullString
	if err := row.Columns(&gp.GlobalProductID, &gtin, &gp.BrandID, &mfr, &gp.Name, &gp.PackQty, &gp.BaseUomID, &key, &gp.Version, &gp.CreatedAt, &gp.UpdatedAt); err != nil {
		return nil, err
	}
	gp.Gtin = gtin.StringVal
	gp.Manufacturer = mfr.StringVal
	gp.NormalizedKey = key.StringVal
	return &gp, nil
}

func (r *SpannerRepository) ListByNormalizedKey(ctx context.Context, key string) ([]GlobalProduct, error) {
	stmt := spanner.Statement{
		SQL: `SELECT GlobalProductId, Gtin, BrandId, Manufacturer, Name, PackQty, BaseUomId, NormalizedKey, Version, CreatedAt, UpdatedAt
		      FROM GlobalProducts WHERE NormalizedKey = @k`,
		Params: map[string]any{"k": key},
	}
	return r.queryMany(ctx, stmt)
}

func (r *SpannerRepository) ListAll(ctx context.Context, limit int) ([]GlobalProduct, error) {
	if limit <= 0 {
		limit = 500
	}
	stmt := spanner.Statement{
		SQL:    `SELECT GlobalProductId, Gtin, BrandId, Manufacturer, Name, PackQty, BaseUomId, NormalizedKey, Version, CreatedAt, UpdatedAt FROM GlobalProducts LIMIT @lim`,
		Params: map[string]any{"lim": int64(limit)},
	}
	return r.queryMany(ctx, stmt)
}

func (r *SpannerRepository) queryMany(ctx context.Context, stmt spanner.Statement) ([]GlobalProduct, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []GlobalProduct
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		gp, err := scanGlobal(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *gp)
	}
	return out, nil
}

func (r *SpannerRepository) UpsertGlobal(ctx context.Context, gp GlobalProduct) error {
	if gp.GlobalProductID == "" {
		gp.GlobalProductID = uuid.NewString()
	}
	m := spanner.InsertOrUpdateMap("GlobalProducts", map[string]any{
		"GlobalProductId": gp.GlobalProductID,
		"Gtin":            nullable(gp.Gtin),
		"BrandId":         gp.BrandID,
		"Manufacturer":    nullable(gp.Manufacturer),
		"Name":            gp.Name,
		"PackQty":         gp.PackQty,
		"BaseUomId":       gp.BaseUomID,
		"NormalizedKey":   nullable(gp.NormalizedKey),
		"Version":         gp.Version,
		"CreatedAt":       spanner.CommitTimestamp,
		"UpdatedAt":       spanner.CommitTimestamp,
	})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func nullable(s string) any {
	if s == "" {
		return spanner.NullString{}
	}
	return s
}

func (r *SpannerRepository) UpsertOffer(ctx context.Context, o Offer) error {
	if o.Version == 0 {
		o.Version = 1
	}
	m := spanner.InsertOrUpdateMap("SupplierProductOffers", map[string]any{
		"SupplierId":      o.SupplierID,
		"ProductId":       o.ProductID,
		"GlobalProductId": o.GlobalProductID,
		"PriceMinor":      o.PriceMinor,
		"Currency":        o.Currency,
		"Moq":             o.Moq,
		"LeadTimeDays":    o.LeadTimeDays,
		"Status":          o.Status,
		"Version":         o.Version,
		"CreatedAt":       spanner.CommitTimestamp,
		"UpdatedAt":       spanner.CommitTimestamp,
	})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) GetOffer(ctx context.Context, supplierID, productID string) (*Offer, error) {
	row, err := r.client.Single().ReadRow(ctx, "SupplierProductOffers", spanner.Key{supplierID, productID},
		[]string{"SupplierId", "ProductId", "GlobalProductId", "PriceMinor", "Currency", "Moq", "LeadTimeDays", "Status", "Version", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var o Offer
	if err := row.Columns(&o.SupplierID, &o.ProductID, &o.GlobalProductID, &o.PriceMinor, &o.Currency, &o.Moq, &o.LeadTimeDays, &o.Status, &o.Version, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *SpannerRepository) ListOffersByGlobal(ctx context.Context, globalProductID string) ([]Offer, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, ProductId, GlobalProductId, PriceMinor, Currency, Moq, LeadTimeDays, Status, Version, CreatedAt, UpdatedAt
		      FROM SupplierProductOffers WHERE GlobalProductId = @gid AND Status = @st`,
		Params: map[string]any{"gid": globalProductID, "st": StatusLinked},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Offer
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var o Offer
		if err := row.Columns(&o.SupplierID, &o.ProductID, &o.GlobalProductID, &o.PriceMinor, &o.Currency, &o.Moq, &o.LeadTimeDays, &o.Status, &o.Version, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (r *SpannerRepository) EnqueueMatch(ctx context.Context, item MatchQueueItem) error {
	if item.QueueID == "" {
		item.QueueID = uuid.NewString()
	}
	m := spanner.InsertOrUpdateMap("ProductMatchQueue", map[string]any{
		"QueueId":                  item.QueueID,
		"SupplierId":               item.SupplierID,
		"ProductId":                item.ProductID,
		"CandidateGlobalProductId": nullable(item.CandidateGlobalProductID),
		"MatchMethod":              item.MatchMethod,
		"Score":                    item.Score,
		"Status":                   item.Status,
		"Reason":                   nullable(item.Reason),
		"Version":                  int64(1),
		"CreatedAt":                spanner.CommitTimestamp,
		"UpdatedAt":                spanner.CommitTimestamp,
	})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository) ListMatchQueue(ctx context.Context, status string, limit int) ([]MatchQueueItem, error) {
	if limit <= 0 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL: `SELECT QueueId, SupplierId, ProductId, CandidateGlobalProductId, MatchMethod, Score, Status, Reason, Version, CreatedAt, UpdatedAt
		      FROM ProductMatchQueue WHERE Status = @st ORDER BY CreatedAt LIMIT @lim`,
		Params: map[string]any{"st": status, "lim": int64(limit)},
	}
	if status == "" {
		stmt.SQL = `SELECT QueueId, SupplierId, ProductId, CandidateGlobalProductId, MatchMethod, Score, Status, Reason, Version, CreatedAt, UpdatedAt
		            FROM ProductMatchQueue WHERE SupplierId IS NOT NULL ORDER BY CreatedAt LIMIT @lim`
		stmt.Params = map[string]any{"lim": int64(limit)}
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []MatchQueueItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		item, err := scanQueue(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func scanQueue(row *spanner.Row) (*MatchQueueItem, error) {
	var item MatchQueueItem
	var cand, reason spanner.NullString
	if err := row.Columns(&item.QueueID, &item.SupplierID, &item.ProductID, &cand, &item.MatchMethod, &item.Score, &item.Status, &reason, &item.Version, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	item.CandidateGlobalProductID = cand.StringVal
	item.Reason = reason.StringVal
	return &item, nil
}

func (r *SpannerRepository) GetMatchQueueItem(ctx context.Context, queueID string) (*MatchQueueItem, error) {
	row, err := r.client.Single().ReadRow(ctx, "ProductMatchQueue", spanner.Key{queueID},
		[]string{"QueueId", "SupplierId", "ProductId", "CandidateGlobalProductId", "MatchMethod", "Score", "Status", "Reason", "Version", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return scanQueue(row)
}

func (r *SpannerRepository) UpdateMatchQueue(ctx context.Context, item MatchQueueItem) error {
	m := spanner.UpdateMap("ProductMatchQueue", map[string]any{
		"QueueId":                  item.QueueID,
		"Status":                   item.Status,
		"CandidateGlobalProductId": nullable(item.CandidateGlobalProductID),
		"Reason":                   nullable(item.Reason),
		"Version":                  item.Version + 1,
		"UpdatedAt":                spanner.CommitTimestamp,
	})
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("update match queue: %w", err)
	}
	return nil
}
