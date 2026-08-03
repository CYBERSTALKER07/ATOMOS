CREATE TABLE Claims (
  ClaimId STRING(36) NOT NULL,
  OrderId STRING(36) NOT NULL,
  RetailerId STRING(36) NOT NULL,
  SupplierId STRING(36) NOT NULL,
  Status STRING(32) NOT NULL,
  Reason STRING(64) NOT NULL,
  RequestedAmountMinor INT64 NOT NULL,
  ApprovedAmountMinor INT64,
  Liability STRING(50),
  Notes STRING(MAX),
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ClaimId);

CREATE INDEX Idx_Claims_ByOrder ON Claims(OrderId);
CREATE INDEX Idx_Claims_BySupplier ON Claims(SupplierId, CreatedAt DESC);
CREATE INDEX Idx_Claims_ByRetailer ON Claims(RetailerId, CreatedAt DESC);

CREATE TABLE ClaimEvidences (
  EvidenceId STRING(36) NOT NULL,
  ClaimId STRING(36) NOT NULL,
  FileUrl STRING(1024) NOT NULL,
  ContentType STRING(128),
  UploadedBy STRING(128) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (EvidenceId);

CREATE INDEX Idx_ClaimEvidences_ByClaim ON ClaimEvidences(ClaimId);
