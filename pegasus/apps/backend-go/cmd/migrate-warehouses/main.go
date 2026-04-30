// Package main — Phantom Node Migration Script
//
// Migrates the system from 1:1 (Supplier = one warehouse) to 1:N by creating
// a "Default Warehouse" for every existing Supplier using their WarehouseLat/Lng.
// Then backfills WarehouseId into Drivers, Vehicles, WarehouseStaff,
// SupplierInventory, InventoryAuditLog, Orders, and RetailerCarts.
//
// Safe to run multiple times (idempotent: checks for existing default warehouse).
//
// Usage:
//
//	SPANNER_PROJECT=... SPANNER_INSTANCE=... SPANNER_DATABASE=... go run ./cmd/migrate-warehouses/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

func main() {
	project := os.Getenv("SPANNER_PROJECT")
	instance := os.Getenv("SPANNER_INSTANCE")
	database := os.Getenv("SPANNER_DATABASE")
	if project == "" || instance == "" || database == "" {
		log.Fatal("SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE must be set")
	}

	ctx := context.Background()
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatalf("Failed to create Spanner client: %v", err)
	}
	defer client.Close()

	log.Println("[PHANTOM NODE] Starting 1:N warehouse migration...")

	// Step 1: Create default warehouses for all suppliers that don't have one yet
	supplierCount, err := createDefaultWarehouses(ctx, client)
	if err != nil {
		log.Fatalf("[PHANTOM NODE] Failed to create default warehouses: %v", err)
	}
	log.Printf("[PHANTOM NODE] Created/verified default warehouses for %d suppliers", supplierCount)

	// Step 2: Backfill WarehouseId on operational tables
	tables := []string{"Drivers", "Vehicles", "WarehouseStaff", "SupplierInventory", "InventoryAuditLog", "Orders", "RetailerCarts"}
	for _, table := range tables {
		count, err := backfillWarehouseId(ctx, client, table)
		if err != nil {
			log.Printf("[PHANTOM NODE] WARNING: Failed to backfill %s: %v", table, err)
			continue
		}
		log.Printf("[PHANTOM NODE] Backfilled %d rows in %s", count, table)
	}

	// Step 3: Create default GLOBAL_ADMIN SupplierUser for each supplier
	userCount, err := createDefaultSupplierUsers(ctx, client)
	if err != nil {
		log.Printf("[PHANTOM NODE] WARNING: Failed to create supplier users: %v", err)
	} else {
		log.Printf("[PHANTOM NODE] Created/verified GLOBAL_ADMIN users for %d suppliers", userCount)
	}

	log.Println("[PHANTOM NODE] Migration complete. System is now technically 1:N, operating as 1:1.")
}

type supplierRow struct {
	SupplierId        string
	Name              string
	WarehouseLocation spanner.NullString
	WarehouseLat      spanner.NullFloat64
	WarehouseLng      spanner.NullFloat64
	Phone             spanner.NullString
	PasswordHash      spanner.NullString
	Email             spanner.NullString
	ContactPerson     spanner.NullString
	FirebaseUid       spanner.NullString
}

func createDefaultWarehouses(ctx context.Context, client *spanner.Client) (int, error) {
	stmt := spanner.Statement{
		SQL: `SELECT s.SupplierId, s.Name, s.WarehouseLocation, s.WarehouseLat, s.WarehouseLng,
		             s.Phone, s.PasswordHash, s.Email, s.ContactPerson, s.FirebaseUid
		      FROM Suppliers s
		      WHERE NOT EXISTS (
		          SELECT 1 FROM Warehouses w WHERE w.SupplierId = s.SupplierId AND w.IsDefault = true
		      )`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var suppliers []supplierRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("query suppliers: %w", err)
		}
		var s supplierRow
		if err := row.Columns(&s.SupplierId, &s.Name, &s.WarehouseLocation, &s.WarehouseLat,
			&s.WarehouseLng, &s.Phone, &s.PasswordHash, &s.Email, &s.ContactPerson, &s.FirebaseUid); err != nil {
			return 0, fmt.Errorf("parse supplier row: %w", err)
		}
		suppliers = append(suppliers, s)
	}

	if len(suppliers) == 0 {
		return 0, nil
	}

	// Write warehouses in batches of 50

	for i := 0; i < len(suppliers); i += 50 {
		end := i + 50
		if end > len(suppliers) {
			end = len(suppliers)
		}
		batch := suppliers[i:end]

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var mutations []*spanner.Mutation
			for _, s := range batch {
				whID := uuid.New().String()
				name := s.Name + " — Default Warehouse"
				address := ""
				if s.WarehouseLocation.Valid {
					address = s.WarehouseLocation.StringVal
				}
				var lat, lng float64
				if s.WarehouseLat.Valid {
					lat = s.WarehouseLat.Float64
				}
				if s.WarehouseLng.Valid {
					lng = s.WarehouseLng.Float64
				}

				mutations = append(mutations, spanner.Insert("Warehouses",
					[]string{"WarehouseId", "SupplierId", "Name", "Address", "Lat", "Lng",
						"CoverageRadiusKm", "IsActive", "IsDefault", "IsOnShift", "CreatedAt"},
					[]interface{}{whID, s.SupplierId, name, address, lat, lng,
						50.0, true, true, true, spanner.CommitTimestamp},
				))
			}
			txn.BufferWrite(mutations)
			return nil
		})
		if err != nil {
			return i, fmt.Errorf("batch write warehouses at offset %d: %w", i, err)
		}
	}

	return len(suppliers), nil
}

