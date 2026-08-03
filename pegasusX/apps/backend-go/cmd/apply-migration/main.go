package main

import (
	"context"
	"flag"
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
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ddlFlag := flag.String("ddl", "", "path to incremental .ddl file (statements separated by ;)")
	verifyFlag := flag.Bool("verify", false, "after apply, verify warehouse stock migration objects")
	flag.Parse()

	ddlPath := strings.TrimSpace(*ddlFlag)
	if ddlPath == "" {
		ddlPath = defaultMigrationPath()
	}

	// Cloud Spanner DDL ops on 100 PU can exceed 5m for large index creates.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load bootstrap config", "err", err)
		os.Exit(1)
	}

	statements, err := loadDDLStatements(ddlPath)
	if err != nil {
		slog.Error("load ddl", "path", ddlPath, "err", err)
		os.Exit(1)
	}
	if len(statements) == 0 {
		slog.Error("no ddl statements found", "path", ddlPath)
		os.Exit(1)
	}

	admin, err := database.NewDatabaseAdminClient(ctx, spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("new database admin client", "err", err)
		os.Exit(1)
	}
	defer admin.Close()

	databaseName := spannerDatabasePath(cfg)
	if _, err := admin.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: databaseName}); err != nil {
		slog.Error("database not found; run cmd/setup first", "database", databaseName, "err", err)
		os.Exit(1)
	}

	slog.Info("applying spanner migration",
		"database", databaseName,
		"ddl_file", ddlPath,
		"statement_count", len(statements),
	)

	if err := applyDDLStatements(ctx, admin, databaseName, statements); err != nil {
		slog.Error("apply migration failed", "err", err)
		os.Exit(1)
	}

	slog.Info("spanner migration applied", "database", databaseName, "ddl_file", ddlPath)

	if *verifyFlag {
		if err := verifyWarehouseStockMigration(ctx, cfg); err != nil {
			slog.Error("verification failed", "err", err)
			os.Exit(1)
		}
	}
}

func defaultMigrationPath() string {
	const name = "20250616_warehouse_stock_policy_supply_items.ddl"
	candidates := []string{
		filepath.Join("schema", "migrations", name),
		filepath.Join("apps", "backend-go", "schema", "migrations", name),
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		candidates = append([]string{
			filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "schema", "migrations", name)),
		}, candidates...)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
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
	for i, statement := range statements {
		op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   databaseName,
			Statements: []string{statement},
		})
		if err != nil {
			if isBenignDDLConflict(err) {
				slog.Info("ddl statement already applied", "index", i+1, "statement", truncateStatement(statement))
				continue
			}
			return fmt.Errorf("update ddl statement %d %q: %w", i+1, truncateStatement(statement), err)
		}
		if err := op.Wait(ctx); err != nil {
			if isBenignDDLConflict(err) {
				slog.Info("ddl statement already applied", "index", i+1, "statement", truncateStatement(statement))
				continue
			}
			return fmt.Errorf("wait ddl statement %d %q: %w", i+1, truncateStatement(statement), err)
		}
		slog.Info("ddl statement applied", "index", i+1, "statement", truncateStatement(statement))
	}
	return nil
}

func verifyWarehouseStockMigration(ctx context.Context, cfg *bootstrap.Config) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	stmt := spanner.Statement{
		SQL: `SELECT TABLE_NAME, COLUMN_NAME
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_NAME IN ('Warehouses', 'SupplierInventoryV2', 'WarehouseSupplyRequests')
			  AND COLUMN_NAME IN (
			    'DefaultOutOfStockPolicy', 'OperatingSchedule',
			    'OutOfStockPolicy', 'ReorderThreshold',
			    'Priority', 'Notes', 'RegionId', 'RequestedDeliveryDate', 'TotalVolumeVU'
			  )
			ORDER BY TABLE_NAME, COLUMN_NAME`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows int
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query information_schema columns: %w", err)
		}
		var tableName, columnName string
		if err := row.Columns(&tableName, &columnName); err != nil {
			return err
		}
		slog.Info("verified column", "table", tableName, "column", columnName)
		rows++
	}

	tableIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'WarehouseSupplyRequestItems'`,
	})
	defer tableIter.Stop()
	tableRow, err := tableIter.Next()
	if err == iterator.Done {
		return fmt.Errorf("WarehouseSupplyRequestItems table missing")
	}
	if err != nil {
		return fmt.Errorf("query information_schema tables: %w", err)
	}
	var tableName string
	if err := tableRow.Columns(&tableName); err != nil {
		return err
	}
	slog.Info("verified table", "table", tableName)

	if rows < 9 {
		return fmt.Errorf("expected at least 9 migrated columns, got %d", rows)
	}
	return nil
}

func truncateStatement(statement string) string {
	const max = 120
	trimmed := strings.TrimSpace(statement)
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "..."
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

func isBenignDDLConflict(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.AlreadyExists, codes.FailedPrecondition:
		return true
	case codes.InvalidArgument:
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "duplicate") ||
			strings.Contains(msg, "already has a constraint")
	default:
		return false
	}
}
