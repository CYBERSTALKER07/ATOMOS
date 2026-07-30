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
