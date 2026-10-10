-- Gate-3 §8.9: AS2 transport config for EDI-lite (not Drummond-certified).

CREATE TABLE PartnerAs2Configs (
  TenantType            STRING(16)   NOT NULL,
  TenantId              STRING(36)   NOT NULL,
  As2Enabled            BOOL         NOT NULL DEFAULT (FALSE),
  OurAs2Id              STRING(128)  NOT NULL,
  PartnerAs2Id          STRING(128)  NOT NULL,
  PartnerUrl            STRING(1024),
  OurCertSecretRef      STRING(256),
  OurKeySecretRef       STRING(256),
  PartnerCertSecretRef  STRING(256),
  SignRequired          BOOL         NOT NULL DEFAULT (TRUE),
  EncryptRequired       BOOL         NOT NULL DEFAULT (TRUE),
  UpdatedAt             TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId);

CREATE UNIQUE INDEX Idx_PartnerAs2Configs_OurAs2Id
  ON PartnerAs2Configs (OurAs2Id);
