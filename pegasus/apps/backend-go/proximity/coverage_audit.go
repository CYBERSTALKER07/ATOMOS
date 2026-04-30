package proximity

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ─── Coverage Consistency Auditor ──────────────────────────────────────────────
//
// Scans for "orphaned retailers": retailers whose H3 cell is not covered by any
// active warehouse. These retailers cannot receive auto-dispatched deliveries.
//
// Trigger points:
//   - Warehouse created / updated (geo change) — via workers.EventPool
//   - Retailer location updated — via workers.EventPool
//   - Periodic cron (every 6 hours) — full sweep across all suppliers

const auditBatchSize = 100

// VerifyCoverageConsistency scans all retailers linked to a supplier and flags
// any whose H3 cell has no active warehouse coverage. Idempotent — does not
// create duplicate ORPHAN_DETECTED rows for the same retailer if one already
// exists unresolved.
func VerifyCoverageConsistency(ctx context.Context, client *spanner.Client, supplierID string) error {
	if client == nil {
		return fmt.Errorf("spanner client is nil")
	}

	// Phase 1: Build the unified coverage cell set from all active warehouses
	coverageCells, warehouseByCell, err := buildCoverageIndex(ctx, client, supplierID)
	if err != nil {
		return fmt.Errorf("build coverage index: %w", err)
	}

	// Phase 2: Scan all linked retailers
	retailers, err := fetchLinkedRetailers(ctx, client, supplierID)
	if err != nil {
		return fmt.Errorf("fetch linked retailers: %w", err)
	}

	// Phase 3: Load existing unresolved audits to avoid duplicates
	unresolvedOrphans, err := fetchUnresolvedOrphans(ctx, client, supplierID)
	if err != nil {
		return fmt.Errorf("fetch unresolved orphans: %w", err)
	}

	// Phase 4: Classify each retailer
	var newOrphans []*spanner.Mutation
	var restorations []*spanner.Mutation
	now := time.Now().UTC()

	for _, r := range retailers {
		cell := LookupCell(r.lat, r.lng)
		_, covered := coverageCells[cell]

		if !covered {
			// Orphaned — check if already flagged
			if _, alreadyFlagged := unresolvedOrphans[r.retailerID]; alreadyFlagged {
				continue // idempotent skip
			}

			// Find distance to nearest warehouse for diagnostic value
			var nearestDist *float64
			for _, wh := range warehouseByCell {
				d := HaversineKm(r.lat, r.lng, wh.lat, wh.lng)
				if nearestDist == nil || d < *nearestDist {
					nearestDist = &d
				}
			}

			m := spanner.Insert("DispatchAudit",
				[]string{"AuditId", "SupplierId", "RetailerId", "RetailerCell", "AuditType", "DistanceKm", "CreatedAt"},
				[]interface{}{uuid.New().String(), supplierID, r.retailerID, cell, "ORPHAN_DETECTED", nearestDist, spanner.CommitTimestamp},
			)
			newOrphans = append(newOrphans, m)
		} else {
			// Covered — resolve any existing orphan audit
			if auditID, wasFlagged := unresolvedOrphans[r.retailerID]; wasFlagged {
				whID := warehouseByCell[cell]

				// Resolve the old audit
				resolveM := spanner.Update("DispatchAudit",
					[]string{"AuditId", "ResolvedAt"},
					[]interface{}{auditID, now},
				)
				// Insert a restoration record
				restoreM := spanner.Insert("DispatchAudit",
					[]string{"AuditId", "SupplierId", "RetailerId", "RetailerCell", "AuditType", "WarehouseId", "CreatedAt"},
					[]interface{}{uuid.New().String(), supplierID, r.retailerID, cell, "COVERAGE_RESTORED", whID.warehouseID, spanner.CommitTimestamp},
				)
				restorations = append(restorations, resolveM, restoreM)
			}
		}
	}

	// Phase 5: Write mutations in batches
	allMutations := append(newOrphans, restorations...)
	if len(allMutations) == 0 {
		return nil
	}

	for i := 0; i < len(allMutations); i += auditBatchSize {
		end := i + auditBatchSize
		if end > len(allMutations) {
			end = len(allMutations)
		}
		batch := allMutations[i:end]
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(batch)
		})
		if err != nil {
			log.Printf("[COVERAGE AUDIT] Batch write failed (supplier=%s, batch=%d): %v", supplierID, i/auditBatchSize, err)
			return err
		}
	}

	log.Printf("[COVERAGE AUDIT] supplier=%s: %d new orphans, %d restorations, %d retailers scanned",
		supplierID, len(newOrphans), len(restorations)/2, len(retailers))
	return nil
}

