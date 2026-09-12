-- Phase 1: refunds (provider-confirmed reversal legs) + supplier payout batches.
-- Immutable ledger style: refunds are reversal rows, never mutation of captures.
CREATE TABLE Refunds (
  RefundId         STRING(36)  NOT NULL,
  OrderId          STRING(36)  NOT NULL,
  SupplierId       STRING(36)  NOT NULL,
  RetailerId       STRING(36)  NOT NULL,
  AmountMinor      INT64       NOT NULL,
  Currency         STRING(8)   NOT NULL,
  ReasonCode       STRING(64)  NOT NULL,
  ReasonText       STRING(MAX),
  Status           STRING(16)  NOT NULL,  -- PENDING | CAPTURED | FAILED
  Gateway          STRING(32),
  ProviderRef      STRING(128),
  CreditNoteId     STRING(36),
  CorrectiveEhfId  STRING(128),
  IdempotencyKey   STRING(128) NOT NULL,
  CreatedBy        STRING(64)  NOT NULL,
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RefundId);

CREATE UNIQUE INDEX UQ_Refunds_IdempotencyKey ON Refunds(IdempotencyKey);
CREATE INDEX Idx_Refunds_ByOrder ON Refunds(OrderId);
CREATE INDEX Idx_Refunds_BySupplierCreated ON Refunds(SupplierId, CreatedAt);

CREATE TABLE PayoutBatches (
  BatchId            STRING(36)  NOT NULL,
  SupplierId         STRING(36)  NOT NULL,
  PeriodStart        DATE        NOT NULL,
  PeriodEnd          DATE        NOT NULL,
  GrossCapturedMinor INT64       NOT NULL,
  RefundedMinor      INT64       NOT NULL,
  CommissionMinor    INT64       NOT NULL,
  NetPayoutMinor     INT64       NOT NULL,
  Currency           STRING(8)   NOT NULL,
  Status             STRING(16)  NOT NULL,  -- DRAFT | EXPORTED | PAID
  ExportFileUri      STRING(MAX),
  IdempotencyKey     STRING(128) NOT NULL,
  CreatedBy          STRING(64)  NOT NULL,
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (BatchId);

CREATE UNIQUE INDEX UQ_PayoutBatches_IdempotencyKey ON PayoutBatches(IdempotencyKey);
CREATE UNIQUE INDEX UQ_PayoutBatches_SupplierPeriod ON PayoutBatches(SupplierId, PeriodStart, PeriodEnd);
