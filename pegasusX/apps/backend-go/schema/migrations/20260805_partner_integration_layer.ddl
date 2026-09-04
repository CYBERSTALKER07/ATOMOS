-- Partner Integration Layer (Gate 3 / PLATFORM_AUDIT §8.9)

CREATE TABLE PartnerApiKeys (
  KeyId STRING(36) NOT NULL,
  TenantType STRING(16) NOT NULL,
  TenantId STRING(36) NOT NULL,
  KeyPrefix STRING(16) NOT NULL,
  KeyHash STRING(128) NOT NULL,
  Scopes ARRAY<STRING(64)>,
  RateLimitClass STRING(32) NOT NULL DEFAULT ('partner_default'),
  Status STRING(16) NOT NULL DEFAULT ('ACTIVE'),
  ExpiresAt TIMESTAMP,
  CreatedBy STRING(64),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  LastUsedAt TIMESTAMP,
) PRIMARY KEY (KeyId);

CREATE UNIQUE INDEX Idx_PartnerApiKeys_ByPrefix ON PartnerApiKeys(KeyPrefix);
CREATE INDEX Idx_PartnerApiKeys_ByTenant ON PartnerApiKeys(TenantType, TenantId, Status);

CREATE TABLE WebhookSubscriptions (
  SubscriptionId STRING(36) NOT NULL,
  TenantType STRING(16) NOT NULL,
  TenantId STRING(36) NOT NULL,
  Url STRING(2048) NOT NULL,
  SigningSecret STRING(128) NOT NULL,
  EventTypes ARRAY<STRING(64)>,
  IsActive BOOL NOT NULL DEFAULT (TRUE),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SubscriptionId);

CREATE INDEX Idx_WebhookSubscriptions_ByTenant ON WebhookSubscriptions(TenantType, TenantId, IsActive);

CREATE TABLE WebhookDeliveryAttempts (
  AttemptId STRING(36) NOT NULL,
  SubscriptionId STRING(36) NOT NULL,
  EventId STRING(64) NOT NULL,
  EventType STRING(64) NOT NULL,
  PayloadJson JSON,
  Status STRING(16) NOT NULL,
  HttpCode INT64,
  NextRetryAt TIMESTAMP,
  AttemptCount INT64 NOT NULL DEFAULT (0),
  LastError STRING(1024),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AttemptId);

CREATE UNIQUE INDEX Idx_WebhookDelivery_BySubEvent ON WebhookDeliveryAttempts(SubscriptionId, EventId);
CREATE INDEX Idx_WebhookDelivery_ByStatusRetry ON WebhookDeliveryAttempts(Status, NextRetryAt);
