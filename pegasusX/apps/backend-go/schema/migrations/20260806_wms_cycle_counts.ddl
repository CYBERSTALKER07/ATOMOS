-- §8.7 Wave 1C: cycle counts + inventory adjustments (stub apply deferred).

CREATE TABLE CycleCounts (
  CountId       STRING(36)    NOT NULL,
  WarehouseId   STRING(36)    NOT NULL,
  LocationId    STRING(36)    NOT NULL,
  ProductId     STRING(36)    NOT NULL,
  LotId         STRING(36),
  ExpectedQty   INT64         NOT NULL DEFAULT (0),
  CountedQty    INT64,
  VarianceQty   INT64,
  ReasonCode    STRING(64),
  Status        STRING(24)    NOT NULL DEFAULT ('OPEN'),
  CountedBy     STRING(36),
  CountedAt     TIMESTAMP,
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CountId);

CREATE INDEX Idx_CycleCounts_ByWarehouseStatus
  ON CycleCounts(WarehouseId, Status, CreatedAt DESC);

CREATE INDEX Idx_CycleCounts_ByWarehouseLocationProduct
  ON CycleCounts(WarehouseId, LocationId, ProductId);

CREATE TABLE InventoryAdjustments (
  AdjustmentId  STRING(36)    NOT NULL,
  WarehouseId   STRING(36)    NOT NULL,
  ProductId     STRING(36)    NOT NULL,
  LotId         STRING(36),
  CountId       STRING(36),
  DeltaQty      INT64         NOT NULL,
  ReasonCode    STRING(64),
  Status        STRING(24)    NOT NULL DEFAULT ('PENDING'),
  ActorId       STRING(36),
  ApprovedBy    STRING(36),
  CreatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AdjustmentId);

CREATE INDEX Idx_InventoryAdjustments_ByWarehouseStatus
  ON InventoryAdjustments(WarehouseId, Status, CreatedAt DESC);
