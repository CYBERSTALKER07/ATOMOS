// Package migrations owns all in-process Spanner DDL migrations and the
// post-DDL H3 backfill. Historically these blocks lived inline in main.go
// (1359 lines, lines 986-2344). They were extracted so that:
//
//  1. Production boots can skip DDL (set MIGRATE_ON_BOOT=false) and instead
//     run cmd/migrate as a one-shot Cloud Run Job, eliminating the risk of
//     N concurrent pods racing UpdateDatabaseDdl on every cold start.
//  2. main.go can shrink toward its 200-line doctrine ceiling.
//  3. cmd/migrate (a separate binary) can re-use the exact same statements
//     so dev-on-boot and prod-out-of-band converge on one source of truth.
//
// Every block in Run() is intentionally idempotent: ALTER ... ADD COLUMN
// / CREATE INDEX / CREATE TABLE statements that have already been applied
// fail silently (errors are intentionally swallowed). This preserves the
// historical "boot until current" behaviour without requiring a migration
// version table.
package migrations

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/option"
)

// Run executes the full in-process migration sequence: schema DDL followed
// by the H3 backfill for legacy rows. All operations are idempotent.
//
// Caller controls invocation policy:
//   - main.go invokes when MIGRATE_ON_BOOT != "false" (default-on for dev).
//   - cmd/migrate invokes unconditionally as a one-shot job (production).
func Run(ctx context.Context, opts []option.ClientOption, dbName string, spannerClient *spanner.Client) {
	var err error
	_ = err

	// ── TEMPORARY MIGRATION: Complete Schema Synchronization ────────────────────
	adminClient, adminErr := database.NewDatabaseAdminClient(ctx, opts...)
	if adminErr == nil {
		columnsToDropIn := []string{
			"ALTER TABLE Orders ADD COLUMN Amount INT64",
			"ALTER TABLE Orders ADD COLUMN PaymentGateway STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN ShopLocation STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Status STRING(20)",
			"ALTER TABLE Orders ADD COLUMN RouteId STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN OrderSource STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN AutoConfirmAt TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN DeliverBefore TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN DeliveryToken STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN ShopName STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN PasswordHash STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN FcmToken STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN TelegramChatId STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Phone STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Latitude FLOAT64",
			"ALTER TABLE Retailers ADD COLUMN Longitude FLOAT64",
			"ALTER TABLE RetailerSupplierSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
			"ALTER TABLE RetailerProductSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
			"ALTER TABLE RetailerVariantSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
		}

		for _, stmt := range columnsToDropIn {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			} else {
				// Ignore errors for already existing columns
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Cart Fan-Out (Phase 1 — MasterInvoices + Orders supplier columns) ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		fanOutDDL := []string{
			`CREATE TABLE MasterInvoices (
				InvoiceId    STRING(36)  NOT NULL,
				RetailerId   STRING(36)  NOT NULL,
				Total     INT64       NOT NULL,
				State        STRING(20)  NOT NULL,
				CreatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (InvoiceId)`,
			`CREATE INDEX Idx_MasterInvoice_Retailer ON MasterInvoices(RetailerId)`,
			"ALTER TABLE Orders ADD COLUMN InvoiceId STRING(36)",
			"ALTER TABLE Orders ADD COLUMN SupplierId STRING(36)",
			`CREATE INDEX Idx_Orders_InvoiceId ON Orders(InvoiceId)`,
			`CREATE INDEX Idx_Orders_SupplierId ON Orders(SupplierId)`,
			"ALTER TABLE Products ADD COLUMN SupplierId STRING(36)",
			`CREATE INDEX Idx_Products_BySupplierId ON Products(SupplierId)`,
		}
		for _, stmt := range fanOutDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Normalize Order state CHECK constraint to the golden path ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		arrivingDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED', 'SCHEDULED'))",
			"ALTER TABLE Orders ADD COLUMN QRValidatedAt TIMESTAMP",
			"ALTER TABLE MasterInvoices ADD COLUMN OrderId STRING(36)",
			"CREATE INDEX Idx_MasterInvoice_OrderId ON MasterInvoices(OrderId)",
		}
		for _, stmt := range arrivingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Empathy Engine — Hierarchical Auto-Order Settings ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		empathyDDL := []string{
			`CREATE TABLE RetailerGlobalSettings (
				RetailerId             STRING(36) NOT NULL,
				GlobalAutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt              TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId)`,
			`CREATE TABLE RetailerSupplierSettings (
				RetailerId       STRING(36) NOT NULL,
				SupplierId       STRING(36) NOT NULL,
				AutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, SupplierId),
			  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,
			`CREATE TABLE RetailerProductSettings (
				RetailerId       STRING(36) NOT NULL,
				ProductId        STRING(36) NOT NULL,
				AutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, ProductId),
			  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,
		}
		for _, stmt := range empathyDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:50]+"...")
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Empathy Engine — Category-level settings ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		categoryDDL := []string{
			`CREATE TABLE RetailerCategorySettings (
				RetailerId         STRING(36) NOT NULL,
				CategoryId         STRING(50) NOT NULL,
				AutoOrderEnabled   BOOL       NOT NULL,
				AnalyticsStartDate TIMESTAMP,
				UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, CategoryId)`,
			`CREATE INDEX Idx_RetailerCategorySettings_ByRetailer ON RetailerCategorySettings(RetailerId)`,
		}
		for _, stmt := range categoryDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── TEMPORARY MIGRATION: The Temporal Brain (AIPredictions) ────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE AIPredictions (
					PredictionId STRING(36) NOT NULL,
					RetailerId STRING(36) NOT NULL,
					PredictedAmount INT64 NOT NULL,
					TriggerDate TIMESTAMP,
					TriggerShard INT64,
					Status STRING(32) NOT NULL,
					CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true),
				) PRIMARY KEY (PredictionId)`,
				`CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId)`,
				`CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: AIPredictions table forged.")
		} else {
			fmt.Printf("DDL migration skipped (table may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Spanner Hotspot Hardening — shard-first time access paths ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		hotspotDDL := []string{
			"ALTER TABLE Orders ADD COLUMN RequestedDeliveryDate TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN ScheduleShard INT64",
			`CREATE INDEX Idx_Orders_ByScheduleShardStateDate ON Orders(ScheduleShard, State, RequestedDeliveryDate DESC)`,
			"DROP INDEX IDX_Orders_Scheduled",
			"ALTER TABLE AIPredictions ADD COLUMN TriggerShard INT64",
			"ALTER TABLE AIPredictions ADD COLUMN CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)",
			`CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId)`,
			`CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC)`,
			"DROP INDEX Idx_AIPredictions_ByStatus",
			`CREATE TABLE AIPredictionItems (
				PredictionId      STRING(36) NOT NULL,
				PredictionItemId  STRING(36) NOT NULL,
				SkuId             STRING(50) NOT NULL,
				PredictedQuantity INT64      NOT NULL,
				UnitPrice      INT64      NOT NULL,
				CreatedAt         TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (PredictionId, PredictionItemId),
			  INTERLEAVE IN PARENT AIPredictions ON DELETE CASCADE`,
			`CREATE INDEX Idx_PredictionItems_BySku ON AIPredictionItems(SkuId)`,
		}
		for _, stmt := range hotspotDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Inventory Ledger — SupplierInventory table ──────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE SupplierInventory (
					ProductId          STRING(36) NOT NULL,
					SupplierId         STRING(36) NOT NULL,
					QuantityAvailable  INT64      NOT NULL,
					UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (ProductId)`,
				`CREATE INDEX Idx_Inventory_BySupplier ON SupplierInventory(SupplierId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: SupplierInventory table forged.")
		} else {
			fmt.Printf("DDL migration skipped (SupplierInventory may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Inventory Audit Log — InventoryAuditLog table ──────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE InventoryAuditLog (
					AuditId      STRING(36)  NOT NULL,
					ProductId    STRING(36)  NOT NULL,
					SupplierId   STRING(36)  NOT NULL,
					AdjustedBy   STRING(36)  NOT NULL,
					PreviousQty  INT64       NOT NULL,
					NewQty       INT64       NOT NULL,
					Delta        INT64       NOT NULL,
					Reason       STRING(50)  NOT NULL,
					AdjustedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (AuditId)`,
				`CREATE INDEX Idx_AuditLog_BySupplier ON InventoryAuditLog(SupplierId)`,
				`CREATE INDEX Idx_AuditLog_ByProduct  ON InventoryAuditLog(ProductId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: InventoryAuditLog table forged.")
		} else {
			fmt.Printf("DDL migration skipped (InventoryAuditLog may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: SupplierReturns — Partial-Qty Reconciliation (Phase 9) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE SupplierReturns (
					ReturnId     STRING(36)  NOT NULL,
					OrderId      STRING(36)  NOT NULL,
					SkuId        STRING(50)  NOT NULL,
					RejectedQty  INT64       NOT NULL,
					Reason       STRING(50)  NOT NULL,
					DriverNotes  STRING(MAX),
					CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (ReturnId)`,
				`CREATE INDEX Idx_Returns_ByOrder ON SupplierReturns(OrderId)`,
				`CREATE INDEX Idx_Returns_BySku   ON SupplierReturns(SkuId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: SupplierReturns table forged.")
		} else {
			fmt.Printf("DDL migration skipped (SupplierReturns may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Drivers Fleet Provisioning — Phone, PIN, Supplier columns (Phase 10) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		fleetDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN Phone STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN PinHash STRING(MAX)",
			"ALTER TABLE Drivers ADD COLUMN SupplierId STRING(36)",
			"ALTER TABLE Drivers ADD COLUMN DriverType STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN VehicleType STRING(50)",
			"ALTER TABLE Drivers ADD COLUMN LicensePlate STRING(30)",
			"ALTER TABLE Drivers ADD COLUMN IsActive BOOL",
			`CREATE INDEX Idx_Drivers_BySupplierId ON Drivers(SupplierId)`,
			`CREATE INDEX Idx_Drivers_ByPhone ON Drivers(Phone)`,
		}
		for _, stmt := range fleetDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Vehicles Table + Drivers extra columns (Phase 10b) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		vehicleDDL := []string{
			`CREATE TABLE Vehicles (
				VehicleId    STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				VehicleClass STRING(10)  NOT NULL,
				Label        STRING(100),
				LicensePlate STRING(30),
				MaxVolumeVU  FLOAT64     NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				UnavailableReason STRING(64),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (VehicleId)`,
			`CREATE INDEX Idx_Vehicles_BySupplier ON Vehicles(SupplierId)`,
			"ALTER TABLE Drivers ADD COLUMN VehicleId STRING(36)",
			"ALTER TABLE Drivers ADD COLUMN TruckStatus STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN DepartedAt TIMESTAMP",
			"ALTER TABLE Drivers ADD COLUMN MaxPalletCapacity INT64",
		}
		for _, stmt := range vehicleDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Payment Settlement — GlobalPayTransactionId on MasterInvoices ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`ALTER TABLE MasterInvoices ADD COLUMN GlobalPayTransactionId STRING(64)`,
				`CREATE INDEX Idx_MasterInvoice_GlobalPayTxn ON MasterInvoices(GlobalPayTransactionId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: GlobalPayTransactionId column added to MasterInvoices.")
		} else {
			fmt.Printf("DDL migration skipped (GlobalPayTransactionId may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Optimistic Concurrency Control & Freeze Locks (Phase 12) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		occDDL := []string{
			"ALTER TABLE Orders ADD COLUMN Version INT64 NOT NULL DEFAULT (1)",
			"ALTER TABLE Orders ADD COLUMN LockedUntil TIMESTAMP",
		}
		for _, stmt := range occDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Registration Pipeline — TaxId, IsConfigured, OperatingCategories, PlatformCategories ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		supplierRegDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN TaxId STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN IsConfigured BOOL",
			"ALTER TABLE Suppliers ADD COLUMN OperatingCategories ARRAY<STRING(MAX)>",
			`CREATE TABLE PlatformCategories (
				CategoryId    STRING(36)  NOT NULL,
				DisplayName   STRING(MAX) NOT NULL,
				IconUrl       STRING(MAX),
				DisplayOrder  INT64       NOT NULL DEFAULT (0)
			) PRIMARY KEY (CategoryId)`,
			`CREATE TABLE Categories (
				CategoryId   STRING(36)  NOT NULL,
				Name         STRING(255) NOT NULL,
				Icon         STRING(100),
				SortOrder    INT64       NOT NULL DEFAULT (0),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (CategoryId)`,
		}
		for _, stmt := range supplierRegDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Extended Profile Columns (Email, Bank, Payment) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		supplierProfileDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN Email STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN ContactPerson STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN CompanyRegNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN BillingAddress STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN BankName STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN AccountNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN CardNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN PaymentGateway STRING(20)",
		}
		for _, stmt := range supplierProfileDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Admins Table ────────────────────────────────────
	{
		adminClient, adminErr := database.NewDatabaseAdminClient(ctx)
		if adminErr == nil {
			adminsDDL := []string{
				`CREATE TABLE Admins (
					AdminId       STRING(36)  NOT NULL,
					Email         STRING(MAX) NOT NULL,
					PasswordHash  STRING(MAX) NOT NULL,
					DisplayName   STRING(MAX),
					CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (AdminId)`,
				`CREATE UNIQUE INDEX Idx_Admins_ByEmail ON Admins(Email)`,
			}
			for _, stmt := range adminsDDL {
				op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
					Database:   dbName,
					Statements: []string{stmt},
				})
				if ddlErr == nil {
					op.Wait(ctx)
					preview := stmt
					if len(preview) > 60 {
						preview = preview[:60] + "..."
					}
					fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
				}
			}
			adminClient.Close()
		}
	}

	// NOTE: auth.SeedDefaultAdmin was previously interleaved here. It is not
	// a schema migration — moved back to main.go (post-Run) where it belongs.

	// ── MIGRATION: Truck State Machine — TruckStatus column on Drivers (Fleet Availability) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		truckStatusDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN TruckStatus STRING(20) DEFAULT ('AVAILABLE')",
			`CREATE INDEX Idx_Drivers_ByTruckStatus ON Drivers(TruckStatus)`,
			"ALTER TABLE Drivers ADD COLUMN DepartedAt TIMESTAMP",
		}
		for _, stmt := range truckStatusDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: SupplierProducts CategoryId + CategoryName, PricingTiers, Warehouse columns ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		catalogPricingDDL := []string{
			"ALTER TABLE SupplierProducts ADD COLUMN CategoryId STRING(36)",
			"ALTER TABLE SupplierProducts ADD COLUMN CategoryName STRING(MAX)",
			"ALTER TABLE SupplierProducts ADD COLUMN PalletFootprint FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN VolumetricUnit FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLocation STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLat FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLng FLOAT64",
			`CREATE TABLE PricingTiers (
				TierId              STRING(36)  NOT NULL,
				SupplierId          STRING(36)  NOT NULL,
				SkuId               STRING(50)  NOT NULL,
				MinPallets          INT64       NOT NULL,
				DiscountPct         INT64       NOT NULL,
				TargetRetailerTier  STRING(20)  NOT NULL,
				ValidUntil          TIMESTAMP,
				IsActive            BOOL        NOT NULL
			) PRIMARY KEY (TierId)`,
			`CREATE INDEX Idx_PricingTiers_BySupplierId ON PricingTiers(SupplierId)`,
		}
		for _, stmt := range catalogPricingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Warehouse Staff (Payloader) Provisioning ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		warehouseStaffDDL := []string{
			`CREATE TABLE WarehouseStaff (
				WorkerId    STRING(36)  NOT NULL,
				SupplierId  STRING(36)  NOT NULL,
				Name        STRING(MAX) NOT NULL,
				Phone       STRING(20)  NOT NULL,
				PinHash     STRING(MAX) NOT NULL,
				IsActive    BOOL        NOT NULL,
				CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (WorkerId)`,
			`CREATE INDEX Idx_WarehouseStaff_BySupplierId ON WarehouseStaff(SupplierId)`,
			`CREATE INDEX Idx_WarehouseStaff_ByPhone ON WarehouseStaff(Phone)`,
		}
		for _, stmt := range warehouseStaffDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Dimensional VU Engine + Registration Expansion ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		dimensionalDDL := []string{
			"ALTER TABLE Vehicles ADD COLUMN LengthCM FLOAT64",
			"ALTER TABLE Vehicles ADD COLUMN WidthCM FLOAT64",
			"ALTER TABLE Vehicles ADD COLUMN HeightCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN LengthCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN WidthCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN HeightCM FLOAT64",
			"ALTER TABLE Retailers ADD COLUMN ReceivingWindowOpen STRING(10)",
			"ALTER TABLE Retailers ADD COLUMN ReceivingWindowClose STRING(10)",
			"ALTER TABLE Retailers ADD COLUMN AccessType STRING(30)",
			"ALTER TABLE Retailers ADD COLUMN StorageCeilingHeightCM FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN FleetColdChainCompliant BOOL",
			"ALTER TABLE Suppliers ADD COLUMN PalletizationStandard STRING(30)",
		}
		for _, stmt := range dimensionalDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Concurrency Crash — BACKORDERED state for partial-fill checkout ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		backorderDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'BACKORDERED'))",
		}
		for _, stmt := range backorderDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phantom Cargo — RejectedQty + ReturnClearedAt on OrderLineItems ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phantooCargoDDL := []string{
			"ALTER TABLE OrderLineItems ADD COLUMN RejectedQty INT64 NOT NULL DEFAULT (0)",
			"ALTER TABLE OrderLineItems ADD COLUMN ReturnClearedAt TIMESTAMP",
		}
		for _, stmt := range phantooCargoDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: UOM Collision — MinimumOrderQty + StepSize on SupplierProducts ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		uomDDL := []string{
			"ALTER TABLE SupplierProducts ADD COLUMN MinimumOrderQty INT64 NOT NULL DEFAULT (1)",
			"ALTER TABLE SupplierProducts ADD COLUMN StepSize INT64 NOT NULL DEFAULT (1)",
		}
		for _, stmt := range uomDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Shift State — OperatingSchedule + ManualOffShift ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		shiftDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN OperatingSchedule JSON",
			"ALTER TABLE Suppliers ADD COLUMN ManualOffShift BOOL NOT NULL DEFAULT (false)",
		}
		for _, stmt := range shiftDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Cash Logistics — PENDING_CASH_COLLECTION state + MasterInvoices cash-custody columns ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		cashLogisticsDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'BACKORDERED', 'QUARANTINE'))",
			"ALTER TABLE MasterInvoices ADD COLUMN PaymentMode STRING(20)",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectorDriverId STRING(36)",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectedAt TIMESTAMP",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectionLat FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectionLng FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN GeofenceDistanceM FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN CustodyStatus STRING(20)",
		}
		for _, stmt := range cashLogisticsDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Multi-vendor Payment — PaymentStatus column + SupplierPaymentConfigs table ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		multiVendorPaymentDDL := []string{
			"ALTER TABLE Orders ADD COLUMN PaymentStatus STRING(30) NOT NULL DEFAULT ('PENDING')",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_PaymentStatus CHECK (PaymentStatus IN ('PENDING', 'PENDING_CASH_COLLECTION', 'AWAITING_GATEWAY_WEBHOOK', 'PAID', 'FAILED'))",
			`CREATE TABLE SupplierPaymentConfigs (
				ConfigId     STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				WarehouseId  STRING(36),
				GatewayName  STRING(20)  NOT NULL,
				MerchantId   STRING(MAX) NOT NULL,
				ServiceId    STRING(MAX),
				SecretKey    BYTES(MAX)  NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'ADYEN', 'AIRWALLEX'))
			) PRIMARY KEY (ConfigId)`,
			"CREATE INDEX Idx_SupplierPaymentConfigs_BySupplierId ON SupplierPaymentConfigs(SupplierId)",
			"CREATE INDEX Idx_SupplierPaymentConfigs_ByWarehouseId ON SupplierPaymentConfigs(WarehouseId)",
			"CREATE UNIQUE INDEX Idx_SupplierPaymentConfigs_Unique ON SupplierPaymentConfigs(SupplierId, WarehouseId, GatewayName)",
			`CREATE TABLE SupplierBillingTiers (
				TierId       STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				MinVolume    INT64       NOT NULL,
				MaxVolume    INT64,
				FeeBasisPts  INT64       NOT NULL,
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (TierId)`,
			// Phase 2 addendum: ServiceId for Cash gateway
			"ALTER TABLE SupplierPaymentConfigs ADD COLUMN ServiceId STRING(MAX)",
			"ALTER TABLE SupplierPaymentConfigs DROP CONSTRAINT CHK_GatewayName",
			"ALTER TABLE SupplierPaymentConfigs ADD CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'ADYEN', 'AIRWALLEX'))",
		}
		for _, stmt := range multiVendorPaymentDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Payment Sessions + Attempts tables (Phase 13) ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		paymentSessionDDL := []string{
			`CREATE TABLE PaymentSessions (
				SessionId         STRING(36)  NOT NULL,
				OrderId           STRING(36)  NOT NULL,
				RetailerId        STRING(36)  NOT NULL,
				SupplierId        STRING(36)  NOT NULL,
				Gateway           STRING(20)  NOT NULL,
				LockedAmount   INT64       NOT NULL,
				Currency          STRING(3)   NOT NULL DEFAULT ('UZS'),
				Status            STRING(30)  NOT NULL DEFAULT ('CREATED'),
				CurrentAttemptNo  INT64       NOT NULL DEFAULT (0),
				InvoiceId         STRING(36),
				RedirectUrl       STRING(MAX),
				ProviderReference STRING(MAX),
				ExpiresAt         TIMESTAMP,
				LastErrorCode     STRING(50),
				LastErrorMessage  STRING(MAX),
				CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt         TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				SettledAt         TIMESTAMP,
				CONSTRAINT CHK_SessionStatus CHECK (Status IN ('CREATED', 'PENDING', 'SETTLED', 'FAILED', 'EXPIRED', 'CANCELLED'))
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_PaymentSessions_ByOrderId ON PaymentSessions(OrderId)",
			"CREATE INDEX Idx_PaymentSessions_BySupplierId ON PaymentSessions(SupplierId)",
			"CREATE INDEX Idx_PaymentSessions_ByStatus ON PaymentSessions(Status)",
			"ALTER TABLE PaymentSessions ADD COLUMN ProviderReference STRING(MAX)",
			`CREATE TABLE PaymentAttempts (
				AttemptId             STRING(36)  NOT NULL,
				SessionId             STRING(36)  NOT NULL,
				AttemptNo             INT64       NOT NULL,
				Gateway               STRING(20)  NOT NULL,
				ProviderTransactionId STRING(64),
				Status                STRING(30)  NOT NULL DEFAULT ('INITIATED'),
				FailureCode           STRING(50),
				FailureMessage        STRING(MAX),
				RequestDigest         STRING(MAX),
				StartedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				FinishedAt            TIMESTAMP,
				CONSTRAINT CHK_AttemptStatus CHECK (Status IN ('INITIATED', 'REDIRECTED', 'PROCESSING', 'SUCCESS', 'FAILED', 'CANCELLED', 'TIMED_OUT'))
			) PRIMARY KEY (AttemptId)`,
			"CREATE INDEX Idx_PaymentAttempts_BySessionId ON PaymentAttempts(SessionId)",
			"CREATE INDEX Idx_PaymentAttempts_ByProviderTxn ON PaymentAttempts(ProviderTransactionId)",
		}
		for _, stmt := range paymentSessionDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Gateway Onboarding Sessions (Supplier Connect) ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		onboardingDDL := []string{
			`CREATE TABLE GatewayOnboardingSessions (
				SessionId      STRING(36)  NOT NULL,
				SupplierId     STRING(36)  NOT NULL,
				Gateway        STRING(20)  NOT NULL,
				Status         STRING(30)  NOT NULL DEFAULT ('CREATED'),
				StateNonce     STRING(128),
				ReturnSurface  STRING(10)  NOT NULL DEFAULT ('web'),
				RedirectUrl    STRING(MAX),
				ErrorMessage   STRING(MAX),
				ExpiresAt      TIMESTAMP   NOT NULL,
				CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt      TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_OnboardStatus CHECK (Status IN ('CREATED', 'PENDING', 'COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED'))
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_GatewayOnboarding_BySupplierId ON GatewayOnboardingSessions(SupplierId)",
			"CREATE INDEX Idx_GatewayOnboarding_ByStatus ON GatewayOnboardingSessions(Status)",
		}
		for _, stmt := range onboardingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Firebase Auth Identity Linking ─────────────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		firebaseUidDDL := []string{
			"ALTER TABLE Admins ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Suppliers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Retailers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Drivers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE WarehouseStaff ADD COLUMN FirebaseUid STRING(128)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Admins_ByFirebaseUid ON Admins(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Suppliers_ByFirebaseUid ON Suppliers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Retailers_ByFirebaseUid ON Retailers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Drivers_ByFirebaseUid ON Drivers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_WarehouseStaff_ByFirebaseUid ON WarehouseStaff(FirebaseUid)",
		}
		for _, stmt := range firebaseUidDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Driver Availability Session Management ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		driverSessionDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN OfflineReason STRING(30)",
			"ALTER TABLE Drivers ADD COLUMN OfflineReasonNote STRING(500)",
			"ALTER TABLE Drivers ADD COLUMN OfflineAt TIMESTAMP",
			"CREATE INDEX Idx_Drivers_ByActiveStatus ON Drivers(SupplierId, IsActive)",
		}
		for _, stmt := range driverSessionDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Geo-Spatial Sovereignty — H3Index on Retailers, Factories & Orders ────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		geoH3DDL := []string{
			"ALTER TABLE Retailers ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Retailers_ByH3Index ON Retailers(H3Index)",
			"ALTER TABLE Factories ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Factories_ByH3Index ON Factories(H3Index)",
			"ALTER TABLE Orders ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Orders_ByH3Index ON Orders(H3Index)",
		}
		for _, stmt := range geoH3DDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier factory planning metadata — ProductTypes ───────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		factoryMetadataDDL := []string{
			"ALTER TABLE Factories ADD COLUMN ProductTypes ARRAY<STRING(MAX)>",
		}
		for _, stmt := range factoryMetadataDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── BACKFILL: H3Index for rows created before the H3 migration ────────────
	// Runs once per boot, a no-op after the first successful pass (all rows
	// already have H3Index populated). Uses h3-go/v4 at resolution 7 to emit
	// 15-char lowercase hex cell IDs compatible with h3-js on the frontend.
	backfillH3Indexes(ctx, spannerClient)

	// ── MIGRATION: Address-as-Label — AddressVerified flag ──────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		addrDDL := []string{
			"ALTER TABLE Retailers ADD COLUMN AddressVerified BOOL",
			"ALTER TABLE Warehouses ADD COLUMN AddressVerified BOOL",
		}
		for _, stmt := range addrDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Warehouse Load Balancing — MaxCapacityThreshold + composite order index ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		loadBalanceDDL := []string{
			"ALTER TABLE Warehouses ADD COLUMN MaxCapacityThreshold INT64",
			"CREATE INDEX Idx_Orders_ByWarehouseStateCreated ON Orders(WarehouseId, State, CreatedAt DESC)",
		}
		for _, stmt := range loadBalanceDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase E — Warehouses, SupplierUsers, Factories (CREATE TABLE) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseEDDL := []string{
			// ── Warehouses ──
			`CREATE TABLE Warehouses (
				WarehouseId      STRING(36)  NOT NULL,
				SupplierId       STRING(36)  NOT NULL,
				Name             STRING(255) NOT NULL,
				Address          STRING(MAX),
				Lat              FLOAT64,
				Lng              FLOAT64,
				H3Indexes        ARRAY<STRING(MAX)>,
				CoverageRadiusKm FLOAT64     NOT NULL DEFAULT (50.0),
				IsActive         BOOL        NOT NULL DEFAULT (true),
				IsDefault        BOOL        NOT NULL DEFAULT (false),
				IsOnShift        BOOL        NOT NULL DEFAULT (true),
				CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (WarehouseId)`,
			`CREATE INDEX Idx_Warehouses_BySupplierId ON Warehouses(SupplierId)`,

			// ── SupplierUsers (RBAC) ──
			`CREATE TABLE SupplierUsers (
				UserId               STRING(36)  NOT NULL,
				SupplierId           STRING(36)  NOT NULL,
				Email                STRING(MAX),
				Phone                STRING(20),
				Name                 STRING(MAX) NOT NULL,
				PasswordHash         STRING(MAX) NOT NULL,
				SupplierRole         STRING(30)  NOT NULL,
				AssignedWarehouseId  STRING(36),
				AssignedFactoryId    STRING(36),
				IsActive             BOOL        NOT NULL DEFAULT (true),
				FirebaseUid          STRING(128),
				CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_SupplierRole CHECK (SupplierRole IN ('GLOBAL_ADMIN', 'NODE_ADMIN', 'FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
			) PRIMARY KEY (UserId)`,
			`CREATE INDEX Idx_SupplierUsers_BySupplierId ON SupplierUsers(SupplierId)`,
			`CREATE INDEX Idx_SupplierUsers_ByPhone ON SupplierUsers(Phone)`,
			`CREATE UNIQUE NULL_FILTERED INDEX Idx_SupplierUsers_ByFirebaseUid ON SupplierUsers(FirebaseUid)`,

			// ── Factories ──
			`CREATE TABLE Factories (
				FactoryId            STRING(36)  NOT NULL,
				SupplierId           STRING(36)  NOT NULL,
				Name                 STRING(255) NOT NULL,
				Address              STRING(MAX),
				Lat                  FLOAT64,
				Lng                  FLOAT64,
				H3Index              STRING(MAX),
				RegionCode           STRING(20),
				LeadTimeDays         INT64       NOT NULL DEFAULT (2),
				ProductionCapacityVU FLOAT64     NOT NULL DEFAULT (0),
				ProductTypes         ARRAY<STRING(MAX)>,
				IsActive             BOOL        NOT NULL DEFAULT (true),
				CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt            TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (FactoryId)`,
			`CREATE INDEX Idx_Factories_BySupplierId ON Factories(SupplierId)`,

			// ── FactoryStaff ──
			`CREATE TABLE FactoryStaff (
				StaffId      STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				Name         STRING(MAX) NOT NULL,
				Phone        STRING(20),
				PasswordHash STRING(MAX) NOT NULL,
				StaffRole    STRING(30)  NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				FirebaseUid  STRING(128),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_FactoryStaffRole CHECK (StaffRole IN ('FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
			) PRIMARY KEY (StaffId)`,
			`CREATE INDEX Idx_FactoryStaff_ByFactoryId ON FactoryStaff(FactoryId)`,
			`CREATE INDEX Idx_FactoryStaff_ByPhone ON FactoryStaff(Phone)`,
			`CREATE UNIQUE NULL_FILTERED INDEX Idx_FactoryStaff_ByFirebaseUid ON FactoryStaff(FirebaseUid)`,

			// ── InternalTransferOrders ──
			`CREATE TABLE InternalTransferOrders (
				TransferId   STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				WarehouseId  STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				State        STRING(20)  NOT NULL DEFAULT ('DRAFT'),
				TotalVolumeVU FLOAT64    NOT NULL DEFAULT (0),
				ManifestId   STRING(36),
				Source       STRING(30)  NOT NULL DEFAULT ('MANUAL_EMERGENCY'),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_TransferState CHECK (State IN ('DRAFT', 'APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVED', 'RECEIVED', 'CANCELLED')),
				CONSTRAINT CHK_TransferSource CHECK (Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED', 'MANUAL_EMERGENCY'))
			) PRIMARY KEY (TransferId)`,
			`CREATE INDEX Idx_Transfers_ByFactoryId ON InternalTransferOrders(FactoryId)`,
			`CREATE INDEX Idx_Transfers_ByWarehouseId ON InternalTransferOrders(WarehouseId)`,
			`CREATE INDEX Idx_Transfers_BySupplierId ON InternalTransferOrders(SupplierId)`,
			`CREATE INDEX Idx_Transfers_ByState ON InternalTransferOrders(State)`,

			// ── InternalTransferItems (interleaved) ──
			`CREATE TABLE InternalTransferItems (
				TransferId STRING(36) NOT NULL,
				ItemId     STRING(36) NOT NULL,
				ProductId  STRING(36) NOT NULL,
				Quantity   INT64      NOT NULL,
				VolumeVU   FLOAT64    NOT NULL DEFAULT (0)
			) PRIMARY KEY (TransferId, ItemId),
			  INTERLEAVE IN PARENT InternalTransferOrders ON DELETE CASCADE`,

			// ── FactoryTruckManifests ──
			`CREATE TABLE FactoryTruckManifests (
				ManifestId   STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				DriverId     STRING(36),
				VehicleId    STRING(36),
				State        STRING(20)  NOT NULL DEFAULT ('PENDING'),
				TotalVolumeVU FLOAT64    NOT NULL DEFAULT (0),
				MaxVolumeVU  FLOAT64     NOT NULL DEFAULT (0),
				StopCount    INT64       NOT NULL DEFAULT (0),
				RegionCode   STRING(20),
				RoutePath    STRING(MAX),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_ManifestState CHECK (State IN ('PENDING', 'READY_FOR_LOADING', 'LOADING', 'DISPATCHED', 'COMPLETED'))
			) PRIMARY KEY (ManifestId)`,
			`CREATE INDEX Idx_FactoryManifests_ByFactoryId ON FactoryTruckManifests(FactoryId)`,
			`CREATE INDEX Idx_FactoryManifests_ByState ON FactoryTruckManifests(State)`,

			// ── ReplenishmentInsights ──
			`CREATE TABLE ReplenishmentInsights (
				InsightId        STRING(36)  NOT NULL,
				WarehouseId      STRING(36)  NOT NULL,
				ProductId        STRING(36)  NOT NULL,
				SupplierId       STRING(36)  NOT NULL,
				CurrentStock     INT64       NOT NULL DEFAULT (0),
				DailyBurnRate    FLOAT64     NOT NULL DEFAULT (0),
				TimeToEmptyDays  FLOAT64     NOT NULL DEFAULT (0),
				SuggestedQuantity INT64      NOT NULL DEFAULT (0),
				UrgencyLevel     STRING(20)  NOT NULL DEFAULT ('STABLE'),
				ReasonCode       STRING(30)  NOT NULL DEFAULT ('LOW_STOCK'),
				Status           STRING(20)  NOT NULL DEFAULT ('PENDING'),
				TargetFactoryId  STRING(36),
				DemandBreakdown  STRING(MAX),
				CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_InsightUrgency CHECK (UrgencyLevel IN ('CRITICAL', 'WARNING', 'STABLE')),
				CONSTRAINT CHK_InsightReason CHECK (ReasonCode IN ('HIGH_VELOCITY', 'LOW_STOCK', 'PREDICTED_SPIKE')),
				CONSTRAINT CHK_InsightStatus CHECK (Status IN ('PENDING', 'APPROVED', 'DISMISSED'))
			) PRIMARY KEY (InsightId)`,
			`CREATE INDEX Idx_Insights_ByWarehouse ON ReplenishmentInsights(WarehouseId)`,
			`CREATE INDEX Idx_Insights_BySupplierId ON ReplenishmentInsights(SupplierId)`,
			`CREATE INDEX Idx_Insights_ByStatus ON ReplenishmentInsights(Status)`,

			// ── Warehouse linkage to operational tables ──
			"ALTER TABLE Drivers ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE Vehicles ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE WarehouseStaff ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE SupplierInventory ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE InventoryAuditLog ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE Orders ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE RetailerCarts ADD COLUMN WarehouseId STRING(36)",
			"CREATE INDEX Idx_Drivers_ByWarehouseId ON Drivers(WarehouseId)",
			"CREATE INDEX Idx_Vehicles_ByWarehouseId ON Vehicles(WarehouseId)",
			"CREATE INDEX Idx_WarehouseStaff_ByWarehouseId ON WarehouseStaff(WarehouseId)",
			"CREATE INDEX Idx_Inventory_ByWarehouseId ON SupplierInventory(SupplierId, WarehouseId)",
			"CREATE INDEX Idx_Orders_ByWarehouseId ON Orders(WarehouseId)",
			"ALTER TABLE AIPredictions ADD COLUMN WarehouseId STRING(36)",
			"CREATE INDEX Idx_AIPredictions_ByWarehouse ON AIPredictions(WarehouseId)",

			// ── Warehouse-Factory linkage ──
			"ALTER TABLE Warehouses ADD COLUMN PrimaryFactoryId STRING(36)",
			"ALTER TABLE Warehouses ADD COLUMN SecondaryFactoryId STRING(36)",
		}
		for _, stmt := range phaseEDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase IV — Pre-order policy columns + Order state expansion ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseIVDDL := []string{
			"ALTER TABLE Orders ADD COLUMN CancelLockedAt TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN CancelLockReason STRING(30)",
			"ALTER TABLE Orders ADD COLUMN ConfirmationNotifiedAt TIMESTAMP",
			`CREATE INDEX Idx_Orders_PreOrderLockPending
				ON Orders(State, RequestedDeliveryDate)
				WHERE CancelLockedAt IS NULL AND ConfirmationNotifiedAt IS NULL
				  AND State IN ('SCHEDULED', 'PENDING_REVIEW')`,
		}
		for _, stmt := range phaseIVDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase V — Temporal Traceability & Notification Correlation ───
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVDDL := []string{
			// 1. Orders.ReplenishmentId — forward-link from replenishment transfer to fulfilled orders
			"ALTER TABLE Orders ADD COLUMN ReplenishmentId STRING(36)",
			`CREATE INDEX Idx_Orders_ByReplenishmentId ON Orders(ReplenishmentId) WHERE ReplenishmentId IS NOT NULL`,

			// 2. Expand Order state machine — add PENDING_CONFIRMATION, LOCKED, AUTO_ACCEPTED,
			//    NO_CAPACITY, STALE_AUDIT (last two already used in application code but unconstrained)
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			`ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN (
				'PENDING', 'PENDING_REVIEW', 'PENDING_CONFIRMATION',
				'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED',
				'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION',
				'COMPLETED', 'CANCELLED',
				'SCHEDULED', 'BACKORDERED', 'QUARANTINE',
				'LOCKED', 'AUTO_ACCEPTED',
				'NO_CAPACITY', 'STALE_AUDIT'
			))`,

			// 3. Notifications.ExpiresAt — soft-expiry for stale alerts
			"ALTER TABLE Notifications ADD COLUMN ExpiresAt TIMESTAMP",

			// 4. Notifications.CorrelationId — links notification to triggering entity (e.g. ord_confirm_{OrderId})
			"ALTER TABLE Notifications ADD COLUMN CorrelationId STRING(36)",
			`CREATE INDEX Idx_Notifications_ByCorrelationId ON Notifications(CorrelationId) WHERE CorrelationId IS NOT NULL`,
			`CREATE INDEX Idx_Notifications_ByExpiresAt ON Notifications(ExpiresAt) WHERE ExpiresAt IS NOT NULL AND ReadAt IS NULL`,
		}
		for _, stmt := range phaseVDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase V.5 — Look-Ahead Source Type ───────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		lookAheadDDL := []string{
			"ALTER TABLE InternalTransferOrders DROP CONSTRAINT CHK_TransferSource",
			`ALTER TABLE InternalTransferOrders ADD CONSTRAINT CHK_TransferSource
				CHECK (Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED', 'MANUAL_EMERGENCY', 'WAREHOUSE_REQUEST', 'SYSTEM_LOOKAHEAD'))`,
		}
		for _, stmt := range lookAheadDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase VI — Fleet Offline State ────────────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVIDDL := []string{
			// 1. Drivers.IsOffline — distinguishes "temporarily offline" (app backgrounded, phone dead)
			//    from "account deactivated" (IsActive=false). Dispatch queries check both.
			"ALTER TABLE Drivers ADD COLUMN IsOffline BOOL DEFAULT (false)",
			`CREATE INDEX Idx_Drivers_ByOffline ON Drivers(SupplierId, IsOffline) WHERE IsOffline = true`,

			// 2. Orders.NudgeNotifiedAt — dedup marker for T-5 soft reminder
			"ALTER TABLE Orders ADD COLUMN NudgeNotifiedAt TIMESTAMP",
		}
		for _, stmt := range phaseVIDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase VII — V.O.I.D. Home Node Lifecycle ─────────────────────
	// Drivers/Vehicles carry a canonical (HomeNodeType, HomeNodeId) tuple so a
	// resource can be home-based at a Warehouse OR a Factory. WarehouseId stays
	// denormalised during the migration window; new writes dual-populate.
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVIIDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN HomeNodeType STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN HomeNodeId STRING(36)",
			"ALTER TABLE Vehicles ADD COLUMN HomeNodeType STRING(20)",
			"ALTER TABLE Vehicles ADD COLUMN HomeNodeId STRING(36)",
			"ALTER TABLE Vehicles ADD COLUMN UnavailableReason STRING(64)",
			"CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId)",
			"CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId)",
			// Transactional Outbox — single mechanism for durable state-change
			// Kafka events. Entity mutations dual-write to OutboxEvents inside
			// the same ReadWriteTransaction; the relay tails this table and
			// publishes. See backend-go/outbox/.
			`CREATE TABLE OutboxEvents (
				EventId       STRING(36)  NOT NULL,
				AggregateType STRING(30)  NOT NULL,
				AggregateId   STRING(36)  NOT NULL,
				TopicName     STRING(100) NOT NULL,
				Payload       BYTES(MAX)  NOT NULL,
				CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				PublishedAt   TIMESTAMP,
			) PRIMARY KEY (EventId)`,
			`CREATE INDEX Idx_OutboxEvents_Unpublished ON OutboxEvents(PublishedAt, CreatedAt)`,
		}
		for _, stmt := range phaseVIIDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}

		// ── Glass Box: OutboxEvents.TraceID for end-to-end trace propagation ──
		glassBoxDDL := []string{
			"ALTER TABLE OutboxEvents ADD COLUMN TraceID STRING(36)",
		}
		for _, stmt := range glassBoxDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}

		adminClient.Close()
	}

	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		telemetryDDL := []string{
			`CREATE TABLE DriverTelemetry (
				TraceId     STRING(64)   NOT NULL,
				DriverId    STRING(36)   NOT NULL,
				SupplierId  STRING(36)   NOT NULL,
				EventTime   TIMESTAMP    NOT NULL,
				Payload     BYTES(MAX)   NOT NULL,
				Lat         FLOAT64,
				Lng         FLOAT64,
				Velocity    FLOAT64,
				Heading     FLOAT64,
				CreatedAt   TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (TraceId)`,
			`CREATE INDEX Idx_DriverTelemetry_ByDriverTime ON DriverTelemetry(DriverId, EventTime DESC)`,
			`CREATE INDEX Idx_DriverTelemetry_BySupplierTime ON DriverTelemetry(SupplierId, EventTime DESC)`,
		}
		for _, stmt := range telemetryDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}

		adminClient.Close()
	}

	// ── MIGRATION: Phase 1 Global Modular — Regions + Regional Configs ─────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		regionalDDL := []string{
			`CREATE TABLE Regions (
				RegionId     STRING(36)  NOT NULL,
				RegionCode   STRING(20)  NOT NULL,
				RegionName   STRING(120) NOT NULL,
				CountryCode  STRING(2)   NOT NULL,
				Timezone     STRING(50)  NOT NULL,
				CurrencyCode STRING(3)   NOT NULL,
				DistanceUnit STRING(10)  NOT NULL DEFAULT ('km'),
				IsDefault    BOOL        NOT NULL DEFAULT (false),
				IsActive     BOOL        NOT NULL DEFAULT (true),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RegionId)`,
			"CREATE UNIQUE INDEX Idx_Regions_ByCode ON Regions(RegionCode)",
			"CREATE INDEX Idx_Regions_ByCountry ON Regions(CountryCode, IsActive)",
			`CREATE TABLE RegionalConfigs (
				RegionId                    STRING(36)  NOT NULL,
				PaymentGateways             STRING(MAX),
				NotificationFallbackOrder   STRING(MAX),
				SMSProvider                 STRING(30),
				MapsProvider                STRING(20),
				LLMProvider                 STRING(20),
				DefaultVUConversion         FLOAT64      NOT NULL DEFAULT (1.0),
				BreachRadiusMeters          FLOAT64      NOT NULL DEFAULT (100.0),
				ShopClosedGraceMinutes      INT64        NOT NULL DEFAULT (5),
				ShopClosedEscalationMinutes INT64        NOT NULL DEFAULT (3),
				OfflineModeDurationMinutes  INT64        NOT NULL DEFAULT (30),
				CashCustodyAlertHours       INT64        NOT NULL DEFAULT (4),
				CreatedAt                   TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt                   TIMESTAMP    OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RegionId)`,
			"ALTER TABLE Suppliers ADD COLUMN RegionId STRING(36)",
			"ALTER TABLE Warehouses ADD COLUMN RegionId STRING(36)",
			"ALTER TABLE Retailers ADD COLUMN RegionId STRING(36)",
			"CREATE INDEX Idx_Suppliers_ByRegionId ON Suppliers(RegionId)",
			"CREATE INDEX Idx_Warehouses_ByRegionId ON Warehouses(RegionId)",
			"CREATE INDEX Idx_Retailers_ByRegionId ON Retailers(RegionId)",
			"ALTER TABLE SupplierPaymentConfigs DROP CONSTRAINT CHK_GatewayName",
			"ALTER TABLE SupplierPaymentConfigs ADD CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'ADYEN', 'AIRWALLEX'))",
			"ALTER TABLE SupplierGlobalPayntConfigs DROP CONSTRAINT CHK_GatewayName",
			"ALTER TABLE SupplierGlobalPayntConfigs ADD CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'ADYEN'))",
		}
		for _, stmt := range regionalDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Autonomous settlement locality + Inventory V2 bootstrap ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		settlementLocalityDDL := []string{
			"ALTER TABLE MasterInvoices ADD COLUMN SettlementTarget STRING(30)",
			"ALTER TABLE MasterInvoices ADD CONSTRAINT CHK_MasterInvoiceSettlementTarget CHECK (SettlementTarget IS NULL OR SettlementTarget IN ('GLOBAL_SUPPLIER', 'LOCAL_WAREHOUSE', 'MIXED_WAREHOUSE'))",
			"ALTER TABLE Warehouses ADD COLUMN PaymentConfigId STRING(36)",
			"CREATE INDEX Idx_Warehouses_ByPaymentConfigId ON Warehouses(PaymentConfigId)",
			"ALTER TABLE SupplierInventory ADD COLUMN H3Cell STRING(15)",
			"CREATE INDEX Idx_Inventory_ByH3Cell ON SupplierInventory(H3Cell)",
			`CREATE TABLE SupplierInventoryV2 (
				SupplierId         STRING(36) NOT NULL,
				WarehouseId        STRING(36) NOT NULL,
				ProductId          STRING(36) NOT NULL,
				QuantityAvailable  INT64      NOT NULL,
				SafetyStockLevel   INT64      NOT NULL DEFAULT (0),
				MinStockLevel      INT64      NOT NULL DEFAULT (0),
				MaxStockLevel      INT64,
				H3Cell             STRING(15),
				UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (SupplierId, WarehouseId, ProductId)`,
			"CREATE INDEX Idx_InventoryV2_ByWarehouse ON SupplierInventoryV2(SupplierId, WarehouseId)",
			"CREATE INDEX Idx_InventoryV2_ByProduct ON SupplierInventoryV2(ProductId)",
			"CREATE INDEX Idx_InventoryV2_ByH3Cell ON SupplierInventoryV2(H3Cell, SupplierId)",
		}
		for _, stmt := range settlementLocalityDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier inventory import staging substrate ───────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		inventoryImportDDL := []string{
			`CREATE TABLE InventoryImportSessions (
				SessionId           STRING(36)  NOT NULL,
				SupplierId          STRING(36)  NOT NULL,
				WarehouseId         STRING(36),
				FileName            STRING(255) NOT NULL,
				ContentType         STRING(120),
				ObjectPath          STRING(MAX) NOT NULL,
				State               STRING(30)  NOT NULL,
				TotalRows           INT64       NOT NULL DEFAULT (0),
				ProcessedRows       INT64       NOT NULL DEFAULT (0),
				FailedRows          INT64       NOT NULL DEFAULT (0),
				PendingCreationRows INT64       NOT NULL DEFAULT (0),
				ApprovedBy          STRING(36),
				ApprovedAt          TIMESTAMP,
				ApplyStartedAt      TIMESTAMP,
				AppliedAt           TIMESTAMP,
				ExpiresAt           TIMESTAMP,
				CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt           TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_InventoryImportSessionState CHECK (
					State IN ('UPLOADED', 'DISCOVERING', 'MAPPING_REQUIRED', 'READY_FOR_REVIEW', 'APPROVED', 'APPLYING', 'APPLIED', 'FAILED', 'EXPIRED')
				)
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_InventoryImportSessions_BySupplierUpdated ON InventoryImportSessions(SupplierId, UpdatedAt DESC)",
			"CREATE INDEX Idx_InventoryImportSessions_BySupplierState ON InventoryImportSessions(SupplierId, State, UpdatedAt DESC)",
			"CREATE INDEX Idx_InventoryImportSessions_ByWarehouse ON InventoryImportSessions(SupplierId, WarehouseId, UpdatedAt DESC)",
			`CREATE TABLE InventoryImportRows (
				SessionId        STRING(36)  NOT NULL,
				RowId            STRING(36)  NOT NULL,
				SupplierId       STRING(36)  NOT NULL,
				WarehouseId      STRING(36),
				RowNumber        INT64       NOT NULL,
				Status           STRING(30)  NOT NULL,
				SourceJson       STRING(MAX),
				SkuId            STRING(36),
				ProductName      STRING(MAX),
				CategoryId       STRING(36),
				Currency         STRING(3),
				BasePrice        INT64,
				QuantityDelta    INT64       NOT NULL DEFAULT (0),
				MinimumOrderQty  INT64,
				StepSize         INT64,
				VolumetricUnit   FLOAT64,
				LengthCM         FLOAT64,
				WidthCM          FLOAT64,
				HeightCM         FLOAT64,
				ErrorsJson       STRING(MAX),
				SuggestionsJson  STRING(MAX),
				ValidationStatus STRING(30),
				CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				AppliedAt        TIMESTAMP,
				CONSTRAINT CHK_InventoryImportRowStatus CHECK (
					Status IN ('UNMAPPED', 'MAPPED_EXISTING', 'PENDING_CREATION', 'INVALID', 'READY_FOR_REVIEW', 'APPROVED', 'APPLIED', 'FAILED')
				)
			) PRIMARY KEY (SessionId, RowId),
			  INTERLEAVE IN PARENT InventoryImportSessions ON DELETE CASCADE`,
			"CREATE INDEX Idx_InventoryImportRows_BySessionRow ON InventoryImportRows(SessionId, RowNumber)",
			"CREATE INDEX Idx_InventoryImportRows_BySessionStatus ON InventoryImportRows(SessionId, Status, RowNumber)",
			"CREATE INDEX Idx_InventoryImportRows_BySupplierStatus ON InventoryImportRows(SupplierId, Status, UpdatedAt DESC)",
		}
		for _, stmt := range inventoryImportDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: SupplierImport sandbox compatibility substrate ───────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		supplierImportSandboxDDL := []string{
			`CREATE TABLE SupplierImportSessions (
				supplier_id    STRING(36)  NOT NULL,
				session_id     STRING(36)  NOT NULL,
				status         STRING(30)  NOT NULL,
				file_name      STRING(255) NOT NULL,
				total_rows     INT64       NOT NULL DEFAULT (0),
				error_summary  JSON,
				created_at     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				updated_at     TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_SupplierImportSessionStatus CHECK (
					status IN (
						'initialized', 'uploaded', 'discovering', 'discovered', 'mapping_required', 'approved', 'applying', 'applied', 'failed',
						'INITIALIZED', 'UPLOADED', 'DISCOVERING', 'DISCOVERED', 'MAPPING_REQUIRED', 'APPROVED', 'APPLYING', 'APPLIED', 'FAILED'
					)
				)
			) PRIMARY KEY (supplier_id, session_id)`,
			"CREATE INDEX Idx_SupplierImportSessions_BySupplierUpdated ON SupplierImportSessions(supplier_id, updated_at DESC)",
			"CREATE INDEX Idx_SupplierImportSessions_BySupplierStatus ON SupplierImportSessions(supplier_id, status, updated_at DESC)",
			`CREATE TABLE SupplierImportStagedRows (
				supplier_id         STRING(36)       NOT NULL,
				session_id          STRING(36)       NOT NULL,
				row_index           INT64            NOT NULL,
				raw_data            JSON,
				cleaned_data        JSON,
				validation_errors   ARRAY<STRING(MAX)>,
				is_new_product      BOOL             NOT NULL DEFAULT (false),
				created_at          TIMESTAMP        NOT NULL OPTIONS (allow_commit_timestamp=true),
				updated_at          TIMESTAMP        OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (supplier_id, session_id, row_index),
			  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE`,
			"CREATE INDEX Idx_SupplierImportStagedRows_BySession ON SupplierImportStagedRows(supplier_id, session_id, row_index)",
			`CREATE TABLE SupplierImportMapping (
				supplier_id    STRING(36)  NOT NULL,
				session_id     STRING(36)  NOT NULL,
				mapping_json   JSON,
				created_at     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				updated_at     TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (supplier_id, session_id),
			  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE`,
		}
		for _, stmt := range supplierImportSandboxDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: S-level dynamic billing metering substrate ─────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		billingMeterDDL := []string{
			`CREATE TABLE BillingMeterEvents (
				OrderId        STRING(36)  NOT NULL,
				SupplierId     STRING(36)  NOT NULL,
				InvoiceId      STRING(36),
				RetailerId     STRING(36),
				PeriodMonth    STRING(7)   NOT NULL,
				Currency       STRING(3)   NOT NULL,
				Amount         INT64       NOT NULL,
				FeeAmount      INT64       NOT NULL,
				FeeBasisPts    INT64       NOT NULL,
				EventTimestamp TIMESTAMP   NOT NULL,
				ProcessedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (OrderId)`,
			"CREATE INDEX Idx_BillingMeterEvents_BySupplierMonth ON BillingMeterEvents(SupplierId, PeriodMonth, ProcessedAt DESC)",
			`CREATE TABLE BillingSupplierMeters (
				SupplierId   STRING(36) NOT NULL,
				PeriodMonth  STRING(7)  NOT NULL,
				Currency     STRING(3)  NOT NULL,
				ShardId      INT64      NOT NULL,
				OrderCount   INT64      NOT NULL DEFAULT (0),
				GrossAmount  INT64      NOT NULL DEFAULT (0),
				FeeAmount    INT64      NOT NULL DEFAULT (0),
				UpdatedAt    TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (SupplierId, PeriodMonth, Currency, ShardId)`,
			"CREATE INDEX Idx_BillingSupplierMeters_ByPeriod ON BillingSupplierMeters(PeriodMonth, SupplierId)",
			`CREATE TABLE BillingGlobalMeters (
				PeriodMonth  STRING(20) NOT NULL,
				Currency     STRING(8)  NOT NULL,
				ShardId      INT64      NOT NULL,
				OrderCount   INT64      NOT NULL DEFAULT (0),
				GrossAmount  INT64      NOT NULL DEFAULT (0),
				FeeAmount    INT64      NOT NULL DEFAULT (0),
				UpdatedAt    TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (PeriodMonth, Currency, ShardId)`,
			"CREATE INDEX Idx_BillingGlobalMeters_ByPeriod ON BillingGlobalMeters(PeriodMonth, Currency)",
		}
		for _, stmt := range billingMeterDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Dynamic delivery handshake sessions ───────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		deliveryHandshakeDDL := []string{
			`CREATE TABLE DeliverySessions (
				SessionId             STRING(36)  NOT NULL,
				OrderId               STRING(36)  NOT NULL,
				RetailerId            STRING(36)  NOT NULL,
				DriverId              STRING(36)  NOT NULL,
				SupplierId            STRING(36),
				State                 STRING(30)  NOT NULL,
				OriginalAmount        INT64       NOT NULL DEFAULT (0),
				AdjustedAmount        INT64       NOT NULL DEFAULT (0),
				FeeBasisPts           INT64       NOT NULL DEFAULT (0),
				FeeAmount             INT64       NOT NULL DEFAULT (0),
				Currency              STRING(3)   NOT NULL DEFAULT ('UZS'),
				HandshakeTokenHash    STRING(MAX),
				RetailerLat           FLOAT64,
				RetailerLng           FLOAT64,
				DriverLat             FLOAT64,
				DriverLng             FLOAT64,
				DistanceM             FLOAT64,
				LastErrorCode         STRING(64),
				LastErrorMessage      STRING(MAX),
				PaymentSessionId      STRING(36),
				InvoiceId             STRING(36),
				HandshakeVerifiedAt   TIMESTAMP,
				SettlementRequiredAt  TIMESTAMP,
				PaymentClearedAt      TIMESTAMP,
				DisputedAt            TIMESTAMP,
				Version               INT64       NOT NULL DEFAULT (1),
				CreatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt             TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_DeliverySessionState CHECK (
					State IN ('PROXIMITY_LOCK', 'HANDSHAKE_START', 'RECONCILIATION', 'SETTLEMENT_AWAIT', 'FINAL_SETTLEMENT', 'DISPUTED')
				)
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_DeliverySessions_ByOrderId ON DeliverySessions(OrderId, UpdatedAt DESC)",
			"CREATE INDEX Idx_DeliverySessions_ByDriverState ON DeliverySessions(DriverId, State, UpdatedAt DESC)",
			"CREATE INDEX Idx_DeliverySessions_ByRetailerState ON DeliverySessions(RetailerId, State, UpdatedAt DESC)",
			"CREATE INDEX Idx_DeliverySessions_BySupplierState ON DeliverySessions(SupplierId, State, UpdatedAt DESC)",
			`CREATE TABLE DeliverySessionAdjustments (
				AdjustmentId          STRING(36)  NOT NULL,
				SessionId             STRING(36)  NOT NULL,
				OrderId               STRING(36)  NOT NULL,
				LineItemId            STRING(36),
				SkuId                 STRING(36)  NOT NULL,
				OriginalQty           INT64       NOT NULL,
				AcceptedQty           INT64       NOT NULL,
				RejectedQty           INT64       NOT NULL,
				UnitPrice             INT64       NOT NULL,
				Reason                STRING(40),
				CreatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (AdjustmentId)`,
			"CREATE INDEX Idx_DeliverySessionAdjustments_BySessionId ON DeliverySessionAdjustments(SessionId, CreatedAt DESC)",
			"CREATE INDEX Idx_DeliverySessionAdjustments_ByOrderId ON DeliverySessionAdjustments(OrderId, CreatedAt DESC)",
		}
		for _, stmt := range deliveryHandshakeDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
