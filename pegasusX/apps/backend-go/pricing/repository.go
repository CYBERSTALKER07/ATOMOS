package pricing

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

var ErrPriceNotFound = errors.New("price_not_found")

type Repository interface {
	GetActiveUnitPriceMinor(ctx context.Context, supplierId, sku string, date time.Time) (int64, error)
}

type spannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) Repository {
	return &spannerRepository{client: client}
}

func (r *spannerRepository) GetActiveUnitPriceMinor(ctx context.Context, supplierId, sku string, date time.Time) (int64, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT i.UnitPriceMinor
			FROM PriceLists p
			JOIN PriceListItems i ON p.PriceListId = i.PriceListId
			WHERE p.SupplierId = @supplierId
			  AND p.EffectiveFrom <= @date
			  AND (p.EffectiveTo IS NULL OR p.EffectiveTo > @date)
			  AND i.Sku = @sku
			ORDER BY p.EffectiveFrom DESC
			LIMIT 1
		`,
		Params: map[string]interface{}{
			"supplierId": supplierId,
			"date":       date,
			"sku":        sku,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return 0, ErrPriceNotFound
	}
	if err != nil {
		return 0, err
	}

	var price int64
	if err := row.ColumnByName("UnitPriceMinor", &price); err != nil {
		return 0, err
	}

	return price, nil
}
