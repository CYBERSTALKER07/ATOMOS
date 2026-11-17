-- ATOMOS P1 Wave 3C (M1.9–M1.10): External Connectors, Shadow Compare, Export Packs & Time-Boxed Auditor Access
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

ALTER TABLE SupplierUserRoles ADD COLUMN ExpiresAt TIMESTAMP;

CREATE TABLE ExternalCredentials (
  TenantId     STRING(64)  NOT NULL,
  CredentialId STRING(64)  NOT NULL,
  Provider     STRING(32)  NOT NULL,
  AccountLabel STRING(128) NOT NULL DEFAULT ('default'),
  AuthType     STRING(32)  NOT NULL,
  SecretRef    STRING(512) NOT NULL,
  BaseUrl      STRING(512),
  MetadataJson STRING(MAX),
  Status       STRING(24)  NOT NULL DEFAULT ('ACTIVE'),
  CreatedBy    STRING(128) NOT NULL,
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, CredentialId);

CREATE INDEX Idx_ExternalCredentials_ByProvider
  ON ExternalCredentials(TenantId, Provider, Status);

CREATE TABLE ConnectorSyncRuns (
  TenantId            STRING(64)  NOT NULL,
  RunId               STRING(64)  NOT NULL,
  Provider            STRING(32)  NOT NULL,
  AccountLabel        STRING(128) NOT NULL DEFAULT ('default'),
  Direction           STRING(32)  NOT NULL,
  StreamName          STRING(64)  NOT NULL,
  Status              STRING(24)  NOT NULL,
  RecordsProcessed    INT64       NOT NULL DEFAULT (0),
  MismatchCount       INT64       NOT NULL DEFAULT (0),
  MismatchSummaryJson STRING(MAX),
  ErrorMessage        STRING(MAX),
  IdempotencyKey      STRING(256),
  StartedAt           TIMESTAMP   NOT NULL,
  CompletedAt         TIMESTAMP,
  CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RunId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_ConnectorSyncRuns_Idempotency
  ON ConnectorSyncRuns(TenantId, IdempotencyKey);

CREATE INDEX Idx_ConnectorSyncRuns_ByProviderStarted
  ON ConnectorSyncRuns(TenantId, Provider, StartedAt DESC);

CREATE TABLE ConnectorEntityMaps (
  TenantId     STRING(64)  NOT NULL,
  Provider     STRING(32)  NOT NULL,
  ObjectType   STRING(64)  NOT NULL,
  ExternalId   STRING(128) NOT NULL,
  InternalId   STRING(128) NOT NULL,
  ContentHash  STRING(64),
  LastRunId    STRING(64),
  LastSyncedAt TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, Provider, ObjectType, ExternalId);

CREATE INDEX Idx_ConnectorEntityMaps_ByInternal
  ON ConnectorEntityMaps(TenantId, Provider, ObjectType, InternalId);

CREATE TABLE ExportPacks (
  TenantId      STRING(64)  NOT NULL,
  PackId        STRING(64)  NOT NULL,
  PackType      STRING(24)  NOT NULL,
  EntityId      STRING(64),
  PeriodId      STRING(32),
  FromDate      TIMESTAMP   NOT NULL,
  ToDate        TIMESTAMP   NOT NULL,
  Format        STRING(16)  NOT NULL DEFAULT ('zip'),
  Status        STRING(24)  NOT NULL,
  ManifestJson  STRING(MAX),
  ArtifactsJson STRING(MAX),
  DownloadToken STRING(128),
  WatermarkText STRING(256),
  RowCount      INT64       NOT NULL DEFAULT (0),
  Sha256Hex     STRING(64),
  CreatedBy     STRING(128) NOT NULL,
  ExpiresAt     TIMESTAMP   NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, PackId);

CREATE INDEX Idx_ExportPacks_ByCreated
  ON ExportPacks(TenantId, CreatedAt DESC);
