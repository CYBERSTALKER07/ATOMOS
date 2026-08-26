-- WMS Warehouse Physical Locations (Aisle, Rack, Shelf, Bin)
CREATE TABLE Locations (
  LocationId    STRING(36)    NOT NULL,
  WarehouseId   STRING(36)    NOT NULL,
  SupplierId    STRING(36)    NOT NULL,
  Aisle         STRING(32)    NOT NULL,
  Rack          STRING(32)    NOT NULL,
  Shelf         STRING(32)    NOT NULL,
  Bin           STRING(32)    NOT NULL,
  Zone          STRING(32)    NOT NULL DEFAULT ('DEFAULT'),
  LocationType  STRING(32)    NOT NULL DEFAULT ('PICK'),
  IsActive      BOOL          NOT NULL DEFAULT (TRUE),
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LocationId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_Locations_Coordinates ON Locations(WarehouseId, Aisle, Rack, Shelf, Bin);
CREATE INDEX Idx_Locations_ByWarehouse ON Locations(WarehouseId, IsActive);
CREATE INDEX Idx_Locations_BySupplier ON Locations(SupplierId, WarehouseId);

-- WMS Stock Lots with Batch Expiration Tracking
CREATE TABLE StockLots (
  LotId              STRING(36)    NOT NULL,
  SupplierId         STRING(36)    NOT NULL,
  WarehouseId        STRING(36)    NOT NULL,
  ProductId          STRING(36)    NOT NULL,
  LocationId         STRING(36)    NOT NULL,
  LotCode            STRING(64)    NOT NULL,
  ManufacturedDate   TIMESTAMP,
  ExpiryDate         TIMESTAMP     NOT NULL,
  QuantityOnHand     INT64         NOT NULL DEFAULT (0),
  QuantityAllocated  INT64         NOT NULL DEFAULT (0),
  Status             STRING(32)    NOT NULL DEFAULT ('AVAILABLE'),
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LotId);

CREATE INDEX Idx_StockLots_ByFefo ON StockLots(WarehouseId, ProductId, Status, ExpiryDate ASC);
CREATE INDEX Idx_StockLots_ByLocation ON StockLots(LocationId);
CREATE INDEX Idx_StockLots_BySupplierProduct ON StockLots(SupplierId, ProductId, ExpiryDate ASC);
CREATE UNIQUE NULL_FILTERED INDEX UQ_StockLots_LotLocation ON StockLots(WarehouseId, ProductId, LocationId, LotCode);
