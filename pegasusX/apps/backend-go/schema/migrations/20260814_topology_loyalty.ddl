-- Topology coverage, node country, retailer loyalty earn (no burn).

ALTER TABLE Warehouses ADD COLUMN CountryCode STRING(2);
ALTER TABLE Factories ADD COLUMN CountryCode STRING(2);

CREATE TABLE WarehouseCoverageCells (
  WarehouseId  STRING(36)  NOT NULL,
  H3Cell       STRING(16)  NOT NULL,
  SupplierId   STRING(36)  NOT NULL,
  CityName     STRING(128),
  Source       STRING(16)  NOT NULL DEFAULT ('CITY'),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId, H3Cell);

CREATE INDEX Idx_WarehouseCoverage_BySupplierCell ON WarehouseCoverageCells(SupplierId, H3Cell);

CREATE TABLE WarehouseCoverageCities (
  WarehouseId  STRING(36)  NOT NULL,
  CityName     STRING(128) NOT NULL,
  Lat          FLOAT64     NOT NULL,
  Lng          FLOAT64     NOT NULL,
  SupplierId   STRING(36)  NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId, CityName);

CREATE TABLE LoyaltyPrograms (
  SupplierId     STRING(36)  NOT NULL,
  EarnBps        INT64       NOT NULL DEFAULT (100),
  TiersJson      STRING(MAX),
  Reason         STRING(512),
  UpdatedBy      STRING(128),
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE TABLE LoyaltyAccounts (
  SupplierId      STRING(36)  NOT NULL,
  RetailerId      STRING(36)  NOT NULL,
  LifetimePoints  INT64       NOT NULL DEFAULT (0),
  AvailablePoints INT64       NOT NULL DEFAULT (0),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, RetailerId);

CREATE TABLE LoyaltyLedger (
  SupplierId   STRING(36)  NOT NULL,
  LedgerId     STRING(64)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  OrderId      STRING(36)  NOT NULL,
  Points       INT64       NOT NULL,
  EarnBps      INT64       NOT NULL,
  AmountMinor  INT64       NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, LedgerId);

CREATE UNIQUE INDEX UQ_LoyaltyLedger_ByOrder ON LoyaltyLedger(SupplierId, OrderId);
CREATE INDEX Idx_LoyaltyLedger_ByRetailer ON LoyaltyLedger(SupplierId, RetailerId, CreatedAt DESC);
