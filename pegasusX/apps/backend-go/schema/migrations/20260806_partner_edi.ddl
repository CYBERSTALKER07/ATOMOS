-- Gate-3 Wave 2B: EDI-lite documents + SFTP dir / enable flags on PartnerSftpConfigs.

ALTER TABLE PartnerSftpConfigs ADD COLUMN InboundDir STRING(512);
ALTER TABLE PartnerSftpConfigs ADD COLUMN OutboundDir STRING(512);
ALTER TABLE PartnerSftpConfigs ADD COLUMN ArchiveDir STRING(512);
ALTER TABLE PartnerSftpConfigs ADD COLUMN EdiEnabled BOOL;

CREATE TABLE PartnerEdiDocuments (
  DocumentId    STRING(36)   NOT NULL,
  TenantType    STRING(16)   NOT NULL,
  TenantId      STRING(36)   NOT NULL,
  Direction     STRING(8)    NOT NULL,
  DocType       STRING(16)   NOT NULL,
  ExternalDocId STRING(128)  NOT NULL,
  OrderId       STRING(36),
  Status        STRING(16)   NOT NULL,
  ObjectPath    STRING(512),
  RemoteName    STRING(255),
  Error         STRING(1024),
  PayloadHash   STRING(64),
  CreatedAt     TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  FinishedAt    TIMESTAMP,
) PRIMARY KEY (DocumentId);

CREATE UNIQUE INDEX Idx_PartnerEdiDocuments_Idempotency
  ON PartnerEdiDocuments (TenantType, TenantId, Direction, DocType, ExternalDocId);

CREATE INDEX Idx_PartnerEdiDocuments_ByTenantCreated
  ON PartnerEdiDocuments (TenantType, TenantId, CreatedAt DESC);

CREATE INDEX Idx_PartnerEdiDocuments_ByStatusDir
  ON PartnerEdiDocuments (Status, Direction, CreatedAt);
