-- Retail OS Phase 4: POS
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
