-- PLATFORM_ADMIN TOTP enrollments (P2-17).
CREATE TABLE IF NOT EXISTS PlatformAdminMFA (
  Subject   STRING(128) NOT NULL,
  Secret    STRING(128) NOT NULL,
  Enabled   BOOL NOT NULL,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  EnabledAt TIMESTAMP,
) PRIMARY KEY (Subject);