func backfillWarehouseId(ctx context.Context, client *spanner.Client, tableName string) (int64, error) {
	// Determine the PK column and SupplierId column for each table
	var pkCol, supplierCol string
	switch tableName {
	case "Drivers":
		pkCol, supplierCol = "DriverId", "SupplierId"
	case "Vehicles":
		pkCol, supplierCol = "VehicleId", "SupplierId"
	case "WarehouseStaff":
		pkCol, supplierCol = "WorkerId", "SupplierId"
	case "SupplierInventory":
		pkCol, supplierCol = "ProductId", "SupplierId"
	case "InventoryAuditLog":
		pkCol, supplierCol = "AuditId", "SupplierId"
	case "Orders":
		pkCol, supplierCol = "OrderId", "SupplierId"
	case "RetailerCarts":
		pkCol, supplierCol = "CartId", "SupplierId"
	default:
		return 0, fmt.Errorf("unknown table: %s", tableName)
	}

	// Use DML to bulk-update all rows that have a SupplierId but no WarehouseId
	sql := fmt.Sprintf(
		`UPDATE %s t SET t.WarehouseId = (
			SELECT w.WarehouseId FROM Warehouses w
			WHERE w.SupplierId = t.%s AND w.IsDefault = true
			LIMIT 1
		) WHERE t.WarehouseId IS NULL AND t.%s IS NOT NULL`,
		tableName, supplierCol, supplierCol,
	)
	_ = pkCol // PK used implicitly by Spanner DML

	var totalRows int64
	// Spanner DML has a mutation limit; process in a single transaction
	// (for large datasets, this would need partitioned DML instead)
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		rowCount, updateErr := txn.Update(ctx, spanner.Statement{SQL: sql})
		if updateErr != nil {
			return updateErr
		}
		totalRows = rowCount
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalRows, nil
}

func createDefaultSupplierUsers(ctx context.Context, client *spanner.Client) (int, error) {
	// For each supplier that doesn't have a SupplierUser yet, create a GLOBAL_ADMIN user
	// using the supplier's existing credentials
	stmt := spanner.Statement{
		SQL: `SELECT s.SupplierId, s.Name, COALESCE(s.ContactPerson, s.Name),
		             s.Phone, s.Email, s.PasswordHash, s.FirebaseUid
		      FROM Suppliers s
		      WHERE NOT EXISTS (
		          SELECT 1 FROM SupplierUsers su WHERE su.SupplierId = s.SupplierId
		      )`,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type migUser struct {
		SupplierId   string
		Name         string
		ContactName  string
		Phone        spanner.NullString
		Email        spanner.NullString
		PasswordHash spanner.NullString
		FirebaseUid  spanner.NullString
	}

	var users []migUser
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("query suppliers for user migration: %w", err)
		}
		var u migUser
		if err := row.Columns(&u.SupplierId, &u.Name, &u.ContactName,
			&u.Phone, &u.Email, &u.PasswordHash, &u.FirebaseUid); err != nil {
			return 0, fmt.Errorf("parse supplier user row: %w", err)
		}
		users = append(users, u)
	}

	if len(users) == 0 {
		return 0, nil
	}

	for i := 0; i < len(users); i += 50 {
		end := i + 50
		if end > len(users) {
			end = len(users)
		}
		batch := users[i:end]

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var mutations []*spanner.Mutation
			for _, u := range batch {
				userID := uuid.New().String()
				pwHash := ""
				if u.PasswordHash.Valid {
					pwHash = u.PasswordHash.StringVal
				}
				phone := ""
				if u.Phone.Valid {
					phone = u.Phone.StringVal
				}
				email := ""
				if u.Email.Valid {
					email = u.Email.StringVal
				}

				cols := []string{"UserId", "SupplierId", "Name", "Phone", "Email",
					"PasswordHash", "SupplierRole", "IsActive", "CreatedAt"}
				vals := []interface{}{userID, u.SupplierId, u.ContactName, phone, email,
					pwHash, "GLOBAL_ADMIN", true, spanner.CommitTimestamp}

				if u.FirebaseUid.Valid {
					cols = append(cols, "FirebaseUid")
					vals = append(vals, u.FirebaseUid.StringVal)
				}

				mutations = append(mutations, spanner.Insert("SupplierUsers", cols, vals))
			}
			txn.BufferWrite(mutations)
			return nil
		})
		if err != nil {
			return i, fmt.Errorf("batch write supplier users at offset %d: %w", i, err)
		}
	}

	return len(users), nil
}

// Unused but required for compilation reference
var _ = time.Now
