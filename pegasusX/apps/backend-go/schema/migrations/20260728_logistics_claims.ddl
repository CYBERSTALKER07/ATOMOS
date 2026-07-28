-- Logistics exception claims (post-delivery damage / OS&D).
-- Additive only. Safe on live SSMR Spanner alongside fiscal tables.
-- Apply via gcloud with separate --ddl flags (or semicolon-separated statements).

CREATE TABLE Claims (
  ClaimId STRING(36) NOT NULL,
  OrderId STRING(36) NOT NULL,
  SupplierId STRING(36) NOT NULL,
  RetailerId STRING(36) NOT NULL,
  FiledBy STRING(128) NOT NULL,
  FiledByRole STRING(32) NOT NULL,
  ClaimType STRING(32) NOT NULL,
  Status STRING(32) NOT NULL,
  Description STRING(MAX),
  AmountMinor INT64,
  Currency STRING(8),
  LineItemsJSON BYTES(MAX),
  ResolutionNote STRING(MAX),
  ResolvedBy STRING(128),
  ResolvedAt TIMESTAMP,
  Source STRING(32) NOT NULL,
  TraceId STRING(64),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true)
) PRIMARY KEY (ClaimId);

CREATE INDEX Idx_Claims_ByOrderCreated ON Claims (OrderId, CreatedAt DESC);

CREATE INDEX Idx_Claims_ByRetailerStatus ON Claims (RetailerId, Status, CreatedAt DESC);

CREATE INDEX Idx_Claims_BySupplierStatus ON Claims (SupplierId, Status, CreatedAt DESC);

CREATE TABLE ClaimEvidences (
  ClaimId STRING(36) NOT NULL,
  EvidenceId STRING(36) NOT NULL,
  EvidenceType STRING(32) NOT NULL,
  Uri STRING(MAX) NOT NULL,
  MimeType STRING(128),
  CapturedAt TIMESTAMP,
  CapturedBy STRING(128),
  MetaJSON BYTES(MAX),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true)
) PRIMARY KEY (ClaimId, EvidenceId),
  INTERLEAVE IN PARENT Claims ON DELETE CASCADE;
