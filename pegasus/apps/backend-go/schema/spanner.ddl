-- ============================================================
-- PEGASUS — CANONICAL SPANNER SCHEMA
-- Last updated: Phase 6 — Abstract Volumetric Units (VU)
-- ============================================================
-- VU SCALE REFERENCE:
--   1.0 VU = 1 standard case of 1L water bottles (universal baseline)
--   Tiny = 0.01 VU | Small = 0.1 VU | Medium = 0.5 VU
--   Standard = 1.0 VU | Bulky = 2.0 VU | Pallet = 50.0 VU
-- VEHICLE CLASS CAPACITY:
--   CLASS_A (Damass/Minivan) = 50 VU
--   CLASS_B (Transit Van)    = 150 VU
--   CLASS_C (Box Truck)      = 400 VU
-- ============================================================

CREATE TABLE Retailers (
    RetailerId              STRING(36)  NOT NULL,
    Name                    STRING(MAX) NOT NULL,
    Phone                   STRING(20),               -- UZ mobile +998XXXXXXXXX
    ShopName                STRING(MAX),              -- Display name for notifications
    ShopLocation            STRING(MAX),
    Latitude                FLOAT64,                  -- GPS latitude from registration map picker
    Longitude               FLOAT64,                  -- GPS longitude from registration map picker
    TaxIdentificationNumber STRING(MAX),
    Status                  STRING(20),
    PasswordHash            STRING(MAX),              -- Bcrypt hash for /v1/auth/retailer/login
    FcmToken                STRING(MAX),              -- Firebase push token (nullable — triggers Telegram fallback)
    TelegramChatId          STRING(MAX),              -- Telegram chat_id for last-resort alerts
    ReceivingWindowOpen     STRING(10),               -- HH:MM — start of receiving window (e.g. "09:00")
    ReceivingWindowClose    STRING(10),               -- HH:MM — end of receiving window (e.g. "18:00")
    AccessType              STRING(30),               -- STREET_PARKING | ALLEYWAY | LOADING_DOCK
    StorageCeilingHeightCM  FLOAT64,                  -- Maximum ceiling height in cm (for tall pallets/vehicles)
    CreatedAt               TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId);

CREATE INDEX Idx_Retailers_ByPhone ON Retailers(Phone);

-- ALTER TABLE statements for existing deployments (run once against emulator):
-- ALTER TABLE Retailers ADD COLUMN ShopName STRING(MAX);
-- ALTER TABLE Retailers ADD COLUMN PasswordHash STRING(MAX);
-- ALTER TABLE Retailers ADD COLUMN FcmToken STRING(MAX);
-- ALTER TABLE Retailers ADD COLUMN TelegramChatId STRING(MAX);

-- ── PHASE F (Registration Expansion) migrations:
-- ALTER TABLE Retailers ADD COLUMN ReceivingWindowOpen STRING(10);
-- ALTER TABLE Retailers ADD COLUMN ReceivingWindowClose STRING(10);
-- ALTER TABLE Retailers ADD COLUMN AccessType STRING(30);
-- ALTER TABLE Retailers ADD COLUMN StorageCeilingHeightCM FLOAT64;

CREATE TABLE Drivers (
    DriverId        STRING(36)  NOT NULL,
    Name            STRING(MAX) NOT NULL,
    Phone           STRING(20),
    PinHash         STRING(MAX),               -- Bcrypt hash of supplier-generated 6-digit PIN
    SupplierId      STRING(36),                -- Owning supplier (fleet provisioning)
    DriverType      STRING(20),                -- IN_HOUSE | CONTRACTOR
    VehicleType     STRING(50),                -- Box Truck, Semi-Trailer, Refrigerated, etc.
    LicensePlate    STRING(30),
    IsActive          BOOL,
    MaxPalletCapacity INT64,                    -- DEPRECATED: use Vehicles.MaxVolumeVU
    VehicleId         STRING(36),                -- FK → Vehicles.VehicleId (nullable = unassigned)
    CurrentLocation   STRING(MAX),
    TruckStatus       STRING(20) NOT NULL DEFAULT ('AVAILABLE'), -- AVAILABLE | LOADING | READY | IN_TRANSIT | RETURNING | MAINTENANCE
    DepartedAt        TIMESTAMP,                 -- Exact departure timestamp for ETA calculations
    CreatedAt         TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (DriverId);

CREATE INDEX Idx_Drivers_BySupplierId ON Drivers(SupplierId);
CREATE INDEX Idx_Drivers_ByPhone ON Drivers(Phone);

-- ── FLEET VEHICLES (PHASE 6: VOLUMETRIC UNITS) ────────────────────────────────
-- Trucks are independent entities, decoupled from drivers.
-- A driver is ASSIGNED to a vehicle; capacity lives on the vehicle.
CREATE TABLE Vehicles (
    VehicleId     STRING(36)  NOT NULL,
    SupplierId    STRING(36)  NOT NULL,
    VehicleClass  STRING(10)  NOT NULL,    -- CLASS_A | CLASS_B | CLASS_C
    Label         STRING(100),             -- Nickname: "White Damass #3"
    LicensePlate  STRING(30),
    MaxVolumeVU   FLOAT64     NOT NULL,    -- 50.0 | 150.0 | 400.0
    LengthCM      FLOAT64,                  -- Physical length in cm (nullable — existing vehicles use direct MaxVolumeVU)
    WidthCM       FLOAT64,                  -- Physical width in cm
    HeightCM      FLOAT64,                  -- Physical height in cm  (VU = L×W×H / 5000)
    IsActive      BOOL        NOT NULL DEFAULT (true),
    CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (VehicleId);

CREATE INDEX Idx_Vehicles_BySupplier ON Vehicles(SupplierId);

-- Migration: driver↔vehicle assignment (nullable — driver can exist without truck)
-- ALTER TABLE Drivers ADD COLUMN VehicleId STRING(36);

-- Legacy columns (deprecated — capacity now lives on Vehicles.MaxVolumeVU):
-- ALTER TABLE Drivers ADD COLUMN MaxPalletCapacity INT64;
-- ALTER TABLE SupplierProducts ADD COLUMN PalletFootprint FLOAT64;

-- ── PHASE A (Dimensional VU Engine) migrations (run once against existing clusters):
-- ALTER TABLE Vehicles ADD COLUMN LengthCM FLOAT64;
-- ALTER TABLE Vehicles ADD COLUMN WidthCM FLOAT64;
-- ALTER TABLE Vehicles ADD COLUMN HeightCM FLOAT64;

CREATE TABLE Orders (
    OrderId        STRING(36)  NOT NULL,
    RetailerId     STRING(36)  NOT NULL,
    DriverId       STRING(36),
    SupplierId     STRING(36),
    InvoiceId      STRING(36),
    State          STRING(30)  NOT NULL, -- PENDING | LOADED | IN_TRANSIT | ARRIVED | COMPLETED
    Amount         INT64,
    Currency       STRING(3)   NOT NULL DEFAULT ('UZS'),
    GlobalPayntGateway STRING(MAX),
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
    GlobalPayntStatus   STRING(30)  NOT NULL DEFAULT ('PENDING'), -- PENDING | AUTHORIZED | PENDING_CASH_COLLECTION | AWAITING_GATEWAY_WEBHOOK | PAID | FAILED
    CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
                     'ARRIVING', 'ARRIVED', 'ARRIVED_SHOP_CLOSED', 'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION',
                     'COMPLETED', 'CANCELLED', 'CANCEL_REQUESTED', 'NO_CAPACITY', 'DELIVERED_ON_CREDIT',
                     'SCHEDULED', 'AUTO_ACCEPTED', 'QUARANTINE', 'STALE_AUDIT', 'REFUNDED')),
    CONSTRAINT CHK_GlobalPayntStatus CHECK (GlobalPayntStatus IN ('PENDING', 'AUTHORIZED', 'PENDING_CASH_COLLECTION', 'AWAITING_GATEWAY_WEBHOOK', 'PAID', 'FAILED'))
) PRIMARY KEY (OrderId);

CREATE INDEX IDX_Orders_RetailerId ON Orders(RetailerId);
CREATE INDEX IDX_Orders_DriverId   ON Orders(DriverId);
CREATE INDEX Idx_Orders_InvoiceId ON Orders(InvoiceId);
CREATE INDEX Idx_Orders_SupplierId ON Orders(SupplierId);
CREATE INDEX Idx_Orders_ByScheduleShardStateDate ON Orders(ScheduleShard, State, RequestedDeliveryDate DESC);

CREATE TABLE MasterInvoices (
    InvoiceId           STRING(36)  NOT NULL,
    RetailerId          STRING(36)  NOT NULL,
    Total               INT64       NOT NULL,
    Currency            STRING(3)   NOT NULL DEFAULT ('UZS'),
    State               STRING(20)  NOT NULL,
    OrderId             STRING(36),
    GlobalPayTransactionId  STRING(64),
    GlobalPayntMode         STRING(20),              -- ELECTRONIC | CASH
    CollectorDriverId   STRING(36),              -- Driver who collected cash (null for electronic)
    CollectedAt         TIMESTAMP,               -- When cash was physically collected
    CollectionLat       FLOAT64,                 -- GPS lat at cash collection
    CollectionLng       FLOAT64,                 -- GPS lng at cash collection
    GeofenceDistanceM   FLOAT64,                 -- Distance to retailer at collection (meters)
    CustodyStatus       STRING(20),              -- HELD_BY_DRIVER | DEPOSITED | null for electronic
    CreatedAt           TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (InvoiceId);

CREATE INDEX Idx_MasterInvoice_Retailer ON MasterInvoices(RetailerId);
CREATE INDEX Idx_MasterInvoice_OrderId ON MasterInvoices(OrderId);
CREATE INDEX Idx_MasterInvoice_GlobalPayTxn ON MasterInvoices(GlobalPayTransactionId);

CREATE TABLE Products (
    ProductId    STRING(36)  NOT NULL,
    Name         STRING(255) NOT NULL,
    Size         STRING(50),
    PackQuantity INT64,
    Price        NUMERIC,
    ImageUrl     STRING(MAX)
) PRIMARY KEY (ProductId);

-- ── SUPPLIER CATALOG (PHASE 4: SUPPLIER COCKPIT) ──────────────────────────
CREATE TABLE SupplierProducts (
    SkuId           STRING(50)  NOT NULL,
    SupplierId      STRING(36)  NOT NULL,
    Name            STRING(255) NOT NULL,
    Description     STRING(MAX),
    ImageUrl        STRING(MAX),            -- The public Google Cloud Storage URL
    SellByBlock     BOOL        NOT NULL,   -- TRUE if wholesale only
    UnitsPerBlock   INT64       NOT NULL,   -- e.g., 24 bottles in a block
    BasePrice       INT64       NOT NULL,   -- Price per Block (or per unit if SellByBlock is false)
    Currency        STRING(3)   NOT NULL DEFAULT ('UZS'),
    PalletFootprint  FLOAT64,                  -- DEPRECATED: use VolumetricUnit
    VolumetricUnit   FLOAT64     NOT NULL DEFAULT (1.0), -- Abstract VU (1.0 = standard case of 1L water bottles)
    LengthCM         FLOAT64,                  -- Physical length in cm (nullable — existing SKUs use direct VolumetricUnit)
    WidthCM          FLOAT64,                  -- Physical width in cm
    HeightCM         FLOAT64,                  -- Physical height in cm  (VU = L×W×H / 5000)
    MinimumOrderQty  INT64       NOT NULL DEFAULT (1),   -- MOQ: minimum units per order (AI Worker enforces ceil)
    StepSize         INT64       NOT NULL DEFAULT (1),   -- Order must be a multiple of this (e.g. 24 for a case)
    IsActive         BOOL        NOT NULL,
    CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SkuId);

-- Index for the Supplier Dashboard to instantly load their catalog
CREATE INDEX Idx_Products_BySupplier ON SupplierProducts(SupplierId);

-- ── PHASE A (Dimensional VU Engine) migrations:
-- ALTER TABLE SupplierProducts ADD COLUMN LengthCM FLOAT64;
-- ALTER TABLE SupplierProducts ADD COLUMN WidthCM FLOAT64;
-- ALTER TABLE SupplierProducts ADD COLUMN HeightCM FLOAT64;

-- ── STANDALONE LINE ITEMS — ANALYTICS-SAFE ────────────────────────────────
-- ARCHITECTURAL DECISION (Phase 4):
--   Previous design: OrderItems INTERLEAVE IN PARENT Orders
--   Problem: cross-order SKU aggregations caused full Orders table scans.
--   Solution: Standalone table with distributed PK (LineItemId = UUID).
--             Two secondary indexes cover both access patterns with O(1) reads.
--
--   Idx_OrderItems_ByOrder → Driver App:      pull all items for one order
--   Idx_OrderItems_BySku   → Analytics Engine: sum SKU volume across country
-- ─────────────────────────────────────────────────────────────────────────

CREATE TABLE OrderLineItems (
    LineItemId   STRING(36) NOT NULL,
    OrderId      STRING(36) NOT NULL,
    SkuId        STRING(50) NOT NULL,
    Quantity     INT64      NOT NULL,
    UnitPrice    INT64      NOT NULL,
    Currency     STRING(3)  NOT NULL DEFAULT ('UZS'),
    Status       STRING(20) NOT NULL  -- PENDING | DELIVERED | REJECTED_DAMAGED
) PRIMARY KEY (LineItemId);

CREATE INDEX Idx_OrderItems_ByOrder ON OrderLineItems(OrderId);
CREATE INDEX Idx_OrderItems_BySku   ON OrderLineItems(SkuId);

-- ── LEDGER ────────────────────────────────────────────────────────────────

CREATE TABLE LedgerEntries (
    TransactionId STRING(100) NOT NULL,
    OrderId       STRING(36)  NOT NULL,
    AccountId     STRING(MAX) NOT NULL,
    Amount        INT64       NOT NULL,
    Currency      STRING(3)   NOT NULL DEFAULT ('UZS'),
    EntryType     STRING(20)  NOT NULL,
    Status        STRING(32),
    IdempotencyKey STRING(64),
    GatewayChargeId STRING(128),
    FailureReason   STRING(MAX),
    SettledAt     TIMESTAMP,
    CreatedAt     TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TransactionId);

-- Idempotency guard: prevents duplicate ledger writes on DLQ replay.
-- Identical (OrderId, EntryType) pairs are rejected at the DB layer on any replay.
CREATE UNIQUE INDEX Idx_Ledger_UniqueOrderEntry ON LedgerEntries(OrderId, EntryType);

-- Allows the gateway worker to query pending charges efficiently.
CREATE INDEX Idx_Ledger_ByStatus ON LedgerEntries(Status);

-- ── FINANCIAL RECONCILIATION (PHASE 5) ────────────────────────────────────
CREATE TABLE LedgerAnomalies (
    OrderId STRING(36) NOT NULL,
    RetailerId STRING(36) NOT NULL,
    SpannerAmount INT64 NOT NULL,
    GatewayAmount INT64 NOT NULL,
    Currency      STRING(3) NOT NULL DEFAULT ('UZS'),
    GatewayProvider STRING(20) NOT NULL, -- 'GLOBAL_PAY' or 'CASH'
    Status STRING(20) NOT NULL,          -- 'DELTA' or 'ORPHANED'
    DetectedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (OrderId);

-- Index for the Admin Dashboard to fetch unresolved anomalies quickly
CREATE INDEX Idx_Anomalies_ByStatus ON LedgerAnomalies(Status);

-- ── PHASE 2: PROXIMITY ENGINE — ARRIVING STATE ───────────────────────────────
-- The CHECK constraint on Orders.State now includes PENDING_REVIEW and ARRIVING.
-- Migration (run against existing deployments):
--   ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
--   ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--     CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_GLOBAL_PAYNT', 'COMPLETED', 'CANCELLED', 'SCHEDULED'));

-- ── PHASE 7: QUARANTINE STATE (REVERSE LOGISTICS) ────────────────────────────
-- QUARANTINE is the state for orders physically returned to the depot undelivered.
-- Triggered by: Desert Protocol REJECTED_DAMAGED sync OR driver route-complete endpoint.
-- VU is NOT released while in QUARANTINE. Supplier must reconcile via /v1/inventory/reconcile-returns.
-- Migration (run against existing deployments):
--   ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
--   ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--     CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED',
--                      'AWAITING_GLOBAL_PAYNT', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'QUARANTINE'));

