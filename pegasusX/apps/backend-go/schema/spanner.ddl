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

CREATE TABLE Retailers (
  RetailerId       STRING(36)    NOT NULL,
  Phone            STRING(32)    NOT NULL,
  Name             STRING(255),
  CountryCode      STRING(2)     NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  H3Cell           STRING(15),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId);

CREATE UNIQUE INDEX UQ_Retailers_Phone ON Retailers(Phone);

CREATE TABLE Orders (
  OrderId          STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  RetailerId       STRING(36)    NOT NULL,
  WarehouseId      STRING(36),
  Status           STRING(20)    NOT NULL,
  LineItemsJson    BYTES(MAX)    NOT NULL,
  TotalMinor       INT64         NOT NULL,
  Currency         STRING(3)     NOT NULL,
  H3Cell           STRING(15),
  Lat              FLOAT64,
  Lng              FLOAT64,
  Version          INT64         NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId);

CREATE INDEX Idx_Orders_ByRetailerCreated ON Orders(RetailerId, CreatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierCreated ON Orders(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_Orders_ByWarehouseCreated ON Orders(WarehouseId, CreatedAt DESC);

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

CREATE TABLE Vehicles (
  VehicleId       STRING(36)    NOT NULL,
  Label           STRING(100),
  LicensePlate    STRING(32)    NOT NULL,
  SupplierId      STRING(36)    NOT NULL,
  HomeNodeType    STRING(20)    NOT NULL,
  HomeNodeId      STRING(36)    NOT NULL,
  IsActive        BOOL          NOT NULL,
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (VehicleId);

CREATE INDEX Idx_Vehicles_BySupplierPlate ON Vehicles(SupplierId, LicensePlate);
CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId, IsActive);

CREATE TABLE Warehouses (
  WarehouseId      STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  CoverageRadiusKm FLOAT64       NOT NULL,
  IsActive         BOOL          NOT NULL,
  IsOnShift        BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplier ON Warehouses(SupplierId);

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
