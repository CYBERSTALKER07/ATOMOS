-- Phase 3: platform tenant lifecycle + feature-flag overrides + admin audit.
CREATE TABLE IF NOT EXISTS PlatformTenants (
  TenantType   STRING(32) NOT NULL,
  TenantId     STRING(64) NOT NULL,
  Status       STRING(32) NOT NULL,
  DisplayName  STRING(255),
  KybNotes     STRING(MAX),
  CreatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  ApprovedAt   TIMESTAMP,
  SuspendedAt  TIMESTAMP,
  OffboardedAt TIMESTAMP,
) PRIMARY KEY (TenantType, TenantId);

CREATE INDEX IF NOT EXISTS Idx_PlatformTenants_ByStatus
  ON PlatformTenants(Status, UpdatedAt DESC);

CREATE TABLE IF NOT EXISTS PlatformAdminAudit (
  AuditId      STRING(36) NOT NULL,
  ActorSubject STRING(128) NOT NULL,
  Action       STRING(64) NOT NULL,
  TenantType   STRING(32),
  TenantId     STRING(64),
  DetailJson   STRING(MAX),
  CreatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AuditId);

CREATE INDEX IF NOT EXISTS Idx_PlatformAdminAudit_ByCreated
  ON PlatformAdminAudit(CreatedAt DESC);

CREATE TABLE IF NOT EXISTS FeatureFlagOverrides (
  FlagKey      STRING(128) NOT NULL,
  TenantType   STRING(32) NOT NULL,
  TenantId     STRING(64) NOT NULL,
  Enabled      BOOL NOT NULL,
  UpdatedBy    STRING(128) NOT NULL,
  UpdatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  Reason       STRING(512),
) PRIMARY KEY (FlagKey, TenantType, TenantId);

CREATE INDEX IF NOT EXISTS Idx_FeatureFlagOverrides_ByTenant
  ON FeatureFlagOverrides(TenantType, TenantId);
