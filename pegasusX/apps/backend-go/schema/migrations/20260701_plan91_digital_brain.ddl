-- PX91 Digital Brain enterprise extension (2026-07-01)
-- Apply via: cmd/apply-migration or cmd/setup idempotent convergence

ALTER TABLE DemandForecastBaseline ADD COLUMN LowUnits INT64;
ALTER TABLE DemandForecastBaseline ADD COLUMN HighUnits INT64;
ALTER TABLE DemandForecastBaseline ADD COLUMN ConfidencePct INT64;
ALTER TABLE DemandForecastBaseline ADD COLUMN BaselineSource STRING(32);
ALTER TABLE DemandForecastBaseline ADD COLUMN BlockedReason STRING(64);

CREATE TABLE SeasonalTemplateOverrides (
  SupplierId   STRING(36) NOT NULL,
  OverrideId   STRING(36) NOT NULL,
  TemplateId   STRING(64) NOT NULL,
  Name         STRING(128),
  StartDate    DATE        NOT NULL,
  EndDate      DATE        NOT NULL,
  IsActive     BOOL        NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, OverrideId);

CREATE INDEX Idx_SeasonalOverrides_Active
  ON SeasonalTemplateOverrides(SupplierId, IsActive, StartDate DESC);

CREATE TABLE PlanningSignalProjections (
  SupplierId   STRING(36) NOT NULL,
  SignalId     STRING(36) NOT NULL,
  Source       STRING(64) NOT NULL,
  PayloadJson  STRING(MAX) NOT NULL,
  IngestedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, SignalId);

CREATE TABLE PlanningPromoSimulations (
  SupplierId     STRING(36) NOT NULL,
  SimulationId   STRING(36) NOT NULL,
  PromotionId    STRING(36) NOT NULL,
  ResultJson     STRING(MAX) NOT NULL,
  CreatedAt      TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, SimulationId);

CREATE INDEX Idx_PlanningPromoSim_ByPromotion
  ON PlanningPromoSimulations(SupplierId, PromotionId, CreatedAt DESC);
