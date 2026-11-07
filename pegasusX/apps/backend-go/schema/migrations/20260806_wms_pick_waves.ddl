-- §8.7 Wave 1B: minimal pick waves + ready-to-seal gate.

ALTER TABLE SupplierTruckManifests ADD COLUMN PickWaveId STRING(36);

CREATE TABLE PickWaves (
  WaveId       STRING(36)    NOT NULL,
  WarehouseId  STRING(36)    NOT NULL,
  SupplierId   STRING(36)    NOT NULL,
  ManifestId   STRING(36)    NOT NULL,
  Strategy     STRING(24)    NOT NULL DEFAULT ('MANIFEST'),
  Status       STRING(24)    NOT NULL DEFAULT ('OPEN'),
  CreatedAt    TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ReleasedAt   TIMESTAMP,
  ReadyAt      TIMESTAMP,
  UpdatedAt    TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WaveId);

CREATE UNIQUE INDEX UQ_PickWaves_ByManifest
  ON PickWaves(ManifestId);

CREATE INDEX Idx_PickWaves_ByWarehouseStatus
  ON PickWaves(WarehouseId, Status, CreatedAt DESC);

CREATE TABLE PickTasks (
  WaveId             STRING(36)    NOT NULL,
  TaskId             STRING(36)    NOT NULL,
  OrderId            STRING(36)    NOT NULL,
  ProductId          STRING(36)    NOT NULL,
  LotId              STRING(36)    NOT NULL,
  LocationId         STRING(36)    NOT NULL,
  QuantityRequested  INT64         NOT NULL,
  QuantityPicked     INT64         NOT NULL DEFAULT (0),
  PickerId           STRING(36),
  Status             STRING(24)    NOT NULL DEFAULT ('PENDING'),
  PickSequence       INT64         NOT NULL DEFAULT (0),
  CreatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WaveId, TaskId),
  INTERLEAVE IN PARENT PickWaves ON DELETE CASCADE;

CREATE INDEX Idx_PickTasks_ByWaveStatus
  ON PickTasks(WaveId, Status, PickSequence);
