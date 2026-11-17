-- Leftover close: CRM email, payout policy, country overrides (UZ).
-- Does not change checkout gateway selection. Payout live rail stays bank-file.

ALTER TABLE Retailers ADD COLUMN Email STRING(320);

ALTER TABLE Factories ALTER COLUMN DailyOutputCapacity SET DEFAULT (700);

CREATE TABLE SupplierPayoutPolicies (
  SupplierId       STRING(36)  NOT NULL,
  PayoutMode       STRING(32)  NOT NULL,
  FeePolicyVersion STRING(64)  NOT NULL,
  EffectiveAt      TIMESTAMP   NOT NULL,
  UpdatedBy        STRING(128),
  UpdatedByType    STRING(32),
  Reason           STRING(512) NOT NULL,
  IsActive         BOOL        NOT NULL DEFAULT (TRUE),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);

CREATE TABLE SupplierCountryOverrides (
  SupplierId                    STRING(36)  NOT NULL,
  CountryCode                   STRING(2)   NOT NULL,
  BreachRadiusMeters            FLOAT64,
  ShopClosedGraceMinutes        INT64,
  ShopClosedEscalationMinutes   INT64,
  OfflineModeDurationMinutes    INT64,
  CashCustodyAlertHours         INT64,
  Reason                        STRING(512) NOT NULL,
  UpdatedBy                     STRING(128),
  UpdatedByType                 STRING(32),
  CreatedAt                     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, CountryCode);
