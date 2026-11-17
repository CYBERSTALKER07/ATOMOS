-- ATOMOS M0.2: Append-Only Balanced General Ledger & Data-Driven Posting Rules
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

CREATE TABLE GlAccounts (
  EntityId     STRING(64)  NOT NULL,
  AccountCode  STRING(64)  NOT NULL,
  Class        STRING(16)  NOT NULL,
  Name         STRING(255) NOT NULL,
  ParentCode   STRING(64),
  Currency     STRING(3)   NOT NULL,
  IsControl    BOOL        NOT NULL DEFAULT (FALSE),
  ActiveFrom   TIMESTAMP,
  ActiveTo     TIMESTAMP,
  CountryPack  STRING(16)  NOT NULL DEFAULT ('UZ'),
  CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CONSTRAINT CHK_GlAccounts_Class CHECK (Class IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE'))
) PRIMARY KEY (EntityId, AccountCode);

CREATE TABLE JournalEntries (
  EntityId         STRING(64)  NOT NULL,
  EntryId          STRING(128) NOT NULL,
  PeriodId         STRING(16)  NOT NULL,
  SourceEventId    STRING(128),
  SourceType       STRING(64)  NOT NULL,
  SourceId         STRING(128) NOT NULL,
  OrderId          STRING(64),
  Memo             STRING(MAX),
  Currency         STRING(3)   NOT NULL,
  CountryPack      STRING(16)  NOT NULL DEFAULT ('UZ'),
  ReversesEntryId  STRING(128),
  IdempotencyKey   STRING(256) NOT NULL,
  PrevHash         STRING(64)  NOT NULL,
  EntryHash        STRING(64)  NOT NULL,
  PostedBy         STRING(128) NOT NULL,
  OccurredAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  PostedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, EntryId);

CREATE UNIQUE INDEX UQ_JournalEntries_EntityIdempotency
  ON JournalEntries(EntityId, IdempotencyKey);

CREATE INDEX Idx_JournalEntries_ByEntityPeriodPosted
  ON JournalEntries(EntityId, PeriodId, PostedAt DESC);

CREATE INDEX Idx_JournalEntries_BySource
  ON JournalEntries(EntityId, SourceType, SourceId, PostedAt DESC);

CREATE INDEX Idx_JournalEntries_ByEntityPosted
  ON JournalEntries(EntityId, PostedAt DESC)
  STORING (EntryHash, PrevHash);

CREATE NULL_FILTERED INDEX Idx_JournalEntries_ByOrder
  ON JournalEntries(EntityId, OrderId, PostedAt DESC);

CREATE NULL_FILTERED INDEX Idx_JournalEntries_ByReversal
  ON JournalEntries(EntityId, ReversesEntryId);

CREATE TABLE JournalPostings (
  EntityId               STRING(64)  NOT NULL,
  EntryId                STRING(128) NOT NULL,
  Line                   INT64       NOT NULL,
  AccountCode            STRING(64)  NOT NULL,
  Direction              STRING(16)  NOT NULL,
  AmountMinor            INT64       NOT NULL,
  Currency               STRING(3)   NOT NULL,
  FunctionalAmountMinor  INT64       NOT NULL DEFAULT (0),
  FxRateId               STRING(64),
  PartyType              STRING(32),
  PartyId                STRING(64),
  WarehouseId            STRING(64),
  ProductId              STRING(64),
  LotId                  STRING(64),
  Description            STRING(MAX),
  CONSTRAINT CHK_JournalPostings_Direction CHECK (Direction IN ('DEBIT', 'CREDIT')),
  CONSTRAINT CHK_JournalPostings_AmountPositive CHECK (AmountMinor > 0)
) PRIMARY KEY (EntityId, EntryId, Line),
  INTERLEAVE IN PARENT JournalEntries ON DELETE NO ACTION;

CREATE INDEX Idx_JournalPostings_ByAccount
  ON JournalPostings(EntityId, AccountCode, EntryId, Line)
  STORING (Direction, AmountMinor, Currency, FunctionalAmountMinor, PartyType, PartyId);

CREATE TABLE GlBalances (
  EntityId     STRING(64)  NOT NULL,
  PeriodId     STRING(16)  NOT NULL,
  AccountCode  STRING(64)  NOT NULL,
  PartyKey     STRING(128) NOT NULL,
  Shard        INT64       NOT NULL DEFAULT (0),
  Currency     STRING(3)   NOT NULL,
  DebitMinor   INT64       NOT NULL DEFAULT (0),
  CreditMinor  INT64       NOT NULL DEFAULT (0),
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (EntityId, PeriodId, AccountCode, PartyKey, Shard);

CREATE TABLE PostingRules (
  CountryPack    STRING(16)   NOT NULL,
  RuleId         STRING(64)   NOT NULL,
  Version        INT64        NOT NULL,
  EventType      STRING(64)   NOT NULL,
  ConditionExpr  STRING(512),
  DebitAccount   STRING(64)   NOT NULL,
  CreditAccount  STRING(64)   NOT NULL,
  AmountExpr     STRING(256)  NOT NULL,
  Description    STRING(1024),
  Active         BOOL         NOT NULL DEFAULT (TRUE),
  UpdatedAt      TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (CountryPack, RuleId, Version);

CREATE INDEX Idx_PostingRules_ByEvent
  ON PostingRules(EventType, CountryPack, Active, Version DESC)
  STORING (ConditionExpr, DebitAccount, CreditAccount, AmountExpr, Description);
