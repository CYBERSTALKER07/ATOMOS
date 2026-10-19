-- P2-4: durable planning scenario workbench (clone → compare → publish).

CREATE TABLE PlanningScenarios (
  SupplierId              STRING(36)  NOT NULL,
  ScenarioId              STRING(36)  NOT NULL,
  Version                 INT64       NOT NULL DEFAULT (1),
  Status                  STRING(16)  NOT NULL,
  ParentScenarioId        STRING(36),
  Label                   STRING(128),
  HorizonDays             INT64       NOT NULL DEFAULT (7),
  FactoryDowntimeHours    INT64       NOT NULL DEFAULT (0),
  DemandDeltaPct          FLOAT64     NOT NULL DEFAULT (0),
  ResultJSON              STRING(MAX) NOT NULL,
  Mode                    STRING(32),
  UnitValueSource         STRING(16),
  SnapshotCapturedAt      TIMESTAMP,
  CreatedBy               STRING(128),
  PublishedBy             STRING(128),
  PublishedAt             TIMESTAMP,
  CreatedAt               TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt               TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ScenarioId);

CREATE INDEX Idx_PlanningScenarios_BySupplierStatus
  ON PlanningScenarios(SupplierId, Status, UpdatedAt DESC);
