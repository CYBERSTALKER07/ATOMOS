-- G5: EDI profiles, master-data parties/plants/DLQ, external doc ledger for adapters.

CREATE TABLE PartnerEdiProfiles (
  TenantType       STRING(16)  NOT NULL,
  TenantId         STRING(36)  NOT NULL,
  PackName         STRING(64)  NOT NULL,
  OurGln           STRING(32),
  TheirGln         STRING(32),
  EnabledDocTypes  STRING(MAX), -- CSV: ORDERS,ORDRSP,DESADV,INVOIC,...
  RequireContrl    BOOL NOT NULL,
  RequireAperak    BOOL NOT NULL,
  AsnAsDesadv      BOOL NOT NULL,
  Transport        STRING(16), -- LOCAL|SFTP|AS2|ANY
  UpdatedAt        TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId);

CREATE TABLE PartnerParties (
  TenantType   STRING(16)  NOT NULL,
  TenantId     STRING(36)  NOT NULL,
  ExternalId   STRING(128) NOT NULL,
  Role         STRING(32)  NOT NULL, -- RETAILER|SUPPLIER|WAREHOUSE|CARRIER
  LegalName    STRING(512),
  Gln          STRING(32),
  Version      INT64 NOT NULL,
  UpdatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId, ExternalId);

CREATE TABLE PartnerPlantMaps (
  TenantType      STRING(16)  NOT NULL,
  TenantId        STRING(36)  NOT NULL,
  ExternalPlantId STRING(128) NOT NULL,
  WarehouseId     STRING(36)  NOT NULL,
  UpdatedAt       TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId, ExternalPlantId);

CREATE TABLE PartnerMasterDataDLQ (
  DlqId        STRING(36)  NOT NULL,
  TenantType   STRING(16)  NOT NULL,
  TenantId     STRING(36)  NOT NULL,
  EntityType   STRING(32)  NOT NULL, -- party|plant|product|price|stock|asn
  ExternalId   STRING(128) NOT NULL,
  Reason       STRING(512) NOT NULL,
  PayloadHash  STRING(64),
  PayloadJson  BYTES(MAX),
  CreatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  ReplayedAt   TIMESTAMP,
) PRIMARY KEY (DlqId);

CREATE INDEX Idx_PartnerMasterDataDLQ_ByTenant
  ON PartnerMasterDataDLQ (TenantType, TenantId, CreatedAt DESC);

CREATE TABLE PartnerExternalDocuments (
  TenantType    STRING(16)  NOT NULL,
  TenantId      STRING(36)  NOT NULL,
  Adapter       STRING(32)  NOT NULL, -- onec|sap|wms_asn
  ExternalDocId STRING(128) NOT NULL,
  Status        STRING(16)  NOT NULL,
  RefId         STRING(64),
  CreatedAt     TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (TenantType, TenantId, Adapter, ExternalDocId);
