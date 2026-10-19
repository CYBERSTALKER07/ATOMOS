-- §8.7 Gate 4 PR-6: cold-chain temperature readings.

CREATE TABLE TemperatureReadings (
  ReadingId    STRING(36)    NOT NULL,
  ManifestId   STRING(36)    NOT NULL,
  SensorId     STRING(64),
  RecordedAt   TIMESTAMP     NOT NULL,
  TempC        FLOAT64       NOT NULL,
  Lat          FLOAT64,
  Lng          FLOAT64,
  CreatedAt    TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ReadingId);

CREATE INDEX Idx_TemperatureReadings_ByManifest
  ON TemperatureReadings(ManifestId, RecordedAt DESC);
