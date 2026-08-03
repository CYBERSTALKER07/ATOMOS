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
