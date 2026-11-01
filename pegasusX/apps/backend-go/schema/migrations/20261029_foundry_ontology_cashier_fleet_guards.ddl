-- Palantir Foundry Semantic-Kinetic-Dynamic Triad, Role 7 (Depot Cashier & AML Vault),
-- Role 9 (Fleet, Yard & DVIR Red-Tag), and the 12 Cross-Role Pre-Execution Edge-Case Ledgers.

CREATE TABLE OntologyObjects (
  CellId            STRING(32)         NOT NULL,
  ObjectType        STRING(64)         NOT NULL,
  ObjectId          STRING(64)         NOT NULL,
  SupplierId        STRING(36)         NOT NULL,
  Version           INT64              NOT NULL DEFAULT (1),
  PropertiesJson    STRING(MAX)        NOT NULL,
  SecurityMarkings  ARRAY<STRING(64)>,
  ValidFrom         TIMESTAMP          NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP          NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CellId, ObjectType, ObjectId);

CREATE INDEX Idx_OntologyObjects_BySupplier ON OntologyObjects(SupplierId, ObjectType);

CREATE TABLE OntologyGraphEdges (
  CellId            STRING(32)   NOT NULL,
  SourceObjectType  STRING(64)   NOT NULL,
  SourceObjectId    STRING(64)   NOT NULL,
  LinkType          STRING(64)   NOT NULL,
  TargetObjectType  STRING(64)   NOT NULL,
  TargetObjectId    STRING(64)   NOT NULL,
  SupplierId        STRING(36)   NOT NULL,
  CreatedByActionId STRING(64)   NOT NULL,
  MetadataJson      STRING(MAX),
  ValidFrom         TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  ValidTo           TIMESTAMP,
) PRIMARY KEY (CellId, SourceObjectType, SourceObjectId, LinkType, TargetObjectType, TargetObjectId);

CREATE INDEX Idx_OntologyGraphEdges_Reverse ON OntologyGraphEdges(CellId, TargetObjectType, TargetObjectId, LinkType);

CREATE TABLE KineticActionExecutions (
  ActionExecutionId      STRING(36)   NOT NULL,
  CellId                 STRING(32)   NOT NULL,
  SupplierId             STRING(36)   NOT NULL,
  ActionType             STRING(64)   NOT NULL,
  ActorId                STRING(64)   NOT NULL,
  ActorRole              STRING(32)   NOT NULL,
  ScenarioId             STRING(36),
  Status                 STRING(24)   NOT NULL,
  PreconditionReportJson STRING(MAX)  NOT NULL,
  DiffPayloadJson        STRING(MAX)  NOT NULL,
  IdempotencyKey         STRING(128)  NOT NULL,
  ExecutedAt             TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ActionExecutionId);

CREATE INDEX Idx_KineticActionExecutions_BySupplier ON KineticActionExecutions(SupplierId, ActionType, ExecutedAt DESC);

CREATE TABLE ScenarioObjectPatches (
  ScenarioId   STRING(36)   NOT NULL,
  ObjectType   STRING(64)   NOT NULL,
  ObjectId     STRING(64)   NOT NULL,
  SupplierId   STRING(36)   NOT NULL,
  Operation    STRING(16)   NOT NULL,
  BaseVersion  INT64        NOT NULL,
  PatchJson    STRING(MAX)  NOT NULL,
  UpdatedAt    TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ScenarioId, ObjectType, ObjectId);

CREATE TABLE CashierBlindCountSessions (
  SessionId                 STRING(36)   NOT NULL,
  SupplierId                STRING(36)   NOT NULL,
  DepotId                   STRING(36)   NOT NULL,
  DriverId                  STRING(36)   NOT NULL,
  RouteId                   STRING(36)   NOT NULL,
  BagSealBarcode            STRING(64)   NOT NULL,
  AttemptNumber             INT64        NOT NULL DEFAULT (1),
  DenominationBreakdownJson STRING(MAX)  NOT NULL,
  CountedCashMinor          INT64        NOT NULL DEFAULT (0),
  ExpectedCashMinor         INT64        NOT NULL DEFAULT (0),
  VarianceMinor             INT64        NOT NULL DEFAULT (0),
  Status                    STRING(32)   NOT NULL,
  ClearanceTokenHash        STRING(64),
  CreatedAt                 TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResolvedAt                TIMESTAMP,
) PRIMARY KEY (SessionId);

