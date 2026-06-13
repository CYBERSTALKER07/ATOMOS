package migrations

import (
	"context"
	"fmt"
	"strings"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/option"
)

// Preflight verifies the Spanner instance and database exist before DDL migration.
// Missing emulator bootstrap must fail loudly — not be reported as "skipped".
func Preflight(ctx context.Context, opts []option.ClientOption, dbName string) error {
	instanceName, err := parentInstanceName(dbName)
	if err != nil {
		return err
	}

	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("preflight instance admin client: %w", err)
	}
	defer instanceAdmin.Close()

	if _, err := instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{Name: instanceName}); err != nil {
		if IsInfrastructureNotFound(err) {
			return fmt.Errorf("preflight: spanner instance missing (%s) — run make spanner-init or pegasusX backend-setup first", instanceName)
		}
		return fmt.Errorf("preflight get instance %s: %w", instanceName, err)
	}

	databaseAdmin, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("preflight database admin client: %w", err)
	}
	defer databaseAdmin.Close()

	if _, err := databaseAdmin.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: dbName}); err != nil {
		if IsInfrastructureNotFound(err) {
			return fmt.Errorf("preflight: spanner database missing (%s) — run make spanner-init or pegasusX backend-setup first", dbName)
		}
		return fmt.Errorf("preflight get database %s: %w", dbName, err)
	}

	return nil
}

func parentInstanceName(dbName string) (string, error) {
	const marker = "/databases/"
	idx := strings.Index(dbName, marker)
	if idx < 0 {
		return "", fmt.Errorf("invalid spanner database path %q", dbName)
	}
	return dbName[:idx], nil
}
