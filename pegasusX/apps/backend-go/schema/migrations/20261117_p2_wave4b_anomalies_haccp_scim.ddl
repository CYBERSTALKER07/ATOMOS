-- ATOMOS P2 Wave 4B: Anomaly Catalogue, What-If E2E Simulator, L2 Autonomy Proposals, Custom Roles & SCIM v2
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

ALTER TABLE Proposals ADD COLUMN AutonomyLevel STRING(8);

CREATE TABLE AnomalyCases (
  TenantId       STRING(64)  NOT NULL,
  CaseId         STRING(64)  NOT NULL,
  RuleCode       STRING(64)  NOT NULL,
  Category       STRING(32)  NOT NULL,
  Severity       STRING(16)  NOT NULL,
  Status         STRING(24)  NOT NULL,
  ObjectType     STRING(64)  NOT NULL,
  ObjectId       STRING(128) NOT NULL,
  ExposureMinor  INT64       NOT NULL DEFAULT (0),
  ScoreBps       INT64       NOT NULL DEFAULT (0),
  EvidenceJson   STRING(MAX),
  PlaybookAction STRING(64),
  ProposalId     STRING(64),
  AssignedTo     STRING(128),
  ResolutionNote STRING(MAX),
  ResolvedBy     STRING(128),
  ResolvedAt     TIMESTAMP,
  DetectedAt     TIMESTAMP   NOT NULL,
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, CaseId);

CREATE INDEX Idx_AnomalyCases_ByStatusSeverity
  ON AnomalyCases(TenantId, Status, Severity, DetectedAt DESC);

CREATE TABLE WhatIfSimulations (
  TenantId                 STRING(64)  NOT NULL,
  SimulationId             STRING(64)  NOT NULL,
  ScenarioId               STRING(64),
  Name                     STRING(160) NOT NULL,
  EntityId                 STRING(64),
  HorizonDays              INT64       NOT NULL DEFAULT (30),
  ShocksJson               STRING(MAX) NOT NULL,
  ForecastDeltaMinor       INT64       NOT NULL DEFAULT (0),
  ReplenishmentCostMinor   INT64       NOT NULL DEFAULT (0),
  DispatchCostDeltaMinor   INT64       NOT NULL DEFAULT (0),
  CreditExposureDeltaMinor INT64       NOT NULL DEFAULT (0),
  NetPnLDeltaMinor         INT64       NOT NULL DEFAULT (0),
  BreakdownJson            STRING(MAX),
  CommittedProposalIdsJson STRING(MAX),
  Status                   STRING(24)  NOT NULL,
  CreatedBy                STRING(128) NOT NULL,
  CreatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, SimulationId);

CREATE INDEX Idx_WhatIfSimulations_ByCreated
  ON WhatIfSimulations(TenantId, CreatedAt DESC);

CREATE TABLE CustomRoles (
  TenantId        STRING(64)  NOT NULL,
  RoleName        STRING(64)  NOT NULL,
  Description     STRING(256),
  PermissionsJson STRING(MAX) NOT NULL,
  ReadOnly        BOOL        NOT NULL DEFAULT (false),
  CreatedBy       STRING(128) NOT NULL,
  Status          STRING(24)  NOT NULL DEFAULT ('ACTIVE'),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RoleName);

CREATE TABLE ScimIdentities (
  TenantId     STRING(64)  NOT NULL,
  ResourceType STRING(16)  NOT NULL,
  ResourceId   STRING(128) NOT NULL,
  ExternalId   STRING(128),
  UserName     STRING(160),
  DisplayName  STRING(160),
  Email        STRING(160),
  Active       BOOL        NOT NULL DEFAULT (true),
  RolesJson    STRING(MAX),
  MembersJson  STRING(MAX),
  MetaJson     STRING(MAX),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, ResourceType, ResourceId);

CREATE INDEX Idx_ScimIdentities_ByUserName
  ON ScimIdentities(TenantId, ResourceType, UserName);
