-- Retail OS Phase 6: sections, staff-section assign, assistance tickets
CREATE TABLE RetailerSections (
  SectionId   STRING(36)  NOT NULL,
  RetailerId  STRING(36)  NOT NULL,
  LocationId  STRING(36)  NOT NULL,
  Name        STRING(128) NOT NULL,
  AisleTag    STRING(64),
  ShelfTag    STRING(64),
  SortOrder   INT64       NOT NULL,
  Status      STRING(16)  NOT NULL,
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt   TIMESTAMP,
) PRIMARY KEY (SectionId);

CREATE INDEX Idx_RetailerSections_ByLocation ON RetailerSections(LocationId, Status, SortOrder);
CREATE INDEX Idx_RetailerSections_ByRetailer ON RetailerSections(RetailerId, Status, Name);

CREATE TABLE RetailerSectionSkus (
  SectionId  STRING(36)  NOT NULL,
  Sku        STRING(128) NOT NULL,
  LocationId STRING(36)  NOT NULL,
  RetailerId STRING(36)  NOT NULL,
  CreatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SectionId, Sku);

CREATE INDEX Idx_RetailerSectionSkus_BySkuLocation ON RetailerSectionSkus(Sku, LocationId);
CREATE INDEX Idx_RetailerSectionSkus_ByLocation ON RetailerSectionSkus(LocationId, Sku);

CREATE TABLE RetailerStaffSections (
  UserId     STRING(36)  NOT NULL,
  SectionId  STRING(36)  NOT NULL,
  RetailerId STRING(36)  NOT NULL,
  AssignedAt TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId, SectionId);

CREATE INDEX Idx_RetailerStaffSections_BySection ON RetailerStaffSections(SectionId, UserId);
CREATE INDEX Idx_RetailerStaffSections_ByRetailer ON RetailerStaffSections(RetailerId, UserId);

CREATE TABLE RetailerAssistanceTickets (
  TicketId          STRING(36)  NOT NULL,
  RetailerId        STRING(36)  NOT NULL,
  LocationId        STRING(36)  NOT NULL,
  SectionId         STRING(36)  NOT NULL,
  Note              STRING(MAX) NOT NULL,
  Status            STRING(16)  NOT NULL,
  CreatedByUserId   STRING(36)  NOT NULL,
  ClaimedByUserId   STRING(36),
  CompletedByUserId STRING(36),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ClaimedAt         TIMESTAMP,
  CompletedAt       TIMESTAMP,
  SlaDueAt          TIMESTAMP,
) PRIMARY KEY (TicketId);

CREATE INDEX Idx_RetailerAssist_ByLocationStatus ON RetailerAssistanceTickets(LocationId, Status, CreatedAt DESC);
CREATE INDEX Idx_RetailerAssist_BySectionStatus ON RetailerAssistanceTickets(SectionId, Status, CreatedAt DESC);
CREATE INDEX Idx_RetailerAssist_ByRetailer ON RetailerAssistanceTickets(RetailerId, CreatedAt DESC);
