-- P8/P10: factory S&OP capacity column + durable ops broadcasts.

ALTER TABLE Factories ADD COLUMN DailyOutputCapacity INT64 NOT NULL DEFAULT (0);

CREATE TABLE OpsBroadcasts (
  BroadcastId  STRING(36)  NOT NULL,
  SupplierId   STRING(36)  NOT NULL,
  WarehouseId  STRING(36),
  Scope        STRING(16)  NOT NULL,
  Title        STRING(255) NOT NULL,
  Body         STRING(MAX) NOT NULL,
  TargetRole   STRING(32)  NOT NULL,
  CreatedBy    STRING(128),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (BroadcastId);

CREATE INDEX Idx_OpsBroadcasts_BySupplierCreated ON OpsBroadcasts(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_OpsBroadcasts_ByWarehouseCreated ON OpsBroadcasts(WarehouseId, CreatedAt DESC);