-- ── PHASE 14: DISPATCHED STATE (PAYLOAD TERMINAL SEAL → DISPATCHED) ──────────
-- DISPATCHED is set when the payload terminal seals an order (JIT token attached).
-- Pipeline: LOADED (approved) → DISPATCHED (sealed on truck) → IN_TRANSIT (driver departs).
-- Migration (run against existing deployments):
--   ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
--   ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--     CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED',
--                      'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'QUARANTINE'));

-- Composite index for fleet/manifest queries that filter by state + route
CREATE INDEX Idx_Orders_ByStateAndRoute ON Orders(State, RouteId);

-- ── EMPATHY ENGINE — HIERARCHICAL AUTO-ORDER SETTINGS ─────────────────────────
-- Three interleaved tables co-located on the same Spanner node per RetailerId.
-- The Field General Cron resolves the hierarchy in one fast read:
-- Product override > Supplier override > Global master switch.

CREATE TABLE RetailerGlobalSettings (
    RetailerId             STRING(36) NOT NULL,
    GlobalAutoOrderEnabled BOOL       NOT NULL,
    AnalyticsStartDate     TIMESTAMP,                -- NULL = use all history; set = "start fresh" cut-off
    UpdatedAt              TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId);

CREATE TABLE RetailerSupplierSettings (
    RetailerId       STRING(36) NOT NULL,
    SupplierId       STRING(36) NOT NULL,
    AutoOrderEnabled BOOL       NOT NULL,
    UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, SupplierId),
  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE;

CREATE TABLE RetailerProductSettings (
    RetailerId       STRING(36) NOT NULL,
    ProductId        STRING(36) NOT NULL,
    AutoOrderEnabled BOOL       NOT NULL,
    UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, ProductId),
  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE;

-- Variant (SKU) level override — highest precedence in the hierarchy.
-- Resolution: Variant > Product > Supplier > Global > OFF
CREATE TABLE RetailerVariantSettings (
    RetailerId       STRING(36) NOT NULL,
    SkuId            STRING(36) NOT NULL,
    AutoOrderEnabled BOOL       NOT NULL,
    UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, SkuId),
  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE;

-- ── AI PREDICTIONS + SKU-LEVEL LINE ITEMS ─────────────────────────────────
-- AIPredictions stores top-level prediction metadata (one row per prediction batch).
-- AIPredictionItems stores individual SKU forecasts (interleaved for fast co-reads).
-- Status: DORMANT (auto-order off, AI still analyzes) | WAITING (armed) | FIRED | REJECTED

CREATE TABLE AIPredictions (
    PredictionId      STRING(36) NOT NULL,
    RetailerId        STRING(36) NOT NULL,
    PredictedAmount    INT64     NOT NULL,
    Currency           STRING(3) NOT NULL DEFAULT ('UZS'),
    TriggerDate       TIMESTAMP,
    TriggerShard      INT64      NOT NULL DEFAULT (0),
    Status            STRING(20) NOT NULL,  -- DORMANT | WAITING | FIRED | REJECTED
    CreatedAt         TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (PredictionId);

CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId);
CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC);

