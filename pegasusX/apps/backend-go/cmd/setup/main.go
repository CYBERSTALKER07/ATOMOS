package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load bootstrap config", "err", err)
		os.Exit(1)
	}

	if err := ensureSpannerSchema(ctx, cfg); err != nil {
		slog.Error("ensure spanner schema", "err", err)
		os.Exit(1)
	}

	if err := ensureSeedSupplier(ctx, cfg); err != nil {
		slog.Error("ensure seed supplier", "err", err)
		os.Exit(1)
	}

	slog.Info("ssmr sandbox bootstrap ready",
		"spanner_database", spannerDatabasePath(cfg),
		"seed_supplier_name", cfg.SeedSupplierName,
	)
}

func ensureSpannerSchema(ctx context.Context, cfg *bootstrap.Config) error {
	clientOptions := spannerClientOptions(cfg)
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, clientOptions...)
	if err != nil {
		return fmt.Errorf("new instance admin client: %w", err)
	}
	defer instanceAdmin.Close()

	databaseAdmin, err := database.NewDatabaseAdminClient(ctx, clientOptions...)
	if err != nil {
		return fmt.Errorf("new database admin client: %w", err)
	}
	defer databaseAdmin.Close()

	if err := ensureSpannerInstance(ctx, instanceAdmin, cfg); err != nil {
		return err
	}

	schemaPath, err := resolveSchemaDDLPath()
	if err != nil {
		return err
	}
	ddlStatements, err := loadDDLStatements(schemaPath)
	if err != nil {
		return fmt.Errorf("load ddl statements: %w", err)
	}

	if err := ensureSpannerDatabase(ctx, databaseAdmin, cfg, ddlStatements); err != nil {
		return err
	}

	return nil
}

func ensureSpannerInstance(ctx context.Context, admin *instance.InstanceAdminClient, cfg *bootstrap.Config) error {
	instanceName := spannerInstancePath(cfg)
	_, err := admin.GetInstance(ctx, &instancepb.GetInstanceRequest{Name: instanceName})
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return fmt.Errorf("get instance %s: %w", instanceName, err)
	}

	op, err := admin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", cfg.SpannerProject),
		InstanceId: cfg.SpannerInstance,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", cfg.SpannerProject),
			DisplayName: cfg.SpannerInstance,
			NodeCount:   1,
		},
	})
	if err != nil {
		return fmt.Errorf("create instance %s: %w", instanceName, err)
	}
	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("wait instance create %s: %w", instanceName, err)
	}

	return nil
}

func ensureSpannerDatabase(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	cfg *bootstrap.Config,
	ddlStatements []string,
) error {
	databaseName := spannerDatabasePath(cfg)
	_, err := admin.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: databaseName})
	if err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("get database %s: %w", databaseName, err)
		}
		if err := createSpannerDatabase(ctx, admin, cfg, ddlStatements); err != nil {
			return err
		}
		return nil
	}

	if err := applyDDLStatements(ctx, admin, databaseName, ddlStatements); err != nil {
		return fmt.Errorf("apply ddl statements: %w", err)
	}

	return nil
}

func createSpannerDatabase(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	cfg *bootstrap.Config,
	ddlStatements []string,
) error {
	op, err := admin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          spannerInstancePath(cfg),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", cfg.SpannerDatabase),
		ExtraStatements: ddlStatements,
	})
	if err != nil {
		return fmt.Errorf("create database %s: %w", spannerDatabasePath(cfg), err)
	}
	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("wait database create %s: %w", spannerDatabasePath(cfg), err)
	}

	return nil
}

func ensureSeedSupplier(ctx context.Context, cfg *bootstrap.Config) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	repo := &seedRepository{client: client}
	if _, err := seed.EnsureSupplier(
		ctx,
		repo,
		cfg.SeedSupplierName,
		cfg.SeedSupplierCountry,
		cfg.SeedSupplierCurrency,
		slog.Default(),
	); err != nil {
		return fmt.Errorf("seed supplier row: %w", err)
	}

	return nil
}

type seedRepository struct {
	client *spanner.Client
}

func (r *seedRepository) UpsertSupplier(ctx context.Context, s seed.Supplier) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt, err := existingSupplierCreatedAt(ctx, txn, s.SupplierID)
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

func existingSupplierCreatedAt(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID string,
) (time.Time, error) {
	row, err := txn.ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"CreatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return time.Now().UTC(), nil
		}
		return time.Time{}, fmt.Errorf("read supplier %s: %w", supplierID, err)
	}

	var createdAt time.Time
	if err := row.Columns(&createdAt); err != nil {
		return time.Time{}, fmt.Errorf("decode supplier %s created_at: %w", supplierID, err)
	}

	return createdAt, nil
}

func resolveSchemaDDLPath() (string, error) {
	candidates := []string{
		"schema/spanner.ddl",
		"apps/backend-go/schema/spanner.ddl",
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		candidates = append([]string{
			filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "schema", "spanner.ddl")),
		}, candidates...)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("spanner.ddl not found (tried %v); run from pegasusX/ or pegasusX/apps/backend-go", candidates)
}

func loadDDLStatements(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cleaned strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleaned.WriteString(line)
		cleaned.WriteString("\n")
	}

	parts := strings.Split(cleaned.String(), ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}

	return statements, nil
}

func applyDDLStatements(
	ctx context.Context,
	admin *database.DatabaseAdminClient,
	databaseName string,
	statements []string,
) error {
	for _, statement := range statements {
		op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   databaseName,
			Statements: []string{statement},
		})
		if err != nil {
			continue
		}
		if err := op.Wait(ctx); err != nil {
			return fmt.Errorf("apply statement %q: %w", statement, err)
		}
	}

	return nil
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

func spannerInstancePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("projects/%s/instances/%s", cfg.SpannerProject, cfg.SpannerInstance)
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("%s/databases/%s", spannerInstancePath(cfg), cfg.SpannerDatabase)
}

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
