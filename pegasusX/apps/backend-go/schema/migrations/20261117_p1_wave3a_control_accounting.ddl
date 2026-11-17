-- ATOMOS P1 Wave 3A (M1.1–M1.4): Period Close, CEL Approvals, Inventory Valuation, and ESF Documents
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

ALTER TABLE JournalEntries ADD COLUMN OriginalPeriodId STRING(16);

CREATE TABLE Periods (
  EntityId      STRING(64)  NOT NULL,
  PeriodId      STRING(16)  NOT NULL,
  Status        STRING(24)  NOT NULL,
  Timezone      STRING(64)  NOT NULL DEFAULT ('Asia/Tashkent'),
  SoftClosedBy  STRING(128),
  SoftClosedAt  TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  ClosedBy      STRING(128),
  ClosedAt      TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  ReopenedBy    STRING(128),
  ReopenedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  ReopenReason  STRING(MAX),
  SnapshotUri   STRING(512),
  SnapshotHash  STRING(64),
  ChecklistJson STRING(MAX),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, PeriodId);

CREATE INDEX Idx_Periods_ByEntityStatus
  ON Periods(EntityId, Status, PeriodId DESC);

CREATE TABLE ApprovalPolicies (
  TenantId      STRING(64)   NOT NULL,
  PolicyId      STRING(64)   NOT NULL,
  Name          STRING(128)  NOT NULL,
  ObjectType    STRING(64)   NOT NULL,
  Action        STRING(128)  NOT NULL,
  ConditionExpr STRING(1024),
  StepsJson     STRING(MAX)  NOT NULL,
  Version       INT64        NOT NULL DEFAULT (1),
  Enabled       BOOL         NOT NULL DEFAULT (true),
  CreatedBy     STRING(128)  NOT NULL,
  CreatedAt     TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, PolicyId);

CREATE INDEX Idx_ApprovalPolicies_ByTenantObjectAction
  ON ApprovalPolicies(TenantId, ObjectType, Action, Enabled);

CREATE TABLE ApprovalRequests (
  TenantId          STRING(64)  NOT NULL,
  RequestId         STRING(64)  NOT NULL,
  PolicyId          STRING(64),
  PolicyVersion     INT64       NOT NULL DEFAULT (1),
  ObjectType        STRING(64)  NOT NULL,
  ObjectId          STRING(128) NOT NULL,
  Action            STRING(128) NOT NULL,
  AmountMinor       INT64       NOT NULL DEFAULT (0),
  RequestedBy       STRING(128) NOT NULL,
  ActorType         STRING(24)  NOT NULL,
  PayloadJson       STRING(MAX) NOT NULL,
  PayloadHash       STRING(64)  NOT NULL,
  Status            STRING(24)  NOT NULL,
  CurrentStep       INT64       NOT NULL DEFAULT (0),
  RequiredApprovers INT64       NOT NULL DEFAULT (1),
  BreakGlass        BOOL        NOT NULL DEFAULT (false),
  BreakGlassReason  STRING(MAX),
  ExpiresAt         TIMESTAMP,
  IdempotencyKey    STRING(256),
  ResolvedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RequestId);

CREATE INDEX Idx_ApprovalRequests_ByTenantStatus
  ON ApprovalRequests(TenantId, Status, CreatedAt DESC);

CREATE INDEX Idx_ApprovalRequests_ByObject
  ON ApprovalRequests(TenantId, ObjectType, ObjectId, CreatedAt DESC);

CREATE UNIQUE NULL_FILTERED INDEX UQ_ApprovalRequests_Idempotency
  ON ApprovalRequests(TenantId, IdempotencyKey);

CREATE TABLE ApprovalDecisions (
  TenantId      STRING(64)  NOT NULL,
  RequestId     STRING(64)  NOT NULL,
  DecisionId    STRING(64)  NOT NULL,
  StepIndex     INT64       NOT NULL DEFAULT (0),
  ApproverId    STRING(128) NOT NULL,
  DelegatedFrom STRING(128),
  Decision      STRING(24)  NOT NULL,
  Comment       STRING(MAX),
  MfaVerified   BOOL        NOT NULL DEFAULT (false),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, RequestId, DecisionId);

CREATE INDEX Idx_ApprovalDecisions_ByApprover
  ON ApprovalDecisions(TenantId, ApproverId, CreatedAt DESC);

