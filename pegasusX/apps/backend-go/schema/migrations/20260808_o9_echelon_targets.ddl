CREATE TABLE EchelonTargets (
  SupplierId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  WarehouseId     STRING(36) NOT NULL,
  Echelon         STRING(16) NOT NULL,
  TargetQty       INT64 NOT NULL,
  SafetyQty       INT64 NOT NULL,
  ServiceLevelBps INT64 NOT NULL,
  HorizonDays     INT64 NOT NULL,
  ComputedAt      TIMESTAMP NOT NULL,
  Source          STRING(32) NOT NULL,
) PRIMARY KEY (SupplierId, Sku, WarehouseId, Echelon);
