CREATE TABLE OrderLineAllocations (
  OrderId       STRING(36) NOT NULL,
  OrderLineId   STRING(36) NOT NULL,
  WarehouseId   STRING(36) NOT NULL,
  Sku           STRING(64) NOT NULL,
  Qty           INT64 NOT NULL,
  CreatedAt     TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderId, OrderLineId, WarehouseId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;
