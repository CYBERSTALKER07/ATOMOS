-- ATOMOS P2 Wave 4A: Multi-Entity & Intercompany, FX Revaluation, Process Templates/Mining, P2P 3-Way Match
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

CREATE TABLE LegalEntities (
  TenantId       STRING(64)  NOT NULL,
  EntityId       STRING(64)  NOT NULL,
  Name           STRING(160) NOT NULL,
  Tin            STRING(32),
  CountryCode    STRING(8)   NOT NULL DEFAULT ('UZ'),
  BaseCurrency   STRING(3)   NOT NULL DEFAULT ('UZS'),
  IsParent       BOOL        NOT NULL DEFAULT (false),
  ParentEntityId STRING(64),
  Status         STRING(24)  NOT NULL DEFAULT ('ACTIVE'),
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, EntityId);

CREATE INDEX Idx_LegalEntities_ByTin
  ON LegalEntities(TenantId, Tin);

CREATE TABLE IntercompanyTransfers (
  TenantId        STRING(64)  NOT NULL,
  TransferId      STRING(64)  NOT NULL,
  FromEntityId    STRING(64)  NOT NULL,
  ToEntityId      STRING(64)  NOT NULL,
  TransferType    STRING(32)  NOT NULL,
  AmountMinor     INT64       NOT NULL,
  Currency        STRING(3)   NOT NULL,
  PeriodId        STRING(32)  NOT NULL,
  ReferenceType   STRING(64),
  ReferenceId     STRING(128),
  DebitJournalId  STRING(64),
  CreditJournalId STRING(64),
  Status          STRING(24)  NOT NULL,
  IdempotencyKey  STRING(256),
  CreatedBy       STRING(128) NOT NULL,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, TransferId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_IntercompanyTransfers_Idempotency
  ON IntercompanyTransfers(TenantId, IdempotencyKey);

CREATE TABLE FxRevaluationRuns (
  TenantId                STRING(64)  NOT NULL,
  RunId                   STRING(64)  NOT NULL,
  EntityId                STRING(64),
  PeriodId                STRING(32)  NOT NULL,
  BaseCurrency            STRING(3)   NOT NULL,
  UnrealizedGainLossMinor INT64       NOT NULL,
  RoundingDeltaMinor      INT64       NOT NULL DEFAULT (0),
  RatesJson               STRING(MAX),
  JournalEntryId          STRING(64),
  IdempotencyKey          STRING(256),
  CreatedBy               STRING(128) NOT NULL,
  CreatedAt               TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RunId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_FxRevaluationRuns_Idempotency
  ON FxRevaluationRuns(TenantId, IdempotencyKey);

ALTER TABLE JournalEntries ADD COLUMN LegalEntityId STRING(64);

CREATE INDEX Idx_JournalEntries_ByEntityPeriod
  ON JournalEntries(EntityId, LegalEntityId, PeriodId);

CREATE TABLE ProcessTemplates (
  TenantId   STRING(64)  NOT NULL,
  TemplateId STRING(64)  NOT NULL,
  Code       STRING(64)  NOT NULL,
  Name       STRING(160) NOT NULL,
  Domain     STRING(24)  NOT NULL,
  StepsJson  STRING(MAX) NOT NULL,
  IsDefault  BOOL        NOT NULL DEFAULT (false),
  Status     STRING(24)  NOT NULL DEFAULT ('ACTIVE'),
  CreatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, TemplateId);

CREATE TABLE ProcessInstances (
  TenantId              STRING(64)  NOT NULL,
  InstanceId            STRING(64)  NOT NULL,
  TemplateId            STRING(64)  NOT NULL,
  EntityId              STRING(64),
  ObjectType            STRING(64)  NOT NULL,
  ObjectId              STRING(128) NOT NULL,
  CurrentStep           STRING(64)  NOT NULL,
  Status                STRING(24)  NOT NULL,
  TransitionHistoryJson STRING(MAX) NOT NULL,
  StartedAt             TIMESTAMP   NOT NULL,
  CompletedAt           TIMESTAMP,
  UpdatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, InstanceId);

CREATE INDEX Idx_ProcessInstances_ByObject
  ON ProcessInstances(TenantId, ObjectType, ObjectId);

CREATE TABLE PurchaseOrders (
  TenantId          STRING(64)  NOT NULL,
  PoId              STRING(64)  NOT NULL,
  EntityId          STRING(64),
  VendorId          STRING(128) NOT NULL,
  VendorName        STRING(160),
  VendorTin         STRING(32),
  Currency          STRING(3)   NOT NULL DEFAULT ('UZS'),
  SubtotalMinor     INT64       NOT NULL,
  VatMinor          INT64       NOT NULL DEFAULT (0),
  TotalMinor        INT64       NOT NULL,
  Status            STRING(24)  NOT NULL,
  ApprovalRequestId STRING(64),
  IdempotencyKey    STRING(256),
  ExpectedAt        TIMESTAMP,
  CreatedBy         STRING(128) NOT NULL,
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, PoId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_PurchaseOrders_Idempotency
  ON PurchaseOrders(TenantId, IdempotencyKey);

CREATE TABLE PurchaseOrderLines (
  TenantId       STRING(64)  NOT NULL,
  PoId           STRING(64)  NOT NULL,
  LineNum        INT64       NOT NULL,
  Sku            STRING(128) NOT NULL,
  ProductName    STRING(256),
  QtyOrdered     INT64       NOT NULL,
  QtyReceived    INT64       NOT NULL DEFAULT (0),
  UnitCostMinor  INT64       NOT NULL,
  VatBps         INT64       NOT NULL DEFAULT (1200),
  LineTotalMinor INT64       NOT NULL
) PRIMARY KEY (TenantId, PoId, LineNum),
  INTERLEAVE IN PARENT PurchaseOrders ON DELETE CASCADE;

CREATE TABLE GoodsReceipts (
  TenantId           STRING(64)  NOT NULL,
  ReceiptId          STRING(64)  NOT NULL,
  PoId               STRING(64)  NOT NULL,
  EntityId           STRING(64),
  WarehouseId        STRING(64)  NOT NULL,
  Status             STRING(24)  NOT NULL,
  LinesJson          STRING(MAX) NOT NULL,
  TotalReceivedMinor INT64       NOT NULL DEFAULT (0),
  JournalEntryId     STRING(64),
  IdempotencyKey     STRING(256),
  ReceivedBy         STRING(128) NOT NULL,
  ReceivedAt         TIMESTAMP   NOT NULL,
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, ReceiptId);

CREATE INDEX Idx_GoodsReceipts_ByPo
  ON GoodsReceipts(TenantId, PoId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_GoodsReceipts_Idempotency
  ON GoodsReceipts(TenantId, IdempotencyKey);

CREATE TABLE VendorInvoices (
  TenantId        STRING(64)  NOT NULL,
  VendorInvoiceId STRING(64)  NOT NULL,
  PoId            STRING(64)  NOT NULL,
  ReceiptId       STRING(64),
  EsfDocId        STRING(64),
  EntityId        STRING(64),
  VendorId        STRING(128) NOT NULL,
  Currency        STRING(3)   NOT NULL DEFAULT ('UZS'),
  SubtotalMinor   INT64       NOT NULL,
  VatMinor        INT64       NOT NULL DEFAULT (0),
  TotalMinor      INT64       NOT NULL,
  MatchStatus     STRING(24)  NOT NULL,
  VarianceMinor   INT64       NOT NULL DEFAULT (0),
  VarianceReason  STRING(MAX),
  JournalEntryId  STRING(64),
  IdempotencyKey  STRING(256),
  CreatedBy       STRING(128) NOT NULL,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, VendorInvoiceId);

CREATE INDEX Idx_VendorInvoices_ByPo
  ON VendorInvoices(TenantId, PoId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_VendorInvoices_Idempotency
  ON VendorInvoices(TenantId, IdempotencyKey);
