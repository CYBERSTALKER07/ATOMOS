CREATE TABLE ControlTowerPlaybooks (
  PlaybookId        STRING(36) NOT NULL,
  SupplierId        STRING(36),
  Name              STRING(128) NOT NULL,
  Description       STRING(MAX),
  IsActive          BOOL NOT NULL,
  Priority          INT64 NOT NULL,
  MatchRulesJson    JSON NOT NULL,
  ActionsJson       JSON NOT NULL,
  AutoExecute       BOOL NOT NULL,
  CreatedAt         TIMESTAMP NOT NULL,
  UpdatedAt         TIMESTAMP NOT NULL,
  CreatedBy         STRING(64) NOT NULL,
) PRIMARY KEY (PlaybookId);

CREATE INDEX ControlTowerPlaybooks_BySupplierActive
ON ControlTowerPlaybooks(SupplierId, IsActive, Priority DESC);

CREATE TABLE ControlTowerPlaybookRuns (
  RunId             STRING(36) NOT NULL,
  PlaybookId        STRING(36) NOT NULL,
  ExceptionId       STRING(36) NOT NULL,
  SupplierId        STRING(36) NOT NULL,
  Mode              STRING(16) NOT NULL,
  Status            STRING(16) NOT NULL,
  ActionsResultJson JSON,
  CreatedAt         TIMESTAMP NOT NULL,
  ExecutedAt        TIMESTAMP,
  ExecutedBy        STRING(64),
) PRIMARY KEY (RunId);

CREATE INDEX ControlTowerPlaybookRuns_ByException
ON ControlTowerPlaybookRuns(ExceptionId, CreatedAt DESC);

CREATE INDEX ControlTowerPlaybookRuns_ByStatus
ON ControlTowerPlaybookRuns(Status, CreatedAt DESC);
