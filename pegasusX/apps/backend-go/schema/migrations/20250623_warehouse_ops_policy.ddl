-- Warehouse-configurable pre-order lead window, order line limits, delivery fee tiers.
ALTER TABLE Warehouses ADD COLUMN PreorderMinLeadDays INT64 NOT NULL DEFAULT (3);
ALTER TABLE Warehouses ADD COLUMN PreorderMaxLeadDays INT64 NOT NULL DEFAULT (90);
ALTER TABLE Warehouses ADD COLUMN OrderLineMinQuantity INT64;
ALTER TABLE Warehouses ADD COLUMN OrderLineMaxQuantity INT64;
ALTER TABLE Warehouses ADD COLUMN DeliveryFeeRules JSON;

-- Supplier catalog pack/case metadata for retailer display and checkout multiples.
ALTER TABLE Products ADD COLUMN SaleUnit STRING(16) NOT NULL DEFAULT ('UNIT');
ALTER TABLE Products ADD COLUMN UnitsPerPack INT64;
