-- G3: supplier/warehouse return (claim) window policies + immutable snapshot on Orders at COMPLETED.
-- Additive only. Apply via gcloud with separate --ddl flags (or semicolon-separated statements).

CREATE TABLE SupplierReturnPolicies (
  SupplierId STRING(36) NOT NULL,
  DefaultWindowHours INT64 NOT NULL,
  ConcealedDamageWindowHours INT64,
  RequirePhoto BOOL NOT NULL DEFAULT (true),
  AllowExpiredClaims BOOL NOT NULL DEFAULT (false),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  UpdatedByUserId STRING(128)
) PRIMARY KEY (SupplierId);

-- SupplierId = "" means warehouse-wide default for all suppliers it serves.
CREATE TABLE WarehouseReturnPolicies (
  WarehouseId STRING(36) NOT NULL,
  SupplierId STRING(36) NOT NULL,
  ReverseDockSLAHours INT64,
  RetailerFileWindowHours INT64,
  CanOverrideRetailerWindow BOOL NOT NULL DEFAULT (false),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  UpdatedByUserId STRING(128)
) PRIMARY KEY (WarehouseId, SupplierId);

ALTER TABLE Orders ADD COLUMN ClaimWindowHours INT64;
ALTER TABLE Orders ADD COLUMN ClaimWindowEndsAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ClaimWindowPolicySource STRING(32);
