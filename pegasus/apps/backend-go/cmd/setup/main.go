package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"config" // local map
)

func main() {
	log.Println("Initializing Pegasus Seed Script...")

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	// 2. Setup Spanner Admin Connections
	// We are hitting the emulator, so we configure GRPC insecure options
	emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorAddr == "" {
		emulatorAddr = "localhost:9010"
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorAddr)
	}

	log.Printf("Connecting to Spanner Emulator at %s", emulatorAddr)

	opts := []option.ClientOption{
		option.WithEndpoint(emulatorAddr),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}

	instanceAdmin, err := instance.NewInstanceAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create instance admin client: %v", err)
	}
	defer instanceAdmin.Close()

	databaseAdmin, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create database admin client: %v", err)
	}
	defer databaseAdmin.Close()

	// 3. Create Spanner Instance (if not exists)
	parentName := fmt.Sprintf("projects/%s", cfg.SpannerProject)
	instanceName := fmt.Sprintf("%s/instances/%s", parentName, cfg.SpannerInstance)

	log.Printf("Checking instance: %s", instanceName)
	_, err = instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})
	if err != nil {
		log.Printf("Instance not found, creating it...")
		req := &instancepb.CreateInstanceRequest{
			Parent:     parentName,
			InstanceId: cfg.SpannerInstance,
			Instance: &instancepb.Instance{
				Config:      fmt.Sprintf("%s/instanceConfigs/emulator-config", parentName),
				DisplayName: "Pegasus Emulator",
				NodeCount:   1,
			},
		}
		op, err := instanceAdmin.CreateInstance(ctx, req)
		if err != nil {
			log.Fatalf("Failed to trigger instance creation: %v", err)
		}
		if _, err := op.Wait(ctx); err != nil {
			log.Fatalf("Failed to create instance: %v", err)
		}
		log.Println("Instance created successfully.")
	}

	// 4. Create Database and apply DDL
	dbName := fmt.Sprintf("%s/databases/%s", instanceName, cfg.SpannerDatabase)

	ddlStatements := []string{
		`CREATE TABLE Retailers (
			RetailerId STRING(36) NOT NULL,
			Name STRING(MAX) NOT NULL,
			Phone STRING(MAX),
			ShopName STRING(MAX),
			ShopLocation STRING(MAX),
			TaxIdentificationNumber STRING(MAX),
			Status STRING(20),
			PasswordHash STRING(MAX),
			FcmToken STRING(MAX),
			TelegramChatId STRING(MAX),
			CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId)`,

		`CREATE TABLE Drivers (
			DriverId STRING(36) NOT NULL,
			Name STRING(MAX) NOT NULL,
			CurrentLocation STRING(MAX),
			CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (DriverId)`,

		`CREATE TABLE Vehicles (
			VehicleId     STRING(36)  NOT NULL,
			SupplierId    STRING(36)  NOT NULL,
			VehicleClass  STRING(10)  NOT NULL,
			Label         STRING(100),
			LicensePlate  STRING(30),
			MaxVolumeVU   FLOAT64     NOT NULL,
			LengthCM      FLOAT64,
			WidthCM       FLOAT64,
			HeightCM      FLOAT64,
			IsActive      BOOL        NOT NULL DEFAULT (true),
			UnavailableReason STRING(64),
			CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (VehicleId)`,
		`CREATE INDEX Idx_Vehicles_BySupplier ON Vehicles(SupplierId)`,

		`CREATE TABLE Orders (
			OrderId        STRING(36)  NOT NULL,
			RetailerId     STRING(36)  NOT NULL,
			DriverId       STRING(36),
			SupplierId     STRING(36),
			InvoiceId      STRING(36),
			State          STRING(30)  NOT NULL,
			TotalAmount    NUMERIC,
			Amount      INT64,
			PaymentGateway STRING(MAX),
			ShopLocation   STRING(MAX),
			RouteId        STRING(MAX),
			OrderSource    STRING(MAX),
			DeliveryToken  STRING(MAX),
			AutoConfirmAt  TIMESTAMP OPTIONS (allow_commit_timestamp=true),
			DeliverBefore  TIMESTAMP OPTIONS (allow_commit_timestamp=true),
			RequestedDeliveryDate TIMESTAMP OPTIONS (allow_commit_timestamp=true),
			ScheduleShard  INT64       NOT NULL DEFAULT (0),
			QRValidatedAt  TIMESTAMP,
			Version        INT64       NOT NULL DEFAULT (1),
			LockedUntil    TIMESTAMP,
			CreatedAt      TIMESTAMP OPTIONS (allow_commit_timestamp=true),
			PaymentStatus   STRING(30)  NOT NULL DEFAULT ('PENDING'),
			CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'BACKORDERED', 'QUARANTINE')),
			CONSTRAINT CHK_PaymentStatus CHECK (PaymentStatus IN ('PENDING', 'PENDING_CASH_COLLECTION', 'AWAITING_GATEWAY_WEBHOOK', 'PAID', 'FAILED'))
		) PRIMARY KEY (OrderId)`,

		`CREATE INDEX IDX_Orders_RetailerId ON Orders(RetailerId)`,
		`CREATE INDEX IDX_Orders_DriverId   ON Orders(DriverId)`,
		`CREATE INDEX Idx_Orders_InvoiceId ON Orders(InvoiceId)`,
		`CREATE INDEX Idx_Orders_SupplierId ON Orders(SupplierId)`,
		`CREATE INDEX Idx_Orders_ByScheduleShardStateDate ON Orders(ScheduleShard, State, RequestedDeliveryDate DESC)`,

		`CREATE TABLE MasterInvoices (
			InvoiceId           STRING(36)  NOT NULL,
			RetailerId          STRING(36)  NOT NULL,
			Total            INT64       NOT NULL,
			State               STRING(20)  NOT NULL,
			OrderId             STRING(36),
			GlobalPayTransactionId  STRING(64),
			PaymentMode         STRING(20),
			CollectorDriverId   STRING(36),
			CollectedAt         TIMESTAMP,
			CollectionLat       FLOAT64,
			CollectionLng       FLOAT64,
			GeofenceDistanceM   FLOAT64,
			CustodyStatus       STRING(20),
			CreatedAt           TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (InvoiceId)`,
		`CREATE INDEX Idx_MasterInvoice_Retailer ON MasterInvoices(RetailerId)`,
		`CREATE INDEX Idx_MasterInvoice_OrderId ON MasterInvoices(OrderId)`,
		`CREATE INDEX Idx_MasterInvoice_GlobalPayTxn ON MasterInvoices(GlobalPayTransactionId)`,

		`CREATE TABLE Products (
			ProductId   STRING(36)  NOT NULL,
			Name        STRING(255) NOT NULL,
			Size        STRING(50),
			PackQuantity INT64,
			Price       NUMERIC,
			ImageUrl    STRING(MAX)
		) PRIMARY KEY (ProductId)`,

		// ── Standalone line-items table (Phase 4 analytics-safe) ──────────────
		// ARCHITECTURAL DECISION: Decoupled from Orders parent.
		// Distributed PK (LineItemId) eliminates write hotspots.
		// ByOrder index   → driver app O(1) order lookup.
		// BySku   index   → analytics engine O(1) SKU aggregation.
		`CREATE TABLE OrderLineItems (
			LineItemId    STRING(36)  NOT NULL,
			OrderId       STRING(36)  NOT NULL,
			SkuId         STRING(50)  NOT NULL,
			Quantity      INT64       NOT NULL,
			UnitPrice  INT64       NOT NULL,
			Status        STRING(20)  NOT NULL
		) PRIMARY KEY (LineItemId)`,

		// Driver App — fetch all items for a given order in O(1)
		`CREATE INDEX Idx_OrderItems_ByOrder ON OrderLineItems(OrderId)`,

		// Analytics Engine — aggregate SKU volume across all orders with zero hotspot
		`CREATE INDEX Idx_OrderItems_BySku ON OrderLineItems(SkuId)`,

		`CREATE TABLE LedgerEntries (
			TransactionId STRING(100) NOT NULL,
			OrderId STRING(36) NOT NULL,
			AccountId STRING(MAX) NOT NULL,
			Amount INT64 NOT NULL,
			EntryType STRING(20) NOT NULL,
			CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (TransactionId)`,

		// ── B2B Dynamic Pricing Engine (Vector G) ─────────────────────────────
		// MULTI-TENANT: Each row is owned by exactly one SupplierId.
		// Nestle rows are invisible to Coca-Cola — the API enforces this via JWT.
		// TargetRetailerTier enables "GOLD gets 15%, SILVER gets 8%" logic.
		// ValidUntil allows time-bound campaign discounts; NULL = no expiry.
		// IsActive lets suppliers soft-disable a tier without deleting history.
		`CREATE TABLE PricingTiers (
			TierId              STRING(36)  NOT NULL,
			SupplierId          STRING(36)  NOT NULL,
			SkuId               STRING(50)  NOT NULL,
			MinPallets          INT64       NOT NULL,
			DiscountPct         INT64       NOT NULL,
			TargetRetailerTier  STRING(20),
			ValidUntil          TIMESTAMP,
			IsActive            BOOL        NOT NULL
		) PRIMARY KEY (TierId)`,

		// Fast supplier-scoped lookup: POST /v1/supplier/pricing/rules → list own tiers
		`CREATE INDEX Idx_PricingTiers_BySupplier ON PricingTiers(SupplierId)`,

		// Checkout engine path: cart/pricing.go queries by SkuId for active tiers
		`CREATE INDEX Idx_PricingTiers_BySkuActive ON PricingTiers(SkuId, IsActive)`,

		// ── Vector H: Field General Route Optimizer ────────────────────────────
		// SequenceIndex is written by the 04:00 AM TSP cron job (routing/optimizer.go).
		// Value 0 = first drop, 1 = second drop, etc. NULL = not yet sequenced.
		// The driver app enforces strict ascending delivery; skipping is a hard block.
		`ALTER TABLE Orders ADD COLUMN SequenceIndex INT64`,

		// Filtered index: the driver app fetches only LOADED orders in depot sequence.
		// Filtering by State='LOADED' keeps the index small — completed rows are excluded.
		`CREATE INDEX Idx_Orders_ByDriverAndSequence ON Orders(DriverId, SequenceIndex)`,

		// ── Financial Reconciliation (Phase 5) ────────────────────────────────
		`CREATE TABLE LedgerAnomalies (
			OrderId STRING(36) NOT NULL,
			RetailerId STRING(36) NOT NULL,
			SpannerUzs INT64 NOT NULL,
			GatewayUzs INT64 NOT NULL,
			GatewayProvider STRING(20) NOT NULL,
			Status STRING(20) NOT NULL,
			DetectedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (OrderId)`,

		`CREATE INDEX Idx_Anomalies_ByStatus ON LedgerAnomalies(Status)`,

		// ── SUPPLIER CATALOG ──────────────────────────────────────────────────
		`CREATE TABLE Suppliers (
			SupplierId          STRING(36)  NOT NULL,
			Name                STRING(255) NOT NULL,
			LogoUrl             STRING(MAX),
			Category            STRING(100),
			Phone               STRING(20),
			Email               STRING(MAX),
			PasswordHash        STRING(MAX),
			TaxId               STRING(MAX),
			ContactPerson       STRING(MAX),
			CompanyRegNumber    STRING(MAX),
			BillingAddress      STRING(MAX),
			IsConfigured        BOOL,
			OperatingCategories ARRAY<STRING(MAX)>,
			WarehouseLocation   STRING(MAX),
			WarehouseLat        FLOAT64,
			WarehouseLng        FLOAT64,
			BankName            STRING(MAX),
			AccountNumber       STRING(MAX),
			CardNumber          STRING(MAX),
			PaymentGateway      STRING(20),
			CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (SupplierId)`,
		`CREATE INDEX Idx_Suppliers_ByPhone ON Suppliers(Phone)`,

		`CREATE TABLE SupplierProducts (
			SkuId         STRING(50)  NOT NULL,
			SupplierId    STRING(36)  NOT NULL,
			Name          STRING(255) NOT NULL,
			Description   STRING(MAX),
			ImageUrl      STRING(MAX),
			SellByBlock   BOOL        NOT NULL,
			UnitsPerBlock INT64       NOT NULL,
			BasePrice  INT64       NOT NULL,
			IsActive      BOOL        NOT NULL,
			CategoryId    STRING(36),
			CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (SkuId)`,
		`CREATE INDEX Idx_Products_BySupplier ON SupplierProducts(SupplierId)`,

		`CREATE TABLE Categories (
			CategoryId STRING(36)  NOT NULL,
			Name       STRING(255) NOT NULL,
			Icon       STRING(100),
			SortOrder  INT64       NOT NULL,
			CreatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (CategoryId)`,

		`CREATE TABLE RetailerSuppliers (
			RetailerId STRING(36) NOT NULL,
			SupplierId STRING(36) NOT NULL,
			AddedAt    TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId, SupplierId)`,
		`CREATE INDEX Idx_RetailerSuppliers_ByRetailer ON RetailerSuppliers(RetailerId)`,

		`CREATE TABLE RetailerGlobalSettings (
			RetailerId             STRING(36) NOT NULL,
			GlobalAutoOrderEnabled BOOL       NOT NULL,
			UpdatedAt              TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId)`,

		`CREATE TABLE RetailerSupplierSettings (
			RetailerId         STRING(36) NOT NULL,
			SupplierId         STRING(36) NOT NULL,
			AutoOrderEnabled   BOOL       NOT NULL,
			AnalyticsStartDate TIMESTAMP,
			UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId, SupplierId),
		  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,

		`CREATE TABLE RetailerProductSettings (
			RetailerId         STRING(36) NOT NULL,
			ProductId          STRING(36) NOT NULL,
			AutoOrderEnabled   BOOL       NOT NULL,
			AnalyticsStartDate TIMESTAMP,
			UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId, ProductId),
		  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,

		`CREATE TABLE RetailerCategorySettings (
			RetailerId         STRING(36) NOT NULL,
			CategoryId         STRING(50) NOT NULL,
			AutoOrderEnabled   BOOL       NOT NULL,
			AnalyticsStartDate TIMESTAMP,
			UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (RetailerId, CategoryId)`,
		`CREATE INDEX Idx_RetailerCategorySettings_ByRetailer ON RetailerCategorySettings(RetailerId)`,

		`CREATE TABLE AIPredictions (
			PredictionId        STRING(36) NOT NULL,
			RetailerId          STRING(36) NOT NULL,
			PredictedAmount  INT64      NOT NULL,
			TriggerDate         TIMESTAMP,
			TriggerShard        INT64      NOT NULL DEFAULT (0),
			Status              STRING(20) NOT NULL,
			CreatedAt           TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (PredictionId)`,
		`CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId)`,
		`CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC)`,
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

		`CREATE TABLE SupplierInventory (
			ProductId         STRING(36) NOT NULL,
			SupplierId        STRING(36) NOT NULL,
			QuantityAvailable INT64      NOT NULL,
			UpdatedAt         TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (ProductId)`,
		`CREATE INDEX Idx_Inventory_BySupplier ON SupplierInventory(SupplierId)`,

		`CREATE TABLE InventoryAuditLog (
			AuditId     STRING(36)  NOT NULL,
			ProductId   STRING(36)  NOT NULL,
			SupplierId  STRING(36)  NOT NULL,
			AdjustedBy  STRING(36)  NOT NULL,
			PreviousQty INT64       NOT NULL,
			NewQty      INT64       NOT NULL,
			Delta       INT64       NOT NULL,
			Reason      STRING(50)  NOT NULL,
			AdjustedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (AuditId)`,
		`CREATE INDEX Idx_AuditLog_BySupplier ON InventoryAuditLog(SupplierId)`,
		`CREATE INDEX Idx_AuditLog_ByProduct  ON InventoryAuditLog(ProductId)`,

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

		// ── SUPPLIER PAYMENT GATEWAY VAULT ──
		`CREATE TABLE SupplierPaymentConfigs (
			ConfigId     STRING(36)  NOT NULL,
			SupplierId   STRING(36)  NOT NULL,
			GatewayName  STRING(20)  NOT NULL,
			MerchantId   STRING(MAX) NOT NULL,
			ServiceId    STRING(MAX),
			SecretKey    BYTES(MAX)  NOT NULL,
			IsActive     BOOL        NOT NULL DEFAULT (true),
			CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
			UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
			CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'GLOBAL_PAY'))
		) PRIMARY KEY (ConfigId)`,
		`CREATE INDEX Idx_SupplierPaymentConfigs_BySupplierId ON SupplierPaymentConfigs(SupplierId)`,
		`CREATE UNIQUE INDEX Idx_SupplierPaymentConfigs_Unique ON SupplierPaymentConfigs(SupplierId, GatewayName)`,

		// ── GATEWAY ONBOARDING SESSIONS (SUPPLIER CONNECT) ──
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
		`CREATE INDEX Idx_GatewayOnboarding_BySupplierId ON GatewayOnboardingSessions(SupplierId)`,
		`CREATE INDEX Idx_GatewayOnboarding_ByStatus ON GatewayOnboardingSessions(Status)`,

		// ── PAYMENT SESSIONS (PHASE 13: DURABLE PAYMENT SESSION ENGINE) ──
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
		`CREATE INDEX Idx_PaymentSessions_ByOrderId ON PaymentSessions(OrderId)`,
		`CREATE INDEX Idx_PaymentSessions_BySupplierId ON PaymentSessions(SupplierId)`,
		`CREATE INDEX Idx_PaymentSessions_ByStatus ON PaymentSessions(Status)`,

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
		`CREATE INDEX Idx_PaymentAttempts_BySessionId ON PaymentAttempts(SessionId)`,
		`CREATE INDEX Idx_PaymentAttempts_ByProviderTxn ON PaymentAttempts(ProviderTransactionId)`,

		// ── Vector H Phase 2: ETA & Returning-to-Warehouse ────────────────────
		// Per-stop ETA columns written by routing/eta.go on driver depart (traffic-aware)
		// and refreshed after each delivery completion from driver's current position.
		`ALTER TABLE Orders ADD COLUMN EstimatedArrivalAt TIMESTAMP`,
		`ALTER TABLE Orders ADD COLUMN EstimatedDurationSec INT64`,
		`ALTER TABLE Orders ADD COLUMN EstimatedDistanceM INT64`,

		// Driver-level return-to-warehouse ETA. Populated when truck enters RETURNING state.
		`ALTER TABLE Drivers ADD COLUMN EstimatedReturnAt TIMESTAMP`,
		`ALTER TABLE Drivers ADD COLUMN ReturnDurationSec INT64`,

		// ── Phase 2: Missing Infrastructure Tables ────────────────────────────

		// Device tokens for FCM / APNs push notifications
		`CREATE TABLE DeviceTokens (
			TokenId   STRING(36)  NOT NULL,
			UserId    STRING(36)  NOT NULL,
			Role      STRING(20)  NOT NULL,
			Platform  STRING(10)  NOT NULL,
			Token     STRING(MAX) NOT NULL,
			CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (TokenId)`,
		`CREATE UNIQUE INDEX Idx_DeviceTokens_ByUserPlatform ON DeviceTokens(UserId, Platform)`,
		`CREATE INDEX Idx_DeviceTokens_ByUser ON DeviceTokens(UserId)`,

		// Notification log
		`CREATE TABLE Notifications (
			NotificationId STRING(36)  NOT NULL,
			RecipientId    STRING(36)  NOT NULL,
			RecipientRole  STRING(20)  NOT NULL,
			Type           STRING(50)  NOT NULL,
			Title          STRING(200) NOT NULL,
			Body           STRING(MAX),
			Payload        STRING(MAX),
			Channel        STRING(20),
			ReadAt         TIMESTAMP,
			CreatedAt      TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (NotificationId)`,
		`CREATE INDEX Idx_Notifications_ByRecipient ON Notifications(RecipientId, CreatedAt DESC)`,

		// Immutable audit trail
		`CREATE TABLE AuditLog (
			LogId        STRING(36)  NOT NULL,
			ActorId      STRING(36)  NOT NULL,
			ActorRole    STRING(20)  NOT NULL,
			Action       STRING(50)  NOT NULL,
			ResourceType STRING(30)  NOT NULL,
			ResourceId   STRING(36)  NOT NULL,
			Metadata     STRING(MAX),
			CreatedAt    TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (LogId)`,
		`CREATE INDEX Idx_AuditLog_ByResource ON AuditLog(ResourceType, ResourceId, CreatedAt DESC)`,
		`CREATE INDEX Idx_AuditLog_ByActor ON AuditLog(ActorId, CreatedAt DESC)`,

		// Server-side retailer cart persistence
		`CREATE TABLE RetailerCarts (
			CartId       STRING(36)  NOT NULL,
			RetailerId   STRING(36)  NOT NULL,
			SupplierId   STRING(36)  NOT NULL,
			SkuId        STRING(36)  NOT NULL,
			Quantity     INT64       NOT NULL,
			UnitPrice INT64       NOT NULL,
			AddedAt      TIMESTAMP OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (CartId)`,
		`CREATE INDEX Idx_RetailerCarts_ByRetailer ON RetailerCarts(RetailerId)`,
		`CREATE INDEX Idx_RetailerCarts_ByRetailerSupplier ON RetailerCarts(RetailerId, SupplierId)`,

		// Distributed cron job locking
		`CREATE TABLE ScheduledJobs (
			JobId      STRING(36)  NOT NULL,
			JobName    STRING(100) NOT NULL,
			LastRunAt  TIMESTAMP,
			NextRunAt  TIMESTAMP,
			Status     STRING(20)  NOT NULL DEFAULT ('IDLE'),
			LockHolder STRING(100),
			LockExpiry TIMESTAMP,
			CONSTRAINT CHK_JobStatus CHECK (Status IN ('IDLE', 'RUNNING', 'FAILED'))
		) PRIMARY KEY (JobId)`,

		// Missing indexes on existing tables
		`CREATE INDEX Idx_Orders_ByRetailerState ON Orders(RetailerId, State)`,
		`CREATE INDEX Idx_PaymentSessions_ByStatusExpiry ON PaymentSessions(Status, ExpiresAt)`,
		`CREATE INDEX Idx_PaymentSessions_ByRetailerId ON PaymentSessions(RetailerId)`,

		// ── Retailer Card Tokens (saved payment cards for tokenized checkout) ──
		`CREATE TABLE RetailerCardTokens (
			TokenId           STRING(36)  NOT NULL,
			RetailerId        STRING(36)  NOT NULL,
			Gateway           STRING(20)  NOT NULL,
			ProviderCardToken STRING(MAX) NOT NULL,
			CardLast4         STRING(4),
			CardType          STRING(20),
			IsDefault         BOOL        NOT NULL DEFAULT (false),
			IsActive          BOOL        NOT NULL DEFAULT (true),
			ExpiresAt         TIMESTAMP,
			CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (TokenId)`,
		`CREATE INDEX Idx_RetailerCardTokens_ByRetailer ON RetailerCardTokens(RetailerId)`,

		// Global Pay split payments: supplier recipient account for distribution
		`ALTER TABLE SupplierPaymentConfigs ADD COLUMN RecipientId STRING(MAX)`,

		// Amendment safeguard: pending supplier approval for large price reductions
		`ALTER TABLE Orders ADD COLUMN AmendmentPendingApproval BOOL NOT NULL DEFAULT (false)`,
		`ALTER TABLE Orders ADD COLUMN PendingAmendmentData STRING(MAX)`,

		// ── Phase E: Warehouses, SupplierUsers, Factories ─────────────────
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

		`CREATE TABLE Factories (
			FactoryId            STRING(36)  NOT NULL,
			SupplierId           STRING(36)  NOT NULL,
			Name                 STRING(255) NOT NULL,
			Address              STRING(MAX),
			Lat                  FLOAT64,
			Lng                  FLOAT64,
			RegionCode           STRING(20),
			LeadTimeDays         INT64       NOT NULL DEFAULT (2),
			ProductionCapacityVU FLOAT64     NOT NULL DEFAULT (0),
			IsActive             BOOL        NOT NULL DEFAULT (true),
			CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
			UpdatedAt            TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
		) PRIMARY KEY (FactoryId)`,
		`CREATE INDEX Idx_Factories_BySupplierId ON Factories(SupplierId)`,

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

		`CREATE TABLE InternalTransferItems (
			TransferId STRING(36) NOT NULL,
			ItemId     STRING(36) NOT NULL,
			ProductId  STRING(36) NOT NULL,
			Quantity   INT64      NOT NULL,
			VolumeVU   FLOAT64    NOT NULL DEFAULT (0)
		) PRIMARY KEY (TransferId, ItemId),
		  INTERLEAVE IN PARENT InternalTransferOrders ON DELETE CASCADE`,

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
		`ALTER TABLE Drivers ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE Vehicles ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE WarehouseStaff ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE SupplierInventory ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE InventoryAuditLog ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE Orders ADD COLUMN WarehouseId STRING(36)`,
		`ALTER TABLE RetailerCarts ADD COLUMN WarehouseId STRING(36)`,
		`CREATE INDEX Idx_Drivers_ByWarehouseId ON Drivers(WarehouseId)`,
		`CREATE INDEX Idx_Vehicles_ByWarehouseId ON Vehicles(WarehouseId)`,
		`CREATE INDEX Idx_WarehouseStaff_ByWarehouseId ON WarehouseStaff(WarehouseId)`,
		`CREATE INDEX Idx_Inventory_ByWarehouseId ON SupplierInventory(SupplierId, WarehouseId)`,
		`CREATE INDEX Idx_Orders_ByWarehouseId ON Orders(WarehouseId)`,
		`ALTER TABLE AIPredictions ADD COLUMN WarehouseId STRING(36)`,
		`CREATE INDEX Idx_AIPredictions_ByWarehouse ON AIPredictions(WarehouseId)`,
		`ALTER TABLE Warehouses ADD COLUMN PrimaryFactoryId STRING(36)`,
		`ALTER TABLE Warehouses ADD COLUMN SecondaryFactoryId STRING(36)`,

		// ── Phase IV: Pre-order policy + state expansion ──
		`ALTER TABLE Orders ADD COLUMN CancelLockedAt TIMESTAMP`,
		`ALTER TABLE Orders ADD COLUMN CancelLockReason STRING(30)`,
		`ALTER TABLE Orders ADD COLUMN ConfirmationNotifiedAt TIMESTAMP`,

		// ── V.5 — H3 spatial indexing on Orders ──
		// Orders carry an H3 cell (resolution 7, 15-char hex). The composite
		// (H3Cell, State) index powers proximity-scoped dispatch queries
		// without a full Orders scan. NULL until backfill hydrates legacy rows.
		`ALTER TABLE Orders ADD COLUMN H3Cell STRING(15)`,
		`CREATE INDEX IDX_Orders_H3Cell_State ON Orders(H3Cell, State)`,

		// ── Transactional Outbox (the atomicity primitive) ──
		// Every state-change event is written here in the same RWTxn as the
		// domain mutation. The outbox.Relay tails CreatedAt-ordered rows,
		// publishes to Kafka with Key = AggregateId (preserves per-entity
		// partition order) and Header `event_type` = EventType (consumer
		// discriminator). PublishedAt is filled on successful publish.
		`CREATE TABLE OutboxEvents (
			EventId       STRING(36)   NOT NULL,
			AggregateType STRING(30)   NOT NULL,
			AggregateId   STRING(36)   NOT NULL,
			EventType     STRING(60)   NOT NULL,
			TopicName     STRING(100)  NOT NULL,
			Payload       BYTES(MAX)   NOT NULL,
			TraceID       STRING(36),
			CreatedAt     TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
			PublishedAt   TIMESTAMP,
		) PRIMARY KEY (EventId)`,
		// Composite index supports the relay's tail query:
		//   WHERE PublishedAt IS NULL ORDER BY CreatedAt LIMIT @lim
		// Unpublished rows (PublishedAt = NULL) sort first under Spanner's
		// NULLS FIRST default ordering, so a LIMIT scan stays bounded.
		`CREATE INDEX Idx_OutboxEvents_Unpublished
			ON OutboxEvents(PublishedAt, CreatedAt)`,

		// ── Phase 2 — Intelligent Dispatch Optimization ──
		// VolumeVU is the per-order volumetric unit total (sum of line-item
		// volume × quantity). First-class column so the optimizer hydrates
		// without a per-call OrderLineItems join. NULL until backfill or
		// next write — hydration COALESCEs to live compute during rollout.
		`ALTER TABLE Orders ADD COLUMN VolumeVU FLOAT64`,

		// Refined dispatch hydration index: H3 cluster + dispatchable state +
		// SLA window. Powers "give me every PENDING order in this H3 ring
		// scheduled for today" in a single index scan.
		`CREATE INDEX Idx_Orders_H3Cell_State_Date
			ON Orders(H3Cell, State, RequestedDeliveryDate DESC)`,

		// Receiving windows on Retailers — Phase 2 hard-constraint inputs to
		// the VRP solver. Stored as TIME (local supplier timezone, Tashkent
		// UTC+5 in v1). Closes the receiving-window slice of Known Gap #13.
		`ALTER TABLE Retailers ADD COLUMN ReceivingWindowOpen STRING(5)`,
		`ALTER TABLE Retailers ADD COLUMN ReceivingWindowClose STRING(5)`,

		// OrderManifests + interleaved OrderManifestStops — supplier/warehouse
		// scoped manifest aggregate. Distinct from the factory-only
		// FactoryTruckManifests table. State machine owned by the supplier
		// dispatch flow:
		//   DRAFT → READY_FOR_LOADING → LOADING → SEALED → DISPATCHED →
		//   COMPLETED (or CANCELLED at any pre-DISPATCHED step).
		// OptimizerSource records which engine produced the draft so we can
		// measure fallback rate over time.
		`CREATE TABLE OrderManifests (
			ManifestId           STRING(36) NOT NULL,
			SupplierId           STRING(36) NOT NULL,
			VehicleId            STRING(36) NOT NULL,
			DriverId             STRING(36),
			HomeNodeType         STRING(20) NOT NULL,
			HomeNodeId           STRING(36) NOT NULL,
			State                STRING(20) NOT NULL DEFAULT ('DRAFT'),
			OptimizerSource      STRING(20) NOT NULL,
			TotalVolumeVU        FLOAT64    NOT NULL DEFAULT (0),
			StopCount            INT64      NOT NULL DEFAULT (0),
			EstimatedDurationSec INT64,
			EstimatedDistanceM   INT64,
			SolveTimeMs          INT64,
			Version              INT64      NOT NULL DEFAULT (1),
			CreatedAt            TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
			CONSTRAINT CHK_OrderManifestState CHECK (
				State IN ('DRAFT','READY_FOR_LOADING','LOADING','SEALED','DISPATCHED','COMPLETED','CANCELLED')
			)
		) PRIMARY KEY (ManifestId)`,
		`CREATE TABLE OrderManifestStops (
			ManifestId    STRING(36) NOT NULL,
			SequenceIndex INT64      NOT NULL,
			OrderId       STRING(36) NOT NULL,
			ResidualVU    FLOAT64,
			ArrivalSec    INT64,
			DepartureSec  INT64,
		) PRIMARY KEY (ManifestId, SequenceIndex),
		  INTERLEAVE IN PARENT OrderManifests ON DELETE CASCADE`,
		`CREATE INDEX Idx_OrderManifests_BySupplierState
			ON OrderManifests(SupplierId, State)`,
		`CREATE INDEX Idx_OrderManifests_ByVehicleState
			ON OrderManifests(VehicleId, State)`,
		`CREATE INDEX Idx_OrderManifestStops_ByOrderId
			ON OrderManifestStops(OrderId)`,
	}

	log.Printf("Checking database: %s", dbName)
	_, err = databaseAdmin.GetDatabase(ctx, &databasepb.GetDatabaseRequest{
		Name: dbName,
	})

	if err != nil {
		log.Println("Database not found, creating schema...")
		req := &databasepb.CreateDatabaseRequest{
			Parent:          instanceName,
			CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", cfg.SpannerDatabase),
			ExtraStatements: ddlStatements,
		}
		op, err := databaseAdmin.CreateDatabase(ctx, req)
		if err != nil {
			log.Fatalf("Failed to trigger database creation: %v", err)
		}
		if _, err := op.Wait(ctx); err != nil {
			log.Fatalf("Failed to create database and apply DDL: %v", err)
		}
		log.Println("Database schema successfully generated.")
	} else {
		log.Println("Database already exists. Skipping DDL...")
	}

	// 5. Insert Seed Data
	log.Println("Inserting Seed Data...")
	spannerClient, err := spanner.NewClient(ctx, dbName, opts...)
	if err != nil {
		log.Fatalf("Failed to create native Spanner client: %v", err)
	}
	defer spannerClient.Close()

	setupSeedData(ctx, spannerClient)

	// 6. Kafka Topic Initialization
	log.Println("Initializing Kafka Topics...")
	setupKafkaTopic(cfg.KafkaBrokerAddress, "orders.completed")
	setupKafkaTopic(cfg.KafkaBrokerAddress, "orders.dispatched")

	log.Println("Setup and Seed Script execution complete. Environment is READY.")
}

