package migrations

import (
	"context"
	"log"

	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// backfillH3Indexes populates the H3Index column on Retailers, Factories, and
// Orders for rows created before the Geo-Spatial Sovereignty migration. It is
// idempotent — subsequent boots see an empty candidate set and exit immediately.
//
// Retailers/Factories derive H3Index from their own Lat/Lng; Orders inherit
// H3Index from their Retailer so that geo-scoped queries can filter on orders
// directly without joining Retailers at read time.
func backfillH3Indexes(ctx context.Context, sc *spanner.Client) {
	if sc == nil {
		return
	}
	n1 := backfillRetailerH3(ctx, sc)
	n2 := backfillFactoryH3(ctx, sc)
	n3 := backfillOrderH3(ctx, sc)
	if n1+n2+n3 > 0 {
		log.Printf("[H3-BACKFILL] retailers=%d factories=%d orders=%d", n1, n2, n3)
	}
}

type h3BackfillUpdate struct {
	id      string
	h3Index string
}

func backfillRetailerH3(ctx context.Context, sc *spanner.Client) int {
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
			log.Printf("[H3-BACKFILL] retailer scan error: %v", err)
			return len(updates)
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

func backfillFactoryH3(ctx context.Context, sc *spanner.Client) int {
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
			log.Printf("[H3-BACKFILL] factory scan error: %v", err)
			return len(updates)
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

func backfillOrderH3(ctx context.Context, sc *spanner.Client) int {
	iter := sc.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT o.OrderId, r.H3Index
		      FROM Orders o JOIN Retailers r ON r.RetailerId = o.RetailerId
		      WHERE (o.H3Index IS NULL OR o.H3Index = '')
		        AND r.H3Index IS NOT NULL AND r.H3Index != ''`,
	})
	defer iter.Stop()

	var updates []h3BackfillUpdate
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[H3-BACKFILL] order scan error: %v", err)
			return len(updates)
		}
		var id, cell string
		if err := row.Columns(&id, &cell); err != nil {
			continue
		}
		if cell == "" {
			continue
		}
		updates = append(updates, h3BackfillUpdate{id: id, h3Index: cell})
	}
	return applyH3Updates(ctx, sc, "Orders", "OrderId", updates)
}

// applyH3Updates writes H3Index values back in batches of 500 to stay under
// Spanner's mutation limit.
func applyH3Updates(ctx context.Context, sc *spanner.Client, table, pkCol string, updates []h3BackfillUpdate) int {
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
			log.Printf("[H3-BACKFILL] %s batch [%d:%d] error: %v", table, i, end, err)
			return i
		}
	}
	return len(updates)
}
