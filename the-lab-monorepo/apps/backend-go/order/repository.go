package order

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Repository defines the data access methods for orders and related entities
type Repository interface {
	GetRetailerDistance(ctx context.Context, retailerID string, currentLoc Location) (float64, error)
	UpdateOrderState(ctx context.Context, orderID string, state string) error
}

type repository struct {
	client *spanner.Client
}

// NewRepository creates a new instance of the Spanner repository
func NewRepository(client *spanner.Client) Repository {
	return &repository{client: client}
}

// GetRetailerDistance calculates the distance in meters between a retailer's shop and a provided location.
func (r *repository) GetRetailerDistance(ctx context.Context, retailerID string, currentLoc Location) (float64, error) {
	// Spanner GEOGRAPHY is constructed using ST_GEOGPOINT(longitude, latitude).
	// We use parameterized queries to prevent SQL injection.
	stmt := spanner.Statement{
		SQL: `
			SELECT ST_DISTANCE(ShopLocation, ST_GEOGPOINT(@longitude, @latitude)) AS DistanceInMeters
			FROM Retailers
			WHERE RetailerId = @retailerId
		`,
		Params: map[string]interface{}{
			"retailerId": retailerID,
			"latitude":   currentLoc.Latitude,
			"longitude":  currentLoc.Longitude,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return 0, fmt.Errorf("retailer with id %s not found", retailerID)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to execute location query: %w", err)
	}

	// Because ShopLocation could theoretically be null, we parse into an intermediate optional.
	// But assuming data consistency, a fallback to decoding directly to float64 is simpler.
	var distance spanner.NullFloat64
	if err := row.Columns(&distance); err != nil {
		return 0, fmt.Errorf("failed to parse distance: %w", err)
	}

	if !distance.Valid {
		return 0, fmt.Errorf("retailer ShopLocation is missing")
	}

	return distance.Float64, nil
}

// UpdateOrderState updates the state of an existing order in Spanner
func (r *repository) UpdateOrderState(ctx context.Context, orderID string, state string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE Orders SET State = @state WHERE OrderId = @orderId`,
			Params: map[string]interface{}{
				"state":   state,
				"orderId": orderID,
			},
		}

		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		if rowCount == 0 {
			return fmt.Errorf("order %s not found for update", orderID)
		}

		return nil
	})

	return err
}
