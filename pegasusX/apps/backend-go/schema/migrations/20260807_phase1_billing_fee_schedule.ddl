-- Phase 1: billing fee schedule + supplier tier.
CREATE TABLE BillingFeeSchedules (
  FeeScheduleId            STRING(36)  NOT NULL,
  SupplierId               STRING(36)  NOT NULL,
  Tier                     STRING(32)  NOT NULL,
  PerOrderMinor            INT64       NOT NULL,
  GmvBps                   INT64       NOT NULL,
  MonthlySubscriptionMinor INT64       NOT NULL,
  Currency                 STRING(8)   NOT NULL,
  EffectiveFrom            TIMESTAMP   NOT NULL,
  EffectiveTo              TIMESTAMP,
  CreatedAt                TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (FeeScheduleId);

CREATE INDEX Idx_BillingFeeSchedules_BySupplier ON BillingFeeSchedules(SupplierId);
CREATE INDEX Idx_BillingFeeSchedules_ByTier ON BillingFeeSchedules(Tier);

ALTER TABLE SupplierProfiles ADD COLUMN Tier STRING(32);
