// verify_ddl is a one-shot operational utility that asserts every Phase-1
// schema migration landed in the local Spanner emulator after a full reset.
// Each query is expected to return exactly one row; a `0 row(s)` result means
// cmd/setup/main.go's hardcoded ddlStatements slice and the doctrine
// drifted apart again.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type check struct {
	label string
	sql   string
}

var checks = []check{
	// ── Outbox header migration (Wave A — done) ──
	{"OutboxEvents.EventType",
		`SELECT column_name, spanner_type FROM information_schema.columns
		   WHERE table_name='OutboxEvents' AND column_name='EventType'`},

	// ── V.5 H3 spatial indexing (done in this pass) ──
	{"Orders.H3Cell",
		`SELECT column_name, spanner_type FROM information_schema.columns
		   WHERE table_name='Orders' AND column_name='H3Cell'`},
	{"index IDX_Orders_H3Cell_State",
		`SELECT index_name FROM information_schema.indexes
		   WHERE index_name='IDX_Orders_H3Cell_State'`},

	// ── Phase 2 — dispatch optimization inputs ──
	{"Orders.VolumeVU",
		`SELECT column_name, spanner_type FROM information_schema.columns
		   WHERE table_name='Orders' AND column_name='VolumeVU'`},
	{"index Idx_Orders_H3Cell_State_Date",
		`SELECT index_name FROM information_schema.indexes
		   WHERE index_name='Idx_Orders_H3Cell_State_Date'`},
	{"Retailers.ReceivingWindowOpen",
		`SELECT column_name, spanner_type FROM information_schema.columns
		   WHERE table_name='Retailers' AND column_name='ReceivingWindowOpen'`},
	{"Retailers.ReceivingWindowClose",
		`SELECT column_name, spanner_type FROM information_schema.columns
		   WHERE table_name='Retailers' AND column_name='ReceivingWindowClose'`},

	// ── Phase 2 — manifest aggregate ──
	{"table OrderManifests",
		`SELECT table_name FROM information_schema.tables
		   WHERE table_name='OrderManifests'`},
	{"table OrderManifestStops",
		`SELECT table_name FROM information_schema.tables
		   WHERE table_name='OrderManifestStops'`},
	{"index Idx_OrderManifests_BySupplierState",
		`SELECT index_name FROM information_schema.indexes
		   WHERE index_name='Idx_OrderManifests_BySupplierState'`},
	{"index Idx_OrderManifests_ByVehicleState",
		`SELECT index_name FROM information_schema.indexes
		   WHERE index_name='Idx_OrderManifests_ByVehicleState'`},
	{"index Idx_OrderManifestStops_ByOrderId",
		`SELECT index_name FROM information_schema.indexes
		   WHERE index_name='Idx_OrderManifestStops_ByOrderId'`},
}

func main() {
	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	}
	ctx := context.Background()
	c, err := spanner.NewClient(ctx, "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	missing := 0
	for _, ch := range checks {
		iter := c.Single().Query(ctx, spanner.Statement{SQL: ch.sql})
		hits := 0
		for {
			_, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			hits++
		}
		iter.Stop()
		status := "✓"
		if hits == 0 {
			status = "✗"
			missing++
		}
		fmt.Printf("  %s %-44s (%d row)\n", status, ch.label, hits)
	}
	if missing > 0 {
		log.Fatalf("%d schema artifact(s) missing — DDL drift detected", missing)
	}
	fmt.Println("All schema artifacts present.")
}
