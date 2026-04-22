// Command backfill-orders-h3cell populates Orders.H3Cell from Retailers.H3Index
// for every order missing a cell. Run after the V.5 schema migration adds the
// Orders.H3Cell column. Idempotent — only writes rows where Orders.H3Cell IS NULL.
//
// Usage:
//
//	SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/backfill-orders-h3cell
//	SPANNER_DB=projects/X/instances/Y/databases/Z go run ./cmd/backfill-orders-h3cell
package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const batchSize = 500

type pair struct {
	orderID string
	h3      string
}

func main() {
	db := os.Getenv("SPANNER_DB")
	if db == "" {
		db = "projects/the-lab-project/instances/lab-instance/databases/the-lab-db"
	}

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()

	stmt := spanner.Statement{
		SQL: `
			SELECT o.OrderId, r.H3Index
			FROM Orders o
			JOIN Retailers r ON r.RetailerId = o.RetailerId
			WHERE o.H3Cell IS NULL AND r.H3Index IS NOT NULL AND r.H3Index != ''
		`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var pending []pair
	total := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("scan orders: %v", err)
		}
		var p pair
		if err := row.Columns(&p.orderID, &p.h3); err != nil {
			log.Fatalf("decode row: %v", err)
		}
		pending = append(pending, p)

		if len(pending) >= batchSize {
			flush(ctx, client, pending)
			total += len(pending)
			pending = pending[:0]
		}
	}
	if len(pending) > 0 {
		flush(ctx, client, pending)
		total += len(pending)
	}

	log.Printf("backfill complete: %d orders updated", total)
}

func flush(ctx context.Context, client *spanner.Client, batch []pair) {
	muts := make([]*spanner.Mutation, 0, len(batch))
	for _, p := range batch {
		muts = append(muts, spanner.Update("Orders",
			[]string{"OrderId", "H3Cell"},
			[]interface{}{p.orderID, p.h3},
		))
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		log.Fatalf("apply batch: %v", err)
	}
	log.Printf("flushed %d rows", len(batch))
}
