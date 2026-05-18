-- pegasusX Spanner DDL (seed)
--
-- Authoritative schema lives under apps/backend-go/schema/. Migrations are
-- applied via apps/backend-go/cmd/setup. This file is the canonical bootstrap
-- definition consumed by emulator setup and production provisioning.
--
-- Single-tenant note: SupplierId remains on every supplier-owned table. One
-- Suppliers row is seeded at bootstrap; the column stays for migration parity
-- with Pegasus.

CREATE TABLE Suppliers (
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  CountryCode      STRING(2)     NOT NULL,
  Currency         STRING(3)     NOT NULL,
  IsConfigured     BOOL          NOT NULL DEFAULT (FALSE),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE TABLE Retailers (
  RetailerId       STRING(36)    NOT NULL,
  Phone            STRING(32)    NOT NULL,
  Name             STRING(255),
  CountryCode      STRING(2)     NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  H3Cell           STRING(15),
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId);

CREATE UNIQUE INDEX UQ_Retailers_Phone ON Retailers(Phone);

CREATE TABLE OutboxEvents (
  EventId          STRING(36)    NOT NULL,
  AggregateType    STRING(64)    NOT NULL,
  AggregateId      STRING(64)    NOT NULL,
  TopicName        STRING(128)   NOT NULL,
  Payload          BYTES(MAX)    NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  PublishedAt      TIMESTAMP,
) PRIMARY KEY (EventId);

CREATE INDEX Idx_OutboxEvents_Unpublished ON OutboxEvents(PublishedAt, CreatedAt);
