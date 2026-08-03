-- Durable audit for auto-order worker ticks (multi-pod / restart safe).
-- Complements RetailerAutoOrderBucket (idempotency) with full run history.
CREATE TABLE RetailerAutoOrderRuns (
  RunId            STRING(64)  NOT NULL,
  RetailerId       STRING(64)  NOT NULL,
  Mode             STRING(16)  NOT NULL,
  Status           STRING(32)  NOT NULL,
  Message          STRING(512),
  ScheduleBucket   STRING(10)  NOT NULL,
  CandidateSource  STRING(32),
  SuggestionsSeen  INT64       NOT NULL DEFAULT (0),
  DraftLines       INT64       NOT NULL DEFAULT (0),
  PlacedLines      INT64       NOT NULL DEFAULT (0),
  SkippedJson      BYTES(MAX),
  PlacedOrdersJson BYTES(MAX),
  StartedAt        TIMESTAMP   NOT NULL,
  FinishedAt       TIMESTAMP,
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RunId);

CREATE INDEX Idx_RetailerAutoOrderRuns_ByRetailerCreated
  ON RetailerAutoOrderRuns (RetailerId, CreatedAt DESC);
