// backfill_volume_vu hydrates Orders.VolumeVU for any row where the column is
// NULL by summing OrderLineItems.Quantity at a default 1.0 VU per unit. The
// binary is idempotent — re-running after completion is a no-op.
//
// Phase 2 simplification: per-SKU volume is not yet a first-class column on
// SupplierProducts, so this binary uses a constant DefaultUnitVolumeVU = 1.0.
// When SupplierProducts.VolumePerUnit lands (a separate migration), update the
// SQL to JOIN that column and re-run; the COALESCE in the backend hydration
// path handles the transition without code changes.
//
// Operational: requires SPANNER_EMULATOR_HOST or production credentials in env.
// Default target is the local emulator project/instance/database.
//
//	go run ./cmd/backfill_volume_vu
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// DefaultUnitVolumeVU is the per-unit volumetric unit assumption used until
// SupplierProducts grows a VolumePerUnit column. Kept here (not in the dispatch
// package) so the backfill remains a self-contained operational tool.
const DefaultUnitVolumeVU = 1.0

// batchSize bounds Spanner mutations per RWTxn (the hard cell-mutation limit
// is 20k; 500 row updates × 1 column = 500 mutations, well within budget).
const batchSize = 500

func main() {
	dbName := flag.String("db",
		"projects/lab-logistics/instances/lab-dev/databases/lab-db",
		"fully-qualified Spanner database name")
	flag.Parse()

	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, *dbName)
	if err != nil {
		logger.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	start := time.Now()
	scanned, updated, err := backfill(ctx, client, logger)
	if err != nil {
		logger.Error("backfill failed", "err", err, "scanned", scanned, "updated", updated)
		os.Exit(1)
	}
	logger.Info("backfill complete",
		"scanned", scanned,
		"updated", updated,
		"elapsed_ms", time.Since(start).Milliseconds(),
	)
}

// orderVolume bundles the result of the SUM(Quantity) join per OrderId.
type orderVolume struct {
	orderID  string
	volumeVU float64
}

// backfill scans every Orders row with NULL VolumeVU, computes its volume
// from joined OrderLineItems, and writes the result back in bounded batches.
// Returns (scanned, updated, error).
func backfill(ctx context.Context, client *spanner.Client, logger *slog.Logger) (int, int, error) {
	stmt := spanner.Statement{SQL: `
		SELECT o.OrderId,
		       COALESCE(SUM(li.Quantity), 0) AS TotalQty
		  FROM Orders o
		  LEFT JOIN OrderLineItems li ON li.OrderId = o.OrderId
		 WHERE o.VolumeVU IS NULL
		 GROUP BY o.OrderId
	`}

	pending := make([]orderVolume, 0, batchSize)
	scanned := 0
	updated := 0

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return scanned, updated, fmt.Errorf("scan orders: %w", err)
		}
		scanned++

		var orderID string
		var qty int64
		if err := row.Columns(&orderID, &qty); err != nil {
			return scanned, updated, fmt.Errorf("decode row: %w", err)
		}

		pending = append(pending, orderVolume{
			orderID:  orderID,
			volumeVU: float64(qty) * DefaultUnitVolumeVU,
		})

		if len(pending) >= batchSize {
			n, err := flush(ctx, client, pending)
			if err != nil {
				return scanned, updated, err
			}
			updated += n
			logger.Info("flushed batch", "rows", n, "total_updated", updated)
			pending = pending[:0]
		}
	}

	if len(pending) > 0 {
		n, err := flush(ctx, client, pending)
		if err != nil {
			return scanned, updated, err
		}
		updated += n
	}

	return scanned, updated, nil
}

// flush applies a single RWTxn that updates VolumeVU for every order in the
// batch. Returns the number of rows written.
func flush(ctx context.Context, client *spanner.Client, batch []orderVolume) (int, error) {
	mutations := make([]*spanner.Mutation, 0, len(batch))
	for _, ov := range batch {
		mutations = append(mutations, spanner.Update(
			"Orders",
			[]string{"OrderId", "VolumeVU"},
			[]interface{}{ov.orderID, ov.volumeVU},
		))
	}
	_, err := client.Apply(ctx, mutations)
	if err != nil {
		return 0, fmt.Errorf("apply batch of %d: %w", len(batch), err)
	}
	return len(batch), nil
}
