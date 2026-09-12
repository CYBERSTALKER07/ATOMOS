-- §8.7 Wave 1A: warehouse bin locations, stock lots, order lot reservations, shelf-life knobs.

ALTER TABLE Products ADD COLUMN MinShelfLifeDays INT64;

ALTER TABLE Retailers ADD COLUMN MinShelfLifeDays INT64;

CREATE TABLE WarehouseLocations (
  WarehouseId   STRING(36)    NOT NULL,
  LocationId    STRING(36)    NOT NULL,
  Zone          STRING(64),
  Aisle         STRING(32),
  Rack          STRING(32),
  Level         STRING(32),
  Bin           STRING(32),
  LocationType  STRING(24)    NOT NULL DEFAULT ('PICK'),
  PickSequence  INT64         NOT NULL DEFAULT (0),
  MaxVolumeVU   FLOAT64,
  IsActive      BOOL          NOT NULL DEFAULT (TRUE),
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId, LocationId);

CREATE INDEX Idx_WarehouseLocations_ByZoneSeq
  ON WarehouseLocations(WarehouseId, Zone, PickSequence);

CREATE TABLE StockLots (
  LotId              STRING(36)    NOT NULL,
  SupplierId         STRING(36)    NOT NULL,
  WarehouseId        STRING(36)    NOT NULL,
  ProductId          STRING(36)    NOT NULL,
  LocationId         STRING(36)    NOT NULL,
  LotCode            STRING(64),
  ExpiryDate         DATE,
  ManufacturedDate   DATE,
  QuantityOnHand     INT64         NOT NULL DEFAULT (0),
  QuantityReserved   INT64         NOT NULL DEFAULT (0),
  ReceivedAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  Status             STRING(24)    NOT NULL DEFAULT ('AVAILABLE'),
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LotId);

CREATE INDEX Idx_StockLots_ByWarehouseProductExpiry
  ON StockLots(WarehouseId, ProductId, ExpiryDate);

CREATE INDEX Idx_StockLots_ByWarehouseLocation
  ON StockLots(WarehouseId, LocationId);

CREATE INDEX Idx_StockLots_BySupplierWarehouseProduct
  ON StockLots(SupplierId, WarehouseId, ProductId, Status);

CREATE TABLE OrderLotReservations (
  OrderId    STRING(36)    NOT NULL,
  LotId      STRING(36)    NOT NULL,
  Quantity   INT64         NOT NULL,
  CreatedAt  TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, LotId);
