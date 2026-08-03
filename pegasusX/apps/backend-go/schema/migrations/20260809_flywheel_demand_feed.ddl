-- B4.4: Durable STORE_POS flywheel DEMAND_SIGNAL feed for supplier REST/UI.
-- Distinct from planning DemandSignals (multipliers). No POS tender/session fields.
CREATE TABLE FlywheelDemandFeed (
  SignalId    STRING(64) NOT NULL,
  SupplierId  STRING(64),
  RetailerId  STRING(64) NOT NULL,
  LocationId  STRING(64),
  SkuId       STRING(128) NOT NULL,
  Day         DATE NOT NULL,
  QtyDelta    INT64 NOT NULL,
  NetSold     INT64 NOT NULL,
  Kind        STRING(16) NOT NULL,
  Source      STRING(32) NOT NULL,
  CreatedAt   TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SignalId);

CREATE NULL_FILTERED INDEX Idx_FlywheelDemandFeed_SupplierCreated
  ON FlywheelDemandFeed (SupplierId, CreatedAt DESC);

CREATE INDEX Idx_FlywheelDemandFeed_SkuDay
  ON FlywheelDemandFeed (SkuId, Day DESC);
