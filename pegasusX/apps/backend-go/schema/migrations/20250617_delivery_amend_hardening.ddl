-- Delivery amend hardening: preserve pre-amend totals and quarantine rejected SKUs.

ALTER TABLE Orders ADD COLUMN OriginalTotalMinor INT64 NOT NULL DEFAULT (0);

CREATE TABLE SupplierReturns (
  ReturnId     STRING(36)  NOT NULL,
  OrderId      STRING(36)  NOT NULL,
  SkuId        STRING(50)  NOT NULL,
  RejectedQty  INT64       NOT NULL,
  Reason       STRING(50)  NOT NULL,
  DriverNotes  STRING(MAX),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ReturnId);

CREATE INDEX Idx_SupplierReturns_ByOrder ON SupplierReturns(OrderId);
CREATE INDEX Idx_SupplierReturns_BySku ON SupplierReturns(SkuId);
