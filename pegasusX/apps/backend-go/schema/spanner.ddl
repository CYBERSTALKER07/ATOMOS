-- pegasusX Spanner DDL (seed)
--
-- Authoritative schema lives under apps/backend-go/schema/. Migrations are
-- applied via apps/backend-go/cmd/setup. This file is the canonical bootstrap
-- definition consumed by emulator setup and production provisioning.
--
-- Single-tenant note: SupplierId remains on every supplier-owned table. One
-- Suppliers row is seeded at bootstrap; the column stays for migration parity
-- with Pegasus.

CREATE TABLE Suppliers (
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  CountryCode      STRING(2)     NOT NULL,
  Currency         STRING(3)     NOT NULL,
  IsConfigured     BOOL          NOT NULL DEFAULT (FALSE),
  RegionId         STRING(36),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE TABLE SupplierProfiles (
  SupplierId              STRING(36)    NOT NULL,
  ContactName             STRING(255),
  Email                   STRING(320),
  Phone                   STRING(32),
  WarehouseName           STRING(255),
  WarehouseAddress        STRING(MAX),
  WarehouseLat            FLOAT64,
  WarehouseLng            FLOAT64,
  BillingSameAsWarehouse  BOOL          NOT NULL DEFAULT (TRUE),
  BillingAddress          STRING(MAX),
  BillingLat              FLOAT64,
  BillingLng              FLOAT64,
  TaxId                   STRING(64),
  CompanyRegNumber        STRING(128),
  FleetVehicleCount       INT64         NOT NULL DEFAULT (0),
  FleetMaxVU              INT64         NOT NULL DEFAULT (0),
  FactoryCount            INT64         NOT NULL DEFAULT (0),
  CategoriesJson          BYTES(MAX),
  IsRegistered            BOOL          NOT NULL DEFAULT (FALSE),
  BankName                STRING(255),
  AccountHolder           STRING(255),
  AccountNumber           STRING(128),
  SwiftBic                STRING(64),
  IBAN                    STRING(128),
  SelectedGatewaysJson    BYTES(MAX),
  PaymentAcceptor         STRING(16)    NOT NULL DEFAULT ('SUPPLIER'),
  RegisteredAt            TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ConfiguredAt            TIMESTAMP,
  UpdatedAt               TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE INDEX Idx_SupplierProfiles_ByUpdatedAt ON SupplierProfiles(UpdatedAt DESC);

CREATE TABLE PaymentConfigs (
  PaymentConfigId      STRING(36)    NOT NULL,
  SupplierId           STRING(36)    NOT NULL,
  WarehouseId          STRING(36)    NOT NULL,
  SelectedGatewaysJson BYTES(MAX)   NOT NULL,
  CreatedAt            TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt            TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PaymentConfigId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_PaymentConfigs_ByWarehouse ON PaymentConfigs(WarehouseId);

CREATE TABLE SupplierPricingRules (
  SupplierId          STRING(36)    NOT NULL,
  BaseMarkupBps       INT64         NOT NULL,
  RetailerDiscountBps INT64         NOT NULL,
  MinMarginBps        INT64         NOT NULL,
  Currency            STRING(3)     NOT NULL,
  RuleVersion         INT64         NOT NULL,
  UpdatedBy           STRING(128),
  UpdatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE INDEX Idx_SupplierPricingRules_ByUpdatedAt ON SupplierPricingRules(UpdatedAt DESC);

-- Per-retailer absolute price overrides (most specific pricing tier).
-- Resolution order: RetailerPricingOverride -> promotions -> catalog list price.
CREATE TABLE RetailerPricingOverrides (
  OverrideId     STRING(36)  NOT NULL,
  SupplierId     STRING(36)  NOT NULL,
  RetailerId     STRING(36)  NOT NULL,
  ProductId      STRING(36)  NOT NULL,
  OverridePrice  INT64       NOT NULL,
  SetBy          STRING(128) NOT NULL,
  SetByRole      STRING(32)  NOT NULL,
  IsActive       BOOL        NOT NULL,
  Notes          STRING(MAX),
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ExpiresAt      TIMESTAMP,
) PRIMARY KEY (OverrideId);

CREATE INDEX Idx_PricingOverrides_ByRetailer ON RetailerPricingOverrides(SupplierId, RetailerId, ProductId, IsActive);
CREATE INDEX Idx_PricingOverrides_BySupplier ON RetailerPricingOverrides(SupplierId, IsActive, CreatedAt DESC);

CREATE TABLE SupplierPromotions (
  PromotionId         STRING(36)    NOT NULL,
  SupplierId          STRING(36)    NOT NULL,
  Name                STRING(255)   NOT NULL,
  Description         STRING(MAX),
  DiscountBps         INT64         NOT NULL,
  ScopeType           STRING(32)    NOT NULL,
  ScopeProductId      STRING(36),
  ScopeCategoryId     STRING(36),
  RetailerScope       STRING(32)    NOT NULL,
  RetailerIdsJson     BYTES(MAX),
  MinLineQuantity     INT64,
  MinOrderAmountMinor INT64,
  StartsAt            TIMESTAMP,
  EndsAt              TIMESTAMP,
  IsActive            BOOL          NOT NULL,
  Priority            INT64         NOT NULL DEFAULT (0),
  Version             INT64         NOT NULL DEFAULT (1),
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PromotionId);

CREATE INDEX Idx_SupplierPromotions_BySupplierActive ON SupplierPromotions(SupplierId, IsActive, UpdatedAt DESC);
CREATE INDEX Idx_SupplierPromotions_ByProduct ON SupplierPromotions(SupplierId, ScopeProductId, IsActive);
CREATE INDEX Idx_SupplierPromotions_ByCategory ON SupplierPromotions(SupplierId, ScopeCategoryId, IsActive);

CREATE TABLE Retailers (
  RetailerId              STRING(36)    NOT NULL,
  Phone                   STRING(32)    NOT NULL,
  Name                    STRING(255),
  CountryCode             STRING(2)     NOT NULL,
  Lat                     FLOAT64,
  Lng                     FLOAT64,
  H3Cell                  STRING(15),
  DeliveryAddress         STRING(MAX),
  PlaceId                 STRING(128),
  ReceivingWindowOpen     STRING(10),
  ReceivingWindowClose    STRING(10),
  RegionId                STRING(36),
  CreatedAt               TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId);

CREATE UNIQUE INDEX UQ_Retailers_Phone ON Retailers(Phone);
CREATE INDEX Idx_Retailers_ByH3Cell ON Retailers(H3Cell);

CREATE TABLE Orders (
  OrderId          STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  RetailerId       STRING(36)    NOT NULL,
  WarehouseId      STRING(36),
  DriverId         STRING(36),
  VehicleId        STRING(36),
  RouteId          STRING(36),
  ManifestId       STRING(36),
  DeliveryToken    STRING(36),
  Status           STRING(32)    NOT NULL,
  OrderSource      STRING(24)    NOT NULL,
  ConfirmationStatus STRING(24)  NOT NULL,
  LineItemsJson    BYTES(MAX)    NOT NULL,
  TotalMinor           INT64         NOT NULL,
  OriginalTotalMinor   INT64         NOT NULL DEFAULT (0),
  Currency             STRING(3)     NOT NULL,
  H3Cell           STRING(15),
  Lat              FLOAT64,
  Lng              FLOAT64,
  RequestedDeliveryDate TIMESTAMP,
  AutoConfirmAt    TIMESTAMP,
  DecisionAt       TIMESTAMP,
  DecisionBy       STRING(64),
  DerivedFromOrderId STRING(36),
  ReceivingWindowOpen  STRING(10),
  ReceivingWindowClose STRING(10),
  ProposedDeliveryDate TIMESTAMP,
  DeliveryProposalAt TIMESTAMP,
  DeliveryProposalBy STRING(128),
  DeliveryProposalReason STRING(512),
  Version          INT64         NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId);

CREATE INDEX Idx_Orders_ByRetailerCreated ON Orders(RetailerId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByRetailerConfirmation ON Orders(RetailerId, ConfirmationStatus, RequestedDeliveryDate DESC, UpdatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierCreated ON Orders(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByWarehouseCreated ON Orders(WarehouseId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByWarehouseRequestedDelivery ON Orders(WarehouseId, RequestedDeliveryDate DESC, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByConfirmationAutoConfirm ON Orders(ConfirmationStatus, AutoConfirmAt, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByDerivedSource ON Orders(DerivedFromOrderId, OrderSource, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByDriverCreated ON Orders(DriverId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByRouteCreated ON Orders(RouteId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByManifestCreated ON Orders(ManifestId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByH3Cell ON Orders(H3Cell, Status, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByStatusWarehouse ON Orders(Status, WarehouseId, CreatedAt DESC);

CREATE TABLE SupplierReturns (
  ReturnId          STRING(36)  NOT NULL,
  OrderId           STRING(36)  NOT NULL,
  SkuId             STRING(50)  NOT NULL,
  RejectedQty       INT64       NOT NULL,
  Reason            STRING(50)  NOT NULL,
  DriverNotes       STRING(MAX),
  Status            STRING(32)  NOT NULL DEFAULT ('PENDING'),
  ResolvedAt        TIMESTAMP,
  ResolutionNotes   STRING(MAX),
  ManifestId        STRING(36),
  DriverId          STRING(36),
  WarehouseId       STRING(36),
  ExpectedQty       INT64,
  ReceivedQty       INT64       NOT NULL DEFAULT (0),
  PhysicalStatus    STRING(32)  NOT NULL DEFAULT ('PENDING'),
  ReceivedAt        TIMESTAMP,
  ReceivedBy          STRING(36),
  ReceiveSessionId  STRING(36),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ReturnId);

CREATE INDEX Idx_SupplierReturns_ByOrder ON SupplierReturns(OrderId);
CREATE INDEX Idx_SupplierReturns_BySku ON SupplierReturns(SkuId);
CREATE INDEX Idx_SupplierReturns_ByStatus ON SupplierReturns(Status, CreatedAt DESC);
CREATE INDEX Idx_SupplierReturns_ByPhysicalStatus ON SupplierReturns(PhysicalStatus, CreatedAt DESC);
CREATE INDEX Idx_SupplierReturns_ByManifest ON SupplierReturns(ManifestId, PhysicalStatus);
CREATE INDEX Idx_SupplierReturns_ByWarehousePhysical ON SupplierReturns(WarehouseId, PhysicalStatus, CreatedAt DESC);
CREATE INDEX Idx_SupplierReturns_ByDriverPhysical ON SupplierReturns(DriverId, PhysicalStatus, CreatedAt DESC);

CREATE TABLE ReturnReceiveSessions (
  SessionId     STRING(36)  NOT NULL,
  WarehouseId   STRING(36)  NOT NULL,
  ManifestId    STRING(36),
  DriverId      STRING(36),
  OperatorId    STRING(36)  NOT NULL,
  OperatorRole  STRING(32)  NOT NULL,
  Status        STRING(32)  NOT NULL DEFAULT ('OPEN'),
  StartedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CompletedAt   TIMESTAMP,
) PRIMARY KEY (SessionId);

CREATE INDEX Idx_ReturnReceiveSessions_ByWarehouse ON ReturnReceiveSessions(WarehouseId, Status, StartedAt DESC);

CREATE TABLE OrderDeliveryProofs (
  ProofId           STRING(36)    NOT NULL,
  OrderId           STRING(36)    NOT NULL,
  SupplierId        STRING(36)    NOT NULL,
  RetailerId        STRING(36)    NOT NULL,
  DriverId          STRING(36)    NOT NULL,
  ProofType         STRING(32)    NOT NULL,
  QRTokenHash       STRING(64),
  ScannedTokenHash  STRING(64),
  Latitude          FLOAT64,
  Longitude         FLOAT64,
  DistanceM         FLOAT64,
  CapturedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProofId);

CREATE INDEX Idx_OrderDeliveryProofs_ByOrderCaptured ON OrderDeliveryProofs(OrderId, CapturedAt DESC);

CREATE TABLE Drivers (
  DriverId        STRING(36)    NOT NULL,
  Name            STRING(255)   NOT NULL,
  Phone           STRING(32)    NOT NULL,
  PinHash         STRING(MAX),
  SupplierId      STRING(36)    NOT NULL,
  HomeNodeType    STRING(20)    NOT NULL,
  HomeNodeId      STRING(36)    NOT NULL,
  VehicleId       STRING(36),
  IsActive        BOOL          NOT NULL,
  OnShift         BOOL          NOT NULL DEFAULT (true),
  UnavailableReason STRING(64),
  UnavailableNote STRING(255),
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DriverId);

CREATE INDEX Idx_Drivers_BySupplierPhone ON Drivers(SupplierId, Phone);
CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId, IsActive);

-- Vehicle classes (volumetric units):
--   CLASS_A (Damass/Minivan) = 50 VU
--   CLASS_B (Transit Van)    = 150 VU
--   CLASS_C (Box Truck)      = 400 VU
CREATE TABLE Vehicles (
  VehicleId       STRING(36)    NOT NULL,
  Label           STRING(100),
  LicensePlate    STRING(32)    NOT NULL,
  SupplierId      STRING(36)    NOT NULL,
  HomeNodeType    STRING(20)    NOT NULL,
  HomeNodeId      STRING(36)    NOT NULL,
  VehicleClass    STRING(10)    NOT NULL DEFAULT ('CLASS_B'),
  MaxVolumeVU     FLOAT64       NOT NULL DEFAULT (150.0),
  IsActive        BOOL          NOT NULL,
  UnavailableReason STRING(64),
  UnavailableNote STRING(255),
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (VehicleId);

CREATE INDEX Idx_Vehicles_BySupplierPlate ON Vehicles(SupplierId, LicensePlate);
CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId, IsActive);

CREATE TABLE Warehouses (
  WarehouseId        STRING(36)    NOT NULL,
  SupplierId         STRING(36)    NOT NULL,
  Name               STRING(255)   NOT NULL,
  Lat                FLOAT64,
  Lng                FLOAT64,
  Address            STRING(MAX),
  PlaceId            STRING(128),
  CoverageRadiusKm   FLOAT64       NOT NULL,
  PrimaryFactoryId   STRING(36),
  SecondaryFactoryId STRING(36),
  TransferMode       STRING(10)    NOT NULL DEFAULT ('TRUCK'),
  CoLocateWithFactoryId STRING(36),
  IsActive           BOOL          NOT NULL,
  IsOnShift          BOOL          NOT NULL,
  RegionId           STRING(36),
  PaymentConfigId    STRING(36),
  AutoDispatchEnabled BOOL         NOT NULL DEFAULT (FALSE),
  DefaultOutOfStockPolicy STRING(24) NOT NULL DEFAULT ('REJECT'),
  ShowStockCountsToRetailers BOOL NOT NULL DEFAULT (FALSE),
  PreorderMinLeadDays INT64 NOT NULL DEFAULT (3),
  PreorderMaxLeadDays INT64 NOT NULL DEFAULT (90),
  OrderLineMinQuantity INT64,
  OrderLineMaxQuantity INT64,
  DeliveryFeeRules JSON,
  OperatingSchedule  JSON,
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplier ON Warehouses(SupplierId);
CREATE INDEX Idx_Warehouses_ByPrimaryFactory ON Warehouses(SupplierId, PrimaryFactoryId);
CREATE INDEX Idx_Warehouses_ByAutoDispatch ON Warehouses(AutoDispatchEnabled, IsActive, WarehouseId) STORING (SupplierId);

CREATE TABLE Factories (
  FactoryId        STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  Address          STRING(MAX),
  PlaceId          STRING(128),
  IsActive         BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (FactoryId);

CREATE INDEX Idx_Factories_BySupplier ON Factories(SupplierId);

CREATE TABLE WarehouseSupplyRequests (
  RequestId                   STRING(36)    NOT NULL,
  SupplierId                  STRING(36)    NOT NULL,
  WarehouseId                 STRING(36)    NOT NULL,
  State                       STRING(24)    NOT NULL,
  RequestedBy                 STRING(128),
  CoverageStartDate           STRING(10)    NOT NULL,
  CoverageDays                INT64         NOT NULL,
  ProjectedUnits              INT64         NOT NULL,
  CommittedUnits              INT64         NOT NULL,
  PendingConfirmationUnits    INT64         NOT NULL,
  FactoryId                   STRING(36),
  TransferMode                STRING(10),
  LinkedTransferId            STRING(36),
  Priority                    STRING(16),
  Notes                       STRING(MAX),
  RegionId                    STRING(36),
  RequestedDeliveryDate       TIMESTAMP,
  DemandBreakdown             JSON,
  TotalVolumeVU               FLOAT64       NOT NULL DEFAULT (0),
  CreatedAt                   TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                   TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId);

CREATE TABLE WarehouseSupplyRequestItems (
  RequestId            STRING(36)  NOT NULL,
  ItemId               STRING(36)  NOT NULL,
  ProductId            STRING(36)  NOT NULL,
  RequestedQuantity    INT64       NOT NULL,
  RecommendedQuantity  INT64       NOT NULL DEFAULT (0),
  UnitVolumeVU         FLOAT64     NOT NULL DEFAULT (0),
  ShippedQuantity      INT64,
  ReceivedQuantity     INT64,
  VarianceReason       STRING(MAX),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId, ItemId),
  INTERLEAVE IN PARENT WarehouseSupplyRequests ON DELETE CASCADE;

CREATE INDEX Idx_WarehouseSupplyRequests_ByWarehouseUpdated ON WarehouseSupplyRequests(WarehouseId, UpdatedAt DESC);
CREATE INDEX Idx_WarehouseSupplyRequests_BySupplierWarehouseCreated ON WarehouseSupplyRequests(SupplierId, WarehouseId, CreatedAt DESC);

CREATE TABLE SupplierUsers (
  UserId               STRING(36)  NOT NULL,
  SupplierId           STRING(36)  NOT NULL,
  Email                STRING(MAX),
  Phone                STRING(32),
  Name                 STRING(MAX) NOT NULL,
  PasswordHash         STRING(MAX) NOT NULL,
  SupplierRole         STRING(30)  NOT NULL,
  AssignedWarehouseId  STRING(36),
  AssignedFactoryId    STRING(36),
  IsActive             BOOL        NOT NULL,
  FirebaseUid          STRING(128),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId);

CREATE INDEX Idx_SupplierUsers_ByPhone ON SupplierUsers(Phone);
CREATE INDEX Idx_SupplierUsers_BySupplierUpdated ON SupplierUsers(SupplierId, UpdatedAt DESC);

CREATE TABLE PaymentSessions (
  SessionId         STRING(36)    NOT NULL,
  OrderId           STRING(36)    NOT NULL,
  SupplierId        STRING(36)    NOT NULL,
  RetailerId        STRING(36)    NOT NULL,
  Gateway           STRING(32)    NOT NULL,
  Currency          STRING(3)     NOT NULL,
  AmountMinor       INT64         NOT NULL,
  Mode              STRING(16)    NOT NULL,
  Status            STRING(32)    NOT NULL,
  CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SessionId);

-- SSMR performance isolation: primary read path is supplier+retailer scoped.
CREATE INDEX Idx_PaymentSessions_BySupplierRetailerCreated ON PaymentSessions(SupplierId, RetailerId, CreatedAt DESC);
CREATE INDEX Idx_PaymentSessions_ByOrderCreated ON PaymentSessions(OrderId, CreatedAt DESC);
CREATE INDEX Idx_PaymentSessions_ByStatusCreated ON PaymentSessions(Status, CreatedAt DESC);

CREATE TABLE PaymentAttempts (
  AttemptId          STRING(36)    NOT NULL,
  SessionId          STRING(36)    NOT NULL,
  Gateway            STRING(32)    NOT NULL,
  ExecutionAction    STRING(40),
  ExecutionMode      STRING(32),
  ProviderReference  STRING(128),
  Status             STRING(32)    NOT NULL,
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AttemptId);

CREATE INDEX Idx_PaymentAttempts_BySessionCreated ON PaymentAttempts(SessionId, CreatedAt DESC);

CREATE TABLE PaymentChargebacks (
  ChargebackId      STRING(36)    NOT NULL,
  OrderId           STRING(36)    NOT NULL,
  RetailerId        STRING(36)    NOT NULL,
  Gateway           STRING(32)    NOT NULL,
  AmountMinor       INT64         NOT NULL,
  Currency          STRING(3)     NOT NULL,
  CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ChargebackId);

CREATE INDEX Idx_PaymentChargebacks_ByOrderCreated ON PaymentChargebacks(OrderId, CreatedAt DESC);

CREATE TABLE PaymentReversals (
  ReversalId        STRING(36)    NOT NULL,
  SessionId         STRING(36)    NOT NULL,
  CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ReversalId);

CREATE INDEX Idx_PaymentReversals_BySessionCreated ON PaymentReversals(SessionId, CreatedAt DESC);

CREATE TABLE PaymentWebhooks (
  WebhookId         STRING(36)    NOT NULL,
  Gateway           STRING(32)    NOT NULL,
  TransactionId     STRING(128)   NOT NULL,
  SessionId         STRING(36),
  OrderId           STRING(36),
  Status            STRING(32)    NOT NULL,
  AmountMinor       INT64         NOT NULL,
  Currency          STRING(3)     NOT NULL,
  SignatureValid    BOOL          NOT NULL,
  ReceivedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WebhookId);

CREATE INDEX Idx_PaymentWebhooks_ByGatewayTxnReceived ON PaymentWebhooks(Gateway, TransactionId, ReceivedAt DESC);
CREATE INDEX Idx_PaymentWebhooks_ByOrderReceived ON PaymentWebhooks(OrderId, ReceivedAt DESC);

CREATE TABLE PaymentLedgerEntries (
  LedgerEntryId     STRING(64)    NOT NULL,
  SessionId         STRING(36),
  OrderId           STRING(36),
  SupplierId        STRING(36),
  RetailerId        STRING(36),
  Gateway           STRING(32)    NOT NULL,
  EntryType         STRING(64)    NOT NULL,
  AmountMinor       INT64         NOT NULL,
  Currency          STRING(3)     NOT NULL,
  ReferenceId       STRING(128),
  Source            STRING(64)    NOT NULL,
  OccurredAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LedgerEntryId);

CREATE INDEX Idx_PaymentLedger_BySupplierOccurred ON PaymentLedgerEntries(SupplierId, OccurredAt DESC);
CREATE INDEX Idx_PaymentLedger_ByOrderOccurred ON PaymentLedgerEntries(OrderId, OccurredAt DESC);
CREATE INDEX Idx_PaymentLedger_BySessionOccurred ON PaymentLedgerEntries(SessionId, OccurredAt DESC);
CREATE INDEX Idx_PaymentLedger_BySupplierGatewayEntryOccurred ON PaymentLedgerEntries(SupplierId, Gateway, EntryType, OccurredAt DESC);

CREATE TABLE OutboxEvents (
  EventId          STRING(36)    NOT NULL,
  AggregateType    STRING(64)    NOT NULL,
  AggregateId      STRING(64)    NOT NULL,
  TopicName        STRING(128)   NOT NULL,
  Payload          BYTES(MAX)    NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  PublishedAt      TIMESTAMP,
) PRIMARY KEY (EventId);

CREATE INDEX Idx_OutboxEvents_Unpublished ON OutboxEvents(PublishedAt, CreatedAt);

CREATE TABLE AIPredictions (
  PredictionId    STRING(36)    NOT NULL,
  AggregateId     STRING(36)    NOT NULL,
  AggregateType   STRING(20)    NOT NULL,
  SupplierId      STRING(36)    NOT NULL,
  PredictionData  BYTES(MAX)    NOT NULL,
  Score           FLOAT64       NOT NULL,
  Status          STRING(20)    NOT NULL,
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (PredictionId);

CREATE INDEX Idx_AIPredictions_BySupplierCreated ON AIPredictions(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_AIPredictions_BySupplierStatusCreated ON AIPredictions(SupplierId, Status, CreatedAt DESC);

-- ───────────────────────────────────────────────────────────────────────────────
-- Phase 1 tables — catalog, inventory, cart, notifications, audit
-- ───────────────────────────────────────────────────────────────────────────────

CREATE TABLE ProductCategories (
  CategoryId       STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  ParentCategoryId STRING(36),
  IconKey          STRING(64),
  SortOrder        INT64         NOT NULL DEFAULT (0),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CategoryId);

CREATE INDEX Idx_ProductCategories_BySupplier ON ProductCategories(SupplierId, SortOrder);

CREATE TABLE Products (
  ProductId      STRING(36)    NOT NULL,
  SupplierId     STRING(36)    NOT NULL,
  CategoryId     STRING(36)    NOT NULL,
  Name           STRING(255)   NOT NULL,
  Description    STRING(MAX),
  ImageURL       STRING(2048),
  PriceMinor     INT64         NOT NULL,
  Currency       STRING(3)     NOT NULL,
  StockQuantity  INT64         NOT NULL DEFAULT (0),
  Unit           STRING(20)    NOT NULL DEFAULT ('UNIT'),
  SaleUnit       STRING(16)    NOT NULL DEFAULT ('UNIT'),
  UnitsPerPack   INT64,
  UnitVolumeVU   FLOAT64       NOT NULL DEFAULT (1.0),
  Barcode        STRING(32),
  IsActive       BOOL          NOT NULL DEFAULT (TRUE),
  Version        INT64         NOT NULL DEFAULT (1),
  CreatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProductId);

CREATE INDEX Idx_Products_BySupplierCategory ON Products(SupplierId, CategoryId, IsActive);
CREATE INDEX Idx_Products_BySupplierActive ON Products(SupplierId, IsActive, UpdatedAt DESC);
CREATE NULL_FILTERED INDEX Idx_Products_BySupplierBarcode ON Products(SupplierId, Barcode);

CREATE TABLE InventoryLevels (
  InventoryId       STRING(36)    NOT NULL,
  ProductId         STRING(36)    NOT NULL,
  WarehouseId       STRING(36)    NOT NULL,
  SupplierId        STRING(36)    NOT NULL,
  QuantityOnHand    INT64         NOT NULL DEFAULT (0),
  QuantityReserved  INT64         NOT NULL DEFAULT (0),
  ReorderThreshold  INT64         NOT NULL DEFAULT (0),
  Version           INT64         NOT NULL DEFAULT (1),
  UpdatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (InventoryId);

CREATE INDEX Idx_InventoryLevels_ByWarehouseProduct ON InventoryLevels(WarehouseId, ProductId);
CREATE INDEX Idx_InventoryLevels_BySupplierProduct ON InventoryLevels(SupplierId, ProductId);

CREATE TABLE CartItems (
  CartItemId     STRING(36)    NOT NULL,
  RetailerId     STRING(36)    NOT NULL,
  SupplierId     STRING(36)    NOT NULL,
  ProductId      STRING(36)    NOT NULL,
  Quantity       INT64         NOT NULL DEFAULT (1),
  PriceSnapshot  INT64         NOT NULL,
  Currency       STRING(3)     NOT NULL,
  UpdatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CartItemId);

CREATE INDEX Idx_CartItems_ByRetailerSupplier ON CartItems(RetailerId, SupplierId);

CREATE TABLE Notifications (
  NotificationId  STRING(36)    NOT NULL,
  RecipientId     STRING(36)    NOT NULL,
  RecipientRole   STRING(20)    NOT NULL,
  EventType       STRING(64)    NOT NULL,
  Title           STRING(512)   NOT NULL,
  Body            STRING(MAX),
  DeepLink        STRING(512),
  IsRead          BOOL          NOT NULL DEFAULT (FALSE),
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (NotificationId);

CREATE INDEX Idx_Notifications_ByRecipientCreated ON Notifications(RecipientId, CreatedAt DESC);
CREATE INDEX Idx_Notifications_ByRecipientUnread ON Notifications(RecipientId, IsRead, CreatedAt DESC);

CREATE TABLE AuditLog (
  AuditId        STRING(36)    NOT NULL,
  SupplierId     STRING(36)    NOT NULL,
  ActorId        STRING(36)    NOT NULL,
  ActorRole      STRING(20)    NOT NULL,
  Action         STRING(64)    NOT NULL,
  AggregateType  STRING(64)    NOT NULL,
  AggregateId    STRING(64)    NOT NULL,
  DetailsJson    BYTES(MAX),
  TraceId        STRING(36),
  CreatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AuditId);

CREATE INDEX Idx_AuditLog_BySupplierCreated ON AuditLog(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_AuditLog_ByAggregateCreated ON AuditLog(AggregateType, AggregateId, CreatedAt DESC);

-- Shop-closed contact protocol: one row per driver report attempt.
CREATE TABLE ShopClosedAttempts (
  AttemptId           STRING(36)  NOT NULL,
  OrderId             STRING(36)  NOT NULL,
  DriverId            STRING(36)  NOT NULL,
  RetailerId          STRING(36)  NOT NULL,
  ReportedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  GPSLat              FLOAT64     NOT NULL,
  GPSLng              FLOAT64     NOT NULL,
  RetailerResponse    STRING(20),
  RetailerRespondedAt TIMESTAMP,
  EscalatedAt         TIMESTAMP,
  EscalatedTo         STRING(36),
  Resolution          STRING(30),
  BypassToken         STRING(6),
  ResolvedAt          TIMESTAMP,
  ResolvedBy          STRING(36)
) PRIMARY KEY (AttemptId);

CREATE INDEX Idx_ShopClosedAttempts_ByOrder ON ShopClosedAttempts(OrderId);
CREATE INDEX Idx_ShopClosedAttempts_ByDriver ON ShopClosedAttempts(DriverId);
CREATE INDEX Idx_ShopClosedAttempts_ByRetailer ON ShopClosedAttempts(RetailerId);
CREATE INDEX Idx_ShopClosedAttempts_Unresolved ON ShopClosedAttempts(Resolution);

-- Driver quantity negotiation (Edge 28): driver proposes → supplier approves/rejects.
CREATE TABLE NegotiationProposals (
  ProposalId    STRING(36)  NOT NULL,
  OrderId       STRING(36)  NOT NULL,
  DriverId      STRING(36)  NOT NULL,
  Status        STRING(20)  NOT NULL,
  ProposedItems STRING(MAX) NOT NULL,
  Resolution    STRING(200),
  ResolvedBy    STRING(36),
  ResolvedAt    TIMESTAMP,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProposalId);

CREATE INDEX Idx_NegotiationProposals_ByOrderId ON NegotiationProposals(OrderId);
CREATE INDEX Idx_NegotiationProposals_Pending ON NegotiationProposals(Status);

-- LEO loading gate: supplier truck manifests (warehouse outbound).
CREATE TABLE SupplierTruckManifests (
  ManifestId        STRING(36)  NOT NULL,
  SupplierId        STRING(36)  NOT NULL,
  WarehouseId       STRING(36),
  RouteId           STRING(MAX),
  TruckId           STRING(36)  NOT NULL,
  DriverId          STRING(36)  NOT NULL,
  State             STRING(20)  NOT NULL,
  TotalVolumeVU     FLOAT64     NOT NULL DEFAULT (0),
  MaxVolumeVU       FLOAT64     NOT NULL DEFAULT (0),
  StopCount         INT64       NOT NULL DEFAULT (0),
  LoadingStartedAt  TIMESTAMP,
  SealedAt          TIMESTAMP,
  DispatchedAt      TIMESTAMP,
  CompletedAt       TIMESTAMP,
  EncodedRoutePolyline STRING(MAX),
  RouteGeometrySource STRING(32),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_SupplierManifests_BySupplierId ON SupplierTruckManifests(SupplierId);
CREATE INDEX Idx_SupplierManifests_ByState ON SupplierTruckManifests(State);
CREATE INDEX Idx_SupplierManifests_ByDriver ON SupplierTruckManifests(DriverId, State);
CREATE INDEX Idx_SupplierManifests_ByWarehouse ON SupplierTruckManifests(WarehouseId, State);

CREATE TABLE ManifestOrders (
  ManifestId     STRING(36)  NOT NULL,
  OrderId        STRING(36)  NOT NULL,
  SequenceIndex  INT64       NOT NULL DEFAULT (0),
  LoadingOrder   INT64       NOT NULL DEFAULT (0),
  VolumeVU       FLOAT64     NOT NULL DEFAULT (0),
  State          STRING(30)  NOT NULL,
  RemovedReason  STRING(100),
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId, OrderId),
  INTERLEAVE IN PARENT SupplierTruckManifests ON DELETE CASCADE;

CREATE TABLE ManifestExceptions (
  ExceptionId   STRING(36)  NOT NULL,
  OrderId       STRING(36)  NOT NULL,
  ManifestId    STRING(36),
  SupplierId    STRING(36)  NOT NULL,
  Reason        STRING(30)  NOT NULL,
  Metadata      STRING(MAX),
  AttemptCount  INT64       NOT NULL DEFAULT (1),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResolvedAt    TIMESTAMP,
  EscalatedAt   TIMESTAMP,
) PRIMARY KEY (ExceptionId);

CREATE INDEX Idx_ManifestExceptions_ByOrder ON ManifestExceptions(OrderId, AttemptCount DESC);
CREATE INDEX Idx_ManifestExceptions_BySupplier ON ManifestExceptions(SupplierId, CreatedAt DESC);

-- Factory outbound truck manifests (inter-hub / factory loading).
CREATE TABLE FactoryTruckManifests (
  ManifestId        STRING(36)  NOT NULL,
  FactoryId         STRING(36)  NOT NULL,
  SupplierId        STRING(36)  NOT NULL,
  DriverId          STRING(36),
  VehicleId         STRING(36),
  State             STRING(20)  NOT NULL,
  TotalVolumeVU     FLOAT64     NOT NULL DEFAULT (0),
  MaxVolumeVU       FLOAT64     NOT NULL DEFAULT (0),
  StopCount         INT64       NOT NULL DEFAULT (0),
  TransferCount     INT64       NOT NULL DEFAULT (0),
  LoadingStartedAt  TIMESTAMP,
  SealedAt          TIMESTAMP,
  DispatchedAt      TIMESTAMP,
  CompletedAt       TIMESTAMP,
  CancelledAt       TIMESTAMP,
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_FactoryManifests_ByFactoryId ON FactoryTruckManifests(FactoryId);
CREATE INDEX Idx_FactoryManifests_BySupplierId ON FactoryTruckManifests(SupplierId);
CREATE INDEX Idx_FactoryManifests_ByState ON FactoryTruckManifests(State);

-- Factory inter-hub transfer orders (pegasusX durable projection of TransferRow).
CREATE TABLE FactoryInternalTransfers (
  TransferId      STRING(36)  NOT NULL,
  FactoryId       STRING(36)  NOT NULL,
  SupplierId      STRING(36)  NOT NULL,
  OrderId         STRING(36),
  ManifestId      STRING(36),
  State           STRING(20)  NOT NULL,
  TotalVolumeVU   FLOAT64     NOT NULL DEFAULT (0),
  DriverId        STRING(36),
  VehicleId       STRING(36),
  ReassignDepth   INT64       NOT NULL DEFAULT (0),
  ExceptionCount  INT64       NOT NULL DEFAULT (0),
  SupplyRequestId STRING(36),
  WarehouseId     STRING(36),
  TransferMode    STRING(10),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TransferId);

CREATE INDEX Idx_FactoryTransfers_ByFactoryId ON FactoryInternalTransfers(FactoryId, UpdatedAt DESC);
CREATE INDEX Idx_FactoryTransfers_BySupplierId ON FactoryInternalTransfers(SupplierId);
CREATE INDEX Idx_FactoryTransfers_ByManifestId ON FactoryInternalTransfers(ManifestId);

-- Predictive / threshold-based replenishment recommendations per warehouse SKU.
CREATE TABLE ReplenishmentInsights (
  InsightId         STRING(36)  NOT NULL,
  WarehouseId       STRING(36)  NOT NULL,
  ProductId         STRING(36)  NOT NULL,
  SupplierId        STRING(36)  NOT NULL,
  CurrentStock      INT64       NOT NULL DEFAULT (0),
  DailyBurnRate     FLOAT64     NOT NULL DEFAULT (0),
  TimeToEmptyDays   FLOAT64     NOT NULL DEFAULT (0),
  SuggestedQuantity INT64       NOT NULL DEFAULT (0),
  UrgencyLevel      STRING(20)  NOT NULL DEFAULT ('STABLE'),
  ReasonCode        STRING(30)  NOT NULL DEFAULT ('LOW_STOCK'),
  Status            STRING(20)  NOT NULL DEFAULT ('PENDING'),
  TargetFactoryId   STRING(36),
  DemandBreakdown   STRING(MAX),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (InsightId);

CREATE INDEX Idx_Insights_ByWarehouse ON ReplenishmentInsights(WarehouseId);
CREATE INDEX Idx_Insights_BySupplierId ON ReplenishmentInsights(SupplierId);
CREATE INDEX Idx_Insights_ByStatus ON ReplenishmentInsights(Status);

-- Client version policy (per role / platform / release channel).
CREATE TABLE ClientVersionPolicies (
  Role               STRING(20)  NOT NULL,
  Platform           STRING(20)  NOT NULL,
  Channel            STRING(30)  NOT NULL,
  MinimumVersion     STRING(32)  NOT NULL,
  RecommendedVersion STRING(32)  NOT NULL,
  UpdateURL          STRING(512),
  ForceUpdate        BOOL        NOT NULL DEFAULT (false),
  UpdatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Role, Platform, Channel);

-- FCM / APNs device tokens for push fallback when WebSocket is offline.
CREATE TABLE DeviceTokens (
  Token     STRING(512) NOT NULL,
  ActorId   STRING(36)  NOT NULL,
  ActorRole STRING(20)  NOT NULL,
  Platform  STRING(20)  NOT NULL,
  UpdatedAt TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Token);

CREATE INDEX Idx_DeviceTokens_ByActor ON DeviceTokens(ActorId, ActorRole);

-- ── Vehicles capacity migration (existing clusters) ─────────────────────────
-- ALTER TABLE Vehicles ADD COLUMN VehicleClass STRING(10) NOT NULL DEFAULT ('CLASS_B');
-- ALTER TABLE Vehicles ADD COLUMN MaxVolumeVU FLOAT64 NOT NULL DEFAULT (150.0);

CREATE TABLE WarehouseDispatchLocks (
  LockId        STRING(36)    NOT NULL,
  WarehouseId   STRING(36)    NOT NULL,
  EntityType    STRING(64)    NOT NULL,
  EntityId      STRING(36)    NOT NULL,
  Reason        STRING(255)   NOT NULL,
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LockId);

CREATE INDEX Idx_WarehouseDispatchLocks_ByWarehouse ON WarehouseDispatchLocks(WarehouseId, EntityType, EntityId);

-- ───────────────────────────────────────────────────────────────────────────────
-- Phase 2+ Enterprise Features (Regions, Billing, Optimizer, Delivery, InventoryV2)
-- ───────────────────────────────────────────────────────────────────────────────

CREATE TABLE Regions (
  RegionId         STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  CountryCode      STRING(2)     NOT NULL,
  IsActive         BOOL          NOT NULL DEFAULT (TRUE),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RegionId);

CREATE TABLE RegionalConfigs (
  RegionId         STRING(36)    NOT NULL,
  ConfigKey        STRING(128)   NOT NULL,
  ConfigValue      STRING(MAX),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RegionId, ConfigKey),
  INTERLEAVE IN PARENT Regions ON DELETE CASCADE;

CREATE TABLE BillingMeterEvents (
  EventId          STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  OrderId          STRING(36)    NOT NULL,
  MeterType        STRING(64)    NOT NULL,
  Amount           FLOAT64       NOT NULL,
  ProcessedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (EventId);

CREATE INDEX Idx_BillingMeterEvents_BySupplier ON BillingMeterEvents(SupplierId, ProcessedAt DESC);

CREATE TABLE BillingSupplierMeters (
  SupplierId       STRING(36)    NOT NULL,
  ShardId          INT64         NOT NULL,
  CurrentValue     FLOAT64       NOT NULL DEFAULT (0),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ShardId);

CREATE TABLE BillingGlobalMeters (
  MeterId          STRING(64)    NOT NULL,
  ShardId          INT64         NOT NULL,
  CurrentValue     FLOAT64       NOT NULL DEFAULT (0),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (MeterId, ShardId);

CREATE TABLE DeliverySessions (
  SessionId           STRING(36)    NOT NULL,
  OrderId             STRING(36)    NOT NULL,
  SupplierId          STRING(36)    NOT NULL,
  RetailerId          STRING(36)    NOT NULL,
  DriverId            STRING(36)    NOT NULL,
  Status              STRING(32)    NOT NULL,
  OriginalAmountMinor INT64         NOT NULL,
  AdjustedAmountMinor INT64         NOT NULL,
  Currency            STRING(3)     NOT NULL,
  PaymentClearedAt    TIMESTAMP,
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SessionId);

CREATE INDEX Idx_DeliverySessions_ByOrder ON DeliverySessions(OrderId);
CREATE INDEX Idx_DeliverySessions_BySupplierStatus ON DeliverySessions(SupplierId, Status);

CREATE TABLE DeliverySessionAdjustments (
  AdjustmentId        STRING(36)    NOT NULL,
  SessionId           STRING(36)    NOT NULL,
  AmountDeltaMinor    INT64         NOT NULL,
  Reason              STRING(255)   NOT NULL,
  AppliedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AdjustmentId);

CREATE TABLE SupplierInventoryV2 (
  SupplierId       STRING(36)    NOT NULL,
  WarehouseId      STRING(36)    NOT NULL,
  ProductId        STRING(36)    NOT NULL,
  H3Cell           STRING(15),
  QuantityOnHand   INT64         NOT NULL DEFAULT (0),
  QuantityReserved INT64         NOT NULL DEFAULT (0),
  OutOfStockPolicy STRING(24),
  ReorderThreshold INT64         NOT NULL DEFAULT (0),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, WarehouseId, ProductId);

CREATE INDEX Idx_SupplierInventoryV2_ByH3Cell ON SupplierInventoryV2(H3Cell);

-- Supplier bulk-import staging substrate (warehouse analytics anomaly queue + future session wizard).
CREATE TABLE SupplierImportSessions (
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
) PRIMARY KEY (supplier_id, session_id);

CREATE INDEX Idx_SupplierImportSessions_BySupplierUpdated
    ON SupplierImportSessions(supplier_id, updated_at DESC);
CREATE INDEX Idx_SupplierImportSessions_BySupplierStatus
    ON SupplierImportSessions(supplier_id, status, updated_at DESC);

CREATE TABLE SupplierImportStagedRows (
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
  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE;

CREATE INDEX Idx_SupplierImportStagedRows_BySession
    ON SupplierImportStagedRows(supplier_id, session_id, row_index);
CREATE INDEX Idx_SupplierImportStagedRows_BySupplierCreated
    ON SupplierImportStagedRows(supplier_id, created_at DESC);

CREATE TABLE SupplierImportMapping (
    supplier_id    STRING(36)  NOT NULL,
    session_id     STRING(36)  NOT NULL,
    mapping_json   JSON,
    created_at     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at     TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, session_id),
  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE;

CREATE TABLE MasterInvoices (
  InvoiceId           STRING(36)    NOT NULL,
  OrderId             STRING(36)    NOT NULL,
  SupplierId          STRING(36)    NOT NULL,
  RetailerId          STRING(36)    NOT NULL,
  Status              STRING(32)    NOT NULL,
  SettlementTarget    STRING(32)    NOT NULL,
  TotalMinor          INT64         NOT NULL,
  Currency            STRING(3)     NOT NULL,
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (InvoiceId);

CREATE INDEX Idx_MasterInvoices_ByOrder ON MasterInvoices(OrderId);

CREATE TABLE OptimizationJobs (
  JobId            STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Status           STRING(32)    NOT NULL,
  RequestType      STRING(32)    NOT NULL,
  PayloadJson      BYTES(MAX)    NOT NULL,
  IdempotencyKey   STRING(128),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (JobId);

CREATE INDEX Idx_OptimizationJobs_BySupplierStatus ON OptimizationJobs(SupplierId, Status, CreatedAt DESC);
CREATE NULL_FILTERED INDEX UQ_OptimizationJobs_Idempotency ON OptimizationJobs(SupplierId, IdempotencyKey);

-- Warehouse stock policy: REJECT blocks retailer checkout when short; ACCEPT_BACKORDER allows with delayed fulfillment.
-- Per-SKU OutOfStockPolicy on SupplierInventoryV2 overrides warehouse DefaultOutOfStockPolicy when set (INHERIT = use warehouse default).
