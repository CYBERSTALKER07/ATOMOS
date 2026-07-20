-- ADR-009 Fiscal hard-gate: immutable fiscal attempts + denormalized order status.
-- Additive only. Apply via backend-setup / phase0 migrate.
-- Do not put semicolons in end-of-line comments (apply-migration splits on ;).
-- Attempt Status values: PENDING | SUCCESS | FAILED | FORCE_SKIPPED (app-enforced).
-- Amounts: integer Tiyin (AmountMinor INT64). Currency UZS.
-- Receipts are supplier-scoped. Today one order equals one supplier leg equals order total.

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
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (OrderId, AttemptId);

CREATE INDEX Idx_OrderFiscalReceipts_ByStatusCreated
  ON OrderFiscalReceipts(Status, CreatedAt DESC);

CREATE INDEX Idx_OrderFiscalReceipts_ByReceiptId
  ON OrderFiscalReceipts(FiscalReceiptId);

CREATE INDEX Idx_OrderFiscalReceipts_BySupplier
  ON OrderFiscalReceipts(SupplierId, CreatedAt DESC);

-- FiscalStatus app values: NONE | PENDING | SUCCESS | FAILED | FORCE_SKIPPED
ALTER TABLE Orders ADD COLUMN FiscalStatus STRING(32);
ALTER TABLE Orders ADD COLUMN LatestFiscalReceiptId STRING(128);
ALTER TABLE Orders ADD COLUMN FiscalizedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN LatestFiscalAttemptId STRING(36);
