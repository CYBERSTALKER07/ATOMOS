-- Wave C3.3: offline count version conflict protocol
CREATE TABLE RetailerStockLocationVersions (
  LocationId  STRING(36) NOT NULL,
  StockBin    STRING(16) NOT NULL,
  Version     INT64       NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LocationId, StockBin);

CREATE TABLE RetailerStockCountForceAudits (
  AuditId       STRING(36) NOT NULL,
  CountId       STRING(36) NOT NULL,
  RetailerId    STRING(36) NOT NULL,
  LocationId    STRING(36) NOT NULL,
  StockBin      STRING(16) NOT NULL,
  BaseVersion   INT64       NOT NULL,
  ServerVersion INT64       NOT NULL,
  ActorUserId   STRING(36) NOT NULL,
  ActorRole     STRING(32) NOT NULL,
  LinesJson     JSON        NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AuditId);
