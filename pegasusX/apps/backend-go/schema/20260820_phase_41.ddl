-- ── Phase 4.1: Retailer Shelf Intelligence & Promotions Lifecycle ───────────

CREATE TABLE RetailerShelfAlerts (
  AlertId             STRING(36)    NOT NULL,
  RetailerId          STRING(36)    NOT NULL,
  LocationId          STRING(36)    NOT NULL,
  GlobalProductId     STRING(36)    NOT NULL,
  Status              STRING(32)    NOT NULL, -- e.g., OPEN, RESOLVED
  CurrentStock        INT64         NOT NULL,
  CapacityThreshold   INT64         NOT NULL,
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResolvedAt          TIMESTAMP,
) PRIMARY KEY (RetailerId, LocationId, AlertId);

CREATE INDEX Idx_RetailerShelfAlerts_ByProduct
  ON RetailerShelfAlerts(GlobalProductId, Status);

CREATE TABLE SupplierPromotionCampaigns (
  CampaignId          STRING(36)    NOT NULL,
  SupplierId          STRING(36)    NOT NULL,
  Name                STRING(128)   NOT NULL,
  BudgetLimitMinor    INT64         NOT NULL,
  BudgetUsedMinor     INT64         NOT NULL DEFAULT (0),
  Status              STRING(32)    NOT NULL, -- e.g., ACTIVE, EXHAUSTED, PAUSED
  CreatedAt           TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CampaignId);

CREATE INDEX Idx_SupplierPromotionCampaigns_BySupplier
  ON SupplierPromotionCampaigns(SupplierId, Status);

CREATE TABLE RetailerPromotionEnrollments (
  EnrollmentId        STRING(36)    NOT NULL,
  CampaignId          STRING(36)    NOT NULL,
  RetailerId          STRING(36)    NOT NULL,
  Status              STRING(32)    NOT NULL, -- e.g., ENROLLED, OPTED_OUT
  EnrolledAt          TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (CampaignId, EnrollmentId);

CREATE INDEX Idx_RetailerPromotionEnrollments_ByRetailer
  ON RetailerPromotionEnrollments(RetailerId, Status);
