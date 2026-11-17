-- GS-L3: supplier-owned regions + store/retailer/region/city pins.
-- Do not reuse the unused global Regions table (no SupplierId).

CREATE TABLE SupplierRegions (
  SupplierId   STRING(36)  NOT NULL,
  RegionId     STRING(36)  NOT NULL,
  CountryCode  STRING(2)   NOT NULL,
  Name         STRING(128) NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, RegionId);

CREATE INDEX Idx_SupplierRegions_ByCountry ON SupplierRegions(SupplierId, CountryCode);

CREATE TABLE ServicePins (
  PinId        STRING(36)  NOT NULL,
  SupplierId   STRING(36)  NOT NULL,
  WarehouseId  STRING(36)  NOT NULL,
  TargetType   STRING(16)  NOT NULL,
  TargetId     STRING(128) NOT NULL,
  Priority     INT64       NOT NULL DEFAULT (0),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PinId);

CREATE UNIQUE INDEX UQ_ServicePins_Edge ON ServicePins(SupplierId, WarehouseId, TargetType, TargetId);
CREATE INDEX Idx_ServicePins_ByTarget ON ServicePins(SupplierId, TargetType, TargetId);
CREATE INDEX Idx_ServicePins_ByWarehouse ON ServicePins(SupplierId, WarehouseId);
