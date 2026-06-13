// Command seed-supplier-prodsim seeds Spanner rows for local prod-simulation smoke tests:
// vet-queue order (PENDING), dispatchable order (CONFIRMED + payment cleared),
// warehouse, driver, vehicle, and payment-ledger credits.
//
// Prerequisite: run `go run ./cmd/setup` once (schema + supplier row).
//
// Usage (safe to re-run — clears orphan manifests, resets dispatchable order bindings):
//   cd pegasusX/apps/backend-go && go run ./cmd/seed-supplier-prodsim
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	prodsimWarehouseID      = "prodsim-wh-1"
	prodsimVehicleID        = "prodsim-vehicle-1"
	prodsimDriverID         = "prodsim-driver-1"
	prodsimRetailerID       = "prodsim-retailer-1"
	prodsimPendingOrderID   = "prodsim-order-pending"
	prodsimDispatchOrderID  = "prodsim-order-dispatch"
	prodsimLedgerPendingID  = "prodsim-ledger-pending"
	prodsimLedgerDispatchID = "prodsim-ledger-dispatch"
	prodsimRetailerPhone    = "+998901234501"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	repo := &seedRepository{client: client}
	supplier, err := seed.EnsureSupplier(ctx, repo, cfg.SeedSupplierName, cfg.SeedSupplierCountry, cfg.SeedSupplierCurrency, logger)
	if err != nil {
		slog.Error("ensure supplier", "err", err)
		os.Exit(1)
	}

	removed, err := cleanupProdsimOrphanManifests(ctx, client, supplier.SupplierID, prodsimWarehouseID)
	if err != nil {
		slog.Error("cleanup orphan manifests", "err", err)
		os.Exit(1)
	}
	if removed > 0 {
		slog.Info("removed orphan prodsim manifests", "count", removed, "warehouse_id", prodsimWarehouseID)
	}

	pendingOrderID := prodsimPendingOrderID
	dispatchOrderID := prodsimDispatchOrderID
	retailerID := prodsimRetailerID
	ts := spanner.CommitTimestamp
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":        prodsimWarehouseID,
				"SupplierId":         supplier.SupplierID,
				"Name":               "ProdSim Warehouse",
				"Lat":                41.2995,
				"Lng":                69.2401,
				"CoverageRadiusKm":   15.0,
				"IsActive":           true,
				"IsOnShift":          true,
				"AutoDispatchEnabled": false,
				"CreatedAt":          ts,
				"UpdatedAt":          ts,
			}),
			spanner.InsertOrUpdateMap("Vehicles", map[string]any{
				"VehicleId":     prodsimVehicleID,
				"SupplierId":    supplier.SupplierID,
				"Label":         "ProdSim Van",
				"LicensePlate":  "PSIM001",
				"HomeNodeType":  "WAREHOUSE",
				"HomeNodeId":    prodsimWarehouseID,
				"VehicleClass":  "CLASS_B",
				"MaxVolumeVU":   150.0,
				"IsActive":      true,
				"CreatedAt":     ts,
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("Drivers", map[string]any{
				"DriverId":     prodsimDriverID,
				"Name":         "ProdSim Driver",
				"Phone":        "+998901234599",
				"SupplierId":   supplier.SupplierID,
				"HomeNodeType": "WAREHOUSE",
				"HomeNodeId":   prodsimWarehouseID,
				"VehicleId":    prodsimVehicleID,
				"IsActive":     true,
				"CreatedAt":    ts,
				"UpdatedAt":    ts,
			}),
			spanner.InsertOrUpdateMap("Retailers", map[string]any{
				"RetailerId":  retailerID,
				"Name":        "ProdSim Retailer",
				"Phone":       prodsimRetailerPhone,
				"CountryCode": supplier.CountryCode,
				"Lat":         41.31,
				"Lng":         69.28,
				"CreatedAt":   ts,
			}),
			spanner.InsertOrUpdateMap("Orders", map[string]any{
				"OrderId":            pendingOrderID,
				"SupplierId":         supplier.SupplierID,
				"RetailerId":         retailerID,
				"WarehouseId":          prodsimWarehouseID,
				"Status":             "PENDING",
				"OrderSource":        "RETAILER_APP",
				"ConfirmationStatus": "PENDING",
				"LineItemsJson":      []byte(`[]`),
				"TotalMinor":         int64(125_000),
				"Currency":           supplier.Currency,
				"Lat":                41.31,
				"Lng":                69.28,
				"Version":            int64(1),
				"CreatedAt":          ts,
				"UpdatedAt":          ts,
			}),
			spanner.InsertOrUpdateMap("Orders", map[string]any{
				"OrderId":            dispatchOrderID,
				"SupplierId":         supplier.SupplierID,
				"RetailerId":         retailerID,
				"WarehouseId":        prodsimWarehouseID,
				"DriverId":           nil,
				"VehicleId":          nil,
				"RouteId":            nil,
				"ManifestId":         nil,
				"Status":             "PENDING",
				"OrderSource":        "RETAILER_APP",
				"ConfirmationStatus": "CONFIRMED",
				"LineItemsJson":      []byte(`[]`),
				"TotalMinor":         int64(210_000),
				"Currency":           supplier.Currency,
				"Lat":                41.312,
				"Lng":                69.281,
				"Version":            int64(1),
				"CreatedAt":          ts,
				"UpdatedAt":          ts,
			}),
			spanner.InsertOrUpdateMap("PaymentLedgerEntries", map[string]any{
				"LedgerEntryId": prodsimLedgerPendingID,
				"OrderId":       pendingOrderID,
				"SupplierId":    supplier.SupplierID,
				"RetailerId":    retailerID,
				"Gateway":       "PAYME",
				"EntryType":     "WEBHOOK_PAID",
				"AmountMinor":   int64(125_000),
				"Currency":      supplier.Currency,
				"ReferenceId":   "prodsim_seed_pending",
				"Source":        "seed-supplier-prodsim",
				"OccurredAt":    ts,
				"CreatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("PaymentLedgerEntries", map[string]any{
				"LedgerEntryId": prodsimLedgerDispatchID,
				"OrderId":       dispatchOrderID,
				"SupplierId":    supplier.SupplierID,
				"RetailerId":    retailerID,
				"Gateway":       "PAYME",
				"EntryType":     "WEBHOOK_PAID",
				"AmountMinor":   int64(210_000),
				"Currency":      supplier.Currency,
				"ReferenceId":   "prodsim_seed_dispatch",
				"Source":        "seed-supplier-prodsim",
				"OccurredAt":    ts,
				"CreatedAt":     ts,
			}),
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		slog.Error("seed prodsim rows", "err", err)
		os.Exit(1)
	}

	slog.Info("supplier prodsim seed complete",
		"supplier_id", supplier.SupplierID,
		"warehouse_id", prodsimWarehouseID,
		"driver_id", prodsimDriverID,
		"vehicle_id", prodsimVehicleID,
		"pending_order_id", pendingOrderID,
		"dispatch_order_id", dispatchOrderID,
		"retailer_id", retailerID,
	)
}

