-- P5 factory planning port: lane plane, network mode, locks, run audit, transfer SLA, transfer source.
-- Apply: go run ./cmd/apply-migration --ddl schema/migrations/20260813_p5_factory_planning.ddl
-- Does not change GET /v1/supplier/supply-lanes JSON (that remains topology warehouse utilization).

CREATE TABLE SupplyLanes (
  LaneId                 STRING(36)  NOT NULL,
  SupplierId             STRING(36)  NOT NULL,
  FactoryId              STRING(36)  NOT NULL,
  WarehouseId            STRING(36)  NOT NULL,
  TransitTimeHours       FLOAT64     NOT NULL DEFAULT (24),
  DampenedTransitHours   FLOAT64     NOT NULL DEFAULT (24),
  FreightCostMinor       INT64       NOT NULL DEFAULT (0),
  CarbonScoreKg          FLOAT64     NOT NULL DEFAULT (0),
  IsActive               BOOL        NOT NULL DEFAULT (true),
  Priority               INT64       NOT NULL DEFAULT (0),
  LastTransitUpdate      TIMESTAMP,
  CreatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt              TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, LaneId);

CREATE INDEX Idx_SupplyLanes_ByFactory ON SupplyLanes(SupplierId, FactoryId);
CREATE INDEX Idx_SupplyLanes_ByWarehouse ON SupplyLanes(SupplierId, WarehouseId);
CREATE UNIQUE INDEX Idx_SupplyLanes_Edge ON SupplyLanes(SupplierId, FactoryId, WarehouseId);

CREATE TABLE ReplenishmentLocks (
  LockKey     STRING(200) NOT NULL,
  AcquiredBy  STRING(36)  NOT NULL,
  SupplierId  STRING(36)  NOT NULL,
  Priority    FLOAT64     NOT NULL DEFAULT (0),
  AcquiredAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ExpiresAt   TIMESTAMP   NOT NULL,
) PRIMARY KEY (LockKey);

CREATE INDEX Idx_ReplenishmentLocks_ByExpiry ON ReplenishmentLocks(ExpiresAt);

CREATE TABLE NetworkOptimizationMode (
  SupplierId  STRING(36)  NOT NULL,
  Mode        STRING(30)  NOT NULL DEFAULT ('BALANCED'),
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedBy   STRING(36)  NOT NULL,
) PRIMARY KEY (SupplierId);

CREATE TABLE PullMatrixRuns (
  RunId              STRING(36)  NOT NULL,
  SupplierId         STRING(36)  NOT NULL,
  RunAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  TransfersGenerated INT64       NOT NULL DEFAULT (0),
  SKUsProcessed      INT64       NOT NULL DEFAULT (0),
  DurationMs         INT64       NOT NULL DEFAULT (0),
  Source             STRING(30)  NOT NULL DEFAULT ('CRON'),
  Notes              STRING(MAX),
) PRIMARY KEY (SupplierId, RunId);

CREATE INDEX Idx_PullMatrixRuns_ByTime ON PullMatrixRuns(SupplierId, RunAt DESC);

CREATE TABLE FactorySLAEvents (
  EventId                 STRING(36)  NOT NULL,
  TransferId              STRING(36)  NOT NULL,
  SupplierId              STRING(36)  NOT NULL,
  FactoryId               STRING(36)  NOT NULL,
  WarehouseId             STRING(36)  NOT NULL,
  EscalationLevel         STRING(20)  NOT NULL,
  PromisedAt              TIMESTAMP,
  ActualAt                TIMESTAMP,
  SLABreachMinutes        INT64       NOT NULL DEFAULT (0),
  ReplacementTransferId   STRING(36),
  CreatedAt               TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (EventId);

CREATE INDEX Idx_FactorySLA_ByTransfer ON FactorySLAEvents(TransferId);
CREATE INDEX Idx_FactorySLA_ByFactory ON FactorySLAEvents(SupplierId, FactoryId, CreatedAt DESC);

ALTER TABLE FactoryInternalTransfers ADD COLUMN Source STRING(30) NOT NULL DEFAULT ('MANUAL_EMERGENCY');

CREATE INDEX Idx_FactoryTransfers_BySupplierSourceState
  ON FactoryInternalTransfers(SupplierId, Source, State);
