// Command seed-warehouse-prodsim seeds warehouse staff login rows for local prod-simulation.
// Run after seed-supplier-prodsim (requires prodsim-wh-1 warehouse).
//
// Usage:
//   cd pegasusX/apps/backend-go && go run ./cmd/seed-warehouse-prodsim
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
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	prodsimWarehouseID   = "prodsim-wh-1"
	prodsimWarehouseUser = "prodsim-wh-admin"
	prodsimWarehousePhone = "+998901234590"
	prodsimWarehousePIN   = "4321"
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

	pinHash, err := bcrypt.GenerateFromPassword([]byte(prodsimWarehousePIN), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash pin", "err", err)
		os.Exit(1)
	}

	ts := spanner.CommitTimestamp
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
				"UserId":              prodsimWarehouseUser,
				"SupplierId":          supplier.SupplierID,
				"Name":                "ProdSim Warehouse Admin",
				"Phone":               prodsimWarehousePhone,
				"PasswordHash":        string(pinHash),
				"SupplierRole":        "WAREHOUSE_ADMIN",
				"AssignedWarehouseId": prodsimWarehouseID,
				"IsActive":            true,
				"CreatedAt":           ts,
				"UpdatedAt":           ts,
			}),
		})
	})
	if err != nil {
		slog.Error("seed warehouse staff", "err", err)
		os.Exit(1)
	}

	slog.Info("warehouse prodsim seed complete",
		"warehouse_id", prodsimWarehouseID,
		"user_id", prodsimWarehouseUser,
		"phone", prodsimWarehousePhone,
		"pin", prodsimWarehousePIN,
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
