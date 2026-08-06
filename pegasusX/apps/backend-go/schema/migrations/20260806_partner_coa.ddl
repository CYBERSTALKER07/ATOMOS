-- Gate-3: configurable 1C chart of accounts for partner journals exports.
CREATE TABLE PartnerCoaMaps (
  TenantType      STRING(16)  NOT NULL,
  TenantId        STRING(36)  NOT NULL,
  AccountAR       STRING(32)  NOT NULL,
  AccountRevenue  STRING(32)  NOT NULL,
  AccountBankCash STRING(32)  NOT NULL,
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedBy       STRING(128),
) PRIMARY KEY (TenantType, TenantId);
