-- Warehouse stock policy, supply-request enrichment, and per-SKU inventory policy.
-- Safe to apply on environments that already have the canonical spanner.ddl snapshot.

ALTER TABLE Warehouses ADD COLUMN DefaultOutOfStockPolicy STRING(24) NOT NULL DEFAULT ('REJECT');
ALTER TABLE Warehouses ADD COLUMN OperatingSchedule JSON;

ALTER TABLE SupplierInventoryV2 ADD COLUMN OutOfStockPolicy STRING(24);
ALTER TABLE SupplierInventoryV2 ADD COLUMN ReorderThreshold INT64 NOT NULL DEFAULT (0);

ALTER TABLE WarehouseSupplyRequests ADD COLUMN Priority STRING(16);
ALTER TABLE WarehouseSupplyRequests ADD COLUMN Notes STRING(MAX);
ALTER TABLE WarehouseSupplyRequests ADD COLUMN RegionId STRING(36);
ALTER TABLE WarehouseSupplyRequests ADD COLUMN RequestedDeliveryDate TIMESTAMP;
ALTER TABLE WarehouseSupplyRequests ADD COLUMN DemandBreakdown JSON;
ALTER TABLE WarehouseSupplyRequests ADD COLUMN TotalVolumeVU FLOAT64 NOT NULL DEFAULT (0);

CREATE TABLE WarehouseSupplyRequestItems (
  RequestId            STRING(36)  NOT NULL,
  ItemId               STRING(36)  NOT NULL,
  ProductId            STRING(36)  NOT NULL,
  RequestedQuantity    INT64       NOT NULL,
  RecommendedQuantity  INT64       NOT NULL DEFAULT (0),
  UnitVolumeVU         FLOAT64     NOT NULL DEFAULT (0),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId, ItemId),
  INTERLEAVE IN PARENT WarehouseSupplyRequests ON DELETE CASCADE;
