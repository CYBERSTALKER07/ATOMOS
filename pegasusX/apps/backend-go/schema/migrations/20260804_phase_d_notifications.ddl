CREATE TABLE NotificationPreferences (
  PrincipalId STRING(36) NOT NULL,
  PrincipalType STRING(16) NOT NULL,
  EventType STRING(64) NOT NULL,
  Channel STRING(16) NOT NULL,
  Enabled BOOL NOT NULL,
  QuietFrom STRING(8),
  QuietTo STRING(8),
  UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (PrincipalId, EventType, Channel);