func setupSeedData(ctx context.Context, client *spanner.Client) {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check if data already exists
		iter := txn.Query(ctx, spanner.Statement{SQL: "SELECT RetailerId FROM Retailers WHERE RetailerId = 'retailer-123'"})
		_, err := iter.Next()
		iter.Stop()
		if err != iterator.Done {
			return err // Already seeded or real error
		}

		// ── Stable IDs ────────────────────────────────────────────────────
		const (
			// Suppliers
			suppCoca     = "SUP-COCA-001"
			suppNestle   = "SUP-NEST-001"
			suppPepsi    = "SUP-PEPS-001"
			suppUnilever = "SUP-UNIL-001"
			// Retailers
			retSamarkand = "retailer-123"
			retTashkent  = "RET-TASH-001"
			retBukhara   = "RET-BUKH-001"
			retNamangan  = "RET-NMGN-001"
			retFergana   = "RET-FERG-001"
			// Drivers
			drvAmir = "DRV-AMIR-001"
			drvRust = "DRV-RUST-001"
			// Categories
			catBeverages = "CAT-BVRG-001"
			catDairy     = "CAT-DIRY-001"
			catSnacks    = "CAT-SNCK-001"
			catHygiene   = "CAT-HYGN-001"
		)

		mutations := []*spanner.Mutation{}

		// ═══════════════════════════════════════════════════════════════════
		// CATEGORIES
		// ═══════════════════════════════════════════════════════════════════
		for _, c := range []struct {
			id, name, icon string
			sort           int64
		}{
			{catBeverages, "Beverages", "local_drink", 1},
			{catDairy, "Dairy & Juice", "egg", 2},
			{catSnacks, "Snacks & Confectionery", "cookie", 3},
			{catHygiene, "Hygiene & Household", "cleaning_services", 4},
		} {
			mutations = append(mutations, spanner.Insert("Categories",
				[]string{"CategoryId", "Name", "Icon", "SortOrder", "CreatedAt"},
				[]interface{}{c.id, c.name, c.icon, c.sort, spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// SUPPLIERS
		// ═══════════════════════════════════════════════════════════════════
		for _, s := range []struct {
			id, name, category string
		}{
			{suppCoca, "Coca-Cola Uzbekistan", "Beverages"},
			{suppNestle, "Nestlé Central Asia", "Dairy & Juice"},
			{suppPepsi, "PepsiCo Tashkent", "Beverages"},
			{suppUnilever, "Unilever UZ", "Hygiene & Household"},
		} {
			mutations = append(mutations, spanner.Insert("Suppliers",
				[]string{"SupplierId", "Name", "Category", "CreatedAt"},
				[]interface{}{s.id, s.name, s.category, spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// RETAILERS (5 shops across Uzbekistan)
		// ═══════════════════════════════════════════════════════════════════
		for _, r := range []struct {
			id, name, loc, tin, status, phone, passwordHash string
		}{
			{retSamarkand, "Target Samarkand", "POINT(66.9750 39.6270)", "UZ1001001", "VERIFIED", "+998901234567", "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"},
			{retTashkent, "Korzinka Yunusabad", "POINT(69.2401 41.2995)", "UZ2002002", "VERIFIED", "+998901234568", "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"},
			{retBukhara, "Makro Bukhara", "POINT(64.4213 39.7745)", "UZ3003003", "VERIFIED", "+998901234569", "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"},
			{retNamangan, "Havas Namangan", "POINT(71.6726 40.9983)", "UZ4004004", "VERIFIED", "+998901234570", "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"},
			{retFergana, "Oltin Bozor Fergana", "POINT(71.7909 40.3838)", "UZ5005005", "PENDING", "+998901234571", "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"},
		} {
			mutations = append(mutations, spanner.Insert("Retailers",
				[]string{"RetailerId", "Name", "ShopLocation", "TaxIdentificationNumber", "Status", "Phone", "PasswordHash"},
				[]interface{}{r.id, r.name, r.loc, r.tin, r.status, r.phone, r.passwordHash}))
		}
		// DRIVERS
		// ═══════════════════════════════════════════════════════════════════
		for _, d := range []struct{ id, name string }{
			{drvAmir, "Amir Karimov"},
			{drvRust, "Rustam Yuldashev"},
		} {
			mutations = append(mutations, spanner.Insert("Drivers",
				[]string{"DriverId", "Name", "CreatedAt"},
				[]interface{}{d.id, d.name, spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// SUPPLIER PRODUCTS (20 SKUs across 4 suppliers)
		// ═══════════════════════════════════════════════════════════════════
		type sku struct {
			id, suppId, name, desc, cat string
			sellByBlock                 bool
			unitsPerBlock, price        int64
		}
		skus := []sku{
			// Coca-Cola (Beverages)
			{"SKU-COKE-500", suppCoca, "Coca-Cola Classic 0.5L", "24-pack PET bottles", catBeverages, true, 24, 143760},
			{"SKU-COKE-15L", suppCoca, "Coca-Cola Classic 1.5L", "6-pack PET bottles", catBeverages, true, 6, 53940},
			{"SKU-FANTA-15L", suppCoca, "Fanta Orange 1.5L", "6-pack PET bottles", catBeverages, true, 6, 47940},
			{"SKU-SPRITE-2L", suppCoca, "Sprite 2.0L", "4-pack PET bottles", catBeverages, true, 4, 39960},
			{"SKU-COKEZR-500", suppCoca, "Coca-Cola Zero 0.5L", "24-pack PET", catBeverages, true, 24, 149760},
			// Nestlé (Dairy)
			{"SKU-NESCAFE-3IN1", suppNestle, "Nescafé 3-in-1 Classic", "Box of 48 sticks", catDairy, true, 48, 119520},
			{"SKU-KITKAT-4F", suppNestle, "KitKat 4-Finger", "Box of 24 bars", catSnacks, true, 24, 239760},
			{"SKU-MAGGI-NOOD", suppNestle, "Maggi Instant Noodles Chicken", "Carton of 40", catSnacks, true, 40, 99800},
			{"SKU-NESQUIK-1KG", suppNestle, "Nesquik Cocoa Powder 1kg", "Box of 6 cans", catDairy, true, 6, 179940},
			{"SKU-PURINA-DOG", suppNestle, "Purina Dog Chow 3kg", "Bag of 4", catSnacks, true, 4, 239960},
			// PepsiCo (Beverages)
			{"SKU-PEPSI-500", suppPepsi, "Pepsi 0.5L", "24-pack PET", catBeverages, true, 24, 137760},
			{"SKU-PEPSI-15L", suppPepsi, "Pepsi 1.5L", "6-pack PET", catBeverages, true, 6, 50940},
			{"SKU-7UP-15L", suppPepsi, "7UP 1.5L", "6-pack PET", catBeverages, true, 6, 47940},
			{"SKU-LAYS-ORIG", suppPepsi, "Lay's Original 150g", "Box of 20", catSnacks, true, 20, 119800},
			{"SKU-DORITOS-NAC", suppPepsi, "Doritos Nacho 180g", "Box of 16", catSnacks, true, 16, 127840},
			// Unilever (Hygiene)
			{"SKU-DOVE-SOAP", suppUnilever, "Dove Beauty Bar 100g", "Box of 48", catHygiene, true, 48, 239520},
			{"SKU-SIGNAL-TOOTH", suppUnilever, "Signal Toothpaste 100ml", "Box of 36", catHygiene, true, 36, 179640},
			{"SKU-SUNSILK-SHMP", suppUnilever, "Sunsilk Shampoo 400ml", "Box of 12", catHygiene, true, 12, 179880},
			{"SKU-REXONA-DEO", suppUnilever, "Rexona Deodorant 150ml", "Box of 24", catHygiene, true, 24, 359760},
			{"SKU-DOMEST-BLCH", suppUnilever, "Domestos Bleach 1L", "Box of 12", catHygiene, true, 12, 107880},
		}
		for _, s := range skus {
			mutations = append(mutations, spanner.Insert("SupplierProducts",
				[]string{"SkuId", "SupplierId", "Name", "Description", "SellByBlock", "UnitsPerBlock", "BasePrice", "IsActive", "CategoryId", "CreatedAt"},
				[]interface{}{s.id, s.suppId, s.name, s.desc, s.sellByBlock, s.unitsPerBlock, s.price, true, s.cat, spanner.CommitTimestamp}))
		}

		// Also insert into legacy Products table for backwards compat
		for _, s := range skus[:4] {
			mutations = append(mutations, spanner.Insert("Products",
				[]string{"ProductId", "Name", "Size", "PackQuantity", "Price"},
				[]interface{}{s.id, s.name, "", s.unitsPerBlock, big.NewRat(s.price, 100)}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// SUPPLIER INVENTORY (stock for all 20 SKUs)
		// ═══════════════════════════════════════════════════════════════════
		inventoryQty := map[string]int64{
			"SKU-COKE-500": 500, "SKU-COKE-15L": 300, "SKU-FANTA-15L": 250,
			"SKU-SPRITE-2L": 200, "SKU-COKEZR-500": 400,
			"SKU-NESCAFE-3IN1": 350, "SKU-KITKAT-4F": 600, "SKU-MAGGI-NOOD": 800,
			"SKU-NESQUIK-1KG": 150, "SKU-PURINA-DOG": 120,
			"SKU-PEPSI-500": 450, "SKU-PEPSI-15L": 280, "SKU-7UP-15L": 220,
			"SKU-LAYS-ORIG": 700, "SKU-DORITOS-NAC": 550,
			"SKU-DOVE-SOAP": 400, "SKU-SIGNAL-TOOTH": 350, "SKU-SUNSILK-SHMP": 200,
			"SKU-REXONA-DEO": 300, "SKU-DOMEST-BLCH": 250,
		}
		skuToSupplier := map[string]string{}
		for _, s := range skus {
			skuToSupplier[s.id] = s.suppId
		}
		for skuId, qty := range inventoryQty {
			mutations = append(mutations, spanner.Insert("SupplierInventory",
				[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{skuId, skuToSupplier[skuId], qty, spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// RETAILER ↔ SUPPLIER FAVORITES
		// ═══════════════════════════════════════════════════════════════════
		favs := [][2]string{
			{retSamarkand, suppCoca}, {retSamarkand, suppNestle}, {retSamarkand, suppPepsi},
			{retTashkent, suppCoca}, {retTashkent, suppNestle}, {retTashkent, suppPepsi}, {retTashkent, suppUnilever},
			{retBukhara, suppCoca}, {retBukhara, suppUnilever},
			{retNamangan, suppPepsi}, {retNamangan, suppNestle},
		}
		for _, f := range favs {
			mutations = append(mutations, spanner.Insert("RetailerSuppliers",
				[]string{"RetailerId", "SupplierId", "AddedAt"},
				[]interface{}{f[0], f[1], spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// RETAILER SETTINGS (auto-order)
		// ═══════════════════════════════════════════════════════════════════
		for _, r := range []string{retSamarkand, retTashkent, retBukhara, retNamangan} {
			mutations = append(mutations, spanner.Insert("RetailerGlobalSettings",
				[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
				[]interface{}{r, false, spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// ORDERS (12 orders across various states for dashboard data)
		// ═══════════════════════════════════════════════════════════════════
		type orderDef struct {
			id, retailer, driver, state, gateway string
			amount                               int64
		}
		orders := []orderDef{
			// Completed orders (historical)
			{"ORD-0001", retTashkent, drvAmir, "COMPLETED", "GLOBAL_PAY", 431_460},
			{"ORD-0002", retSamarkand, drvRust, "COMPLETED", "CASH", 287_520},
			{"ORD-0003", retBukhara, drvAmir, "COMPLETED", "GLOBAL_PAY", 599_280},
			{"ORD-0004", retTashkent, drvRust, "COMPLETED", "CASH", 143_760},
			{"ORD-0005", retNamangan, drvAmir, "COMPLETED", "GLOBAL_PAY", 239_760},
			// In-progress orders
			{"ORD-0006", retTashkent, drvAmir, "IN_TRANSIT", "GLOBAL_PAY", 323_520},
			{"ORD-0007", retSamarkand, drvRust, "LOADED", "CASH", 179_940},
			// Pending orders (visible in supplier order queue)
			{"ORD-0008", retBukhara, "", "PENDING", "GLOBAL_PAY", 479_040},
			{"ORD-0009", retTashkent, "", "PENDING", "CASH", 269_560},
			{"ORD-0010", retNamangan, "", "PENDING", "GLOBAL_PAY", 119_800},
			{"ORD-0011", retSamarkand, "", "PENDING", "CASH", 359_760},
			{"ORD-0012", retTashkent, "", "PENDING", "GLOBAL_PAY", 539_520},
		}
		for _, o := range orders {
			cols := []string{"OrderId", "RetailerId", "State", "Amount", "PaymentGateway", "CreatedAt"}
			vals := []interface{}{o.id, o.retailer, o.state, o.amount, o.gateway, spanner.CommitTimestamp}
			if o.driver != "" {
				cols = append(cols, "DriverId")
				vals = append(vals, o.driver)
			}
			mutations = append(mutations, spanner.Insert("Orders", cols, vals))
		}

		// ═══════════════════════════════════════════════════════════════════
		// ORDER LINE ITEMS
		// ═══════════════════════════════════════════════════════════════════
		type lineItem struct {
			orderId, skuId, status string
			qty, price             int64
		}
		items := []lineItem{
			// ORD-0001 (Completed)
			{"ORD-0001", "SKU-COKE-500", "DELIVERED", 3, 143_760},
			// ORD-0002 (Completed)
			{"ORD-0002", "SKU-FANTA-15L", "DELIVERED", 2, 47_940},
			{"ORD-0002", "SKU-SPRITE-2L", "DELIVERED", 2, 39_960},
			{"ORD-0002", "SKU-COKE-15L", "REJECTED_DAMAGED", 1, 53_940},
			// ORD-0003 (Completed)
			{"ORD-0003", "SKU-KITKAT-4F", "DELIVERED", 2, 239_760},
			{"ORD-0003", "SKU-NESCAFE-3IN1", "DELIVERED", 1, 119_520},
			// ORD-0004 (Completed)
			{"ORD-0004", "SKU-COKE-500", "DELIVERED", 1, 143_760},
			// ORD-0005 (Completed)
			{"ORD-0005", "SKU-KITKAT-4F", "DELIVERED", 1, 239_760},
			// ORD-0006 (In Transit)
			{"ORD-0006", "SKU-PEPSI-500", "PENDING", 2, 137_760},
			{"ORD-0006", "SKU-FANTA-15L", "PENDING", 1, 47_940},
			// ORD-0007 (Loaded)
			{"ORD-0007", "SKU-NESQUIK-1KG", "PENDING", 1, 179_940},
			// ORD-0008 (Pending — supplier sees these)
			{"ORD-0008", "SKU-DOVE-SOAP", "PENDING", 1, 239_520},
			{"ORD-0008", "SKU-SIGNAL-TOOTH", "PENDING", 1, 179_640},
			{"ORD-0008", "SKU-DOMEST-BLCH", "PENDING", 1, 107_880},
			// ORD-0009 (Pending)
			{"ORD-0009", "SKU-COKE-500", "PENDING", 1, 143_760},
			{"ORD-0009", "SKU-LAYS-ORIG", "PENDING", 1, 119_800},
			// ORD-0010 (Pending)
			{"ORD-0010", "SKU-LAYS-ORIG", "PENDING", 1, 119_800},
			// ORD-0011 (Pending)
			{"ORD-0011", "SKU-REXONA-DEO", "PENDING", 1, 359_760},
			// ORD-0012 (Pending)
			{"ORD-0012", "SKU-DOVE-SOAP", "PENDING", 1, 239_520},
			{"ORD-0012", "SKU-SUNSILK-SHMP", "PENDING", 1, 179_880},
			{"ORD-0012", "SKU-NESCAFE-3IN1", "PENDING", 1, 119_520},
		}
		for _, li := range items {
			mutations = append(mutations, spanner.Insert("OrderLineItems",
				[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Status"},
				[]interface{}{uuid.New().String(), li.orderId, li.skuId, li.qty, li.price, li.status}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// LEDGER ENTRIES (for completed orders)
		// ═══════════════════════════════════════════════════════════════════
		for i, o := range orders[:5] { // first 5 are COMPLETED
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{fmt.Sprintf("TXN-%04d-CR", i+1), o.id, o.retailer, o.amount, "CREDIT", spanner.CommitTimestamp}))
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{fmt.Sprintf("TXN-%04d-DB", i+1), o.id, "PEGASUS-TREASURY", o.amount, "DEBIT", spanner.CommitTimestamp}))
		}

		// ═══════════════════════════════════════════════════════════════════
		// LEDGER ANOMALIES (for admin reconciliation dashboard)
		// ═══════════════════════════════════════════════════════════════════
		mutations = append(mutations,
			spanner.Insert("LedgerAnomalies",
				[]string{"OrderId", "RetailerId", "SpannerUzs", "GatewayUzs", "GatewayProvider", "Status", "DetectedAt"},
				[]interface{}{"ORD-0003", retBukhara, int64(599_280), int64(589_280), "GLOBAL_PAY", "DELTA", spanner.CommitTimestamp}),
			spanner.Insert("LedgerAnomalies",
				[]string{"OrderId", "RetailerId", "SpannerUzs", "GatewayUzs", "GatewayProvider", "Status", "DetectedAt"},
				[]interface{}{"ORD-0005", retNamangan, int64(239_760), int64(0), "GLOBAL_PAY", "ORPHANED", spanner.CommitTimestamp}),
		)

		// ═══════════════════════════════════════════════════════════════════
		// PRICING TIERS (B2B discount rules)
		// ═══════════════════════════════════════════════════════════════════
		type tier struct {
			suppId, skuId, retailerTier string
			minPallets, discountPct     int64
		}
		tiers := []tier{
			{suppCoca, "SKU-COKE-500", "GOLD", 10, 15},
			{suppCoca, "SKU-COKE-500", "SILVER", 5, 8},
			{suppCoca, "SKU-FANTA-15L", "GOLD", 8, 12},
			{suppNestle, "SKU-KITKAT-4F", "GOLD", 5, 10},
			{suppPepsi, "SKU-LAYS-ORIG", "SILVER", 10, 7},
			{suppUnilever, "SKU-DOVE-SOAP", "GOLD", 3, 20},
		}
		for _, t := range tiers {
			mutations = append(mutations, spanner.Insert("PricingTiers",
				[]string{"TierId", "SupplierId", "SkuId", "MinPallets", "DiscountPct", "TargetRetailerTier", "IsActive"},
				[]interface{}{uuid.New().String(), t.suppId, t.skuId, t.minPallets, t.discountPct, t.retailerTier, true}))
		}

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		log.Fatalf("Failed to inject seed data: %v", err)
	}

	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("  SEED DATA INJECTED SUCCESSFULLY")
	log.Println("  • 4 Suppliers (Coca-Cola, Nestlé, PepsiCo, Unilever)")
	log.Println("  • 4 Categories (Beverages, Dairy, Snacks, Hygiene)")
	log.Println("  • 20 SKUs with inventory stock")
	log.Println("  • 5 Retailers across Uzbekistan")
	log.Println("  • 2 Drivers")
	log.Println("  • 12 Orders (5 completed, 2 in-progress, 5 pending)")
	log.Println("  • 21 Line items with delivery/damage statuses")
	log.Println("  • 10 Ledger entries + 2 anomalies")
	log.Println("  • 6 Pricing tiers")
	log.Println("═══════════════════════════════════════════════════════════")
}

func setupKafkaTopic(brokerAddress string, topic string) {
	conn, err := kafka.Dial("tcp", brokerAddress)
	if err != nil {
		log.Fatalf("Failed to connect to naive Kafka bootstrap %s: %v", brokerAddress, err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Fatalf("Failed to extract controller metrics: %v", err)
	}

	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		log.Fatalf("Failed to dial controller: %v", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil && !strings.Contains(err.Error(), "Topic with this name already exists") {
		log.Printf("Failed to create topic %s: %v", topic, err)
	} else {
		log.Printf("Topic '%s' prepared successfully.", topic)
	}
}
