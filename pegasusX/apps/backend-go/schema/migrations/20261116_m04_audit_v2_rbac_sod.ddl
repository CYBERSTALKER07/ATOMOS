-- ATOMOS M0.4: Immutable Audit v2 Hash-Chain, WORM Anchors, Legal Holds & RBAC SoD
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

CREATE TABLE AuditEvents (
  TenantId         STRING(64)  NOT NULL,
  DayBucket        STRING(10)  NOT NULL,
  EventId          STRING(64)  NOT NULL,
  SeqNum           INT64       NOT NULL,
  ActorId          STRING(128) NOT NULL,
  ActorType        STRING(24)  NOT NULL,
  Role             STRING(64),
  Action           STRING(128) NOT NULL,
  ObjectType       STRING(64),
  ObjectId         STRING(128),
  BeforeJson       STRING(MAX),
  AfterJson        STRING(MAX),
  DiffGcsUri       STRING(512),
  ReasonCode       STRING(64),
  ApprovalId       STRING(64),
  TraceId          STRING(64),
  Ip               STRING(64),
  DeviceId         STRING(128),
  PrevHash         STRING(64)  NOT NULL,
  EntryHash        STRING(64)  NOT NULL,
  LegalHold        BOOL        NOT NULL DEFAULT (false),
  LegalHoldReason  STRING(MAX),
  RetentionUntil   TIMESTAMP,
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, DayBucket, EventId);

CREATE INDEX Idx_AuditEvents_ByObject
  ON AuditEvents(TenantId, ObjectType, ObjectId, CreatedAt DESC);

CREATE INDEX Idx_AuditEvents_ByActor
  ON AuditEvents(TenantId, ActorId, CreatedAt DESC);

CREATE INDEX Idx_AuditEvents_ByCreated
  ON AuditEvents(TenantId, CreatedAt DESC);

CREATE INDEX Idx_AuditEvents_ByDaySeq
  ON AuditEvents(TenantId, DayBucket, SeqNum DESC)
  STORING (EntryHash, PrevHash);

CREATE TABLE AuditChainAnchors (
  TenantId     STRING(64)  NOT NULL,
  DayBucket    STRING(10)  NOT NULL,
  HeadEventId  STRING(64)  NOT NULL,
  EventCount   INT64       NOT NULL,
  RootHash     STRING(64)  NOT NULL,
  WormUri      STRING(512),
  SignerKeyId  STRING(128),
  AnchoredBy   STRING(128) NOT NULL,
  AnchoredAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, DayBucket);

CREATE TABLE AuditLegalHolds (
  TenantId    STRING(64)  NOT NULL,
  HoldId      STRING(64)  NOT NULL,
  ObjectType  STRING(64)  NOT NULL,
  ObjectId    STRING(128) NOT NULL,
  Reason      STRING(MAX) NOT NULL,
  Status      STRING(24)  NOT NULL,
  CreatedBy   STRING(128) NOT NULL,
  ExpiresAt   TIMESTAMP,
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, HoldId);

CREATE TABLE RolePermissions (
  TenantId       STRING(64)  NOT NULL,
  RoleName       STRING(64)  NOT NULL,
  PermissionKey  STRING(64)  NOT NULL,
  GrantedBy      STRING(128),
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RoleName, PermissionKey);

CREATE TABLE SupplierUserRoles (
  TenantId    STRING(64)  NOT NULL,
  UserId      STRING(128) NOT NULL,
  RoleName    STRING(64)  NOT NULL,
  AssignedBy  STRING(128),
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, UserId, RoleName);

CREATE TABLE SodPolicies (
  TenantId           STRING(64)  NOT NULL,
  PolicyId           STRING(64)  NOT NULL,
  Name               STRING(128) NOT NULL,
  PermissionA        STRING(64)  NOT NULL,
  PermissionB        STRING(64)  NOT NULL,
  EnforceSameObject  BOOL        NOT NULL DEFAULT (true),
  Enabled            BOOL        NOT NULL DEFAULT (true),
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, PolicyId);

CREATE TABLE SodViolations (
  TenantId     STRING(64)  NOT NULL,
  ViolationId  STRING(64)  NOT NULL,
  ActorId      STRING(128) NOT NULL,
  Action       STRING(128) NOT NULL,
  ObjectType   STRING(64),
  ObjectId     STRING(128),
  MakerId      STRING(128),
  RuleCode     STRING(64)  NOT NULL,
  RoutePath    STRING(256),
  TraceId      STRING(64),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, ViolationId);

CREATE INDEX Idx_SodViolations_ByTenantCreated
  ON SodViolations(TenantId, CreatedAt DESC);
