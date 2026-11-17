-- ATOMOS P1 Wave 3B (M1.5–M1.8): Forecast v2, Credit Score History, Governed Agent Proposals, and ROI Ledger
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

ALTER TABLE RetailerPaymentTerms ADD COLUMN ConsentVerified BOOL NOT NULL DEFAULT (FALSE);

CREATE TABLE ForecastRuns (
  EntityId           STRING(64)  NOT NULL,
  RunId              STRING(64)  NOT NULL,
  SupplierId         STRING(64)  NOT NULL,
  LocationId         STRING(64),
  HorizonDays        INT64       NOT NULL DEFAULT (14),
  Status             STRING(24)  NOT NULL,
  ChampionModel      STRING(64)  NOT NULL,
  ChallengerModel    STRING(64)  NOT NULL,
  WinnerModel        STRING(64)  NOT NULL,
  ChampionWapeBps    INT64       NOT NULL DEFAULT (0),
  ChallengerWapeBps  INT64       NOT NULL DEFAULT (0),
  ChampionBiasBps    INT64       NOT NULL DEFAULT (0),
  ChallengerBiasBps  INT64       NOT NULL DEFAULT (0),
  WapeImprovementBps INT64       NOT NULL DEFAULT (0),
  SkuCount           INT64       NOT NULL DEFAULT (0),
  LocationCount      INT64       NOT NULL DEFAULT (0),
  RegressorsJson     STRING(MAX),
  FeatureSource      STRING(64),
  TriggeredBy        STRING(128) NOT NULL,
  FailureReason      STRING(MAX),
  IdempotencyKey     STRING(256),
  StartedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CompletedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, RunId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_ForecastRuns_Idempotency
  ON ForecastRuns(EntityId, IdempotencyKey);

CREATE INDEX Idx_ForecastRuns_ByEntityCreated
  ON ForecastRuns(EntityId, CreatedAt DESC);

CREATE INDEX Idx_ForecastRuns_BySupplierCreated
  ON ForecastRuns(SupplierId, CreatedAt DESC);

CREATE TABLE ForecastValues (
  EntityId         STRING(64)  NOT NULL,
  RunId            STRING(64)  NOT NULL,
  ForecastDate     DATE        NOT NULL,
  LocationId       STRING(64)  NOT NULL,
  LocationType     STRING(24)  NOT NULL DEFAULT ('WAREHOUSE'),
  Sku              STRING(64)  NOT NULL,
  ProductId        STRING(64),
  Model            STRING(64)  NOT NULL,
  IsWinner         BOOL        NOT NULL DEFAULT (false),
  DemandClass      STRING(24),
  P10Milli         INT64       NOT NULL DEFAULT (0),
  P50Milli         INT64       NOT NULL DEFAULT (0),
  P90Milli         INT64       NOT NULL DEFAULT (0),
  BaselineP50Milli INT64       NOT NULL DEFAULT (0),
  WapeBps          INT64       NOT NULL DEFAULT (0),
  BiasBps          INT64       NOT NULL DEFAULT (0),
  ConfidenceBps    INT64       NOT NULL DEFAULT (0),
  RegressorsJson   STRING(MAX),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, RunId, ForecastDate, LocationId, Sku, Model);

CREATE INDEX Idx_ForecastValues_ByEntitySkuLocationDate
  ON ForecastValues(EntityId, Sku, LocationId, ForecastDate DESC)
  STORING (IsWinner, P10Milli, P50Milli, P90Milli, WapeBps, ConfidenceBps);

CREATE INDEX Idx_ForecastValues_ByRunWinner
  ON ForecastValues(EntityId, RunId, IsWinner, ForecastDate);

CREATE TABLE ForecastExplanations (
  EntityId            STRING(64)  NOT NULL,
  ExplanationId       STRING(64)  NOT NULL,
  RunId               STRING(64),
  LocationId          STRING(64)  NOT NULL,
  Sku                 STRING(64)  NOT NULL,
  WindowStart         DATE,
  WindowEnd           DATE,
  Source              STRING(32)  NOT NULL,
  ModelName           STRING(64),
  SummaryText         STRING(MAX) NOT NULL,
  EvidenceRefsJson    STRING(MAX) NOT NULL,
  AdjustmentBps       INT64       NOT NULL DEFAULT (0),
  AdjustmentReason    STRING(MAX),
  Status              STRING(24)  NOT NULL,
  ProposedBy          STRING(128) NOT NULL,
  ApprovedBy          STRING(128),
  ApprovedAt          TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  ApprovalNote        STRING(MAX),
  ApprovalRequestId   STRING(64),
  BaselineP50Milli    INT64       NOT NULL DEFAULT (0),
  AdjustedP50Milli    INT64       NOT NULL DEFAULT (0),
  ActualQtyMilli      INT64,
  ForecastValueAddBps INT64,
  CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, ExplanationId);

CREATE INDEX Idx_ForecastExplanations_ByEntitySkuCreated
  ON ForecastExplanations(EntityId, Sku, LocationId, CreatedAt DESC);

CREATE INDEX Idx_ForecastExplanations_ByEntityStatus
  ON ForecastExplanations(EntityId, Status, CreatedAt DESC);

CREATE TABLE CreditScoreHistory (
  RetailerId            STRING(64)  NOT NULL,
  ScoreId               STRING(64)  NOT NULL,
  Scope                 STRING(24)  NOT NULL,
  SupplierId            STRING(64),
  ConsentVerified       BOOL        NOT NULL DEFAULT (false),
  ConsentScope          STRING(64),
  ModelVersion          STRING(64)  NOT NULL,
  ScorerSource          STRING(32)  NOT NULL,
  Score                 INT64       NOT NULL,
  PdBps                 INT64       NOT NULL DEFAULT (0),
  RiskTier              STRING(24)  NOT NULL,
  SuggestedLimitMinor   INT64       NOT NULL DEFAULT (0),
  CurrentLimitMinor     INT64       NOT NULL DEFAULT (0),
  ReasonCodesJson       STRING(MAX) NOT NULL,
  FactorsJson           STRING(MAX),
  FeaturesHash          STRING(64)  NOT NULL,
  DisputesIsolatedCount INT64       NOT NULL DEFAULT (0),
  WindowStart           TIMESTAMP   NOT NULL,
  WindowEnd             TIMESTAMP   NOT NULL,
  ComputedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CreatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, ScoreId);

CREATE INDEX Idx_CreditScoreHistory_ByRetailerComputed
  ON CreditScoreHistory(RetailerId, ComputedAt DESC)
  STORING (Scope, SupplierId, Score, PdBps, RiskTier, SuggestedLimitMinor, ModelVersion);

CREATE NULL_FILTERED INDEX Idx_CreditScoreHistory_BySupplierRetailer
  ON CreditScoreHistory(SupplierId, RetailerId, ComputedAt DESC);

CREATE TABLE Proposals (
  TenantId             STRING(64)  NOT NULL,
  ProposalId           STRING(64)  NOT NULL,
  AgentId              STRING(64)  NOT NULL,
  ProposalType         STRING(64)  NOT NULL,
  TargetObjectType     STRING(64)  NOT NULL,
  TargetObjectId       STRING(128) NOT NULL,
  Title                STRING(255) NOT NULL,
  Summary              STRING(MAX),
  Rationale            STRING(MAX) NOT NULL,
  PayloadJson          STRING(MAX) NOT NULL,
  PayloadHash          STRING(64)  NOT NULL,
  EvidenceRefsJson     STRING(MAX) NOT NULL,
  ConfidenceBps        INT64       NOT NULL DEFAULT (0),
  EstimatedImpactMinor INT64       NOT NULL DEFAULT (0),
  Currency             STRING(3)   NOT NULL DEFAULT ('UZS'),
  RoiCategory          STRING(64),
  Status               STRING(24)  NOT NULL,
  ApprovalRequestId    STRING(64),
  CreatedBy            STRING(128) NOT NULL,
  ActorType            STRING(24)  NOT NULL,
  DecidedBy            STRING(128),
  DecisionComment      STRING(MAX),
  ExecutedBy           STRING(128),
  ExecutionResultJson  STRING(MAX),
  ExecutionError       STRING(MAX),
  RoiEventId           STRING(64),
  IdempotencyKey       STRING(256),
  ExpiresAt            TIMESTAMP,
  DecidedAt            TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  ExecutedAt           TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, ProposalId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_Proposals_Idempotency
  ON Proposals(TenantId, IdempotencyKey);

CREATE INDEX Idx_Proposals_ByStatusAgent
  ON Proposals(TenantId, Status, AgentId, CreatedAt DESC);

CREATE INDEX Idx_Proposals_ByTarget
  ON Proposals(TenantId, TargetObjectType, TargetObjectId, Status);

CREATE NULL_FILTERED INDEX Idx_Proposals_ByApprovalRequest
  ON Proposals(TenantId, ApprovalRequestId);

CREATE TABLE RoiBaselines (
  TenantId               STRING(64)  NOT NULL,
  BaselineId             STRING(64)  NOT NULL,
  WindowDays             INT64       NOT NULL DEFAULT (90),
  BaselineArMinor        INT64       NOT NULL DEFAULT (0),
  BaselineBadDebtMinor   INT64       NOT NULL DEFAULT (0),
  BaselineExpiredMinor   INT64       NOT NULL DEFAULT (0),
  BaselineStockoutMinor  INT64       NOT NULL DEFAULT (0),
  BaselineLogisticsMinor INT64       NOT NULL DEFAULT (0),
  BaselineLabourHours    INT64       NOT NULL DEFAULT (0),
  SurveyHourlyWageMinor  INT64       NOT NULL DEFAULT (0),
  SurveyNotes            STRING(MAX),
  SurveyJson             STRING(MAX),
  Currency               STRING(3)   NOT NULL DEFAULT ('UZS'),
  CapturedBy             STRING(128) NOT NULL,
  PeriodStart            TIMESTAMP,
  PeriodEnd              TIMESTAMP,
  CreatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, BaselineId);

CREATE INDEX Idx_RoiBaselines_ByTenantCreated
  ON RoiBaselines(TenantId, CreatedAt DESC);

CREATE TABLE RoiEvents (
  TenantId        STRING(64)  NOT NULL,
  EventId         STRING(64)  NOT NULL,
  Category        STRING(48)  NOT NULL,
  SavingsMinor    INT64       NOT NULL,
  Currency        STRING(3)   NOT NULL DEFAULT ('UZS'),
  Methodology     STRING(MAX) NOT NULL,
  ConfidenceBps   INT64       NOT NULL DEFAULT (0),
  IsEstimate      BOOL        NOT NULL DEFAULT (false),
  HoldoutControl  BOOL        NOT NULL DEFAULT (false),
  ProposalId      STRING(64),
  EvidenceRefType STRING(64),
  EvidenceRefId   STRING(128),
  Notes           STRING(MAX),
  RecordedBy      STRING(128) NOT NULL,
  IdempotencyKey  STRING(256),
  OccurredAt      TIMESTAMP   NOT NULL,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, EventId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_RoiEvents_Idempotency
  ON RoiEvents(TenantId, IdempotencyKey);

CREATE INDEX Idx_RoiEvents_ByCategoryOccurred
  ON RoiEvents(TenantId, Category, OccurredAt DESC)
  STORING (SavingsMinor, Currency, ConfidenceBps, IsEstimate, HoldoutControl);

CREATE NULL_FILTERED INDEX Idx_RoiEvents_ByProposal
  ON RoiEvents(TenantId, ProposalId);
