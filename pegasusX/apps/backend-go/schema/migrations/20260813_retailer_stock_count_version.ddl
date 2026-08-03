-- Wave C3.3: offline stock count version + force audit.
-- base_version on commit must match server Version or force (MANAGER/OWNER).
-- Flag: OFFLINE_COUNT_ENABLED (default off → commit path 404; legacy /stock/counts still works).

CREATE TABLE RetailerStockLocationVersions (
  RetailerId  STRING(36)  NOT NULL,
  LocationId  STRING(36)  NOT NULL,
  StockBin    STRING(16)  NOT NULL,
  Version     INT64       NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocationId, StockBin);

CREATE TABLE RetailerStockCountForceAudit (
  AuditId       STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  LocationId    STRING(36)  NOT NULL,
  StockBin      STRING(16)  NOT NULL,
  CountId       STRING(36)  NOT NULL,
  ActorUserId   STRING(36)  NOT NULL,
  BaseVersion   INT64       NOT NULL,
  ServerVersion INT64       NOT NULL,
  Reason        STRING(MAX),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (AuditId);

CREATE INDEX Idx_RetailerStockCountForceAudit_ByRetailer
  ON RetailerStockCountForceAudit (RetailerId, CreatedAt DESC);
