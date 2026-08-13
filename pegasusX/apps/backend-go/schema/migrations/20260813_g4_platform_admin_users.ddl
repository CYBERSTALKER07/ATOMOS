-- G4.B: durable PLATFORM_ADMIN password login (replace token-paste as primary path).
-- Apply: go run ./cmd/apply-migration --ddl schema/migrations/20260813_g4_platform_admin_users.ddl

CREATE TABLE PlatformAdminUsers (
  Subject      STRING(128) NOT NULL,
  Email        STRING(320),
  PasswordHash STRING(255) NOT NULL,
  Enabled      BOOL NOT NULL,
  CreatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt    TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Subject);

CREATE UNIQUE NULL_FILTERED INDEX UQ_PlatformAdminUsers_Email
  ON PlatformAdminUsers(Email);
