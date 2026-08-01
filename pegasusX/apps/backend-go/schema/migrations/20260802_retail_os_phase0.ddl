-- Retail OS Phase 0 foundations (users, capability packs, durable auto-order/favorites)
CREATE TABLE RetailerUsers (
  UserId        STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  Phone         STRING(32)  NOT NULL,
  Name          STRING(255),
  PasswordHash  STRING(MAX),
  FirebaseUid   STRING(128),
  RetailerRole  STRING(32)  NOT NULL,
  IsOwner       BOOL        NOT NULL,
  IsActive      BOOL        NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId);

CREATE UNIQUE INDEX UQ_RetailerUsers_ByRetailerPhone ON RetailerUsers(RetailerId, Phone);
CREATE INDEX Idx_RetailerUsers_ByPhone ON RetailerUsers(Phone);
CREATE INDEX Idx_RetailerUsers_ByRetailer ON RetailerUsers(RetailerId, IsActive, UpdatedAt DESC);

CREATE TABLE RetailerCapabilityPacks (
  RetailerId      STRING(36)  NOT NULL,
  PackId          STRING(32)  NOT NULL,
  Enabled         BOOL        NOT NULL,
  EnabledByUserId STRING(36),
  EnabledAt       TIMESTAMP,
  ConfigJson      STRING(MAX),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, PackId);

CREATE TABLE RetailerAutoOrderSettings (
  RetailerId      STRING(36)  NOT NULL,
  SettingsJson    STRING(MAX) NOT NULL,
  UpdatedByUserId STRING(36),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId);

CREATE TABLE RetailerFavoriteSuppliers (
  RetailerId  STRING(36)  NOT NULL,
  SupplierId  STRING(36)  NOT NULL,
  IsFavorite  BOOL        NOT NULL,
  UpdatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, SupplierId);
