package migrations

import (
	"context"
	"fmt"
	"log"

	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// backfillH3Indexes populates the H3Index column on Retailers and Factories
// for rows created before the Geo-Spatial Sovereignty migration.
func backfillH3Indexes(ctx context.Context, sc *spanner.Client) error {
	if sc == nil {
		return nil
	}
	n1, err := backfillRetailerH3(ctx, sc)
	if err != nil {
		return err
	}
	n2, err := backfillFactoryH3(ctx, sc)
	if err != nil {
		return err
	}
	if n1+n2 > 0 {
		log.Printf("[H3-BACKFILL] retailers=%d factories=%d", n1, n2)
	}
	return nil
}

type h3BackfillUpdate struct {
	id      string
	h3Index string
}

func backfillRetailerH3(ctx context.Context, sc *spanner.Client) (int, error) {
	iter := sc.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, Latitude, Longitude FROM Retailers
		      WHERE (H3Index IS NULL OR H3Index = '')
		        AND Latitude IS NOT NULL AND Longitude IS NOT NULL`,
	})
	defer iter.Stop()

	var updates []h3BackfillUpdate
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if IsInfrastructureNotFound(err) {
				return 0, fmt.Errorf("h3 backfill retailers: %w", err)
			}
			log.Printf("[H3-BACKFILL] retailer scan error: %v", err)
			return len(updates), nil
		}
		var id string
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&id, &lat, &lng); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}
		cell := proximity.LookupCell(lat.Float64, lng.Float64)
		if cell == "" {
			continue
		}
		updates = append(updates, h3BackfillUpdate{id: id, h3Index: cell})
	}
	return applyH3Updates(ctx, sc, "Retailers", "RetailerId", updates)
}

func backfillFactoryH3(ctx context.Context, sc *spanner.Client) (int, error) {
	iter := sc.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FactoryId, Lat, Lng FROM Factories
		      WHERE (H3Index IS NULL OR H3Index = '')
		        AND Lat IS NOT NULL AND Lng IS NOT NULL`,
	})
	defer iter.Stop()

	var updates []h3BackfillUpdate
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if IsInfrastructureNotFound(err) {
				return 0, fmt.Errorf("h3 backfill factories: %w", err)
			}
			log.Printf("[H3-BACKFILL] factory scan error: %v", err)
			return len(updates), nil
		}
		var id string
		var lat, lng spanner.NullFloat64
		if err := row.Columns(&id, &lat, &lng); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}
		cell := proximity.LookupCell(lat.Float64, lng.Float64)
		if cell == "" {
			continue
		}
		updates = append(updates, h3BackfillUpdate{id: id, h3Index: cell})
	}
	return applyH3Updates(ctx, sc, "Factories", "FactoryId", updates)
}

func applyH3Updates(ctx context.Context, sc *spanner.Client, table, pkCol string, updates []h3BackfillUpdate) (int, error) {
	const batchSize = 500
	for i := 0; i < len(updates); i += batchSize {
		end := i + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		muts := make([]*spanner.Mutation, 0, end-i)
		for _, u := range updates[i:end] {
			muts = append(muts, spanner.Update(table,
				[]string{pkCol, "H3Index"},
				[]interface{}{u.id, u.h3Index}))
		}
		if _, err := sc.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(muts)
		}); err != nil {
			if IsInfrastructureNotFound(err) {
				return i, fmt.Errorf("h3 backfill %s: %w", table, err)
			}
			log.Printf("[H3-BACKFILL] %s batch [%d:%d] error: %v", table, i, end, err)
			return i, nil
		}
	}
	return len(updates), nil
}