CREATE UNIQUE INDEX Idx_CashierBlindCount_BagSeal ON CashierBlindCountSessions(BagSealBarcode);
CREATE INDEX Idx_CashierBlindCount_ByDriver ON CashierBlindCountSessions(DriverId, CreatedAt DESC);

CREATE TABLE DepotVaultLedgerEntries (
  VaultEntryId        STRING(36)  NOT NULL,
  DepotId             STRING(36)  NOT NULL,
  SupplierId          STRING(36)  NOT NULL,
  SessionId           STRING(36),
  CITManifestId       STRING(36),
  EntryType           STRING(32)  NOT NULL,
  AmountMinor         INT64       NOT NULL,
  RunningBalanceMinor INT64       NOT NULL,
  CourierBadgeId      STRING(64),
  SealBarcode         STRING(64),
  CreatedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (VaultEntryId);

CREATE INDEX Idx_DepotVaultLedger_ByDepot ON DepotVaultLedgerEntries(DepotId, CreatedAt DESC);

CREATE TABLE MerchantDailyCashExposure (
  LegalEntityTIN       STRING(32)  NOT NULL,
  SupplierId           STRING(36)  NOT NULL,
  CalendarDate         DATE        NOT NULL,
  AccumulatedCashMinor INT64       NOT NULL DEFAULT (0),
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LegalEntityTIN, SupplierId, CalendarDate);

CREATE TABLE FleetDVIRInspections (
  InspectionId           STRING(36)   NOT NULL,
  SupplierId             STRING(36)   NOT NULL,
  VehicleId              STRING(36)   NOT NULL,
  DriverId               STRING(36)   NOT NULL,
  InspectionType         STRING(16)   NOT NULL,
  ChecklistJson          STRING(MAX)  NOT NULL,
  CriticalDefectFound    BOOL         NOT NULL DEFAULT (FALSE),
  RedTagApplied          BOOL         NOT NULL DEFAULT (FALSE),
  OdometerKm             INT64        NOT NULL,
  ReeferTempCelsius      FLOAT64,
  Status                 STRING(32)   NOT NULL,
  MaintenanceWorkOrderId STRING(36),
  CreatedAt              TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (InspectionId);

CREATE INDEX Idx_FleetDVIRInspections_ByVehicle ON FleetDVIRInspections(VehicleId, CreatedAt DESC);

CREATE TABLE YardGateCheckEvents (
  GateEventId     STRING(36)  NOT NULL,
  DepotId         STRING(36)  NOT NULL,
  SupplierId      STRING(36)  NOT NULL,
  VehicleId       STRING(36)  NOT NULL,
  LicensePlate    STRING(24)  NOT NULL,
  Direction       STRING(16)  NOT NULL,
  BoltSealBarcode STRING(64),
  SealIntact      BOOL        NOT NULL DEFAULT (TRUE),
  AssignedBaySlot STRING(32)  NOT NULL,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (GateEventId);

CREATE TABLE ReturnableContainerLedger (
  AccountId          STRING(64)  NOT NULL,
  ContainerSku       STRING(64)  NOT NULL,
  SupplierId         STRING(36)  NOT NULL,
  HolderType         STRING(24)  NOT NULL,
  BalanceQty         INT64       NOT NULL DEFAULT (0),
  DepositLockedMinor INT64       NOT NULL DEFAULT (0),
  UpdatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (AccountId, ContainerSku);

CREATE TABLE OfflineQuotaLeases (
  LeaseId               STRING(36)  NOT NULL,
  OrderId               STRING(36)  NOT NULL,
  DriverId              STRING(36)  NOT NULL,
  SupplierId            STRING(36)  NOT NULL,
  LegalEntityTIN        STRING(32)  NOT NULL,
  MaxCashAllowedMinor   INT64       NOT NULL,
  MaxCreditAllowedMinor INT64       NOT NULL,
  ExpiresAt             TIMESTAMP   NOT NULL,
  Status                STRING(24)  NOT NULL,
  CreatedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LeaseId);
