CREATE TABLE RetailerSegments (
  RetailerId      STRING(36) NOT NULL,
  Segment         STRING(16) NOT NULL,
  Reason          STRING(256),
  EffectiveFrom   TIMESTAMP NOT NULL,
  EffectiveTo     TIMESTAMP,
  UpdatedBy       STRING(64) NOT NULL,
  UpdatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (RetailerId);

CREATE TABLE SkuClasses (
  SupplierId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  VelocityClass   STRING(8) NOT NULL,
  StrategicFlag   BOOL NOT NULL DEFAULT (FALSE),
  UpdatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (SupplierId, Sku);

CREATE TABLE ServicePolicies (
  PolicyId              STRING(36) NOT NULL,
  SupplierId            STRING(36) NOT NULL,
  RetailerSegment       STRING(16) NOT NULL,
  SkuClass              STRING(8) NOT NULL,
  PriorityWeight        INT64 NOT NULL,
  TargetServiceLevelBps INT64 NOT NULL,
  MaxFairShareBps       INT64 NOT NULL,
  MinFairShareBps       INT64 NOT NULL,
  CreditRiskBoost       INT64 NOT NULL,
  Enabled               BOOL NOT NULL DEFAULT (TRUE),
  UpdatedAt             TIMESTAMP NOT NULL,
) PRIMARY KEY (PolicyId);

CREATE INDEX ServicePolicies_BySupplier ON ServicePolicies(SupplierId);

ALTER TABLE OrderLineAllocations ADD COLUMN AllocationMode STRING(16);
ALTER TABLE OrderLineAllocations ADD COLUMN PriorityScore INT64;
ALTER TABLE OrderLineAllocations ADD COLUMN FairShareBps INT64;
ALTER TABLE OrderLineAllocations ADD COLUMN PolicyId STRING(36);

CREATE TABLE AllocationDecisions (
  DecisionId      STRING(36) NOT NULL,
  OrderId         STRING(36) NOT NULL,
  OrderLineId     STRING(36) NOT NULL,
  SupplierId      STRING(36) NOT NULL,
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  WarehouseId     STRING(36) NOT NULL,
  Qty             INT64 NOT NULL,
  AllocationMode  STRING(16) NOT NULL,
  PriorityScore   INT64 NOT NULL,
  FairShareBps    INT64,
  PolicyId        STRING(36),
  RetailerSegment STRING(16),
  SkuClass        STRING(8),
  RiskTier        STRING(16),
  CreatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (DecisionId);

CREATE INDEX AllocationDecisions_ByOrder ON AllocationDecisions(OrderId, CreatedAt DESC);
