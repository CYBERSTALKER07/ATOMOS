-- Orders extensions (partial line qty lives in LineItemsJson — no OrderLines table).
ALTER TABLE Orders ADD COLUMN ShopClosedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedReason STRING(64);
ALTER TABLE Orders ADD COLUMN ShopClosedGraceEndsAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedResolution STRING(32);
ALTER TABLE Orders ADD COLUMN PartialDelivery BOOL DEFAULT (false);
ALTER TABLE Orders ADD COLUMN ProximityUnlockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ProximityMethod STRING(16);

CREATE TABLE OrderShopClosedLog (
  OrderId     STRING(36) NOT NULL,
  EventId     STRING(36) NOT NULL,
  Actor       STRING(64) NOT NULL,
  Action      STRING(32) NOT NULL,
  Payload     JSON,
  CreatedAt   TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, EventId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

CREATE INDEX Idx_OrderShopClosedLog_ByOrderCreated
  ON OrderShopClosedLog(OrderId, CreatedAt DESC);