type seedRepository struct {
	client *spanner.Client
}

func (r *seedRepository) UpsertSupplier(ctx context.Context, s seed.Supplier) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt, err := supplierCreatedAt(ctx, txn, s.SupplierID)
		if err != nil {
			return err
		}
		mutation := spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":   s.SupplierID,
			"Name":         s.Name,
			"CountryCode":  s.CountryCode,
			"Currency":     s.Currency,
			"IsConfigured": false,
			"CreatedAt":    createdAt,
			"UpdatedAt":    spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		return fmt.Errorf("upsert supplier %s: %w", s.SupplierID, err)
	}
	return nil
}

func supplierCreatedAt(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID string,
) (any, error) {
	row, err := txn.ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"CreatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return spanner.CommitTimestamp, nil
		}
		return nil, fmt.Errorf("read supplier %s: %w", supplierID, err)
	}
	var createdAt time.Time
	if err := row.Columns(&createdAt); err != nil {
		return nil, err
	}
	return createdAt, nil
}

func spannerClientOptions(cfg *bootstrap.Config) []option.ClientOption {
	if strings.TrimSpace(cfg.SpannerEmulatorHost) == "" {
		return nil
	}
	return []option.ClientOption{
		option.WithEndpoint(cfg.SpannerEmulatorHost),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)
}

// cleanupProdsimOrphanManifests deletes non-terminal manifests at the prodsim warehouse
// and unbinds any orders still pointing at them so re-seed restores a dispatchable loop.
func cleanupProdsimOrphanManifests(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (int, error) {
	supplierID = strings.TrimSpace(supplierID)
	warehouseID = strings.TrimSpace(warehouseID)
	if client == nil || supplierID == "" || warehouseID == "" {
		return 0, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT ManifestId
		      FROM SupplierTruckManifests@{FORCE_INDEX=Idx_SupplierManifests_ByWarehouse}
		      WHERE WarehouseId = @warehouseId
		        AND SupplierId = @supplierId
		        AND State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED')`,
		Params: map[string]any{
			"warehouseId": warehouseID,
			"supplierId":  supplierID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	manifestIDs := make([]string, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("list prodsim orphan manifests: %w", err)
		}
		var manifestID string
		if err := row.Columns(&manifestID); err != nil {
			return 0, fmt.Errorf("scan prodsim orphan manifest: %w", err)
		}
		if manifestID = strings.TrimSpace(manifestID); manifestID != "" {
			manifestIDs = append(manifestIDs, manifestID)
		}
	}
	if len(manifestIDs) == 0 {
		return 0, nil
	}

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE Orders
			      SET ManifestId = NULL,
			          DriverId = NULL,
			          VehicleId = NULL,
			          RouteId = NULL,
			          Status = 'PENDING',
			          UpdatedAt = PENDING_COMMIT_TIMESTAMP()
			      WHERE ManifestId IN UNNEST(@manifestIds)`,
			Params: map[string]any{"manifestIds": manifestIDs},
		}); err != nil {
			return fmt.Errorf("unbind orders from orphan manifests: %w", err)
		}

		mutations := make([]*spanner.Mutation, 0, len(manifestIDs))
		for _, manifestID := range manifestIDs {
			mutations = append(mutations, spanner.Delete("SupplierTruckManifests", spanner.Key{manifestID}))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return 0, err
	}
	return len(manifestIDs), nil
}
