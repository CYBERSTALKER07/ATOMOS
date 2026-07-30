-- Enhanced shop-closed protocol + proximity settlement + partial offload.
-- Additive only. Safe on live Spanner. Line-level qty lives in LineItemsJson
-- (no separate OrderLines table — see order.LineItem DeliveredQty/RemainingQty).
-- Wire status ARRIVED_SHOP_CLOSED maps to design SHOP_CLOSED_PENDING.

ALTER TABLE Orders ADD COLUMN ShopClosedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedReason STRING(64);
ALTER TABLE Orders ADD COLUMN ShopClosedGraceEndsAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ShopClosedResolution STRING(32);
ALTER TABLE Orders ADD COLUMN PartialDelivery BOOL;
ALTER TABLE Orders ADD COLUMN ProximityUnlockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ProximityMethod STRING(16);

CREATE TABLE OrderShopClosedLog (
  OrderId   STRING(36) NOT NULL,
  EventId   STRING(36) NOT NULL,
  Actor     STRING(64) NOT NULL,
  Action    STRING(32) NOT NULL,
  Payload   BYTES(MAX),
  CreatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderId, EventId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

CREATE INDEX Idx_OrderShopClosedLog_ByOrderCreated
  ON OrderShopClosedLog(OrderId, CreatedAt DESC);