CREATE TABLE AIPredictionItems (
    PredictionId      STRING(36) NOT NULL,
    PredictionItemId  STRING(36) NOT NULL,
    SkuId             STRING(50) NOT NULL,
    PredictedQuantity INT64      NOT NULL,
    UnitPrice         INT64      NOT NULL,
    Currency          STRING(3)  NOT NULL DEFAULT ('UZS'),
    CreatedAt         TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (PredictionId, PredictionItemId),
  INTERLEAVE IN PARENT AIPredictions ON DELETE CASCADE;

CREATE INDEX Idx_PredictionItems_BySku ON AIPredictionItems(SkuId);

-- ── PHASE 6: RETAILER CATALOG & SUPPLIER DISCOVERY ────────────────────────

-- Categories for product taxonomy (Beverages, Dairy, Snacks, etc.)
CREATE TABLE Categories (
    CategoryId   STRING(36)  NOT NULL,
    Name         STRING(255) NOT NULL,
    Icon         STRING(100),              -- SF Symbol / Material icon name
    SortOrder    INT64       NOT NULL DEFAULT (0),
    CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (CategoryId);

-- Add Category reference to SupplierProducts
-- ALTER TABLE SupplierProducts ADD COLUMN CategoryId STRING(36);

-- Retailer-Supplier relationship (favorites / "My Suppliers")
CREATE TABLE RetailerSuppliers (
    RetailerId   STRING(36) NOT NULL,
    SupplierId   STRING(36) NOT NULL,
    AddedAt      TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, SupplierId);

CREATE INDEX Idx_RetailerSuppliers_ByRetailer ON RetailerSuppliers(RetailerId);

-- Suppliers table (supplier identity — sourced from onboarding)
CREATE TABLE Suppliers (
    SupplierId           STRING(36)  NOT NULL,
    Name                 STRING(255) NOT NULL,
    LogoUrl              STRING(MAX),
    Category             STRING(100),              -- Primary category (e.g. "Beverages")
    Phone                STRING(20),               -- Login phone number
    Email                STRING(MAX),              -- Contact email
    PasswordHash         STRING(MAX),              -- Bcrypt hash for /v1/auth/supplier/login
    TaxId                STRING(MAX),              -- Tax identification number (onboarding)
    ContactPerson        STRING(MAX),              -- Primary contact full name
    CompanyRegNumber     STRING(MAX),              -- Company registration / STIR number
    BillingAddress       STRING(MAX),              -- Legal / billing address
    IsConfigured         BOOL,                     -- TRUE after /v1/supplier/configure completes
    OperatingCategories  ARRAY<STRING(MAX)>,       -- Category IDs this supplier operates in
    WarehouseLocation    STRING(MAX),              -- Warehouse address text
    WarehouseLat         FLOAT64,                  -- Warehouse GPS latitude
    WarehouseLng         FLOAT64,                  -- Warehouse GPS longitude
    BankName             STRING(MAX),              -- Payout bank name
    AccountNumber        STRING(MAX),              -- Bank account / IBAN
    CardNumber           STRING(MAX),              -- Card number for payouts (masked in API)
    GlobalPayntGateway       STRING(20),               -- GLOBAL_PAY | CASH | BANK_TRANSFER
    OperatingSchedule    JSON,                     -- {"mon":{"open":"09:00","close":"18:00"},...} — null = always open
    ManualOffShift           BOOL        NOT NULL DEFAULT (false),  -- Master override: true = force CLOSED regardless of schedule
    FleetColdChainCompliant  BOOL,                     -- TRUE if fleet is capable of cold-chain delivery
    PalletizationStandard    STRING(30),               -- LOOSE_CARTONS | EURO_PALLETS | MIXED
    CreatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SupplierId);

CREATE INDEX Idx_Suppliers_ByPhone ON Suppliers(Phone);

-- ── PHASE F (Registration Expansion) migrations:
-- ALTER TABLE Suppliers ADD COLUMN FleetColdChainCompliant BOOL;
-- ALTER TABLE Suppliers ADD COLUMN PalletizationStandard STRING(30);

-- Platform-wide operating categories for mobile app onboarding
CREATE TABLE PlatformCategories (
    CategoryId    STRING(36)  NOT NULL,
    DisplayName   STRING(MAX) NOT NULL,
    IconUrl       STRING(MAX),
    DisplayOrder  INT64       NOT NULL DEFAULT (0)
) PRIMARY KEY (CategoryId);

-- ── ADMIN USERS ──────────────────────────────────────────────────────────
CREATE TABLE Admins (
    AdminId       STRING(36)  NOT NULL,
    Email         STRING(MAX) NOT NULL,
    PasswordHash  STRING(MAX) NOT NULL,
    DisplayName   STRING(MAX),
    CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (AdminId);

CREATE UNIQUE INDEX Idx_Admins_ByEmail ON Admins(Email);

-- ── INVENTORY LEDGER (PHASE 7: STOCK LOCKING) ────────────────────────────────
-- Keyed by ProductId (SkuId). Read + decremented inside the unified checkout
-- ReadWriteTransaction. Spanner serializes concurrent reads on the same key,
-- so two retailers can never oversell the last pallet.

CREATE TABLE SupplierInventory (
    ProductId          STRING(36) NOT NULL,
    SupplierId         STRING(36) NOT NULL,
    QuantityAvailable  INT64      NOT NULL,
    UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (ProductId);

CREATE INDEX Idx_Inventory_BySupplier ON SupplierInventory(SupplierId);

-- ── INVENTORY AUDIT LOG (PHASE 8: SUPPLIER OPS) ──────────────────────────────
-- Every stock adjustment (restock, damage write-off, correction, return) is
-- immutably logged here. HandleInventoryAuditLog reads last 100 entries.

CREATE TABLE InventoryAuditLog (
    AuditId      STRING(36)  NOT NULL,
    ProductId    STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    AdjustedBy   STRING(36)  NOT NULL,
    PreviousQty  INT64       NOT NULL,
    NewQty       INT64       NOT NULL,
    Delta        INT64       NOT NULL,
    Reason       STRING(50)  NOT NULL,
    AdjustedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (AuditId);

CREATE INDEX Idx_AuditLog_BySupplier ON InventoryAuditLog(SupplierId);
CREATE INDEX Idx_AuditLog_ByProduct  ON InventoryAuditLog(ProductId);

-- ── SUPPLIER RETURNS (PHASE 9: PARTIAL-QTY RECONCILIATION) ────────────────
-- Every rejected item from driver delivery correction inserts a row here.
-- Warehouse managers process returns via /v1/supplier/returns endpoints.

CREATE TABLE SupplierReturns (
    ReturnId     STRING(36)  NOT NULL,
    OrderId      STRING(36)  NOT NULL,
    SkuId        STRING(50)  NOT NULL,
    RejectedQty  INT64       NOT NULL,
    Reason       STRING(50)  NOT NULL,  -- DAMAGED | MISSING | WRONG_ITEM | OTHER
    DriverNotes  STRING(MAX),
    CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (ReturnId);

CREATE INDEX Idx_Returns_ByOrder ON SupplierReturns(OrderId);
CREATE INDEX Idx_Returns_BySku   ON SupplierReturns(SkuId);

-- ── WAREHOUSE STAFF (PHASE 10: PAYLOADER PROVISIONING) ────────────────────
-- Supplier-provisioned warehouse workers who operate the Payload Terminal.
-- Auth: Phone + 6-digit PIN (bcrypt), minted by supplier admin.

CREATE TABLE WarehouseStaff (
    WorkerId    STRING(36)  NOT NULL,
    SupplierId  STRING(36)  NOT NULL,
    Name        STRING(MAX) NOT NULL,
    Phone       STRING(20)  NOT NULL,
    PinHash     STRING(MAX) NOT NULL,
    IsActive    BOOL        NOT NULL,
    CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (WorkerId);

CREATE INDEX Idx_WarehouseStaff_BySupplierId ON WarehouseStaff(SupplierId);
CREATE INDEX Idx_WarehouseStaff_ByPhone      ON WarehouseStaff(Phone);

-- ── PHASE 11: SCHEDULED ORDERS + DISPATCH SNAPSHOT ────────────────────────
-- ALTER TABLE Orders ADD COLUMN RequestedDeliveryDate TIMESTAMP;
-- ALTER TABLE Orders ADD COLUMN ScheduleShard INT64;
-- ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
-- ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--   CHECK (State IN ('PENDING','LOADED','IN_TRANSIT','ARRIVING','ARRIVED','COMPLETED','EN_ROUTE','CANCELLED','SCHEDULED'));
-- CREATE INDEX Idx_Orders_ByScheduleShardStateDate ON Orders(ScheduleShard, State, RequestedDeliveryDate DESC);

-- ── PHASE 12: OPTIMISTIC CONCURRENCY + FREEZE LOCKS ──────────────────────
-- ALTER TABLE Orders ADD COLUMN Version INT64 NOT NULL DEFAULT (1);
-- ALTER TABLE Orders ADD COLUMN LockedUntil TIMESTAMP;

-- ── SUPPLIER GLOBAL_PAYNT GATEWAY VAULT (MULTI-VENDOR) ─────────────────────────
-- One supplier can have multiple active gateways (Cash + GlobalPay simultaneously).
-- SecretKey is stored AES-256-GCM encrypted; plaintext NEVER leaves the backend.
CREATE TABLE SupplierGlobalPayntConfigs (
    ConfigId     STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    GatewayName  STRING(20)  NOT NULL,  -- CASH | GLOBAL_PAY | GLOBAL_PAY
    MerchantId   STRING(MAX) NOT NULL,
    ServiceId    STRING(MAX),           -- Cash service_id (NULL for GlobalPay/Global Pay)
    SecretKey    BYTES(MAX)  NOT NULL,  -- AES-256-GCM encrypted at rest
    IsActive     BOOL        NOT NULL DEFAULT (true),
    CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'GLOBAL_PAY'))
) PRIMARY KEY (ConfigId);

CREATE INDEX Idx_SupplierGlobalPayntConfigs_BySupplierId ON SupplierGlobalPayntConfigs(SupplierId);
CREATE UNIQUE INDEX Idx_SupplierGlobalPayntConfigs_Unique ON SupplierGlobalPayntConfigs(SupplierId, GatewayName);

-- ── GLOBAL_PAYNT SESSIONS (PHASE 13: DURABLE GLOBAL_PAYNT SESSION ENGINE) ────────────
-- Canonical record for every global_paynt attempt lifecycle.
-- One active session per order. Retry creates a new attempt, not a new session.
-- Amount locks after ConfirmOffload (driver damage adjustments finalized).

CREATE TABLE GlobalPayntSessions (
    SessionId         STRING(36)  NOT NULL,
    OrderId           STRING(36)  NOT NULL,
    RetailerId        STRING(36)  NOT NULL,
    SupplierId        STRING(36)  NOT NULL,
    Gateway           STRING(20)  NOT NULL,     -- CASH | GLOBAL_PAY | CASH | UZCARD | GLOBAL_PAY
    LockedAmount      INT64       NOT NULL,     -- Immutable after creation (except AmendOrder adjustments)
    Currency          STRING(3)   NOT NULL DEFAULT ('UZS'),
    Status            STRING(30)  NOT NULL DEFAULT ('CREATED'),
    -- CREATED | PENDING | AUTHORIZED | SETTLED | FAILED | EXPIRED | CANCELLED
    CurrentAttemptNo  INT64       NOT NULL DEFAULT (0),
    InvoiceId         STRING(36),               -- FK → MasterInvoices.InvoiceId (nullable for CASH)
    RedirectUrl       STRING(MAX),              -- Deep-link URL for Cash/GlobalPay
    ProviderReference STRING(MAX),              -- Gateway-specific session token / provider reference
    AuthorizationId   STRING(MAX),              -- Global Pay global_paynt_id from auth hold (null for non-GP)
    AuthorizedAmount  INT64,                    -- Original authorized max amount in tiyins (null for non-GP)
    CapturedAmount    INT64,                    -- Actual captured amount in tiyins (set at capture time)
    ExpiresAt         TIMESTAMP,                -- Session expiry (nullable = no expiry)
    LastErrorCode     STRING(50),               -- Provider error code on failure
    LastErrorMessage  STRING(MAX),              -- Human-readable failure reason
    CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt         TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    SettledAt         TIMESTAMP,                -- When global_paynt was confirmed
    CONSTRAINT CHK_SessionStatus CHECK (Status IN ('CREATED', 'PENDING', 'AUTHORIZED', 'SETTLED', 'FAILED', 'EXPIRED', 'CANCELLED', 'PARTIALLY_PAID'))
) PRIMARY KEY (SessionId);

-- Active session lookup by order (most common query path)
CREATE INDEX Idx_GlobalPayntSessions_ByOrderId ON GlobalPayntSessions(OrderId);
-- Supplier ops: filter sessions by status for admin dashboards
CREATE INDEX Idx_GlobalPayntSessions_BySupplierId ON GlobalPayntSessions(SupplierId);
-- Retry queue: find all FAILED/EXPIRED sessions for ops follow-up
CREATE INDEX Idx_GlobalPayntSessions_ByStatus ON GlobalPayntSessions(Status);
-- Prevent duplicate active sessions for the same order
CREATE UNIQUE INDEX Idx_GlobalPayntSessions_ActiveOrder ON GlobalPayntSessions(OrderId, Status)
    WHERE Status IN ('CREATED', 'PENDING');

-- ── GLOBAL_PAYNT ATTEMPTS (AUDIT TRAIL) ───────────────────────────────────────────
-- Every retry/webhook callback creates a row here. Immutable after creation.
-- Provider transaction IDs live here, not on the session.

CREATE TABLE GlobalPayntAttempts (
    AttemptId            STRING(36)  NOT NULL,
    SessionId            STRING(36)  NOT NULL,
    AttemptNo            INT64       NOT NULL,
    Gateway              STRING(20)  NOT NULL,
    ProviderTransactionId STRING(64),           -- Cash trans_id or GlobalPay transaction ID
    Status               STRING(30)  NOT NULL DEFAULT ('INITIATED'),
    -- INITIATED | REDIRECTED | PROCESSING | SUCCESS | FAILED | CANCELLED | TIMED_OUT
    FailureCode          STRING(50),
    FailureMessage       STRING(MAX),
    RequestDigest        STRING(MAX),           -- SHA256 of request payload for audit
    StartedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    FinishedAt           TIMESTAMP,
    CONSTRAINT CHK_AttemptStatus CHECK (Status IN ('INITIATED', 'REDIRECTED', 'PROCESSING', 'SUCCESS', 'FAILED', 'CANCELLED', 'TIMED_OUT'))
) PRIMARY KEY (AttemptId);

-- Session lookup: get all attempts for a session
CREATE INDEX Idx_GlobalPayntAttempts_BySessionId ON GlobalPayntAttempts(SessionId);
-- Provider reconciliation: find attempt by provider transaction ID
CREATE INDEX Idx_GlobalPayntAttempts_ByProviderTxn ON GlobalPayntAttempts(ProviderTransactionId);

-- ── SYSTEM CONFIG (ADMIN) ─────────────────────────────────────────────────
CREATE TABLE SystemConfig (
    ConfigKey   STRING(100) NOT NULL,
    ConfigValue STRING(MAX) NOT NULL,
    UpdatedAt   TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (ConfigKey);

-- ── DEVICE TOKENS (FCM / APNs) ───────────────────────────────────────────
CREATE TABLE DeviceTokens (
    TokenId   STRING(36)  NOT NULL,
    UserId    STRING(36)  NOT NULL,
    Role      STRING(20)  NOT NULL, -- RETAILER | DRIVER | SUPPLIER | PAYLOADER
    Platform  STRING(10)  NOT NULL, -- ANDROID | IOS | WEB
    Token     STRING(MAX) NOT NULL,
    CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TokenId);

CREATE UNIQUE INDEX Idx_DeviceTokens_ByUserPlatform ON DeviceTokens(UserId, Platform);
CREATE INDEX Idx_DeviceTokens_ByUser ON DeviceTokens(UserId);

-- ── NOTIFICATIONS ─────────────────────────────────────────────────────────
CREATE TABLE Notifications (
    NotificationId STRING(36)  NOT NULL,
    RecipientId    STRING(36)  NOT NULL,
    RecipientRole  STRING(20)  NOT NULL,
    Type           STRING(50)  NOT NULL, -- ORDER_UPDATE | GLOBAL_PAYNT_REQUIRED | DELIVERY_ETA | SYSTEM
    Title          STRING(200) NOT NULL,
    Body           STRING(MAX),
    Payload        STRING(MAX), -- JSON metadata
    Channel        STRING(20), -- FCM | TELEGRAM | WS | EMAIL
    ReadAt         TIMESTAMP,
    CreatedAt      TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (NotificationId);

CREATE INDEX Idx_Notifications_ByRecipient ON Notifications(RecipientId, CreatedAt DESC);

-- ── AUDIT LOG ─────────────────────────────────────────────────────────────
CREATE TABLE AuditLog (
    LogId        STRING(36)  NOT NULL,
    ActorId      STRING(36)  NOT NULL,
    ActorRole    STRING(20)  NOT NULL,
    Action       STRING(50)  NOT NULL, -- CREATE | UPDATE | DELETE | STATE_CHANGE | LOGIN
    ResourceType STRING(30)  NOT NULL, -- ORDER | GLOBAL_PAYNT | ROUTE | DRIVER | VEHICLE
    ResourceId   STRING(36)  NOT NULL,
    Metadata     STRING(MAX), -- JSON (diff, old/new values, context)
    CreatedAt    TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (LogId);

CREATE INDEX Idx_AuditLog_ByResource ON AuditLog(ResourceType, ResourceId, CreatedAt DESC);
CREATE INDEX Idx_AuditLog_ByActor ON AuditLog(ActorId, CreatedAt DESC);

-- ── RETAILER CARTS (server-side persistence) ──────────────────────────────
CREATE TABLE RetailerCarts (
    CartId       STRING(36)  NOT NULL,
    RetailerId   STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    SkuId        STRING(36)  NOT NULL,
    Quantity     INT64       NOT NULL,
    UnitPrice    INT64       NOT NULL,
    Currency     STRING(3)   NOT NULL DEFAULT ('UZS'),
    AddedAt      TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (CartId);

CREATE INDEX Idx_RetailerCarts_ByRetailer ON RetailerCarts(RetailerId);
CREATE INDEX Idx_RetailerCarts_ByRetailerSupplier ON RetailerCarts(RetailerId, SupplierId);

-- ── SCHEDULED JOBS (distributed locking) ──────────────────────────────────
CREATE TABLE ScheduledJobs (
    JobId      STRING(36)  NOT NULL,
    JobName    STRING(100) NOT NULL,
    LastRunAt  TIMESTAMP,
    NextRunAt  TIMESTAMP,
    Status     STRING(20)  NOT NULL DEFAULT ('IDLE'), -- IDLE | RUNNING | FAILED
    LockHolder STRING(100),
    LockExpiry TIMESTAMP,
    CONSTRAINT CHK_JobStatus CHECK (Status IN ('IDLE', 'RUNNING', 'FAILED'))
) PRIMARY KEY (JobId);

-- ── MISSING INDEXES ON EXISTING TABLES ────────────────────────────────────
-- Retailer order listing: filter by state
CREATE INDEX Idx_Orders_ByRetailerState ON Orders(RetailerId, State);
-- GlobalPaynt session cleanup cron: find expired/stale sessions by status and time
CREATE INDEX Idx_GlobalPayntSessions_ByStatusExpiry ON GlobalPayntSessions(Status, ExpiresAt);
-- GlobalPaynt session listing by retailer (for pending-global_paynts endpoint)
CREATE INDEX Idx_GlobalPayntSessions_ByRetailerId ON GlobalPayntSessions(RetailerId);

-- ── RETAILER CARD TOKENS (saved global_paynt cards for tokenized checkout) ─────
-- One retailer can have multiple saved cards across gateways.
-- Cards are soft-deleted (IsActive=false), never hard-deleted.
CREATE TABLE RetailerCardTokens (
    TokenId           STRING(36)  NOT NULL,
    RetailerId        STRING(36)  NOT NULL,
    Gateway           STRING(20)  NOT NULL,  -- GLOBAL_PAY (expandable to other gateways later)
    ProviderCardToken STRING(MAX) NOT NULL,  -- Reusable card token from gateway
    CardLast4         STRING(4),             -- Display-only masked digits
    CardType          STRING(20),            -- UZCARD | HUMO | VISA | MASTERCARD
    IsDefault         BOOL        NOT NULL DEFAULT (false),
    IsActive          BOOL        NOT NULL DEFAULT (true),
    ExpiresAt         TIMESTAMP,             -- Card expiry if provider returns it
    CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TokenId);

CREATE INDEX Idx_RetailerCardTokens_ByRetailer ON RetailerCardTokens(RetailerId);
CREATE UNIQUE INDEX Idx_RetailerCardTokens_Active ON RetailerCardTokens(RetailerId, Gateway, ProviderCardToken)
    WHERE IsActive = true;

-- ── GLOBAL PAY SPLIT GLOBAL_PAYNT: Supplier recipient ID ──────────────────────
ALTER TABLE SupplierGlobalPayntConfigs ADD COLUMN RecipientId STRING(MAX);

-- ── AMENDMENT SAFEGUARD: pending supplier approval for large reductions ───
ALTER TABLE Orders ADD COLUMN AmendmentPendingApproval BOOL NOT NULL DEFAULT (false);
ALTER TABLE Orders ADD COLUMN PendingAmendmentData STRING(MAX);

-- ── FIREBASE AUTH IDENTITY LINKING ───────────────────────────────────────
-- Links each role's Spanner identity to its Firebase Auth UID.
-- NULL until user is migrated/created in Firebase Auth.
ALTER TABLE Admins ADD COLUMN FirebaseUid STRING(128);
ALTER TABLE Suppliers ADD COLUMN FirebaseUid STRING(128);
ALTER TABLE Retailers ADD COLUMN FirebaseUid STRING(128);
ALTER TABLE Drivers ADD COLUMN FirebaseUid STRING(128);
ALTER TABLE WarehouseStaff ADD COLUMN FirebaseUid STRING(128);

CREATE UNIQUE NULL_FILTERED INDEX Idx_Admins_ByFirebaseUid ON Admins(FirebaseUid);
CREATE UNIQUE NULL_FILTERED INDEX Idx_Suppliers_ByFirebaseUid ON Suppliers(FirebaseUid);
CREATE UNIQUE NULL_FILTERED INDEX Idx_Retailers_ByFirebaseUid ON Retailers(FirebaseUid);
CREATE UNIQUE NULL_FILTERED INDEX Idx_Drivers_ByFirebaseUid ON Drivers(FirebaseUid);
CREATE UNIQUE NULL_FILTERED INDEX Idx_WarehouseStaff_ByFirebaseUid ON WarehouseStaff(FirebaseUid);

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE W: 1:N SUPPLIER → WAREHOUSE HIERARCHICAL MIGRATION
-- Moves from flat 1:1 (Supplier = one warehouse) to hierarchical 1:N
-- (Supplier HQ → N Warehouse execution nodes).
-- ══════════════════════════════════════════════════════════════════════════════

-- ── WAREHOUSES TABLE ──────────────────────────────────────────────────────────
-- Each supplier has one or more warehouses (execution nodes).
-- The first warehouse is auto-created during the Phantom Node migration as
-- IsDefault=true using the supplier's existing WarehouseLat/WarehouseLng.
-- Products (SupplierProducts) remain supplier-wide; INVENTORY is per-warehouse.
CREATE TABLE Warehouses (
    WarehouseId      STRING(36)  NOT NULL,
    SupplierId       STRING(36)  NOT NULL,
    Name             STRING(255) NOT NULL,
    Address          STRING(MAX),
    Lat              FLOAT64,
    Lng              FLOAT64,
    H3Indexes        ARRAY<STRING(MAX)>,       -- H3 hex IDs at resolution 7 covering the service area
    CoverageRadiusKm FLOAT64     NOT NULL DEFAULT (50.0),
    IsActive         BOOL        NOT NULL DEFAULT (true),
    IsDefault        BOOL        NOT NULL DEFAULT (false),
    IsOnShift        BOOL        NOT NULL DEFAULT (true),
    CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplierId ON Warehouses(SupplierId);

-- ── SUPPLIER USERS (RBAC: GLOBAL_ADMIN vs NODE_ADMIN) ────────────────────────
-- Universal identity table for the supplier organization pyramid.
-- GLOBAL_ADMIN: full access to all warehouses + factories under the supplier.
-- NODE_ADMIN: scoped to one warehouse — middleware silently appends WHERE WarehouseId.
-- FACTORY_ADMIN: manages a factory facility (loading bays, production, staff).
-- FACTORY_PAYLOADER: handles manifest scanning + loading at a factory.
CREATE TABLE SupplierUsers (
    UserId               STRING(36)  NOT NULL,
    SupplierId           STRING(36)  NOT NULL,
    Email                STRING(MAX),
    Phone                STRING(20),
    Name                 STRING(MAX) NOT NULL,
    PasswordHash         STRING(MAX) NOT NULL,
    SupplierRole         STRING(30)  NOT NULL,         -- GLOBAL_ADMIN | NODE_ADMIN | FACTORY_ADMIN | FACTORY_PAYLOADER
    AssignedWarehouseId  STRING(36),                   -- NULL for GLOBAL_ADMIN, required for NODE_ADMIN
    AssignedFactoryId    STRING(36),                   -- NULL unless FACTORY_ADMIN or FACTORY_PAYLOADER
    IsActive             BOOL        NOT NULL DEFAULT (true),
    FirebaseUid          STRING(128),
    CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_SupplierRole CHECK (SupplierRole IN ('GLOBAL_ADMIN', 'NODE_ADMIN', 'FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
) PRIMARY KEY (UserId);

CREATE INDEX Idx_SupplierUsers_BySupplierId ON SupplierUsers(SupplierId);
CREATE INDEX Idx_SupplierUsers_ByPhone ON SupplierUsers(Phone);
CREATE UNIQUE NULL_FILTERED INDEX Idx_SupplierUsers_ByFirebaseUid ON SupplierUsers(FirebaseUid);

-- ── ADD WarehouseId TO OPERATIONAL TABLES ─────────────────────────────────────
-- All nullable initially → backfilled → then enforced by application logic.
ALTER TABLE Drivers ADD COLUMN WarehouseId STRING(36);
ALTER TABLE Vehicles ADD COLUMN WarehouseId STRING(36);
ALTER TABLE WarehouseStaff ADD COLUMN WarehouseId STRING(36);
ALTER TABLE SupplierInventory ADD COLUMN WarehouseId STRING(36);
ALTER TABLE InventoryAuditLog ADD COLUMN WarehouseId STRING(36);
ALTER TABLE Orders ADD COLUMN WarehouseId STRING(36);
ALTER TABLE RetailerCarts ADD COLUMN WarehouseId STRING(36);

-- ── WAREHOUSE-SCOPED INDEXES ──────────────────────────────────────────────────
CREATE INDEX Idx_Drivers_ByWarehouseId ON Drivers(WarehouseId);
CREATE INDEX Idx_Vehicles_ByWarehouseId ON Vehicles(WarehouseId);
CREATE INDEX Idx_WarehouseStaff_ByWarehouseId ON WarehouseStaff(WarehouseId);
CREATE INDEX Idx_Inventory_ByWarehouseId ON SupplierInventory(SupplierId, WarehouseId);
CREATE INDEX Idx_Orders_ByWarehouseId ON Orders(WarehouseId);

-- ── AI PREDICTIONS — WAREHOUSE SCOPING ────────────────────────────────────────
ALTER TABLE AIPredictions ADD COLUMN WarehouseId STRING(36);
CREATE INDEX Idx_AIPredictions_ByWarehouse ON AIPredictions(WarehouseId);

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE F: FACTORY-TO-WAREHOUSE REPLENISHMENT LAYER
-- Internal supply chain: Factory production sites → Warehouse execution nodes.
-- Manages replenishment transfers, factory manifests, and predictive insights.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── FACTORIES ────────────────────────────────────────────────────────────────
-- Each supplier has zero or more factory / production sites.
-- Factories produce goods and ship internal transfer orders to warehouses.
CREATE TABLE Factories (
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
) PRIMARY KEY (FactoryId);

CREATE INDEX Idx_Factories_BySupplierId ON Factories(SupplierId);

-- ── FACTORY STAFF ────────────────────────────────────────────────────────────
-- Workers at a factory: FACTORY_ADMIN manages the facility, FACTORY_PAYLOADER
-- handles loading bays and manifest scanning.
CREATE TABLE FactoryStaff (
    StaffId      STRING(36)  NOT NULL,
    FactoryId    STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    Name         STRING(MAX) NOT NULL,
    Phone        STRING(20),
    PasswordHash STRING(MAX) NOT NULL,
    StaffRole    STRING(30)  NOT NULL,  -- FACTORY_ADMIN | FACTORY_PAYLOADER
    IsActive     BOOL        NOT NULL DEFAULT (true),
    FirebaseUid  STRING(128),
    CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_FactoryStaffRole CHECK (StaffRole IN ('FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
) PRIMARY KEY (StaffId);

CREATE INDEX Idx_FactoryStaff_ByFactoryId ON FactoryStaff(FactoryId);
CREATE INDEX Idx_FactoryStaff_ByPhone ON FactoryStaff(Phone);
CREATE UNIQUE NULL_FILTERED INDEX Idx_FactoryStaff_ByFirebaseUid ON FactoryStaff(FirebaseUid);

-- ── INTERNAL TRANSFER ORDERS ─────────────────────────────────────────────────
-- A request to move goods from a Factory to a Warehouse.
-- Created by the replenishment engine (system) or manually by operator.
CREATE TABLE InternalTransferOrders (
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
) PRIMARY KEY (TransferId);

CREATE INDEX Idx_Transfers_ByFactoryId ON InternalTransferOrders(FactoryId);
CREATE INDEX Idx_Transfers_ByWarehouseId ON InternalTransferOrders(WarehouseId);
CREATE INDEX Idx_Transfers_BySupplierId ON InternalTransferOrders(SupplierId);
CREATE INDEX Idx_Transfers_ByState ON InternalTransferOrders(State);

-- ── INTERNAL TRANSFER ITEMS ──────────────────────────────────────────────────
-- Line items within a transfer order. Interleaved for efficient per-transfer reads.
CREATE TABLE InternalTransferItems (
    TransferId STRING(36) NOT NULL,
    ItemId     STRING(36) NOT NULL,
    ProductId  STRING(36) NOT NULL,
    Quantity   INT64      NOT NULL,
    VolumeVU   FLOAT64    NOT NULL DEFAULT (0),
) PRIMARY KEY (TransferId, ItemId),
  INTERLEAVE IN PARENT InternalTransferOrders ON DELETE CASCADE;

-- ── FACTORY TRUCK MANIFESTS ──────────────────────────────────────────────────
-- Aggregated loading manifests for factory outbound trucks.
-- Groups multiple InternalTransferOrders onto a single vehicle.
CREATE TABLE FactoryTruckManifests (
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
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_FactoryManifests_ByFactoryId ON FactoryTruckManifests(FactoryId);
CREATE INDEX Idx_FactoryManifests_ByState ON FactoryTruckManifests(State);

-- ── REPLENISHMENT INSIGHTS ───────────────────────────────────────────────────
-- Predictive / threshold-based recommendations generated by the replenishment engine.
-- Each row represents "warehouse X needs product Y restocked from factory Z".
CREATE TABLE ReplenishmentInsights (
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
    DemandBreakdown  STRING(MAX),  -- JSON: {"unfulfilled":N, "preorders":N, "ai_projected":N, "in_transit":N}
    CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_InsightUrgency CHECK (UrgencyLevel IN ('CRITICAL', 'WARNING', 'STABLE')),
    CONSTRAINT CHK_InsightReason CHECK (ReasonCode IN ('HIGH_VELOCITY', 'LOW_STOCK', 'PREDICTED_SPIKE')),
    CONSTRAINT CHK_InsightStatus CHECK (Status IN ('PENDING', 'APPROVED', 'DISMISSED'))
) PRIMARY KEY (InsightId);

CREATE INDEX Idx_Insights_ByWarehouse ON ReplenishmentInsights(WarehouseId);
CREATE INDEX Idx_Insights_BySupplierId ON ReplenishmentInsights(SupplierId);
CREATE INDEX Idx_Insights_ByStatus ON ReplenishmentInsights(Status);

-- ── LINK WAREHOUSES TO FACTORIES ─────────────────────────────────────────────
ALTER TABLE Warehouses ADD COLUMN PrimaryFactoryId STRING(36);
ALTER TABLE Warehouses ADD COLUMN SecondaryFactoryId STRING(36);

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE G: GEO-SPATIAL SOVEREIGNTY — H3 Index on Retailers & Factories
-- Adds single H3 grid cell ID for each entity's physical location.
-- Warehouses already have H3Indexes ARRAY<STRING> as coverage polygons.
-- ══════════════════════════════════════════════════════════════════════════════
ALTER TABLE Retailers ADD COLUMN H3Index STRING(MAX);
CREATE INDEX Idx_Retailers_ByH3Index ON Retailers(H3Index);

ALTER TABLE Factories ADD COLUMN H3Index STRING(MAX);
CREATE INDEX Idx_Factories_ByH3Index ON Factories(H3Index);

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE H: EDGE-CASE HARDENING — Schema Additions
-- STALE_AUDIT state, dispatch metadata columns, partial-global_paynt support,
-- and stale-order audit indexes.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── H.1: STALE_AUDIT Order State ──────────────────────────────────────────────
-- Orders stuck IN_TRANSIT/ARRIVING for >12h are automatically transitioned to
-- STALE_AUDIT by the StaleOrderAuditor cron. Requires manual admin resolution.
-- Migration:
--   ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
--   ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--     CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
--                      'ARRIVING', 'ARRIVED', 'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION',
--                      'COMPLETED', 'CANCELLED', 'SCHEDULED', 'QUARANTINE', 'STALE_AUDIT'));

-- ── H.2: Dispatch Metadata Columns ───────────────────────────────────────────
-- Persists ephemeral dispatch anomaly flags from the AutoDispatch engine.
-- CapacityOverflow: Phase 4 force-assigned to an already-full truck.
-- LogisticsIsolated: beyond MaxDetourRadius (10km) from any cluster.
-- DispatchWarnings: JSON array of warning strings for admin visibility.
ALTER TABLE Orders ADD COLUMN CapacityOverflow BOOL;
ALTER TABLE Orders ADD COLUMN LogisticsIsolated BOOL;
ALTER TABLE Orders ADD COLUMN DispatchWarnings STRING(MAX);

-- ── H.3: Stale Audit Timestamp ───────────────────────────────────────────────
ALTER TABLE Orders ADD COLUMN StaleAuditAt TIMESTAMP;

-- ── H.4: Routing Metadata ────────────────────────────────────────────────────
-- Tracks whether route optimization fell back to Haversine when Google Maps
-- returned ZERO_RESULTS.
ALTER TABLE Orders ADD COLUMN RoutingMethod STRING(30);

-- ── H.5: Stale Order Sweep Index ─────────────────────────────────────────────
-- Covers the StaleOrderAuditor cron: find IN_TRANSIT/ARRIVING orders older than X hours.
CREATE INDEX Idx_Orders_ByStateUpdatedAt ON Orders(State, CreatedAt);

-- ── H.6: Partial GlobalPaynt Support ─────────────────────────────────────────────
-- PaidAmount tracks actual collected amount when global_paynt is partial.
-- PARTIALLY_PAID status added to GlobalPayntSessions CHECK.
ALTER TABLE GlobalPayntSessions ADD COLUMN PaidAmount INT64;
-- Migration:
--   ALTER TABLE GlobalPayntSessions DROP CONSTRAINT CHK_SessionStatus;
--   ALTER TABLE GlobalPayntSessions ADD CONSTRAINT CHK_SessionStatus
--     CHECK (Status IN ('CREATED', 'PENDING', 'SETTLED', 'FAILED', 'EXPIRED', 'CANCELLED', 'PARTIALLY_PAID'));

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE I: GLOBAL ENTERPRISE ADAPTATION + SHOP-CLOSED PROTOCOL (FRIDAY v2.2)
-- Multi-country config, per-supplier overrides, order event audit trail,
-- shop-closed contact protocol tables, device fingerprinting, and 3 new order states.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── I.1: COUNTRY CONFIGURATION ────────────────────────────────────────────────
-- Per-country operational parameters. Replaces all hardcoded values (BreachRadius,
-- currency, timezone, notification chain, etc.) with config-driven behavior.
-- Seeded with UZ defaults. Expandable to KZ, TR, ID, IN, etc.
CREATE TABLE CountryConfigs (
    CountryCode              STRING(2)   NOT NULL, -- ISO 3166-1 alpha-2
    CountryName              STRING(100) NOT NULL,
    Timezone                 STRING(50)  NOT NULL, -- IANA: Asia/Tashkent
    CurrencyCode             STRING(3)   NOT NULL, -- ISO 4217: UZS, KZT, TRY
    CurrencyDecimalPlaces    INT64       NOT NULL DEFAULT (0),
    DistanceUnit             STRING(10)  NOT NULL DEFAULT ('km'),  -- km | miles
    DefaultVUConversion      FLOAT64     NOT NULL DEFAULT (1.0),
    MapsProvider             STRING(20)  NOT NULL DEFAULT ('GOOGLE'), -- GOOGLE | YANDEX | BAIDU | HERE
    LLMProvider              STRING(20)  NOT NULL DEFAULT ('GEMINI'), -- GEMINI | OPENAI | ANTHROPIC
    GlobalPayntGateways          STRING(MAX),           -- JSON: ["GLOBAL_PAY","CASH"]
    SMSProvider              STRING(30),             -- ESKIZ | TWILIO | null
    NotificationFallbackOrder STRING(MAX) NOT NULL DEFAULT ('["FCM","TELEGRAM"]'), -- JSON array
    LegalRetentionDays       INT64       NOT NULL DEFAULT (365),
    GridSystem               STRING(10)  NOT NULL DEFAULT ('H3'), -- H3 | CUSTOM
    BreachRadiusMeters       FLOAT64     NOT NULL DEFAULT (100.0),
    ShopClosedGraceMinutes   INT64       NOT NULL DEFAULT (5),
    ShopClosedEscalationMinutes INT64    NOT NULL DEFAULT (3),
    OfflineModeDurationMinutes  INT64    NOT NULL DEFAULT (30),
    CashCustodyAlertHours    INT64       NOT NULL DEFAULT (4),
    IsActive                 BOOL        NOT NULL DEFAULT (true),
    CreatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt                TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (CountryCode);

-- ── I.2: SUPPLIER COUNTRY OVERRIDES ──────────────────────────────────────────
-- Per-supplier overrides on top of CountryConfigs. Nullable fields = use country default.
-- Merge logic: SupplierOverride ?? CountryConfig ?? hardcoded fallback.
CREATE TABLE SupplierCountryOverrides (
    SupplierId                  STRING(36) NOT NULL,
    CountryCode                 STRING(2)  NOT NULL,
    BreachRadiusMeters          FLOAT64,
    ShopClosedGraceMinutes      INT64,
    ShopClosedEscalationMinutes INT64,
    OfflineModeDurationMinutes  INT64,
    CashCustodyAlertHours       INT64,
    GlobalPayntGateways             STRING(MAX),  -- JSON override
    NotificationFallbackOrder   STRING(MAX),  -- JSON override
    SMSProvider                 STRING(30),
    MapsProvider                STRING(20),
    LLMProvider                 STRING(20),
    CreatedAt                   TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt                   TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SupplierId, CountryCode);

CREATE INDEX Idx_SupplierCountryOverrides_ByCountry ON SupplierCountryOverrides(CountryCode);

-- ── I.3: ORDER EVENTS AUDIT TABLE ────────────────────────────────────────────
-- Immutable, chronological event log for every significant order action.
-- Separate from AuditLog (which is generic). OrderEvents is order-specific
-- with GPS, actor context, and structured metadata for timeline rendering.
CREATE TABLE OrderEvents (
    EventId     STRING(36)  NOT NULL, -- UUID
    OrderId     STRING(36)  NOT NULL,
    ActorId     STRING(36)  NOT NULL,
    ActorRole   STRING(20)  NOT NULL, -- DRIVER | RETAILER | SUPPLIER | SYSTEM
    EventType   STRING(50)  NOT NULL,
    -- Lifecycle: STATE_CHANGE, DISPATCHED, LOADED, ARRIVED, COMPLETED, CANCELLED
    -- Shop-Closed: SHOP_CLOSED_REPORTED, RETAILER_RESPONDED, ADMIN_ESCALATED,
    --              BYPASS_ISSUED, BYPASS_USED, RETURN_TO_DEPOT
    -- Edge Cases: CANCEL_REQUESTED, CANCEL_APPROVED, CANCEL_REJECTED,
    --             GLOBAL_PAYNT_BYPASS_ISSUED, GLOBAL_PAYNT_BYPASS_USED,
    --             NO_CAPACITY_FLAGGED, CAPACITY_RESTORED
    -- Audit: STALE_AUDIT_TRIGGERED, QUARANTINE_ENTERED
    Metadata    STRING(MAX),           -- JSON: contextual data per event type
    GPSLat      FLOAT64,               -- Actor location at event time (nullable)
    GPSLng      FLOAT64,
    CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EventId);

CREATE INDEX Idx_OrderEvents_ByOrder ON OrderEvents(OrderId, CreatedAt DESC);
CREATE INDEX Idx_OrderEvents_ByActorId ON OrderEvents(ActorId, CreatedAt DESC);
CREATE INDEX Idx_OrderEvents_ByType ON OrderEvents(EventType, CreatedAt DESC);

-- ── I.4: SHOP-CLOSED ATTEMPTS ────────────────────────────────────────────────
-- Tracks each shop-closed contact attempt lifecycle: report → response → escalation → resolution.
-- One row per attempt. An order can have multiple attempts (driver re-arrives).
CREATE TABLE ShopClosedAttempts (
    AttemptId           STRING(36)  NOT NULL, -- UUID
    OrderId             STRING(36)  NOT NULL,
    DriverId            STRING(36)  NOT NULL,
    RetailerId          STRING(36)  NOT NULL,
    ReportedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    GPSLat              FLOAT64     NOT NULL, -- Driver GPS at report time
    GPSLng              FLOAT64     NOT NULL,
    RetailerResponse    STRING(20),            -- OPEN_NOW | 5_MIN | CALL_ME | CLOSED_TODAY | NO_RESPONSE
    RetailerRespondedAt TIMESTAMP,
    EscalatedAt         TIMESTAMP,
    EscalatedTo         STRING(36),            -- Admin user ID who received escalation
    Resolution          STRING(30),            -- RETAILER_OPENED | BYPASS_ISSUED | RETURN_TO_DEPOT | WAITING
    BypassToken         STRING(6),             -- 6-digit numeric, issued by admin
    ResolvedAt          TIMESTAMP,
    ResolvedBy          STRING(36)             -- Actor who resolved (admin ID or SYSTEM)
) PRIMARY KEY (AttemptId);

CREATE INDEX Idx_ShopClosedAttempts_ByOrder ON ShopClosedAttempts(OrderId);
CREATE INDEX Idx_ShopClosedAttempts_ByDriver ON ShopClosedAttempts(DriverId);
CREATE INDEX Idx_ShopClosedAttempts_ByRetailer ON ShopClosedAttempts(RetailerId);
CREATE INDEX Idx_ShopClosedAttempts_Unresolved ON ShopClosedAttempts(Resolution)
    WHERE Resolution IS NULL;

-- ── I.5: DEVICE FINGERPRINTS (Edge 24) ───────────────────────────────────────
-- Tracks device identity per user session. Enforces single-device policy:
-- login on device B → force-logout on device A via WebSocket FORCE_LOGOUT push.
CREATE TABLE DeviceFingerprints (
    FingerprintId STRING(36)  NOT NULL, -- UUID
    UserId        STRING(36)  NOT NULL,
    Role          STRING(20)  NOT NULL, -- DRIVER | RETAILER | SUPPLIER | PAYLOADER
    DeviceId      STRING(100) NOT NULL, -- X-Device-Id header value
    Platform      STRING(10)  NOT NULL, -- ANDROID | IOS | WEB
    AppVersion    STRING(20),           -- e.g. "2.1.0"
    LastSeenAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    Active        BOOL        NOT NULL DEFAULT (true)
) PRIMARY KEY (FingerprintId);

CREATE INDEX Idx_DeviceFingerprints_ByUser ON DeviceFingerprints(UserId, Active);
CREATE INDEX Idx_DeviceFingerprints_ByDeviceId ON DeviceFingerprints(DeviceId);

-- ── I.6: NEW ORDER STATES ────────────────────────────────────────────────────
-- ARRIVED_SHOP_CLOSED: Driver arrived but shop is closed/unresponsive.
--   Triggered by POST /v1/delivery/shop-closed. Enters shop-closed contact protocol.
-- CANCEL_REQUESTED: Retailer requested cancellation before IN_TRANSIT.
--   Requires supplier approval to transition to CANCELLED.
-- NO_CAPACITY: Dispatch engine found zero available trucks/capacity.
--   Auto-retried by StaleOrderAuditor cron every 15 minutes.
-- Migration:
--   ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
--   ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
--     CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
--                      'ARRIVING', 'ARRIVED', 'ARRIVED_SHOP_CLOSED', 'AWAITING_GLOBAL_PAYNT',
--                      'PENDING_CASH_COLLECTION', 'COMPLETED', 'CANCELLED',
--                      'CANCEL_REQUESTED', 'NO_CAPACITY', 'DELIVERED_ON_CREDIT',
--                      'SCHEDULED', 'QUARANTINE', 'STALE_AUDIT'));

-- ── I.7: SUPPLIER COUNTRY CODE ───────────────────────────────────────────────
-- Links supplier to their operating country for config resolution.
-- Default 'UZ' for backward compatibility with existing Uzbekistan suppliers.
ALTER TABLE Suppliers ADD COLUMN CountryCode STRING(2) DEFAULT ('UZ');
CREATE INDEX Idx_Suppliers_ByCountryCode ON Suppliers(CountryCode);

-- ── I.8: RETAILER PREFERRED WAREHOUSE (Edge 17) ─────────────────────────────
-- Optional override: bypasses H3/haversine proximity resolution.
-- If set and warehouse is active+on-shift, used directly.
ALTER TABLE Retailers ADD COLUMN PreferredWarehouseId STRING(36);

-- ── I.9: ORDER GLOBAL_PAYNT BYPASS SUPPORT (Edge 5) ──────────────────────────────
-- Stores bypass token issued by supplier admin for AWAITING_GLOBAL_PAYNT stuck orders.
ALTER TABLE Orders ADD COLUMN GlobalPayntBypassToken STRING(6);
ALTER TABLE Orders ADD COLUMN GlobalPayntBypassIssuedBy STRING(36);
ALTER TABLE Orders ADD COLUMN GlobalPayntBypassAt TIMESTAMP;

-- ── I.10: ORDER CANCEL REQUEST METADATA (Edge 7) ────────────────────────────
-- Tracks who requested cancellation and the reason.
ALTER TABLE Orders ADD COLUMN CancelRequestedBy STRING(36);
ALTER TABLE Orders ADD COLUMN CancelRequestedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN CancelReason STRING(MAX);

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE II: HUMAN-CENTRIC EDGE CASES (FRIDAY v3.1)
-- Credit delivery, negotiation proposals, family member sub-profiles,
-- AI confirmation gate, split global_paynt support, early route complete.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── II.1: CREDIT DELIVERY SUPPORT (Edge 32) ─────────────────────────────────
-- Per retailer-supplier pair: enable/disable credit delivery and set limits.
ALTER TABLE RetailerSupplierSettings ADD COLUMN CreditEnabled BOOL NOT NULL DEFAULT (false);
ALTER TABLE RetailerSupplierSettings ADD COLUMN CreditLimit INT64 NOT NULL DEFAULT (0);
ALTER TABLE RetailerSupplierSettings ADD COLUMN CreditBalance INT64 NOT NULL DEFAULT (0);
ALTER TABLE RetailerSupplierSettings ADD COLUMN CreditCurrency STRING(3) NOT NULL DEFAULT ('UZS');

-- ── II.2: AI CONFIRMATION GATE (Edge 34) ────────────────────────────────────
-- When cron creates orders from AI predictions, AiPendingConfirmation = true.
-- Dispatch skips these until retailer confirms or AutoConfirmAt passes.
ALTER TABLE Orders ADD COLUMN AiPendingConfirmation BOOL;

-- ── II.3: NEGOTIATION PROPOSALS (Edge 28) ───────────────────────────────────
-- Driver proposes quantity changes → supplier approves/rejects → AmendOrder executes.
CREATE TABLE NegotiationProposals (
    ProposalId    STRING(36)  NOT NULL,
    OrderId       STRING(36)  NOT NULL,
    DriverId      STRING(36)  NOT NULL,
    Status        STRING(20)  NOT NULL DEFAULT ('PENDING'),
    ProposedItems STRING(MAX),           -- JSON: [{"sku_id":"...","original_qty":10,"proposed_qty":7}]
    Resolution    STRING(200),           -- Supplier comment on resolve
    ResolvedBy    STRING(36),
    ResolvedAt    TIMESTAMP,
    CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_NegotiationStatus CHECK (Status IN ('PENDING', 'APPROVED', 'REJECTED'))
) PRIMARY KEY (ProposalId);

CREATE INDEX Idx_NegotiationProposals_ByOrderId ON NegotiationProposals(OrderId);
CREATE INDEX Idx_NegotiationProposals_Pending ON NegotiationProposals(Status)
    WHERE Status = 'PENDING';

-- ── II.4: RETAILER FAMILY MEMBERS (Edge 29) ─────────────────────────────────
-- Lightweight sub-profiles for family-run shops. No auth changes — cosmetic only.
-- All actions still attributed to main RetailerId; family_member logged in OrderEvents.
CREATE TABLE RetailerFamilyMembers (
    RetailerId    STRING(36)  NOT NULL,
    MemberId      STRING(36)  NOT NULL,
    Nickname      STRING(100) NOT NULL,
    PhotoUrl      STRING(MAX),
    CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, MemberId),
  INTERLEAVE IN PARENT Retailers ON DELETE CASCADE;

-- ── II.5: CREDIT DELIVERY PHOTO PROOF (Edge 32) ─────────────────────────────
-- Store photo proof URL on the order when delivered on credit.
ALTER TABLE Orders ADD COLUMN CreditPhotoProofUrl STRING(MAX);

-- ── II.6: EARLY ROUTE COMPLETE METADATA (Edge 27) ───────────────────────────
-- Track fatigue/early-complete reason on the order for supplier analytics.
ALTER TABLE Orders ADD COLUMN EarlyCompleteReason STRING(30);
ALTER TABLE Orders ADD COLUMN EarlyCompleteNote STRING(MAX);

-- ═══════════════════════════════════════════════════════════════════════════
-- PHASE III: ENTERPRISE HARDENING
-- ═══════════════════════════════════════════════════════════════════════════

-- ── III.1: AI RLHF CORRECTION WEIGHTS ───────────────────────────────────────
-- Persistent per-retailer, per-warehouse, per-SKU correction factors for the
-- Empathy Engine. Survives AI Worker restarts. Written by AI Worker on
-- AI_PREDICTION_CORRECTED / AI_PLAN_SKU_MODIFIED events.
CREATE TABLE CorrectionWeights (
    RetailerId          STRING(36)  NOT NULL,
    WarehouseId         STRING(36)  NOT NULL,
    SkuId               STRING(36)  NOT NULL,
    Factor              FLOAT64     NOT NULL,
    TriggerDateShiftH   FLOAT64     NOT NULL DEFAULT (0),   -- hours to shift trigger date
    CorrectionCount     INT64       NOT NULL DEFAULT (0),
    LastCorrectedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, WarehouseId, SkuId);

CREATE INDEX Idx_CorrectionWeights_ByRetailer ON CorrectionWeights(RetailerId);

-- ── III.2: REFUND TRACKING ──────────────────────────────────────────────────
-- Explicit refund records for auditable global_paynt reversals.
CREATE TABLE Refunds (
    RefundId        STRING(36)   NOT NULL,
    OrderId         STRING(36)   NOT NULL,
    SessionId       STRING(36),            -- original GlobalPayntSession if electronic refund
    Gateway         STRING(30),            -- CASH, GLOBAL_PAY, GLOBAL_PAY, CASH
    AmountUZS       INT64        NOT NULL,
    Reason          STRING(MAX)  NOT NULL,
    Status          STRING(20)   NOT NULL,
    ProviderRefundId STRING(MAX),           -- gateway-specific refund transaction ID
    InitiatedBy     STRING(36)   NOT NULL, -- SupplierId who triggered refund
    CreatedAt       TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
    SettledAt       TIMESTAMP,
    CONSTRAINT CHK_RefundStatus CHECK (Status IN ('PENDING', 'SETTLED', 'FAILED', 'MANUAL_REQUIRED'))
) PRIMARY KEY (RefundId);

CREATE INDEX Idx_Refunds_ByOrder ON Refunds(OrderId);
CREATE INDEX Idx_Refunds_Pending ON Refunds(Status) WHERE Status = 'PENDING';

-- ── III.3: ORDER STATE EXPANSION ────────────────────────────────────────────
-- Adds ARRIVED_SHOP_CLOSED, CANCEL_REQUESTED, NO_CAPACITY, STALE_AUDIT, REFUNDED.
-- Applied to CREATE TABLE above. Run against existing deployments:
ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State
    CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
                     'ARRIVING', 'ARRIVED', 'ARRIVED_SHOP_CLOSED', 'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION',
                     'COMPLETED', 'CANCELLED', 'CANCEL_REQUESTED', 'NO_CAPACITY', 'DELIVERED_ON_CREDIT',
                     'SCHEDULED', 'QUARANTINE', 'STALE_AUDIT', 'REFUNDED'));

-- ═══════════════════════════════════════════════════════════════════════════
-- PHASE IV: WAREHOUSE SUPPLY CHAIN & PRE-ORDER POLICY
-- ═══════════════════════════════════════════════════════════════════════════

-- ── IV.1: PRE-ORDER CONFIRMATION POLICY ─────────────────────────────────────
-- Enables T-4/T-3 notification + auto-lock lifecycle for scheduled and AI orders.
ALTER TABLE Orders ADD COLUMN CancelLockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN CancelLockReason STRING(30);    -- AI_POLICY | MANUAL_POLICY | ADMIN_OVERRIDE
ALTER TABLE Orders ADD COLUMN ConfirmationNotifiedAt TIMESTAMP;

CREATE INDEX Idx_Orders_PreOrderLockPending
    ON Orders(State, RequestedDeliveryDate)
    WHERE CancelLockedAt IS NULL AND ConfirmationNotifiedAt IS NULL
      AND State IN ('SCHEDULED', 'PENDING_REVIEW');

-- ── IV.2: WAREHOUSE STAFF EXPANSION ─────────────────────────────────────────
-- Link each warehouse worker to a specific warehouse.
ALTER TABLE WarehouseStaff ADD COLUMN WarehouseId STRING(36);
CREATE INDEX Idx_WarehouseStaff_ByWarehouseId ON WarehouseStaff(WarehouseId);

-- Add a Role column to differentiate admin, staff, and payloader privileges.
ALTER TABLE WarehouseStaff ADD COLUMN Role STRING(20) DEFAULT ('WAREHOUSE_STAFF');
-- WAREHOUSE_ADMIN: full access (create supply requests, manage settings)
-- WAREHOUSE_STAFF: read-only + order handling
-- PAYLOADER: loading bay operations

-- ── IV.3: SUPPLY REQUESTS ───────────────────────────────────────────────────
-- Warehouse-to-factory supply requests with demand-aware recommendations.
-- Created by warehouse admin, fulfilled by factory through InternalTransferOrders.
CREATE TABLE SupplyRequests (
    RequestId            STRING(36)  NOT NULL,
    WarehouseId          STRING(36)  NOT NULL,
    FactoryId            STRING(36)  NOT NULL,
    SupplierId           STRING(36)  NOT NULL,
    State                STRING(30)  NOT NULL DEFAULT ('DRAFT'),
    Priority             STRING(20)  NOT NULL DEFAULT ('NORMAL'),
    RequestedDeliveryDate TIMESTAMP,
    TotalVolumeVU        FLOAT64    NOT NULL DEFAULT (0),
    Notes                STRING(MAX),
    DemandBreakdown      JSON,        -- {"incoming":N,"ai_predicted":N,"preorder":N,"burn_rate":N}
    TransferOrderId      STRING(36),  -- links to InternalTransferOrders once factory marks READY
    CreatedBy            STRING(36),  -- WarehouseStaff WorkerId
    CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt            TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_SupplyRequest_State CHECK (State IN (
        'DRAFT', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'
    )),
    CONSTRAINT CHK_SupplyRequest_Priority CHECK (Priority IN ('NORMAL', 'URGENT', 'CRITICAL'))
) PRIMARY KEY (RequestId);

CREATE INDEX Idx_SupplyRequests_ByWarehouse ON SupplyRequests(WarehouseId, State);
CREATE INDEX Idx_SupplyRequests_ByFactory ON SupplyRequests(FactoryId, State);
CREATE INDEX Idx_SupplyRequests_BySupplierId ON SupplyRequests(SupplierId);

-- ── IV.4: SUPPLY REQUEST ITEMS ──────────────────────────────────────────────
-- Per-SKU line items within a supply request.
CREATE TABLE SupplyRequestItems (
    RequestId          STRING(36)  NOT NULL,
    ItemId             STRING(36)  NOT NULL,
    ProductId          STRING(36)  NOT NULL,
    RequestedQuantity  INT64       NOT NULL,
    RecommendedQuantity INT64      NOT NULL DEFAULT (0),
    UnitVolumeVU       FLOAT64    NOT NULL DEFAULT (0),
    CreatedAt          TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId, ItemId),
  INTERLEAVE IN PARENT SupplyRequests ON DELETE CASCADE;

-- ── IV.5: DISPATCH LOCKS ────────────────────────────────────────────────────
-- Prevents concurrent dispatch operations and enables order buffering.
CREATE TABLE DispatchLocks (
    LockId       STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    WarehouseId  STRING(36),             -- NULL = all warehouses for supplier
    FactoryId    STRING(36),             -- NULL = supplier-level dispatch lock
    LockType     STRING(30)  NOT NULL,   -- AUTO_DISPATCH | MANUAL_DISPATCH | FACTORY_DISPATCH
    LockedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UnlockedAt   TIMESTAMP,              -- NULL = still locked
    LockedBy     STRING(36)  NOT NULL,   -- user ID who acquired lock
) PRIMARY KEY (LockId);

CREATE INDEX Idx_DispatchLocks_Active
    ON DispatchLocks(SupplierId, LockedAt DESC)
    WHERE UnlockedAt IS NULL;

-- ── IV.6: TRANSFER ORDER BACK-LINK ─────────────────────────────────────────
-- Links InternalTransferOrders to the originating SupplyRequest.
ALTER TABLE InternalTransferOrders ADD COLUMN SupplyRequestId STRING(36);
ALTER TABLE InternalTransferOrders DROP CONSTRAINT CHK_TransferSource;
ALTER TABLE InternalTransferOrders ADD CONSTRAINT CHK_TransferSource
    CHECK (Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED', 'MANUAL_EMERGENCY', 'WAREHOUSE_REQUEST'));

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE V: LEO — LOGISTICS EXECUTION ORCHESTRATOR (Loading Gate)
-- Formalizes the DRAFT → LOADING → SEALED → DISPATCHED → COMPLETED state
-- machine for supplier truck manifests (retail delivery routes).
-- Adds volumetric validation, exception re-injection, DLQ, and Ghost Stop gate.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── V.1: SUPPLIER TRUCK MANIFESTS ───────────────────────────────────────────
-- The "Loading Gate" entity. Each manifest represents a truck's planned route
-- with a strict lifecycle enforced by LEO.
CREATE TABLE SupplierTruckManifests (
    ManifestId     STRING(36)  NOT NULL,
    SupplierId     STRING(36)  NOT NULL,
    WarehouseId    STRING(36),               -- originating warehouse (nullable = supplier-wide)
    RouteId        STRING(MAX),              -- route identifier (AUTO-XXXX-NNNNN)
    TruckId        STRING(36)  NOT NULL,     -- Vehicles.VehicleId
    DriverId       STRING(36)  NOT NULL,     -- Drivers.DriverId
    State          STRING(20)  NOT NULL DEFAULT ('DRAFT'),
    TotalVolumeVU  FLOAT64     NOT NULL DEFAULT (0),
    MaxVolumeVU    FLOAT64     NOT NULL DEFAULT (0),   -- truck capacity × 0.95 (Tetris Buffer)
    StopCount      INT64       NOT NULL DEFAULT (0),
    RegionCode     STRING(20),
    RoutePath      STRING(MAX),              -- JSON: ordered GPS waypoints (populated on SEAL)
    SealedAt       TIMESTAMP,                -- set on LOADING → SEALED transition
    SealedBy       STRING(36),               -- payloader WorkerId who sealed
    DispatchedAt   TIMESTAMP,                -- set on SEALED → DISPATCHED transition
    CompletedAt    TIMESTAMP,                -- set on route completion
    CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_SupplierManifestState CHECK (
        State IN ('DRAFT', 'LOADING', 'SEALED', 'DISPATCHED', 'COMPLETED', 'CANCELLED')
    )
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_SupplierManifests_BySupplierId ON SupplierTruckManifests(SupplierId);
CREATE INDEX Idx_SupplierManifests_ByState ON SupplierTruckManifests(State);
CREATE INDEX Idx_SupplierManifests_ByDriver ON SupplierTruckManifests(DriverId, State);
CREATE INDEX Idx_SupplierManifests_ByWarehouse ON SupplierTruckManifests(WarehouseId, State);

-- ── V.2: MANIFEST ORDERS (Manifest ↔ Order junction) ───────────────────────
-- Links orders to supplier manifests with LIFO loading metadata.
CREATE TABLE ManifestOrders (
    ManifestId     STRING(36)  NOT NULL,
    OrderId        STRING(36)  NOT NULL,
    SequenceIndex  INT64       NOT NULL DEFAULT (0),   -- delivery stop order (1-based)
    LoadingOrder   INT64       NOT NULL DEFAULT (0),   -- LIFO position (1 = back of truck)
    VolumeVU       FLOAT64     NOT NULL DEFAULT (0),
    State          STRING(30)  NOT NULL DEFAULT ('ASSIGNED'),
    RemovedAt      TIMESTAMP,
    RemovedReason  STRING(100),
    CONSTRAINT CHK_ManifestOrderState CHECK (
        State IN ('ASSIGNED', 'REMOVED_OVERFLOW', 'REMOVED_DAMAGED', 'REMOVED_MANUAL', 'SEALED', 'DELIVERED')
    )
) PRIMARY KEY (ManifestId, OrderId),
  INTERLEAVE IN PARENT SupplierTruckManifests ON DELETE CASCADE;

-- ── V.3: MANIFEST EXCEPTIONS (DLQ Tracking) ────────────────────────────────
-- Tracks repeated assignment failures per order for Dead Letter Queue escalation.
CREATE TABLE ManifestExceptions (
    ExceptionId    STRING(36)  NOT NULL,
    OrderId        STRING(36)  NOT NULL,
    ManifestId     STRING(36),               -- manifest it was removed from (nullable for DLQ)
    SupplierId     STRING(36)  NOT NULL,
    Reason         STRING(30)  NOT NULL,     -- OVERFLOW | DAMAGED | MANUAL | NO_CAPACITY
    Metadata       STRING(MAX),              -- JSON: additional context
    AttemptCount   INT64       NOT NULL DEFAULT (1),  -- cumulative overflow count for this order
    CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    ResolvedAt     TIMESTAMP,                -- set when order is successfully re-dispatched
    EscalatedAt    TIMESTAMP,                -- set when AttemptCount >= 3 (DLQ threshold)
    CONSTRAINT CHK_ExceptionReason CHECK (
        Reason IN ('OVERFLOW', 'DAMAGED', 'MANUAL', 'NO_CAPACITY')
    )
) PRIMARY KEY (ExceptionId);

CREATE INDEX Idx_ManifestExceptions_ByOrder ON ManifestExceptions(OrderId, AttemptCount DESC);
CREATE INDEX Idx_ManifestExceptions_BySupplierId ON ManifestExceptions(SupplierId, EscalatedAt)
    WHERE EscalatedAt IS NOT NULL;
CREATE INDEX Idx_ManifestExceptions_Unresolved ON ManifestExceptions(SupplierId, CreatedAt DESC)
    WHERE ResolvedAt IS NULL;

-- ── V.4: ALTER ORDERS — LEO Columns ────────────────────────────────────────
ALTER TABLE Orders ADD COLUMN ManifestId STRING(36);
ALTER TABLE Orders ADD COLUMN DispatchPriority INT64 DEFAULT (0);
ALTER TABLE Orders ADD COLUMN OverflowCount INT64 DEFAULT (0);
CREATE INDEX Idx_Orders_ByManifestId ON Orders(ManifestId);
CREATE INDEX Idx_Orders_ByDispatchPriority ON Orders(SupplierId, DispatchPriority DESC, CreatedAt ASC)
    WHERE State IN ('PENDING', 'NO_CAPACITY');

-- ── V.5: ALTER ORDERS — H3 Spatial Index + DELAYED State ───────────────────
-- H3Cell: H3 resolution-7 cell (15-char hex) for spatial geo-batching during
-- dispatch. Mirrors Retailers.H3Index from the order's delivery address. Backfill
-- via cmd/backfill-orders-h3cell after deployment.
ALTER TABLE Orders ADD COLUMN H3Cell STRING(15);
CREATE INDEX IDX_Orders_H3Cell_State ON Orders(H3Cell, State);
-- LEO state extension: DELAYED = capacity-overflow at loading bay, awaiting
-- next manifest cycle. Spanner CHECK constraints are immutable — drop+recreate.
ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
    'ARRIVING', 'ARRIVED', 'ARRIVED_SHOP_CLOSED', 'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION',
    'COMPLETED', 'CANCELLED', 'CANCEL_REQUESTED', 'NO_CAPACITY', 'DELIVERED_ON_CREDIT',
    'SCHEDULED', 'AUTO_ACCEPTED', 'QUARANTINE', 'STALE_AUDIT', 'REFUNDED', 'DELAYED'));

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE VI: WAREHOUSE SOVEREIGNTY — Advanced Configuration, Per-Retailer
-- Pricing, Stock Limits, Operating Schedule, and Spatial Event Materialisation.
-- ══════════════════════════════════════════════════════════════════════════════

-- ── VI.1: WAREHOUSE ADVANCED COLUMNS ─────────────────────────────────────────
-- MaxCapacityThreshold: configurable daily order capacity (used by load balancer).
-- OperatingSchedule: per-warehouse business hours (overrides supplier-level schedule).
-- DisabledReason: optional operator note when IsActive=false.
ALTER TABLE Warehouses ADD COLUMN MaxCapacityThreshold INT64 DEFAULT (100);
ALTER TABLE Warehouses ADD COLUMN OperatingSchedule STRING(MAX);    -- JSON: {"mon":{"open":"09:00","close":"18:00"}, ...}
ALTER TABLE Warehouses ADD COLUMN DisabledReason STRING(MAX);

-- ── VI.2: PER-RETAILER PRICING OVERRIDES ─────────────────────────────────────
-- Individual price overrides per retailer per SKU. Most specific pricing tier.
-- Resolution order: RetailerPricingOverride → PricingTier (tier-based) → Base Price.
-- GLOBAL_ADMIN can set for any retailer; NODE_ADMIN scoped to their warehouse's retailers.
CREATE TABLE RetailerPricingOverrides (
    OverrideId   STRING(36)  NOT NULL,
    SupplierId   STRING(36)  NOT NULL,
    WarehouseId  STRING(36),               -- NULL = global override (applies regardless of warehouse)
    RetailerId   STRING(36)  NOT NULL,
    SkuId        STRING(36)  NOT NULL,
    OverridePrice INT64      NOT NULL,     -- Absolute price in smallest currency unit (e.g. UZS)
    SetBy        STRING(36)  NOT NULL,     -- UserId who set this override
    SetByRole    STRING(20)  NOT NULL,     -- GLOBAL_ADMIN | NODE_ADMIN
    IsActive     BOOL        NOT NULL DEFAULT (true),
    Notes        STRING(MAX),              -- Optional reason / label
    CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    ExpiresAt    TIMESTAMP                 -- NULL = no expiry
) PRIMARY KEY (OverrideId);

CREATE INDEX Idx_PricingOverrides_ByRetailer ON RetailerPricingOverrides(SupplierId, RetailerId, SkuId)
    WHERE IsActive = true;
CREATE INDEX Idx_PricingOverrides_ByWarehouse ON RetailerPricingOverrides(SupplierId, WarehouseId)
    WHERE IsActive = true;
CREATE UNIQUE INDEX Idx_PricingOverrides_Unique ON RetailerPricingOverrides(SupplierId, RetailerId, SkuId)
    WHERE IsActive = true;

-- ── VI.3: SUPPLIER-RETAILER CLIENT MATERIALISED WAREHOUSE ASSIGNMENT ─────────
-- PrimaryWarehouseId is a read-cache updated by the spatial engine and Kafka
-- background workers. Used for NODE_ADMIN RBAC scoping without running geo
-- queries on every admin page load.
CREATE TABLE SupplierRetailerClients (
    SupplierId         STRING(36) NOT NULL,
    RetailerId         STRING(36) NOT NULL,
    PrimaryWarehouseId STRING(36),          -- Materialised from spatial engine
    RetailerTier       STRING(20) NOT NULL DEFAULT ('BRONZE'), -- BRONZE | SILVER | GOLD
    CreatedAt          TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SupplierId, RetailerId);

CREATE INDEX Idx_SupplierRetailers_ByWarehouse ON SupplierRetailerClients(SupplierId, PrimaryWarehouseId);
CREATE INDEX Idx_SupplierRetailers_ByRetailer ON SupplierRetailerClients(RetailerId);

-- ── VI.4: STOCK ALERTS / MIN-STOCK THRESHOLDS ───────────────────────────────
-- Per-SKU, per-warehouse minimum stock level. When QuantityAvailable drops below
-- MinStockLevel, the replenishment engine flags it and can trigger supply requests.
ALTER TABLE SupplierInventory ADD COLUMN MinStockLevel INT64 DEFAULT (0);
ALTER TABLE SupplierInventory ADD COLUMN MaxStockLevel INT64;

-- ═══════════════════════════════════════════════════════════════════════════════
-- Phase VII: Cross-Warehouse Freight Surcharge — DELIVERY ZONES + ORDER FEE
-- ═══════════════════════════════════════════════════════════════════════════════

-- VII.1: DeliveryFee + FulfillmentWarehouseId on Orders
-- DeliveryFee is INT64 minor currency (sum). FulfillmentWarehouseId tracks which
-- warehouse shipped the order (NULL = primary / local warehouse, non-NULL = cross-warehouse).
ALTER TABLE Orders ADD COLUMN DeliveryFee INT64 DEFAULT (0);
ALTER TABLE Orders ADD COLUMN FulfillmentWarehouseId STRING(36);

-- VII.2: DeliveryZones — supplier-defined distance bands with per-band fee
-- Allows suppliers to define tiered freight charges by distance from warehouse.
CREATE TABLE DeliveryZones (
    ZoneId       STRING(36) NOT NULL,
    SupplierId   STRING(36) NOT NULL,
    WarehouseId  STRING(36),               -- NULL = default zones for all warehouses
    ZoneName     STRING(100) NOT NULL,      -- e.g. "City Center", "Suburban", "Remote"
    MinDistanceKm FLOAT64 NOT NULL DEFAULT (0),
    MaxDistanceKm FLOAT64 NOT NULL,
    FeeMinor     INT64 NOT NULL DEFAULT (0), -- delivery fee in minor currency units
    Priority     INT64 NOT NULL DEFAULT (0), -- for overlapping zone resolution
    IsActive     BOOL NOT NULL DEFAULT (true),
    CreatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt    TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SupplierId, ZoneId);

CREATE INDEX Idx_DeliveryZones_Lookup ON DeliveryZones(SupplierId, WarehouseId, IsActive);

-- VII.3: WarehouseId on SupplierInventory (warehouse-scoped stock)
ALTER TABLE SupplierInventory ADD COLUMN WarehouseId STRING(36);

-- ═══════════════════════════════════════════════════════════════════════════════
-- Phase VIII: REPLENISHMENT GRAPH HARDENING — Supply Lanes, Distributed Locks,
-- Factory SLA, Network Optimization Mode, Pull Matrix Audit.
-- ═══════════════════════════════════════════════════════════════════════════════

-- VIII.1: SUPPLY LANES — directed factory→warehouse routing edges
-- Each lane carries transit time, freight cost, and carbon weight for
-- multi-objective optimization (SPEED / ECONOMY / BALANCED / LOW_CARBON).
-- DampenedTransitHours is an EMA-smoothed value resistant to minute-level noise.
CREATE TABLE SupplyLanes (
    LaneId              STRING(36) NOT NULL,
    SupplierId          STRING(36) NOT NULL,
    FactoryId           STRING(36) NOT NULL,
    WarehouseId         STRING(36) NOT NULL,
    TransitTimeHours    FLOAT64 NOT NULL DEFAULT (24),
    DampenedTransitHours FLOAT64 NOT NULL DEFAULT (24),
    FreightCostMinor    INT64 NOT NULL DEFAULT (0),
    CarbonScoreKg       FLOAT64 NOT NULL DEFAULT (0),
    IsActive            BOOL NOT NULL DEFAULT (true),
    Priority            INT64 NOT NULL DEFAULT (0),
    LastTransitUpdate   TIMESTAMP,
    CreatedAt           TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt           TIMESTAMP OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SupplierId, LaneId);

CREATE INDEX Idx_SupplyLanes_ByFactory ON SupplyLanes(SupplierId, FactoryId)
    WHERE IsActive = true;
CREATE INDEX Idx_SupplyLanes_ByWarehouse ON SupplyLanes(SupplierId, WarehouseId)
    WHERE IsActive = true;
CREATE UNIQUE INDEX Idx_SupplyLanes_Edge ON SupplyLanes(SupplierId, FactoryId, WarehouseId)
    WHERE IsActive = true;

-- VIII.2: REPLENISHMENT LOCKS — distributed advisory lock for concurrent
-- warehouse threshold triggers competing for same factory capacity.
-- Priority is based on 30-day sales velocity score.
CREATE TABLE ReplenishmentLocks (
    LockKey     STRING(200) NOT NULL,  -- format: "SKU:{skuId}:FACTORY:{factoryId}"
    AcquiredBy  STRING(36) NOT NULL,   -- WarehouseId that holds the lock
    SupplierId  STRING(36) NOT NULL,
    Priority    FLOAT64 NOT NULL DEFAULT (0),  -- sales velocity score
    AcquiredAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    ExpiresAt   TIMESTAMP NOT NULL
) PRIMARY KEY (LockKey);

CREATE INDEX Idx_ReplenishmentLocks_ByExpiry ON ReplenishmentLocks(ExpiresAt);

-- VIII.3: FACTORY SLA EVENTS — tracks factory promise vs delivery performance.
-- Escalation levels: WARNING (50%), CRITICAL (100%), AUTO_REROUTE (150%).
CREATE TABLE FactorySLAEvents (
    EventId            STRING(36) NOT NULL,
    TransferId         STRING(36) NOT NULL,
    SupplierId         STRING(36) NOT NULL,
    FactoryId          STRING(36) NOT NULL,
    WarehouseId        STRING(36) NOT NULL,
    EscalationLevel    STRING(20) NOT NULL,  -- WARNING | CRITICAL | AUTO_REROUTE
    PromisedAt         TIMESTAMP,
    ActualAt           TIMESTAMP,
    SLABreachMinutes   INT64 NOT NULL DEFAULT (0),
    ReplacementTransferId STRING(36),        -- set if AUTO_REROUTE created a replacement
    CreatedAt          TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EventId);

CREATE INDEX Idx_FactorySLA_ByTransfer ON FactorySLAEvents(TransferId);
CREATE INDEX Idx_FactorySLA_ByFactory ON FactorySLAEvents(SupplierId, FactoryId, CreatedAt DESC);

-- VIII.4: NETWORK OPTIMIZATION MODE — per-supplier toggle for routing objective.
-- SPEED: fastest transit. ECONOMY: cheapest freight. BALANCED: weighted mix.
-- LOW_CARBON: minimum emissions. MANUAL_ONLY: kill switch — no automated transfers.
CREATE TABLE NetworkOptimizationMode (
    SupplierId  STRING(36) NOT NULL,
    Mode        STRING(30) NOT NULL DEFAULT ('BALANCED'),  -- SPEED|ECONOMY|BALANCED|LOW_CARBON|MANUAL_ONLY
    UpdatedAt   TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedBy   STRING(36) NOT NULL
) PRIMARY KEY (SupplierId);

-- VIII.5: PULL MATRIX RUNS — audit trail for automated replenishment cycles.
CREATE TABLE PullMatrixRuns (
    RunId              STRING(36) NOT NULL,
    SupplierId         STRING(36) NOT NULL,
    RunAt              TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    TransfersGenerated INT64 NOT NULL DEFAULT (0),
    SKUsProcessed      INT64 NOT NULL DEFAULT (0),
    DurationMs         INT64 NOT NULL DEFAULT (0),
    Source             STRING(30) NOT NULL DEFAULT ('CRON'),  -- CRON | EVENT_TRIGGERED | MANUAL
    Notes              STRING(MAX)
) PRIMARY KEY (SupplierId, RunId);

CREATE INDEX Idx_PullMatrixRuns_ByTime ON PullMatrixRuns(SupplierId, RunAt DESC);

-- VIII.6: Safety stock configuration columns
ALTER TABLE Warehouses ADD COLUMN SafetyStockDays INT64 DEFAULT (3);
ALTER TABLE SupplierInventory ADD COLUMN SafetyStockLevel INT64 DEFAULT (0);

-- VIII.7: Factory daily output capacity (for concurrency collision priority)
ALTER TABLE Factories ADD COLUMN DailyOutputCapacity INT64 DEFAULT (0);

-- VIII.8: Dynamic Supply Chain — factory load tracking & lane enrichment
ALTER TABLE Factories ADD COLUMN CurrentLoad INT64 DEFAULT (0); -- Rolling 24h factory output counter
ALTER TABLE Factories ADD COLUMN LastLoadUpdate DATE; -- JIT self-healing: tracks last calendar day CurrentLoad was written
ALTER TABLE SupplyLanes ADD COLUMN ExternalEnrichmentEnabled BOOL DEFAULT (false); -- API shadow flag (future Distance Matrix)
ALTER TABLE SupplyLanes ADD COLUMN DirectDistanceKm FLOAT64; -- Haversine-computed straight-line distance at creation

-- VIII.9: Force-seal rate limiting audit ledger
CREATE TABLE SupplierOverrides (
    SupplierId   STRING(36)  NOT NULL,
    OverrideId   STRING(36)  NOT NULL,
    OverrideType STRING(30)  NOT NULL DEFAULT ('FORCE_SEAL'), -- FORCE_SEAL | other future types
    Timestamp    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    Reason       STRING(MAX)
) PRIMARY KEY (SupplierId, OverrideId);

CREATE INDEX Idx_SupplierOverrides_ByTime ON SupplierOverrides(SupplierId, Timestamp DESC);

-- ── SOVEREIGNTY PROTOCOL: SupplierUsers expansion ─────────────────────────────
-- Adds factory scope column and expands role CHECK to universal identity.
-- Apply these ALTER statements for existing deployments (new deploys use the
-- updated CREATE TABLE above).
ALTER TABLE SupplierUsers ADD COLUMN AssignedFactoryId STRING(36);
ALTER TABLE SupplierUsers DROP CONSTRAINT CHK_SupplierRole;
ALTER TABLE SupplierUsers ADD CONSTRAINT CHK_SupplierRole
    CHECK (SupplierRole IN ('GLOBAL_ADMIN', 'NODE_ADMIN', 'FACTORY_ADMIN', 'FACTORY_PAYLOADER'));
ALTER TABLE SupplierUsers ALTER COLUMN SupplierRole STRING(30) NOT NULL;

-- ══════════════════════════════════════════════════════════════════════════════
-- Phase IX: COVERAGE INTEGRITY — DispatchAudit & Orphan Retailer Detection
-- ══════════════════════════════════════════════════════════════════════════════
-- The VerifyCoverageConsistency background job writes to this table whenever
-- warehouse H3 coverage or retailer location changes. Flags "orphaned" retailers
-- whose H3 cell has zero warehouse coverage — preventing un-routable orders.

CREATE TABLE DispatchAudit (
    AuditId        STRING(36)  NOT NULL,
    SupplierId     STRING(36)  NOT NULL,
    RetailerId     STRING(36)  NOT NULL,
    RetailerCell   STRING(100) NOT NULL,     -- H3 cell ID from LookupCell(lat, lng)
    AuditType      STRING(30)  NOT NULL,     -- ORPHAN_DETECTED | COVERAGE_RESTORED | COVERAGE_GAP
    WarehouseId    STRING(36),               -- NULL when orphaned, set when coverage restored
    DistanceKm     FLOAT64,                  -- Haversine distance to nearest warehouse (NULL if none)
    ResolvedAt     TIMESTAMP,                -- set when a warehouse later covers this cell
    CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_DispatchAuditType CHECK (
        AuditType IN ('ORPHAN_DETECTED', 'COVERAGE_RESTORED', 'COVERAGE_GAP')
    )
) PRIMARY KEY (AuditId);

CREATE INDEX Idx_DispatchAudit_BySupplierId ON DispatchAudit(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_DispatchAudit_Unresolved ON DispatchAudit(SupplierId, AuditType, ResolvedAt)
    WHERE ResolvedAt IS NULL;
CREATE INDEX Idx_DispatchAudit_ByRetailer ON DispatchAudit(RetailerId, CreatedAt DESC);

-- ── V.O.I.D. Phase VII: Home Node Lifecycle ──────────────────────────────────
-- Drivers and Vehicles are home-based at a specific Warehouse OR Factory node.
-- HomeNodeType/HomeNodeId are the canonical fields; WarehouseId is retained as
-- a denormalised field during the migration window and MUST stay populated
-- whenever HomeNodeType = 'WAREHOUSE'.
--
-- HomeNodeType CHECK: WAREHOUSE | FACTORY
-- HomeNodeId: resolves to Warehouses.WarehouseId or Factories.FactoryId
-- depending on HomeNodeType.
ALTER TABLE Drivers  ADD COLUMN HomeNodeType STRING(20);
ALTER TABLE Drivers  ADD COLUMN HomeNodeId   STRING(36);
ALTER TABLE Vehicles ADD COLUMN HomeNodeType STRING(20);
ALTER TABLE Vehicles ADD COLUMN HomeNodeId   STRING(36);
CREATE INDEX Idx_Drivers_ByHomeNode  ON Drivers(HomeNodeType, HomeNodeId);
CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId);

-- ── V.O.I.D. Phase VII: Transactional Outbox ────────────────────────────────
-- The single mechanism for publishing durable state-change Kafka events. A
-- handler that mutates an entity writes both the entity row AND an
-- OutboxEvents row inside the same spanner.ReadWriteTransaction; the outbox
-- relay (backend-go/outbox.Relay) tails unpublished rows and publishes them
-- to Kafka, marking PublishedAt on success. If the handler transaction
-- aborts, both rows disappear — no "ghost" events possible.
CREATE TABLE OutboxEvents (
    EventId       STRING(36)   NOT NULL,
    AggregateType STRING(30)   NOT NULL,           -- Driver | Vehicle | Order | Factory | Warehouse | ...
    AggregateId   STRING(36)   NOT NULL,           -- Partition key for Kafka (enforces per-entity ordering)
    EventType     STRING(60)   NOT NULL,           -- Discriminator (e.g. ORDER_DISPATCHED) — relay injects as Kafka header `event_type`
    TopicName     STRING(100)  NOT NULL,           -- Destination Kafka topic
    Payload       BYTES(MAX)   NOT NULL,           -- JSON-encoded event body
    TraceID       STRING(36),                       -- Request-scoped correlation token (Glass Box tracing)
    CreatedAt     TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
    PublishedAt   TIMESTAMP,                        -- NULL until relay publishes
) PRIMARY KEY (EventId);

CREATE INDEX Idx_OutboxEvents_Unpublished
    ON OutboxEvents(CreatedAt)
    WHERE PublishedAt IS NULL;

-- ══════════════════════════════════════════════════════════════════════════════
-- PHASE IX: PRE-ORDER LIFECYCLE & NOTIFICATION HARDENING
-- ══════════════════════════════════════════════════════════════════════════════

-- NudgeNotifiedAt — tracks when the T-5 soft reminder was sent (used by cron to avoid re-sends).
ALTER TABLE Orders ADD COLUMN NudgeNotifiedAt TIMESTAMP;

-- PreorderReminderSentAt — tracks periodic 2-day reminder cadence for long-horizon preorders (>1 week).
ALTER TABLE Orders ADD COLUMN PreorderReminderSentAt TIMESTAMP;

-- Index for the auto-accept sweep: SCHEDULED orders past T-4 midnight with cancel lock but not yet promoted.
CREATE INDEX Idx_Orders_ByAutoAcceptEligible
    ON Orders(State, RequestedDeliveryDate)
    WHERE State = 'SCHEDULED' AND CancelLockedAt IS NOT NULL;

-- ══ PHASE X: GLOBAL PIN UNIQUENESS & FACTORY PIN MIGRATION ══════════════════

-- GlobalPins stores a deterministic SHA-256 hash of every active PIN in the
-- system. bcrypt is salted (same PIN → different hash each time), so a
-- UNIQUE index on PinHash is impossible. SHA-256(plaintext) is deterministic
-- and serves as the collision-detection key. The bcrypt hash lives in the
-- entity table for authentication; the SHA-256 lives here for uniqueness.
CREATE TABLE GlobalPins (
    PinSha256   STRING(64)  NOT NULL,  -- hex-encoded SHA-256 of the plaintext PIN
    EntityType  STRING(30)  NOT NULL,  -- DRIVER | WAREHOUSE_STAFF | FACTORY_STAFF
    EntityId    STRING(36)  NOT NULL,
    CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PinSha256);

-- Reverse lookup: find the pin record for a given entity so we can DELETE the
-- old row when a PIN is rotated.
CREATE INDEX Idx_GlobalPins_ByEntity ON GlobalPins(EntityType, EntityId);

-- FactoryStaff dual-auth migration: add PinHash for PIN-based login alongside
-- the existing PasswordHash. Nullable during the migration window — factory
-- staff created before this migration retain password-only auth until their
-- PIN is provisioned.
ALTER TABLE FactoryStaff ADD COLUMN PinHash STRING(MAX);

-- ══════════════════════════════════════════════════════════════════════════════
-- Phase VI-B: Exception Recovery States + IsRecovery Flag
-- ══════════════════════════════════════════════════════════════════════════════

-- Add READY_FOR_DISPATCH (overflow-bounced orders) and CANCELLED_BY_ORIGIN
-- (admin hard-kill) to the Order state machine.
ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State;
ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN (
    'PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT',
    'ARRIVING', 'ARRIVED', 'ARRIVED_SHOP_CLOSED', 'AWAITING_GLOBAL_PAYNT', 'PENDING_CASH_COLLECTION',
    'COMPLETED', 'CANCELLED', 'CANCEL_REQUESTED', 'NO_CAPACITY', 'DELIVERED_ON_CREDIT',
    'SCHEDULED', 'AUTO_ACCEPTED', 'QUARANTINE', 'STALE_AUDIT', 'REFUNDED', 'DELAYED',
    'READY_FOR_DISPATCH', 'CANCELLED_BY_ORIGIN'));

-- IsRecovery: set TRUE when an overflow-bounced order returns to the dispatch
-- pool. Clarke-Wright solver reads this to add a priority savings boost
-- (recovery orders get first dibs on vehicle volume).
ALTER TABLE Orders ADD COLUMN IsRecovery BOOL;

