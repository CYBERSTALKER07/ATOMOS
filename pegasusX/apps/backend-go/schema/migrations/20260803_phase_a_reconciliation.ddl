CREATE TABLE CreditNotes (
  CreditNoteId        STRING(36) NOT NULL,
  OrderId             STRING(36) NOT NULL,
  Type                STRING(32) NOT NULL,
  Status              STRING(16) NOT NULL,
  ReasonCode          STRING(64) NOT NULL,
  ReasonText          STRING(MAX),
  TotalNetMinor       INT64 NOT NULL,
  TotalVatMinor       INT64 NOT NULL,
  TotalGrossMinor     INT64 NOT NULL,
  RegimeId            STRING(36),
  OriginalEhfId       STRING(128),
  CorrectiveEhfId     STRING(128),
  CreatedBy           STRING(64) NOT NULL,
  CreatedAt           TIMESTAMP NOT NULL,
  IssuedAt            TIMESTAMP,
  CompletedAt         TIMESTAMP,
) PRIMARY KEY (CreditNoteId);

CREATE INDEX CreditNotes_ByOrder ON CreditNotes(OrderId);

CREATE TABLE CreditNoteLines (
  CreditNoteId        STRING(36) NOT NULL,
  LineId              STRING(36) NOT NULL,
  OrderLineId         STRING(36) NOT NULL,
  Sku                 STRING(64) NOT NULL,
  Qty                 INT64 NOT NULL,
  UnitNetMinor        INT64 NOT NULL,
  VatRateBps          INT64 NOT NULL,
  LineNetMinor        INT64 NOT NULL,
  LineVatMinor        INT64 NOT NULL,
  LineGrossMinor      INT64 NOT NULL,
) PRIMARY KEY (CreditNoteId, LineId),
  INTERLEAVE IN PARENT CreditNotes ON DELETE CASCADE;

CREATE TABLE ReverseLogisticsTasks (
  TaskId              STRING(36) NOT NULL,
  CreditNoteId        STRING(36) NOT NULL,
  OrderId             STRING(36) NOT NULL,
  Status              STRING(16) NOT NULL,
  WarehouseId         STRING(36),
  DriverId            STRING(36),
  ExpectedQtyJson     JSON,
  ReceivedQtyJson     JSON,
  CreatedAt           TIMESTAMP NOT NULL,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (TaskId);

CREATE INDEX ReverseLogisticsTasks_ByStatus ON ReverseLogisticsTasks(Status);

CREATE TABLE CashReconciliations (
  ReconciliationId    STRING(36) NOT NULL,
  DriverId            STRING(36) NOT NULL,
  RouteId             STRING(36),
  ShiftDate           DATE NOT NULL,
  ExpectedCashMinor   INT64 NOT NULL,
  DeclaredCashMinor   INT64 NOT NULL,
  DifferenceMinor     INT64 NOT NULL,
  Status              STRING(16) NOT NULL,
  DriverNote          STRING(MAX),
  FinanceNote         STRING(MAX),
  CreatedAt           TIMESTAMP NOT NULL,
  ResolvedAt          TIMESTAMP,
  ResolvedBy          STRING(64),
) PRIMARY KEY (ReconciliationId);

CREATE INDEX CashReconciliations_ByDriverDate ON CashReconciliations(DriverId, ShiftDate);
CREATE INDEX CashReconciliations_ByStatus ON CashReconciliations(Status);
