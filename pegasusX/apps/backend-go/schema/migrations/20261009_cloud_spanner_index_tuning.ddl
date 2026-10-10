-- 20261009_cloud_spanner_index_tuning.ddl
-- Cloud Spanner High-Throughput Index Tuning & Covering Index Optimization
--
-- 1) Drop redundant PK-prefix index on OrderStatusTransitions and create covering index
--    for CreatedAt range queries (pulse feed / timeline queries).
DROP INDEX Idx_OrderStatusTransitions_ByOrder;
CREATE INDEX Idx_OrderStatusTransitions_ByCreatedAt
  ON OrderStatusTransitions(CreatedAt DESC, OrderId)
  STORING (NewStatus, Reason, EventKind, PreviousStatus, ActorRole, ActorId);

-- 2) Covering index on OutboxEvents to eliminate base-table back-joins and read-lock contention
--    during outbox dispatcher candidate lease loops.
DROP INDEX Idx_OutboxEvents_Unpublished;
CREATE INDEX Idx_OutboxEvents_Unpublished
  ON OutboxEvents(PublishedAt, CreatedAt)
  STORING (ClaimedUntil, AggregateType, AggregateId, TopicName, SupplierId);

-- 3) Covering indexes on StockLots for high-frequency FEFO reservation and inventory rollup
--    without base table read-lock acquisition inside RW transactions.
DROP INDEX Idx_StockLots_BySupplierWarehouseProduct;
CREATE INDEX Idx_StockLots_BySupplierWarehouseProduct
  ON StockLots(SupplierId, WarehouseId, ProductId, Status)
  STORING (QuantityOnHand, QuantityReserved, ExpiryDate, ReceivedAt);

DROP INDEX Idx_StockLots_ByWarehouseProductExpiry;
CREATE INDEX Idx_StockLots_ByWarehouseProductExpiry
  ON StockLots(WarehouseId, ProductId, ExpiryDate)
  STORING (QuantityOnHand, QuantityReserved, Status, LocationId);

-- 4) Covering index on RetailerStockBalances for real-time inventory aggregation.
DROP INDEX Idx_RetailerStockBalances_ByRetailerSku;
CREATE INDEX Idx_RetailerStockBalances_ByRetailerSku
  ON RetailerStockBalances(RetailerId, Sku, LocationId)
  STORING (OnHand, Reserved);

-- 5) NULL_FILTERED covering index on OrderFiscalReceipts to eliminate NULL entries for
--    un-receipted attempts and cover fiscal receipt verification reads.
DROP INDEX Idx_OrderFiscalReceipts_ByReceiptId;
CREATE NULL_FILTERED INDEX Idx_OrderFiscalReceipts_ByReceiptId
  ON OrderFiscalReceipts(FiscalReceiptId)
  STORING (Status, Provider, AmountMinor, Currency);
