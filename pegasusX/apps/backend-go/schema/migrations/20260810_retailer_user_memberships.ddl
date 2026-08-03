-- Wave C1.1: Multi-org staff memberships (person → many retailer orgs).
-- Additive dual-write alongside RetailerUsers. Login dual-read in C1.2.
-- Prefer global UserId + multiple membership rows over duplicate phones per org long-term.

CREATE TABLE RetailerUserMemberships (
  UserId          STRING(36)  NOT NULL,
  RetailerId      STRING(36)  NOT NULL,
  RetailerRole    STRING(32)  NOT NULL,
  IsActive        BOOL        NOT NULL DEFAULT (true),
  LocationIdsJson STRING(MAX),
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId, RetailerId);

CREATE INDEX Idx_RetailerUserMemberships_ByRetailer
  ON RetailerUserMemberships (RetailerId, IsActive);

CREATE INDEX Idx_RetailerUserMemberships_ByUserActive
  ON RetailerUserMemberships (UserId, IsActive);
