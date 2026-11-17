-- ATOMOS M0.3: Four-Way Financial Reconciliation Runs & Breaks
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

CREATE TABLE ReconciliationRuns (
  EntityId       STRING(64)  NOT NULL,
  RunId          STRING(128) NOT NULL,
  PeriodId       STRING(16)  NOT NULL,
  Mode           STRING(24)  NOT NULL,
  Status         STRING(24)  NOT NULL,
  ChecksJson     STRING(MAX),
  SummaryJson    STRING(MAX),
  BreakCount     INT64       NOT NULL DEFAULT (0),
  ExposureMinor  INT64       NOT NULL DEFAULT (0),
  TriggeredBy    STRING(128) NOT NULL,
  StartedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CompletedAt    TIMESTAMP
) PRIMARY KEY (EntityId, RunId);

CREATE INDEX Idx_ReconciliationRuns_ByEntityStarted
  ON ReconciliationRuns(EntityId, StartedAt DESC);

CREATE TABLE ReconciliationBreaks (
  EntityId                 STRING(64)  NOT NULL,
  BreakId                  STRING(128) NOT NULL,
  RunId                    STRING(128) NOT NULL,
  Dimension                STRING(32)  NOT NULL,
  ObjectType               STRING(64)  NOT NULL,
  ObjectId                 STRING(128) NOT NULL,
  ExpectedMinor            INT64       NOT NULL DEFAULT (0),
  ActualMinor              INT64       NOT NULL DEFAULT (0),
  DeltaMinor               INT64       NOT NULL DEFAULT (0),
  Currency                 STRING(3)   NOT NULL,
  Severity                 STRING(16)  NOT NULL,
  Status                   STRING(24)  NOT NULL,
  OwnerId                  STRING(128),
  SlaDueAt                 TIMESTAMP,
  ResolutionNote           STRING(MAX),
  ControlTowerExceptionId  STRING(64),
  CreatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, BreakId);

CREATE INDEX Idx_ReconciliationBreaks_ByEntityStatus
  ON ReconciliationBreaks(EntityId, Status, DeltaMinor);

CREATE INDEX Idx_ReconciliationBreaks_ByRun
  ON ReconciliationBreaks(EntityId, RunId);
