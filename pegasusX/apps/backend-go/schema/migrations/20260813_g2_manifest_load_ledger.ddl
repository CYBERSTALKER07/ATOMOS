-- G2.B: durable payload line-level load ledger (required vs scanned qty).
-- Apply: go run ./cmd/apply-migration --ddl schema/migrations/20260813_g2_manifest_load_ledger.ddl

CREATE TABLE ManifestLoadLines (
  ManifestId   STRING(36)  NOT NULL,
  OrderId      STRING(36)  NOT NULL,
  LineItemId   STRING(64)  NOT NULL,
  SkuId        STRING(64)  NOT NULL,
  RequiredQty  INT64       NOT NULL,
  ScannedQty   INT64       NOT NULL,
  Status       STRING(32)  NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId, OrderId, LineItemId);

CREATE INDEX Idx_ManifestLoadLines_ByManifestStatus
  ON ManifestLoadLines(ManifestId, Status);
