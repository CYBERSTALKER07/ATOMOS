-- Durable multi-pod idempotency for auto-order draft/place buckets.
-- PK: one row per retailer + calendar day + mode + SKU.
CREATE TABLE RetailerAutoOrderBucket (
  RetailerId STRING(64) NOT NULL,
  Day        STRING(10) NOT NULL,
  Mode       STRING(16) NOT NULL,
  Sku        STRING(128) NOT NULL,
  RunId      STRING(64),
  OrderId    STRING(64),
  CreatedAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, Day, Mode, Sku);
