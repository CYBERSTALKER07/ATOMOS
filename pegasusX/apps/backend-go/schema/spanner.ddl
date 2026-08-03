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
  MaxRedemptions      INT64,
  CurrentRedemptions  INT64         NOT NULL DEFAULT (0),
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
  Timezone              STRING(64),

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
  Timezone              STRING(64),

  ProposedDeliveryDate TIMESTAMP,
  DeliveryProposalAt TIMESTAMP,
  DeliveryProposalBy STRING(128),
  DeliveryProposalReason STRING(512),
  BuyerAcceptanceStatus STRING(MAX),
  BuyerAcceptanceDeadline TIMESTAMP,
  Version          INT64         NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId);

CREATE INDEX Idx_Orders_ByRetailerCreated ON Orders(RetailerId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByRetailerConfirmation ON Orders(RetailerId, ConfirmationStatus, RequestedDeliveryDate DESC, UpdatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierCreated ON Orders(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierUpdated ON Orders(SupplierId, UpdatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierStatusUpdated ON Orders(SupplierId, Status, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByWarehouseCreated ON Orders(WarehouseId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByWarehouseRequestedDelivery ON Orders(WarehouseId, RequestedDeliveryDate DESC, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByConfirmationAutoConfirm ON Orders(ConfirmationStatus, AutoConfirmAt, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByDerivedSource ON Orders(DerivedFromOrderId, OrderSource, UpdatedAt DESC);
CREATE INDEX Idx_Orders_ByDriverCreated ON Orders(DriverId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByRouteCreated ON Orders(RouteId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByManifestCreated ON Orders(ManifestId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByH3Cell ON Orders(H3Cell, Status, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByStatusWarehouse ON Orders(Status, WarehouseId, CreatedAt DESC);
CREATE INDEX Idx_Orders_BuyerAcceptance ON Orders(FiscalStatus, BuyerAcceptanceStatus, BuyerAcceptanceDeadline);

CREATE TABLE SupplierReturns (
  ReturnId          STRING(36)  NOT NULL,
  OrderId           STRING(36)  NOT NULL,
  SkuId             STRING(50)  NOT NULL,
  RejectedQty       INT64       NOT NULL,
  Reason            STRING(50)  NOT NULL,
  DriverNotes       STRING(MAX),
  Status            STRING(32)  NOT NULL DEFAULT ('PENDING'),
  ResolvedAt        TIMESTAMP OPTIONS (allow_commit_timestamp=true),
  ResolutionNotes   STRING(MAX),
  ManifestId        STRING(36),
  DriverId          STRING(36),
  WarehouseId       STRING(36),
  ExpectedQty       INT64,
  ReceivedQty       INT64       NOT NULL DEFAULT (0),
  PhysicalStatus    STRING(32)  NOT NULL DEFAULT ('PENDING'),
  ReceivedAt        TIMESTAMP OPTIONS (allow_commit_timestamp=true),
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
  CompletedAt   TIMESTAMP OPTIONS (allow_commit_timestamp=true),
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

-- Structured quality/condition reports linked to orders and optionally line items.
CREATE TABLE OrderConditionReports (
  ReportId          STRING(36)    NOT NULL,
  OrderId           STRING(36)    NOT NULL,
  SupplierId        STRING(36)    NOT NULL,
  RetailerId        STRING(36)    NOT NULL,
  LineItemIndex     INT64,
  SKU               STRING(50),
  ConditionType     STRING(32)    NOT NULL,
  Severity          STRING(16)    NOT NULL DEFAULT ('MEDIUM'),
  Description       STRING(MAX),
  PhotoURLsJson     BYTES(MAX),
  ProofIdsJson      BYTES(MAX),
  ReportedBy        STRING(36)    NOT NULL,
  ReportedByRole    STRING(20)    NOT NULL,
  ResolutionStatus  STRING(20)    NOT NULL DEFAULT ('OPEN'),
  ResolvedBy        STRING(36),
  ResolvedAt        TIMESTAMP OPTIONS (allow_commit_timestamp=true),
  ResolutionNotes   STRING(MAX),
  CreatedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ReportId);

CREATE INDEX Idx_OrderConditionReports_ByOrder ON OrderConditionReports(OrderId, CreatedAt DESC);
CREATE INDEX Idx_OrderConditionReports_BySupplierStatus ON OrderConditionReports(SupplierId, ResolutionStatus, CreatedAt DESC);
CREATE INDEX Idx_OrderConditionReports_ByRetailer ON OrderConditionReports(RetailerId, ResolutionStatus, CreatedAt DESC);

-- Retailer credit profile and risk engine state per supplier.
CREATE TABLE RetailerCreditProfiles (
  RetailerId          STRING(36)    NOT NULL,
  SupplierId          STRING(36)    NOT NULL,
  CreditLimitMinor    INT64         NOT NULL DEFAULT (0),
  CurrentBalanceMinor INT64         NOT NULL DEFAULT (0),
  AvailableCreditMinor INT64        NOT NULL DEFAULT (0),
  RiskScore           INT64         NOT NULL DEFAULT (0),
  DelinquencyCount    INT64         NOT NULL DEFAULT (0),
  Status              STRING(20)    NOT NULL DEFAULT ('ACTIVE'),
  LastEvaluatedAt     TIMESTAMP,
  Version             INT64         NOT NULL DEFAULT (1),
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, SupplierId);

CREATE INDEX Idx_RetailerCreditProfiles_BySupplier ON RetailerCreditProfiles(SupplierId, Status, UpdatedAt DESC);
CREATE INDEX Idx_RetailerCreditProfiles_ByRisk ON RetailerCreditProfiles(SupplierId, RiskScore DESC, UpdatedAt DESC);

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
  H3Cell             STRING(15),
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplier ON Warehouses(SupplierId);
CREATE INDEX Idx_Warehouses_ByH3Cell ON Warehouses(SupplierId, H3Cell);
CREATE INDEX Idx_Warehouses_ByPrimaryFactory ON Warehouses(SupplierId, PrimaryFactoryId);
CREATE INDEX Idx_Warehouses_ByAutoDispatch ON Warehouses(AutoDispatchEnabled, IsActive, WarehouseId) STORING (SupplierId);

CREATE TABLE Factories (
  FactoryId        STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  H3Cell           STRING(15),
  Address          STRING(MAX),
  PlaceId          STRING(128),
  IsActive         BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (FactoryId);

CREATE INDEX Idx_Factories_BySupplier ON Factories(SupplierId);
CREATE INDEX Idx_Factories_ByH3Cell ON Factories(SupplierId, H3Cell);

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

CREATE TABLE WebhookInbox (
  WebhookId     STRING(36)    NOT NULL,
  Gateway       STRING(32)    NOT NULL,
  RecordJson    JSON          NOT NULL,
  Source        STRING(128)   NOT NULL,
  Status        STRING(24)    NOT NULL,
  Attempts      INT64         NOT NULL,
  NextRetryAt   TIMESTAMP,
  LastError     STRING(MAX),
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WebhookId);

CREATE INDEX Idx_WebhookInbox_Pending ON WebhookInbox(Status, NextRetryAt);

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
  ProductId          STRING(36)    NOT NULL,
  SupplierId         STRING(36)    NOT NULL,
  CategoryId         STRING(36)    NOT NULL,
  Name               STRING(255)   NOT NULL,
  Description        STRING(MAX),
  ImageURL           STRING(2048),
  PriceMinor         INT64         NOT NULL,
  Currency           STRING(3)     NOT NULL,
  StockQuantity      INT64         NOT NULL DEFAULT (0),
  Unit               STRING(20)    NOT NULL DEFAULT ('UNIT'),
  SaleUnit           STRING(16)    NOT NULL DEFAULT ('UNIT'),
  UnitsPerPack       INT64,
  UnitVolumeVU       FLOAT64       NOT NULL DEFAULT (1.0),
  Barcode            STRING(32),
  HandlingClass      STRING(20)    NOT NULL DEFAULT ('GENERAL'),
  RequiresColdChain  BOOL          NOT NULL DEFAULT (FALSE),
  IsHazardous        BOOL          NOT NULL DEFAULT (FALSE),
  IsPerishable       BOOL          NOT NULL DEFAULT (FALSE),
  StorageTempMinC    FLOAT64,
  StorageTempMaxC    FLOAT64,
  IsActive           BOOL          NOT NULL DEFAULT (TRUE),
  Version            INT64         NOT NULL DEFAULT (1),
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProductId);

CREATE INDEX Idx_Products_BySupplierCategory ON Products(SupplierId, CategoryId, IsActive);
CREATE INDEX Idx_Products_BySupplierActive ON Products(SupplierId, IsActive, UpdatedAt DESC);
CREATE NULL_FILTERED INDEX Idx_Products_BySupplierBarcode ON Products(SupplierId, Barcode);
CREATE INDEX Idx_Products_ByHandlingClass ON Products(SupplierId, HandlingClass, IsActive);

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
  MetadataJson    BYTES(MAX),
  IsRead          BOOL          NOT NULL DEFAULT (FALSE),
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (NotificationId);

CREATE INDEX Idx_Notifications_ByRecipientCreated ON Notifications(RecipientId, CreatedAt DESC);
CREATE INDEX Idx_Notifications_ByRecipientUnread ON Notifications(RecipientId, IsRead, CreatedAt DESC);

CREATE TABLE DeviceTokens (
  Token     STRING(255) NOT NULL,
  ActorId   STRING(36)  NOT NULL,
  ActorRole STRING(20)  NOT NULL,
  Platform  STRING(20)  NOT NULL,
  UpdatedAt TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Token);

CREATE INDEX Idx_DeviceTokens_ByActorRole ON DeviceTokens(ActorId, ActorRole);

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
  ExpiresAt     TIMESTAMP,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProposalId);

CREATE INDEX Idx_NegotiationProposals_ByOrderId ON NegotiationProposals(OrderId);
CREATE INDEX Idx_NegotiationProposals_Pending ON NegotiationProposals(Status);
CREATE INDEX Idx_NegotiationProposals_Expiry ON NegotiationProposals(Status, ExpiresAt);

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
  LastReplannedAt   TIMESTAMP,
  ReplanCount       INT64       NOT NULL DEFAULT (0),
  ReplanReason      STRING(64),
  EncodedRoutePolyline STRING(MAX),
  RouteGeometrySource STRING(32),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId);

CREATE INDEX Idx_SupplierManifests_BySupplierId ON SupplierTruckManifests(SupplierId);
CREATE INDEX Idx_SupplierManifests_ByState ON SupplierTruckManifests(State);
CREATE INDEX Idx_SupplierManifests_ByDriver ON SupplierTruckManifests(DriverId, State);
CREATE INDEX Idx_SupplierManifests_ByWarehouse ON SupplierTruckManifests(WarehouseId, State);

CREATE TABLE ManifestReplanLog (
  ManifestId      STRING(36) NOT NULL,
  ReplanId        STRING(36) NOT NULL,
  Reason          STRING(64) NOT NULL,
  OldSequenceJson BYTES(MAX),
  NewSequenceJson BYTES(MAX),
  TriggeredBy     STRING(64) NOT NULL,
  CreatedAt       TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId, ReplanId),
  INTERLEAVE IN PARENT SupplierTruckManifests ON DELETE CASCADE;


CREATE TABLE DispatchRuns (
  RunId           STRING(36)  NOT NULL,
  WarehouseId     STRING(36)  NOT NULL,
  SupplierId      STRING(36)  NOT NULL,
  ActorId         STRING(36),
  Mode            STRING(20)  NOT NULL,
  Status          STRING(32)  NOT NULL,
  ManifestCount   INT64       NOT NULL DEFAULT (0),
  OrdersAssigned  INT64       NOT NULL DEFAULT (0),
  WarningsJson    BYTES(MAX),
  ManifestsJson   BYTES(MAX),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RunId);

CREATE INDEX Idx_DispatchRuns_ByWarehouseCreated ON DispatchRuns(WarehouseId, CreatedAt DESC);

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
  SourceInsightId STRING(36),
  WarehouseId     STRING(36),
  TransferMode    STRING(10),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TransferId);

CREATE INDEX Idx_FactoryTransfers_ByFactoryId ON FactoryInternalTransfers(FactoryId, UpdatedAt DESC);
CREATE INDEX Idx_FactoryTransfers_BySupplierId ON FactoryInternalTransfers(SupplierId);
CREATE INDEX Idx_FactoryTransfers_ByManifestId ON FactoryInternalTransfers(ManifestId);
CREATE INDEX Idx_FactoryTransfers_BySourceInsight ON FactoryInternalTransfers(SourceInsightId);

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

CREATE TABLE OrderStockReservationMarkers (
  OrderId     STRING(36) NOT NULL,
  ReservedAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId);

-- Immutable order status audit trail (delays, promotions, cancellations).
CREATE TABLE OrderStatusTransitions (
  OrderId          STRING(36)  NOT NULL,
  TransitionId     STRING(36)  NOT NULL,
  PreviousStatus   STRING(32),
  NewStatus        STRING(32)  NOT NULL,
  Reason           STRING(512),
  ActorRole        STRING(32),
  ActorId          STRING(128),
  EventKind        STRING(32),
  MetadataJson     BYTES(MAX),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, CreatedAt DESC, TransitionId);

CREATE INDEX Idx_OrderStatusTransitions_ByOrder ON OrderStatusTransitions(OrderId, CreatedAt DESC);

CREATE TABLE WarehouseBroadcastTemplates (
  WarehouseId   STRING(36)  NOT NULL,
  TemplateId    STRING(36)  NOT NULL,
  SupplierId    STRING(36)  NOT NULL,
  Title         STRING(255) NOT NULL,
  Body          STRING(MAX) NOT NULL,
  DefaultRole   STRING(32)  NOT NULL,
  Category      STRING(64),
  CreatedBy     STRING(36),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId, TemplateId);

CREATE INDEX Idx_WarehouseBroadcastTemplates_ByWarehouseUpdated
  ON WarehouseBroadcastTemplates(WarehouseId, UpdatedAt DESC);
ALTER TABLE Orders ADD COLUMN CancelLockExpiresAt TIMESTAMP;

CREATE TABLE Payers (
  PayerId          STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Email            STRING(255)   NOT NULL,
  Phone            STRING(32),
  BillingAddress   STRING(MAX),
  TaxId            STRING(64),
  IsActive         BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PayerId);

-- PX90-A1: Touchless replenishment policy knobs per supplier.
CREATE TABLE ReplenishmentPolicies (
  SupplierId                 STRING(36)  NOT NULL,
  AutoApproveStable          BOOL        NOT NULL DEFAULT (true),
  AutoApprovePredictivePush  BOOL        NOT NULL DEFAULT (true),
  MaxDailyTransferUnits      INT64       NOT NULL DEFAULT (500),
  MinConfidenceScore         FLOAT64     NOT NULL DEFAULT (0.85),
  UpdatedAt                  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

-- PX90-B1: Actionable control-tower zone overrides (GeoJSON polygon + TTL).
CREATE TABLE ControlTowerZoneOverrides (
  OverrideId       STRING(36)  NOT NULL,
  SupplierId       STRING(36)  NOT NULL,
  WarehouseId      STRING(36),
  Action           STRING(32)  NOT NULL,
  PolygonGeoJSON   STRING(MAX) NOT NULL,
  TtlExpiresAt     TIMESTAMP   NOT NULL,
  CreatedBy        STRING(36),
  IsActive         BOOL        NOT NULL DEFAULT (true),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OverrideId);

CREATE INDEX Idx_ControlTowerOverrides_BySupplier
  ON ControlTowerZoneOverrides(SupplierId, IsActive, TtlExpiresAt DESC);

-- PX90-C1: One-number demand baseline per supplier/day/warehouse/SKU.
CREATE TABLE DemandForecastBaseline (
  SupplierId    STRING(36)  NOT NULL,
  ForecastDate  DATE        NOT NULL,
  WarehouseId   STRING(36)  NOT NULL,
  ProductId     STRING(36)  NOT NULL,
  BaselineQty   INT64       NOT NULL,
  Confidence    FLOAT64     NOT NULL DEFAULT (0),
  Source        STRING(32)  NOT NULL,
  LowUnits      INT64,
  HighUnits     INT64,
  ConfidencePct INT64,
  BaselineSource STRING(32),
  BlockedReason STRING(64),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ForecastDate, WarehouseId, ProductId);

CREATE INDEX Idx_DemandBaseline_ByWarehouseDate
  ON DemandForecastBaseline(WarehouseId, ForecastDate DESC);

CREATE TABLE SeasonalTemplateOverrides (
  SupplierId   STRING(36) NOT NULL,
  OverrideId   STRING(36) NOT NULL,
  TemplateId   STRING(64) NOT NULL,
  Name         STRING(128),
  StartDate    DATE        NOT NULL,
  EndDate      DATE        NOT NULL,
  IsActive     BOOL        NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, OverrideId);

CREATE INDEX Idx_SeasonalOverrides_Active
  ON SeasonalTemplateOverrides(SupplierId, IsActive, StartDate DESC);

CREATE TABLE PlanningSignalProjections (
  SupplierId   STRING(36) NOT NULL,
  SignalId     STRING(36) NOT NULL,
  Source       STRING(64) NOT NULL,
  PayloadJson  STRING(MAX) NOT NULL,
  IngestedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, SignalId);

CREATE TABLE PlanningPromoSimulations (
  SupplierId     STRING(36) NOT NULL,
  SimulationId   STRING(36) NOT NULL,
  PromotionId    STRING(36) NOT NULL,
  ResultJson     STRING(MAX) NOT NULL,
  CreatedAt      TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, SimulationId);

CREATE INDEX Idx_PlanningPromoSim_ByPromotion
  ON PlanningPromoSimulations(SupplierId, PromotionId, CreatedAt DESC);

-- ── ADR-009 Fiscal hard-gate ────────────────────────────────────────────────
CREATE TABLE OrderFiscalReceipts (
  OrderId              STRING(36)  NOT NULL,
  AttemptId            STRING(36)  NOT NULL,
  SupplierId           STRING(36)  NOT NULL,
  RetailerId           STRING(36),
  Provider             STRING(32)  NOT NULL,
  Status               STRING(32)  NOT NULL,
  FiscalReceiptId      STRING(128),
  FiscalQR             STRING(MAX),
  AmountMinor          INT64       NOT NULL,
  Currency             STRING(8)   NOT NULL,
  PaymentMethod        STRING(16),
  ProviderPayloadJSON  BYTES(MAX),
  ErrorCode            STRING(64),
  ErrorMessage         STRING(1024),
  ReasonCode           STRING(64),
  ActorId              STRING(128),
  TraceId              STRING(64),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, AttemptId);

CREATE INDEX Idx_OrderFiscalReceipts_ByStatusCreated
  ON OrderFiscalReceipts(Status, CreatedAt DESC);

CREATE INDEX Idx_OrderFiscalReceipts_ByReceiptId
  ON OrderFiscalReceipts(FiscalReceiptId);

CREATE INDEX Idx_OrderFiscalReceipts_BySupplier
  ON OrderFiscalReceipts(SupplierId, CreatedAt DESC);

ALTER TABLE Orders ADD COLUMN FiscalStatus STRING(32);
ALTER TABLE Orders ADD COLUMN LatestFiscalReceiptId STRING(128);
ALTER TABLE Orders ADD COLUMN FiscalizedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN LatestFiscalAttemptId STRING(36);

-- Enhanced shop-closed + proximity settlement + partial offload (2026-07-29).
-- Wire status ARRIVED_SHOP_CLOSED ≡ design SHOP_CLOSED_PENDING.
-- Partial line qty lives in LineItemsJson (DeliveredQty/RemainingQty/OffloadStatus).
ALTER TABLE Orders ADD COLUMN ShopClosedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedReason STRING(64);
ALTER TABLE Orders ADD COLUMN ShopClosedGraceEndsAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedResolution STRING(32);
ALTER TABLE Orders ADD COLUMN PartialDelivery BOOL;
ALTER TABLE Orders ADD COLUMN ProximityUnlockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ProximityMethod STRING(16);

CREATE TABLE OrderShopClosedLog (
  OrderId   STRING(36) NOT NULL,
  EventId   STRING(36) NOT NULL,
  Actor     STRING(64) NOT NULL,
  Action    STRING(32) NOT NULL,
  Payload   BYTES(MAX),
  CreatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderId, EventId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

CREATE INDEX Idx_OrderShopClosedLog_ByOrderCreated
  ON OrderShopClosedLog(OrderId, CreatedAt DESC);

-- ───────────────────────────────────────────────────────────────────────────────
-- Tax Regime Versioning — versioned tax configurations + per-line fiscal snapshots
-- ───────────────────────────────────────────────────────────────────────────────

CREATE TABLE TaxRegimeVersions (
  Id             STRING(36)  NOT NULL,
  CountryCode    STRING(2)   NOT NULL,
  EffectiveFrom  TIMESTAMP   NOT NULL,
  EffectiveTo    TIMESTAMP,
  Currency       STRING(3)   NOT NULL,
  VatRatesBps    ARRAY<INT64>,
  SimplifiedRules JSON,
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CreatedBy      STRING(64)  NOT NULL,
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Id);

CREATE INDEX Idx_TaxRegimeVersions_Effective
  ON TaxRegimeVersions(CountryCode, EffectiveFrom DESC);

CREATE TABLE OrderLineFiscalSnapshots (
  OrderId           STRING(36) NOT NULL,
  LineSku           STRING(64) NOT NULL,
  RegimeVersionId   STRING(36) NOT NULL,
  TaxableMinor      INT64      NOT NULL,
  VatMinor          INT64      NOT NULL,
  TotalMinor        INT64      NOT NULL,
  AppliedVatRateBps INT64      NOT NULL,
) PRIMARY KEY (OrderId, LineSku),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

CREATE TABLE ExceptionTickets (
    TicketId STRING(36) NOT NULL,
    Type STRING(64) NOT NULL,
    OrderId STRING(36) NOT NULL,
    EhfId STRING(64),
    Severity STRING(16) NOT NULL,
    Status STRING(32) NOT NULL,
    Title STRING(256) NOT NULL,
    Description STRING(MAX),
    AssignedRole STRING(64),
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    CreatedBy STRING(128),
    Payload JSON
) PRIMARY KEY (TicketId);

CREATE TABLE DemandSignals (
  SignalId        STRING(36) NOT NULL,
  Type            STRING(32) NOT NULL,
  Scope           STRING(64) NOT NULL,
  Sku             STRING(64),
  StartAt         TIMESTAMP NOT NULL,
  EndAt           TIMESTAMP NOT NULL,
  Multiplier      FLOAT64 NOT NULL,
  Meta            JSON,
  CreatedAt       TIMESTAMP NOT NULL,
  CreatedBy       STRING(64) NOT NULL,
) PRIMARY KEY (SignalId);

CREATE INDEX DemandSignals_ByScopeTime ON DemandSignals (Scope, StartAt, EndAt);

CREATE TABLE DemandAdjustments (
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  Date            DATE NOT NULL,
  BaseVelocity    FLOAT64 NOT NULL,
  Adjustment      FLOAT64 NOT NULL,
  AdjustedDemand  FLOAT64 NOT NULL,
  FactorsJson     JSON,
  ComputedAt      TIMESTAMP NOT NULL,
) PRIMARY KEY (RetailerId, Sku, Date);
-- Money movement ledger (append-only)
CREATE TABLE OrderPaymentLegs (
  OrderId           STRING(36) NOT NULL,
  LegId             STRING(36) NOT NULL,
  Method            STRING(16) NOT NULL,
  AmountMinor       INT64 NOT NULL,
  Status            STRING(16) NOT NULL,
  IdempotencyKey    STRING(64) NOT NULL,
  ProviderRef       STRING(128),
  CreatedAt         TIMESTAMP NOT NULL,
  CapturedAt        TIMESTAMP,
) PRIMARY KEY (OrderId, LegId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

-- Explicit shortfalls / discrepancies
CREATE TABLE OrderSettlementExceptions (
  OrderId           STRING(36) NOT NULL,
  ExceptionId       STRING(36) NOT NULL,
  Type              STRING(32) NOT NULL,
  AmountMinor       INT64 NOT NULL,
  Status            STRING(16) NOT NULL,
  Reason            STRING(MAX),
  CreatedBy         STRING(64) NOT NULL,
  CreatedAt         TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderId, ExceptionId);

CREATE TABLE CreditNotes (
  CreditNoteId        STRING(36) NOT NULL,
  OrderId             STRING(36) NOT NULL,
  Type                STRING(32) NOT NULL,
  Status              STRING(16) NOT NULL,
  ReasonCode          STRING(64) NOT NULL,
  ReasonText          STRING(MAX),
  TotalNetMinor       INT64 NOT NULL,
  TotalVatMinor       INT64 NOT NULL,
  TotalGrossMinor     INT64 NOT NULL,
  RegimeId            STRING(36),
  OriginalEhfId       STRING(128),
  CorrectiveEhfId     STRING(128),
  CreatedBy           STRING(64) NOT NULL,
  CreatedAt           TIMESTAMP NOT NULL,
  IssuedAt            TIMESTAMP,
  CompletedAt         TIMESTAMP,
) PRIMARY KEY (CreditNoteId);

CREATE INDEX CreditNotes_ByOrder ON CreditNotes(OrderId);

CREATE TABLE CreditNoteLines (
  CreditNoteId        STRING(36) NOT NULL,
  LineId              STRING(36) NOT NULL,
  OrderLineId         STRING(36) NOT NULL,
  Sku                 STRING(64) NOT NULL,
  Qty                 INT64 NOT NULL,
  UnitNetMinor        INT64 NOT NULL,
  VatRateBps          INT64 NOT NULL,
  LineNetMinor        INT64 NOT NULL,
  LineVatMinor        INT64 NOT NULL,
  LineGrossMinor      INT64 NOT NULL,
) PRIMARY KEY (CreditNoteId, LineId),
  INTERLEAVE IN PARENT CreditNotes ON DELETE CASCADE;

CREATE TABLE ReverseLogisticsTasks (
  TaskId              STRING(36) NOT NULL,
  CreditNoteId        STRING(36) NOT NULL,
  OrderId             STRING(36) NOT NULL,
  Status              STRING(16) NOT NULL,
  WarehouseId         STRING(36),
  DriverId            STRING(36),
  ExpectedQtyJson     JSON,
  ReceivedQtyJson     JSON,
  CreatedAt           TIMESTAMP NOT NULL,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (TaskId);

CREATE INDEX ReverseLogisticsTasks_ByStatus ON ReverseLogisticsTasks(Status);

CREATE TABLE CashReconciliations (
  ReconciliationId    STRING(36) NOT NULL,
  DriverId            STRING(36) NOT NULL,
  RouteId             STRING(36),
  ShiftDate           DATE NOT NULL,
  ExpectedCashMinor   INT64 NOT NULL,
  DeclaredCashMinor   INT64 NOT NULL,
  DifferenceMinor     INT64 NOT NULL,
  Status              STRING(16) NOT NULL,
  DriverNote          STRING(MAX),
  FinanceNote         STRING(MAX),
  CreatedAt           TIMESTAMP NOT NULL,
  ResolvedAt          TIMESTAMP,
  ResolvedBy          STRING(64),
) PRIMARY KEY (ReconciliationId);

CREATE INDEX CashReconciliations_ByDriverDate ON CashReconciliations(DriverId, ShiftDate);
CREATE INDEX CashReconciliations_ByStatus ON CashReconciliations(Status);

CREATE TABLE ReorderSuggestions (
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  SuggestedQty    INT64 NOT NULL,
  AdjustedDemand  FLOAT64 NOT NULL,
  CurrentStock    INT64 NOT NULL,
  InFlightQty     INT64 NOT NULL,
  SafetyStock     FLOAT64 NOT NULL,
  SuggestedByDate DATE NOT NULL,
  ComputedAt      TIMESTAMP NOT NULL,
  Status          STRING(16) NOT NULL,
) PRIMARY KEY (RetailerId, Sku);

CREATE TABLE RetailerCreditScores (
  RetailerId STRING(MAX) NOT NULL,
  Score INT64 NOT NULL,
  RiskTier STRING(64) NOT NULL,
  SuggestedLimitMinor INT64 NOT NULL,
  FactorsJson JSON,
  WindowStart TIMESTAMP NOT NULL,
  WindowEnd TIMESTAMP NOT NULL,
  ComputedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(RetailerId);

CREATE TABLE RoutePerformanceAnalytics (
  RouteId STRING(36) NOT NULL,
  DriverId STRING(36),
  PlannedStops INT64,
  ActualStops INT64,
  PlannedDurationSec INT64,
  ActualDurationSec INT64,
  ReplanCount INT64,
  ComputedAt TIMESTAMP,
) PRIMARY KEY (RouteId);

CREATE TABLE NotificationPreferences (
  PrincipalId STRING(36) NOT NULL,
  PrincipalType STRING(16) NOT NULL,
  EventType STRING(64) NOT NULL,
  Channel STRING(16) NOT NULL,
  Enabled BOOL NOT NULL,
  QuietFrom STRING(8),
  QuietTo STRING(8),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (PrincipalId, EventType, Channel);

CREATE TABLE PriceLists (
  PriceListId     STRING(36) NOT NULL,
  SupplierId      STRING(36) NOT NULL,
  Name            STRING(128) NOT NULL,
  EffectiveFrom   TIMESTAMP NOT NULL,
  EffectiveTo     TIMESTAMP,
) PRIMARY KEY (PriceListId);

CREATE TABLE PriceListItems (
  PriceListId     STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  UnitPriceMinor  INT64 NOT NULL,
  MinQty          INT64,
) PRIMARY KEY (PriceListId, Sku),
  INTERLEAVE IN PARENT PriceLists ON DELETE CASCADE;

CREATE TABLE EventDensitySignals (
  ZoneH3 STRING(15) NOT NULL,
  Date DATE NOT NULL,
  DensityScore FLOAT64,
  EventsJson JSON,
  ComputedAt TIMESTAMP,
) PRIMARY KEY (ZoneH3, Date);
CREATE TABLE RetailerSegments (
  RetailerId      STRING(36) NOT NULL,
  Segment         STRING(16) NOT NULL,
  Reason          STRING(256),
  EffectiveFrom   TIMESTAMP NOT NULL,
  EffectiveTo     TIMESTAMP,
  UpdatedBy       STRING(64) NOT NULL,
  UpdatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (RetailerId);

CREATE TABLE SkuClasses (
  SupplierId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  VelocityClass   STRING(8) NOT NULL,
  StrategicFlag   BOOL NOT NULL DEFAULT (FALSE),
  UpdatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (SupplierId, Sku);

CREATE TABLE ServicePolicies (
  PolicyId              STRING(36) NOT NULL,
  SupplierId            STRING(36) NOT NULL,
  RetailerSegment       STRING(16) NOT NULL,
  SkuClass              STRING(8) NOT NULL,
  PriorityWeight        INT64 NOT NULL,
  TargetServiceLevelBps INT64 NOT NULL,
  MaxFairShareBps       INT64 NOT NULL,
  MinFairShareBps       INT64 NOT NULL,
  CreditRiskBoost       INT64 NOT NULL,
  Enabled               BOOL NOT NULL DEFAULT (TRUE),
  UpdatedAt             TIMESTAMP NOT NULL,
) PRIMARY KEY (PolicyId);

CREATE INDEX ServicePolicies_BySupplier ON ServicePolicies(SupplierId);

CREATE TABLE AllocationDecisions (
  DecisionId      STRING(36) NOT NULL,
  OrderId         STRING(36) NOT NULL,
  OrderLineId     STRING(36) NOT NULL,
  SupplierId      STRING(36) NOT NULL,
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  WarehouseId     STRING(36) NOT NULL,
  Qty             INT64 NOT NULL,
  AllocationMode  STRING(16) NOT NULL,
  PriorityScore   INT64 NOT NULL,
  FairShareBps    INT64,
  PolicyId        STRING(36),
  RetailerSegment STRING(16),
  SkuClass        STRING(8),
  RiskTier        STRING(16),
  CreatedAt       TIMESTAMP NOT NULL,
  ConstraintReason STRING(64),
  RequestedQty    INT64,
  AllocatedQty    INT64,
) PRIMARY KEY (DecisionId);

CREATE INDEX AllocationDecisions_ByOrder ON AllocationDecisions(OrderId, CreatedAt DESC);

CREATE INDEX AllocationDecisions_ByRetailer ON AllocationDecisions(RetailerId, CreatedAt DESC);

CREATE TABLE OrderLineAllocations (
  OrderId         STRING(36) NOT NULL,
  OrderLineId     STRING(36) NOT NULL,
  WarehouseId     STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  Qty             INT64 NOT NULL,
  CreatedAt       TIMESTAMP NOT NULL,
  AllocationMode  STRING(16),
  PriorityScore   INT64,
  FairShareBps    INT64,
  PolicyId        STRING(36),
) PRIMARY KEY (OrderId, OrderLineId, WarehouseId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

-- O9-2: Control Tower playbooks — declarative exception → action orchestration.
CREATE TABLE ControlTowerPlaybooks (
  PlaybookId        STRING(36) NOT NULL,
  SupplierId        STRING(36),
  Name              STRING(128) NOT NULL,
  Description       STRING(MAX),
  IsActive          BOOL NOT NULL,
  Priority          INT64 NOT NULL,
  MatchRulesJson    JSON NOT NULL,
  ActionsJson       JSON NOT NULL,
  AutoExecute       BOOL NOT NULL,
  CreatedAt         TIMESTAMP NOT NULL,
  UpdatedAt         TIMESTAMP NOT NULL,
  CreatedBy         STRING(64) NOT NULL,
) PRIMARY KEY (PlaybookId);

CREATE INDEX ControlTowerPlaybooks_BySupplierActive
ON ControlTowerPlaybooks(SupplierId, IsActive, Priority DESC);

CREATE TABLE ControlTowerPlaybookRuns (
  RunId             STRING(36) NOT NULL,
  PlaybookId        STRING(36) NOT NULL,
  ExceptionId       STRING(36) NOT NULL,
  SupplierId        STRING(36) NOT NULL,
  Mode              STRING(16) NOT NULL,
  Status            STRING(16) NOT NULL,
  ActionsResultJson JSON,
  CreatedAt         TIMESTAMP NOT NULL,
  ExecutedAt        TIMESTAMP,
  ExecutedBy        STRING(64),
) PRIMARY KEY (RunId);

CREATE INDEX ControlTowerPlaybookRuns_ByException
ON ControlTowerPlaybookRuns(ExceptionId, CreatedAt DESC);

CREATE INDEX ControlTowerPlaybookRuns_ByStatus
ON ControlTowerPlaybookRuns(Status, CreatedAt DESC);

CREATE TABLE EchelonTargets (
  SupplierId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  WarehouseId     STRING(36) NOT NULL,
  Echelon         STRING(16) NOT NULL,
  TargetQty       INT64 NOT NULL,
  SafetyQty       INT64 NOT NULL,
  ServiceLevelBps INT64 NOT NULL,
  HorizonDays     INT64 NOT NULL,
  ComputedAt      TIMESTAMP NOT NULL,
  Source          STRING(32) NOT NULL,
) PRIMARY KEY (SupplierId, Sku, WarehouseId, Echelon);

-- Retail OS Phase 0: multi-user identity, capability packs, durable settings
CREATE TABLE RetailerUsers (
  UserId        STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  Phone         STRING(32)  NOT NULL,
  Name          STRING(255),
  PasswordHash  STRING(MAX),
  FirebaseUid   STRING(128),
  RetailerRole  STRING(32)  NOT NULL,
  IsOwner       BOOL        NOT NULL,
  IsActive      BOOL        NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId);

CREATE UNIQUE INDEX UQ_RetailerUsers_ByRetailerPhone ON RetailerUsers(RetailerId, Phone);
CREATE INDEX Idx_RetailerUsers_ByPhone ON RetailerUsers(Phone);
CREATE INDEX Idx_RetailerUsers_ByRetailer ON RetailerUsers(RetailerId, IsActive, UpdatedAt DESC);

-- Wave C1.1: multi-org membership (UserId may belong to many RetailerIds)
CREATE TABLE RetailerUserMemberships (
  UserId          STRING(36)  NOT NULL,
  RetailerId      STRING(36)  NOT NULL,
  RetailerRole    STRING(32)  NOT NULL,
  IsActive        BOOL        NOT NULL DEFAULT (true),
  LocationIdsJson STRING(MAX),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId, RetailerId);

CREATE INDEX Idx_RetailerUserMemberships_ByRetailer
  ON RetailerUserMemberships (RetailerId, IsActive);

CREATE INDEX Idx_RetailerUserMemberships_ByUserActive
  ON RetailerUserMemberships (UserId, IsActive);

CREATE TABLE RetailerCapabilityPacks (
  RetailerId      STRING(36)  NOT NULL,
  PackId          STRING(32)  NOT NULL,
  Enabled         BOOL        NOT NULL,
  EnabledByUserId STRING(36),
  EnabledAt       TIMESTAMP,
  ConfigJson      STRING(MAX),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, PackId);

CREATE TABLE RetailerAutoOrderSettings (
  RetailerId      STRING(36)  NOT NULL,
  SettingsJson    STRING(MAX) NOT NULL,
  UpdatedByUserId STRING(36),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId);

CREATE TABLE RetailerFavoriteSuppliers (
  RetailerId  STRING(36)  NOT NULL,
  SupplierId  STRING(36)  NOT NULL,
  IsFavorite  BOOL        NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, SupplierId);

-- Retail OS Phase 2: multi-location stores + staff scope
CREATE TABLE RetailerLocations (
  LocationId             STRING(36)  NOT NULL,
  RetailerId             STRING(36)  NOT NULL,
  Name                   STRING(255) NOT NULL,
  DeliveryAddress        STRING(MAX),
  PlaceId                STRING(128),
  Lat                    FLOAT64,
  Lng                    FLOAT64,
  H3Cell                 STRING(15),
  ReceivingWindowOpen    STRING(10),
  ReceivingWindowClose   STRING(10),
  IsPrimary              BOOL        NOT NULL,
  IsActive               BOOL        NOT NULL,
  CreatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LocationId);

CREATE INDEX Idx_RetailerLocations_ByRetailer ON RetailerLocations(RetailerId, IsActive, IsPrimary DESC, UpdatedAt DESC);

CREATE TABLE RetailerUserLocations (
  UserId      STRING(36) NOT NULL,
  LocationId  STRING(36) NOT NULL,
  RetailerId  STRING(36) NOT NULL,
  CreatedAt   TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId, LocationId);

CREATE INDEX Idx_RetailerUserLocations_ByRetailerUser ON RetailerUserLocations(RetailerId, UserId);
CREATE INDEX Idx_RetailerUserLocations_ByLocation ON RetailerUserLocations(LocationId);

-- Retail OS Phase 3: retailer store inventory ledger
CREATE TABLE RetailerStockBalances (
  LocationId  STRING(36)  NOT NULL,
  StockBin    STRING(16)  NOT NULL,
  Sku         STRING(64)  NOT NULL,
  RetailerId  STRING(36)  NOT NULL,
  OnHand      INT64       NOT NULL,
  Reserved    INT64       NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LocationId, StockBin, Sku);

CREATE INDEX Idx_RetailerStockBalances_ByRetailerSku ON RetailerStockBalances(RetailerId, Sku, LocationId);

CREATE TABLE RetailerStockMovements (
  MovementId   STRING(36)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  StockBin     STRING(16)  NOT NULL,
  Sku          STRING(64)  NOT NULL,
  Qty          INT64       NOT NULL,
  MovementType STRING(32)  NOT NULL,
  RefType      STRING(32),
  RefId        STRING(64),
  ActorUserId  STRING(36),
  Note         STRING(MAX),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (MovementId);

CREATE INDEX Idx_RetailerStockMovements_ByLocationCreated ON RetailerStockMovements(LocationId, CreatedAt DESC);
CREATE INDEX Idx_RetailerStockMovements_ByRetailerSku ON RetailerStockMovements(RetailerId, Sku, CreatedAt DESC);
CREATE INDEX Idx_RetailerStockMovements_ByRef ON RetailerStockMovements(RefType, RefId);

CREATE TABLE RetailerReceiveSessions (
  SessionId    STRING(36)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  OrderId      STRING(36)  NOT NULL,
  Status       STRING(16)  NOT NULL,
  LinesJson    STRING(MAX) NOT NULL,
  CreatedBy    STRING(36),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ConfirmedAt  TIMESTAMP,
) PRIMARY KEY (SessionId);

CREATE UNIQUE INDEX UQ_RetailerReceiveSessions_ByOrder ON RetailerReceiveSessions(OrderId);
CREATE INDEX Idx_RetailerReceiveSessions_ByRetailer ON RetailerReceiveSessions(RetailerId, CreatedAt DESC);

CREATE TABLE RetailerStockCounts (
  CountId      STRING(36)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  Status       STRING(16)  NOT NULL,
  LinesJson    STRING(MAX) NOT NULL,
  CreatedBy    STRING(36),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CommittedAt  TIMESTAMP,
) PRIMARY KEY (CountId);

CREATE INDEX Idx_RetailerStockCounts_ByLocation ON RetailerStockCounts(LocationId, CreatedAt DESC);

-- Wave C3.3: offline count version etag + force audit
CREATE TABLE RetailerStockLocationVersions (
  RetailerId  STRING(36)  NOT NULL,
  LocationId  STRING(36)  NOT NULL,
  StockBin    STRING(16)  NOT NULL,
  Version     INT64       NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocationId, StockBin);

CREATE TABLE RetailerStockCountForceAudit (
  AuditId       STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  LocationId    STRING(36)  NOT NULL,
  StockBin      STRING(16)  NOT NULL,
  CountId       STRING(36)  NOT NULL,
  ActorUserId   STRING(36)  NOT NULL,
  BaseVersion   INT64       NOT NULL,
  ServerVersion INT64       NOT NULL,
  Reason        STRING(MAX),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (AuditId);

CREATE INDEX Idx_RetailerStockCountForceAudit_ByRetailer
  ON RetailerStockCountForceAudit (RetailerId, CreatedAt DESC);

-- Retail OS Phase 4: POS registers, sessions, sales
CREATE TABLE RetailerRegisters (
  RegisterId  STRING(36)  NOT NULL,
  RetailerId  STRING(36)  NOT NULL,
  LocationId  STRING(36)  NOT NULL,
  Label       STRING(128) NOT NULL,
  Status      STRING(16)  NOT NULL,
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RegisterId);

CREATE INDEX Idx_RetailerRegisters_ByLocation ON RetailerRegisters(LocationId, Status, UpdatedAt DESC);
CREATE INDEX Idx_RetailerRegisters_ByRetailer ON RetailerRegisters(RetailerId, UpdatedAt DESC);

CREATE TABLE RetailerPosSessions (
  SessionId          STRING(36)  NOT NULL,
  RegisterId         STRING(36)  NOT NULL,
  LocationId         STRING(36)  NOT NULL,
  RetailerId         STRING(36)  NOT NULL,
  OpenedByUserId     STRING(36)  NOT NULL,
  ClosedByUserId     STRING(36),
  Status             STRING(16)  NOT NULL,
  OpeningFloatMinor  INT64       NOT NULL,
  ClosingCashMinor   INT64,
  ExpectedCashMinor  INT64,
  VarianceMinor      INT64,
  Currency           STRING(3)   NOT NULL,
  OpenedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClosedAt           TIMESTAMP,
) PRIMARY KEY (SessionId);

CREATE INDEX Idx_RetailerPosSessions_ByRegister ON RetailerPosSessions(RegisterId, Status, OpenedAt DESC);
CREATE INDEX Idx_RetailerPosSessions_ByRetailer ON RetailerPosSessions(RetailerId, OpenedAt DESC);

CREATE TABLE RetailerPosSales (
  SaleId           STRING(36)  NOT NULL,
  SessionId        STRING(36)  NOT NULL,
  RegisterId       STRING(36)  NOT NULL,
  LocationId       STRING(36)  NOT NULL,
  RetailerId       STRING(36)  NOT NULL,
  CashierUserId    STRING(36)  NOT NULL,
  Status           STRING(16)  NOT NULL,
  TotalMinor       INT64       NOT NULL,
  Currency         STRING(3)   NOT NULL,
  ReceiptNumber    STRING(32)  NOT NULL,
  LinesJson        STRING(MAX) NOT NULL,
  TendersJson      STRING(MAX) NOT NULL,
  StockBin         STRING(16)  NOT NULL,
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  VoidedAt         TIMESTAMP,
  VoidedByUserId   STRING(36),
  VoidReason       STRING(MAX),
) PRIMARY KEY (SaleId);

CREATE INDEX Idx_RetailerPosSales_BySession ON RetailerPosSales(SessionId, CreatedAt DESC);
CREATE INDEX Idx_RetailerPosSales_ByRetailer ON RetailerPosSales(RetailerId, CreatedAt DESC);
CREATE UNIQUE INDEX UQ_RetailerPosSales_Receipt ON RetailerPosSales(RetailerId, ReceiptNumber);

-- Wave C3.1: parked POS carts (holds) — never touch OnHand
CREATE TABLE RetailerPosHolds (
  HoldId        STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  LocationId    STRING(36)  NOT NULL,
  RegisterId    STRING(36),
  UserId        STRING(36)  NOT NULL,
  Status        STRING(16)  NOT NULL,
  CartJson      STRING(MAX) NOT NULL,
  Note          STRING(512),
  ExpiresAt     TIMESTAMP   NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResumedAt     TIMESTAMP,
  VoidedAt      TIMESTAMP
) PRIMARY KEY (RetailerId, HoldId);

CREATE INDEX Idx_RetailerPosHolds_ByLocationStatus
  ON RetailerPosHolds (RetailerId, LocationId, Status, ExpiresAt);

CREATE INDEX Idx_RetailerPosHolds_ByExpires
  ON RetailerPosHolds (Status, ExpiresAt);

-- Wave C2.1: HQ analytics projections (sale/void writers same Apply as ledger)
CREATE TABLE RetailerHqSalesDaily (
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  Day          DATE        NOT NULL,
  SkuId        STRING(128) NOT NULL,
  QtySold      INT64       NOT NULL,
  QtyVoided    INT64       NOT NULL,
  GrossMinor   INT64       NOT NULL,
  NetMinor     INT64       NOT NULL,
  Currency     STRING(8)   NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, Day, LocationId, SkuId);

CREATE INDEX Idx_RetailerHqSalesDaily_ByDay
  ON RetailerHqSalesDaily (RetailerId, Day, LocationId);

CREATE TABLE RetailerHqStockSnapshot (
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  SkuId        STRING(128) NOT NULL,
  OnHand       INT64       NOT NULL,
  Reserved     INT64       NOT NULL,
  AsOf         TIMESTAMP   NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocationId, SkuId);

-- Retail OS Phase 5: shifts & time clock
CREATE TABLE RetailerTimeEntries (
  EntryId      STRING(36)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  UserId       STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  Status       STRING(16)  NOT NULL,
  ClockInAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClockOutAt   TIMESTAMP,
  AutoClosed   BOOL        NOT NULL,
  Note         STRING(MAX),
) PRIMARY KEY (EntryId);

CREATE INDEX Idx_RetailerTimeEntries_ByUserStatus ON RetailerTimeEntries(UserId, Status, ClockInAt DESC);
CREATE INDEX Idx_RetailerTimeEntries_ByRetailer ON RetailerTimeEntries(RetailerId, ClockInAt DESC);

CREATE TABLE RetailerShifts (
  ShiftId            STRING(36)  NOT NULL,
  RetailerId         STRING(36)  NOT NULL,
  LocationId         STRING(36)  NOT NULL,
  RegisterId         STRING(36),
  OpenedByUserId     STRING(36)  NOT NULL,
  ClosedByUserId     STRING(36),
  Status             STRING(16)  NOT NULL,
  OpeningFloatMinor  INT64       NOT NULL,
  ClosingCashMinor   INT64,
  ExpectedCashMinor  INT64,
  VarianceMinor      INT64,
  Currency           STRING(3)   NOT NULL,
  LinkedPosSessionId STRING(36),
  OpenedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClosedAt           TIMESTAMP,
) PRIMARY KEY (ShiftId);

CREATE INDEX Idx_RetailerShifts_ByLocationStatus ON RetailerShifts(LocationId, Status, OpenedAt DESC);
CREATE INDEX Idx_RetailerShifts_ByRetailer ON RetailerShifts(RetailerId, OpenedAt DESC);
CREATE INDEX Idx_RetailerShifts_ByPosSession ON RetailerShifts(LinkedPosSessionId);

-- Retail OS Phase 6: sections + assist
CREATE TABLE RetailerSections (
  SectionId   STRING(36)  NOT NULL,
  RetailerId  STRING(36)  NOT NULL,
  LocationId  STRING(36)  NOT NULL,
  Name        STRING(128) NOT NULL,
  AisleTag    STRING(64),
  ShelfTag    STRING(64),
  SortOrder   INT64       NOT NULL,
  Status      STRING(16)  NOT NULL,
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt   TIMESTAMP,
) PRIMARY KEY (SectionId);

CREATE INDEX Idx_RetailerSections_ByLocation ON RetailerSections(LocationId, Status, SortOrder);
CREATE INDEX Idx_RetailerSections_ByRetailer ON RetailerSections(RetailerId, Status, Name);

CREATE TABLE RetailerSectionSkus (
  SectionId  STRING(36)  NOT NULL,
  Sku        STRING(128) NOT NULL,
  LocationId STRING(36)  NOT NULL,
  RetailerId STRING(36)  NOT NULL,
  CreatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SectionId, Sku);

CREATE INDEX Idx_RetailerSectionSkus_BySkuLocation ON RetailerSectionSkus(Sku, LocationId);
CREATE INDEX Idx_RetailerSectionSkus_ByLocation ON RetailerSectionSkus(LocationId, Sku);

CREATE TABLE RetailerStaffSections (
  UserId     STRING(36)  NOT NULL,
  SectionId  STRING(36)  NOT NULL,
  RetailerId STRING(36)  NOT NULL,
  AssignedAt TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId, SectionId);

CREATE INDEX Idx_RetailerStaffSections_BySection ON RetailerStaffSections(SectionId, UserId);
CREATE INDEX Idx_RetailerStaffSections_ByRetailer ON RetailerStaffSections(RetailerId, UserId);

CREATE TABLE RetailerAssistanceTickets (
  TicketId              STRING(36)  NOT NULL,
  RetailerId            STRING(36)  NOT NULL,
  LocationId            STRING(36)  NOT NULL,
  SectionId             STRING(36)  NOT NULL,
  Note                  STRING(MAX) NOT NULL,
  Status                STRING(16)  NOT NULL,
  CreatedByUserId       STRING(36)  NOT NULL,
  ClaimedByUserId       STRING(36),
  CompletedByUserId     STRING(36),
  CreatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClaimedAt             TIMESTAMP,
  CompletedAt           TIMESTAMP,
  SlaDueAt              TIMESTAMP,
  SlaBreachNotifiedAt   TIMESTAMP,
) PRIMARY KEY (TicketId);

CREATE INDEX Idx_RetailerAssist_ByLocationStatus ON RetailerAssistanceTickets(LocationId, Status, CreatedAt DESC);
CREATE INDEX Idx_RetailerAssist_BySectionStatus ON RetailerAssistanceTickets(SectionId, Status, CreatedAt DESC);
CREATE INDEX Idx_RetailerAssist_ByRetailer ON RetailerAssistanceTickets(RetailerId, CreatedAt DESC);
