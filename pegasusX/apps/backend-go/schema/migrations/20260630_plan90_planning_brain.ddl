-- PX90 planning brain: MEIO policies, control-tower overrides, one-number demand baseline.

CREATE TABLE ReplenishmentPolicies (
  SupplierId                 STRING(36)  NOT NULL,
  AutoApproveStable          BOOL        NOT NULL DEFAULT (true),
  AutoApprovePredictivePush  BOOL        NOT NULL DEFAULT (true),
  MaxDailyTransferUnits      INT64       NOT NULL DEFAULT (500),
  MinConfidenceScore         FLOAT64     NOT NULL DEFAULT (0.85),
  UpdatedAt                  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE TABLE ControlTowerZoneOverrides (
  OverrideId       STRING(36)  NOT NULL,
  SupplierId       STRING(36)  NOT NULL,
  WarehouseId      STRING(36),
  Action           STRING(32)  NOT NULL,
  PolygonGeoJSON   STRING(MAX) NOT NULL,
  TtlExpiresAt     TIMESTAMP   NOT NULL,
  CreatedBy        STRING(36),
  IsActive         BOOL        NOT NULL DEFAULT (true),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OverrideId);

CREATE INDEX Idx_ControlTowerOverrides_BySupplier
  ON ControlTowerZoneOverrides(SupplierId, IsActive, TtlExpiresAt DESC);

CREATE TABLE DemandForecastBaseline (
  SupplierId    STRING(36)  NOT NULL,
  ForecastDate  DATE        NOT NULL,
  WarehouseId   STRING(36)  NOT NULL,
  ProductId     STRING(36)  NOT NULL,
  BaselineQty   INT64       NOT NULL,
  Confidence    FLOAT64     NOT NULL DEFAULT (0),
  Source        STRING(32)  NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ForecastDate, WarehouseId, ProductId);

CREATE INDEX Idx_DemandBaseline_ByWarehouseDate
  ON DemandForecastBaseline(WarehouseId, ForecastDate DESC);
