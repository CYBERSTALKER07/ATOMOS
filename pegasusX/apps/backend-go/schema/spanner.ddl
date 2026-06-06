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
  RegisteredAt            TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ConfiguredAt            TIMESTAMP,
  UpdatedAt               TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE INDEX Idx_SupplierProfiles_ByUpdatedAt ON SupplierProfiles(UpdatedAt DESC);

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

CREATE TABLE Retailers (
  RetailerId              STRING(36)    NOT NULL,
  Phone                   STRING(32)    NOT NULL,
  Name                    STRING(255),
  CountryCode             STRING(2)     NOT NULL,
  Lat                     FLOAT64,
  Lng                     FLOAT64,
  H3Cell                  STRING(15),
  ReceivingWindowOpen     STRING(10),
  ReceivingWindowClose    STRING(10),
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
  Status           STRING(20)    NOT NULL,
  OrderSource      STRING(24)    NOT NULL,
  ConfirmationStatus STRING(24)  NOT NULL,
  LineItemsJson    BYTES(MAX)    NOT NULL,
  TotalMinor       INT64         NOT NULL,
  Currency         STRING(3)     NOT NULL,
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
  CoverageRadiusKm   FLOAT64       NOT NULL,
  PrimaryFactoryId   STRING(36),
  SecondaryFactoryId STRING(36),
  IsActive           BOOL          NOT NULL,
  IsOnShift          BOOL          NOT NULL,
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplier ON Warehouses(SupplierId);
CREATE INDEX Idx_Warehouses_ByPrimaryFactory ON Warehouses(SupplierId, PrimaryFactoryId);

CREATE TABLE Factories (
  FactoryId        STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
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
  CreatedAt                   TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                   TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId);

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
  IsActive       BOOL          NOT NULL DEFAULT (TRUE),
  Version        INT64         NOT NULL DEFAULT (1),
  CreatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProductId);

CREATE INDEX Idx_Products_BySupplierCategory ON Products(SupplierId, CategoryId, IsActive);
CREATE INDEX Idx_Products_BySupplierActive ON Products(SupplierId, IsActive, UpdatedAt DESC);

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
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TransferId);

CREATE INDEX Idx_FactoryTransfers_ByFactoryId ON FactoryInternalTransfers(FactoryId, UpdatedAt DESC);
CREATE INDEX Idx_FactoryTransfers_BySupplierId ON FactoryInternalTransfers(SupplierId);
CREATE INDEX Idx_FactoryTransfers_ByManifestId ON FactoryInternalTransfers(ManifestId);

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
