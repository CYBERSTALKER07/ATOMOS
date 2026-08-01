-- Retail OS Phase 5: shifts & time clock
CREATE TABLE RetailerTimeEntries (
  EntryId      STRING(36)  NOT NULL,
  RetailerId   STRING(36)  NOT NULL,
  UserId       STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  Status       STRING(16)  NOT NULL,
  ClockInAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClockOutAt   TIMESTAMP,
  AutoClosed   BOOL        NOT NULL,
  Note         STRING(MAX),
) PRIMARY KEY (EntryId);

CREATE INDEX Idx_RetailerTimeEntries_ByUserStatus ON RetailerTimeEntries(UserId, Status, ClockInAt DESC);
CREATE INDEX Idx_RetailerTimeEntries_ByRetailer ON RetailerTimeEntries(RetailerId, ClockInAt DESC);

CREATE TABLE RetailerShifts (
  ShiftId            STRING(36)  NOT NULL,
  RetailerId         STRING(36)  NOT NULL,
  LocationId         STRING(36)  NOT NULL,
  RegisterId         STRING(36),
  OpenedByUserId     STRING(36)  NOT NULL,
  ClosedByUserId     STRING(36),
  Status             STRING(16)  NOT NULL,
  OpeningFloatMinor  INT64       NOT NULL,
  ClosingCashMinor   INT64,
  ExpectedCashMinor  INT64,
  VarianceMinor      INT64,
  Currency           STRING(3)   NOT NULL,
  LinkedPosSessionId STRING(36),
  OpenedAt           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClosedAt           TIMESTAMP,
) PRIMARY KEY (ShiftId);

CREATE INDEX Idx_RetailerShifts_ByLocationStatus ON RetailerShifts(LocationId, Status, OpenedAt DESC);
CREATE INDEX Idx_RetailerShifts_ByRetailer ON RetailerShifts(RetailerId, OpenedAt DESC);
CREATE INDEX Idx_RetailerShifts_ByPosSession ON RetailerShifts(LinkedPosSessionId);