CREATE TABLE ApprovalDelegations (
  TenantId     STRING(64)  NOT NULL,
  DelegationId STRING(64)  NOT NULL,
  DelegatorId  STRING(128) NOT NULL,
  DelegateId   STRING(128) NOT NULL,
  RoleScope    STRING(64),
  StartsAt     TIMESTAMP   NOT NULL,
  EndsAt       TIMESTAMP   NOT NULL,
  Reason       STRING(MAX),
  Enabled      BOOL        NOT NULL DEFAULT (true),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (TenantId, DelegationId);

CREATE INDEX Idx_ApprovalDelegations_ByDelegate
  ON ApprovalDelegations(TenantId, DelegateId, Enabled, EndsAt DESC);

CREATE TABLE InventoryCostLayers (
  EntityId       STRING(64)  NOT NULL,
  WarehouseId    STRING(64)  NOT NULL,
  ProductId      STRING(64)  NOT NULL,
  LayerId        STRING(64)  NOT NULL,
  LotId          STRING(64),
  ReceiptRefType STRING(64),
  ReceiptRefId   STRING(128),
  QtyRemaining   INT64       NOT NULL DEFAULT (0),
  OriginalQty    INT64       NOT NULL,
  UnitCostMinor  INT64       NOT NULL,
  Currency       STRING(3)   NOT NULL DEFAULT ('UZS'),
  FxRateId       STRING(64),
  ReceivedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ExpiryDate     DATE,
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, WarehouseId, ProductId, LayerId);

CREATE INDEX Idx_InventoryCostLayers_FEFO
  ON InventoryCostLayers(EntityId, WarehouseId, ProductId, ExpiryDate, ReceivedAt)
  STORING (LotId, QtyRemaining, UnitCostMinor, Currency);

CREATE NULL_FILTERED INDEX Idx_InventoryCostLayers_ByLot
  ON InventoryCostLayers(EntityId, LotId)
  STORING (QtyRemaining, UnitCostMinor, Currency);

CREATE TABLE InventoryMovements (
  EntityId       STRING(64)  NOT NULL,
  MovementId     STRING(128) NOT NULL,
  MovementType   STRING(32)  NOT NULL,
  ReferenceType  STRING(64)  NOT NULL,
  ReferenceId    STRING(128) NOT NULL,
  WarehouseId    STRING(64)  NOT NULL,
  ProductId      STRING(64)  NOT NULL,
  LotId          STRING(64),
  LayerId        STRING(64),
  QtyDelta       INT64       NOT NULL,
  UnitCostMinor  INT64       NOT NULL,
  TotalCostMinor INT64       NOT NULL,
  Currency       STRING(3)   NOT NULL DEFAULT ('UZS'),
  GlEntryId      STRING(128),
  IdempotencyKey STRING(256),
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, MovementId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_InventoryMovements_Idempotency
  ON InventoryMovements(EntityId, IdempotencyKey);

CREATE INDEX Idx_InventoryMovements_ByWarehouseProductCreated
  ON InventoryMovements(EntityId, WarehouseId, ProductId, CreatedAt DESC);

CREATE TABLE EsfDocuments (
  EntityId         STRING(64)  NOT NULL,
  EsfId            STRING(64)  NOT NULL,
  Direction        STRING(16)  NOT NULL,
  OrderId          STRING(64),
  PoId             STRING(64),
  CounterpartyTin  STRING(32)  NOT NULL,
  CounterpartyName STRING(255),
  TotalMinor       INT64       NOT NULL,
  VatMinor         INT64       NOT NULL DEFAULT (0),
  VatRateBps       INT64       NOT NULL DEFAULT (0),
  Status           STRING(24)  NOT NULL,
  Operator         STRING(32)  NOT NULL,
  OperatorRef      STRING(128),
  SignedPayloadUri STRING(512),
  PayloadHash      STRING(64),
  FailureReason    STRING(MAX),
  PeriodId         STRING(16)  NOT NULL,
  IssuedAt         TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  AcceptedAt       TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, EsfId);

CREATE INDEX Idx_EsfDocuments_ByOrder
  ON EsfDocuments(EntityId, OrderId);

CREATE INDEX Idx_EsfDocuments_ByStatus
  ON EsfDocuments(EntityId, Status, CreatedAt DESC);

CREATE INDEX Idx_EsfDocuments_ByPeriod
  ON EsfDocuments(EntityId, PeriodId, Direction, Status)
  STORING (TotalMinor, VatMinor, VatRateBps);

CREATE TABLE EsfStatusHistory (
  EntityId            STRING(64)  NOT NULL,
  EsfId               STRING(64)  NOT NULL,
  HistoryId           STRING(64)  NOT NULL,
  FromStatus          STRING(24),
  ToStatus            STRING(24)  NOT NULL,
  ActorId             STRING(128) NOT NULL,
  Reason              STRING(MAX),
  OperatorPayloadJson STRING(MAX),
  CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, EsfId, HistoryId);
