-- Gate-3 Wave 2A: partner bulk export jobs + SFTP destination config.
CREATE TABLE PartnerExportJobs (
  JobId       STRING(36)   NOT NULL,
  TenantType  STRING(16)   NOT NULL,
  TenantId    STRING(36)   NOT NULL,
  Resource    STRING(32)   NOT NULL,
  Format      STRING(16)   NOT NULL,
  Status      STRING(16)   NOT NULL,
  FromDate    DATE,
  ToDate      DATE,
  ObjectPath  STRING(512),
  RowCount    INT64        NOT NULL DEFAULT (0),
  Error       STRING(1024),
  SftpStatus  STRING(16),
  CreatedAt   TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  FinishedAt  TIMESTAMP,
) PRIMARY KEY (JobId);

CREATE INDEX Idx_PartnerExportJobs_ByTenantCreated
  ON PartnerExportJobs (TenantType, TenantId, CreatedAt DESC);

CREATE INDEX Idx_PartnerExportJobs_ByStatus
  ON PartnerExportJobs (Status, CreatedAt);

CREATE TABLE PartnerSftpConfigs (
  TenantType STRING(16)   NOT NULL,
  TenantId   STRING(36)   NOT NULL,
  Host       STRING(255)  NOT NULL,
  Port       INT64        NOT NULL DEFAULT (22),
  Username   STRING(128)  NOT NULL,
  SecretRef  STRING(256)  NOT NULL,
  RemoteDir  STRING(512)  NOT NULL DEFAULT ('/'),
  IsActive   BOOL         NOT NULL DEFAULT (TRUE),
  UpdatedAt  TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId);
