-- ATOMOS M0.5: BigQuery & Analytical Sink Checkpoints
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

CREATE TABLE BigQuerySinkCheckpoints (
  SinkId        STRING(64)  NOT NULL,
  Topic         STRING(128) NOT NULL,
  PartitionNum  INT64       NOT NULL,
  OffsetVal     INT64       NOT NULL,
  LastEventId   STRING(64),
  RowsExported  INT64       NOT NULL DEFAULT (0),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (SinkId, Topic, PartitionNum);