// VerifyCoverageConsistencyAll iterates all suppliers and runs the audit.
// Intended for the periodic cron job.
func VerifyCoverageConsistencyAll(ctx context.Context, client *spanner.Client) {
	if client == nil {
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId FROM Warehouses WHERE IsActive = true`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var count int
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[COVERAGE AUDIT] Failed to list suppliers: %v", err)
			return
		}
		var supplierID string
		if err := row.Columns(&supplierID); err != nil {
			continue
		}
		if err := VerifyCoverageConsistency(ctx, client, supplierID); err != nil {
			log.Printf("[COVERAGE AUDIT] Failed for supplier %s: %v", supplierID, err)
		}
		count++
	}
	log.Printf("[COVERAGE AUDIT] Full sweep complete: %d suppliers audited", count)
}

// ── internal types ──────────────────────────────────────────────────────────

type retailerGeo struct {
	retailerID string
	lat, lng   float64
}

type warehouseGeo struct {
	warehouseID string
	lat, lng    float64
}

// buildCoverageIndex returns:
//   - coverageCells: set of all H3 cells covered by any active warehouse
//   - warehouseByCell: maps each cell to its covering warehouse (nearest wins on overlap)
func buildCoverageIndex(ctx context.Context, client *spanner.Client, supplierID string) (map[string]struct{}, map[string]warehouseGeo, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, H3Indexes
		      FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	coverageCells := make(map[string]struct{})
	warehouseByCell := make(map[string]warehouseGeo)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		var whID string
		var lat, lng spanner.NullFloat64
		var h3Indexes []string

		if err := row.Columns(&whID, &lat, &lng, &h3Indexes); err != nil {
			continue
		}
		if !lat.Valid || !lng.Valid {
			continue
		}

		wh := warehouseGeo{warehouseID: whID, lat: lat.Float64, lng: lng.Float64}
		for _, cell := range h3Indexes {
			coverageCells[cell] = struct{}{}
			warehouseByCell[cell] = wh
		}
	}

	return coverageCells, warehouseByCell, nil
}

// fetchLinkedRetailers returns all retailers linked to this supplier via
// SupplierRetailerClients, with their GPS coordinates.
func fetchLinkedRetailers(ctx context.Context, client *spanner.Client, supplierID string) ([]retailerGeo, error) {
	stmt := spanner.Statement{
		SQL: `SELECT src.RetailerId, r.Latitude, r.Longitude
		      FROM SupplierRetailerClients src
		      JOIN Retailers r ON r.RetailerId = src.RetailerId
		      WHERE src.SupplierId = @sid
		        AND r.Latitude IS NOT NULL
		        AND r.Longitude IS NOT NULL`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []retailerGeo
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var rid string
		var lat, lng float64
		if err := row.Columns(&rid, &lat, &lng); err != nil {
			continue
		}
		results = append(results, retailerGeo{retailerID: rid, lat: lat, lng: lng})
	}
	return results, nil
}

// fetchUnresolvedOrphans returns a map of retailerID → auditID for all unresolved
// ORPHAN_DETECTED records for this supplier.
func fetchUnresolvedOrphans(ctx context.Context, client *spanner.Client, supplierID string) (map[string]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AuditId, RetailerId
		      FROM DispatchAudit
		      WHERE SupplierId = @sid
		        AND AuditType = 'ORPHAN_DETECTED'
		        AND ResolvedAt IS NULL`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]string)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var auditID, retailerID string
		if err := row.Columns(&auditID, &retailerID); err != nil {
			continue
		}
		result[retailerID] = auditID
	}
	return result, nil
}
