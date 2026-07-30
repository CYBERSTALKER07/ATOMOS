-- Orders extensions
ALTER TABLE Orders ADD COLUMN ShopClosedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedReason STRING(64);
ALTER TABLE Orders ADD COLUMN ShopClosedGraceEndsAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedResolution STRING(32);
ALTER TABLE Orders ADD COLUMN PartialDelivery BOOL DEFAULT (false);
ALTER TABLE Orders ADD COLUMN ProximityUnlockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ProximityMethod STRING(16);

-- OrderLines extensions
ALTER TABLE OrderLines ADD COLUMN DeliveredQty INT64;
ALTER TABLE OrderLines ADD COLUMN RemainingQty INT64;
ALTER TABLE OrderLines ADD COLUMN OffloadStatus STRING(16);
ALTER TABLE OrderLines ADD COLUMN OffloadReason STRING(64);

-- Shop-closed interaction log
CREATE TABLE OrderShopClosedLog (
  OrderId     STRING(36) NOT NULL,
  EventId     STRING(36) NOT NULL,
  Actor       STRING(64) NOT NULL,
  Action      STRING(32) NOT NULL,
  Payload     JSON,
  CreatedAt   TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, EventId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;
