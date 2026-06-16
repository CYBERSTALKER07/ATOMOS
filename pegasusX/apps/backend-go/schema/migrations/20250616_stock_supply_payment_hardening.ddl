-- Stock UX, supply variance, and factory-driver transfer hardening.

ALTER TABLE Warehouses ADD COLUMN ShowStockCountsToRetailers BOOL NOT NULL DEFAULT (FALSE);

ALTER TABLE WarehouseSupplyRequestItems ADD COLUMN ShippedQuantity INT64;
ALTER TABLE WarehouseSupplyRequestItems ADD COLUMN ReceivedQuantity INT64;
ALTER TABLE WarehouseSupplyRequestItems ADD COLUMN VarianceReason STRING(MAX);
