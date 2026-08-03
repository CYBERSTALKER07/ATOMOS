-- Durable org-level policy flags (e.g. family_writes_gone after Team migrate).
CREATE TABLE RetailerOrgFlags (
  RetailerId STRING(64) NOT NULL,
  FlagKey    STRING(64) NOT NULL,
  FlagValue  STRING(256),
  UpdatedAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, FlagKey);
