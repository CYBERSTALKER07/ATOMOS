-- Gate-0: outbox relay leases + schema migration ledger.
ALTER TABLE OutboxEvents ADD COLUMN ClaimedBy STRING(64);
ALTER TABLE OutboxEvents ADD COLUMN ClaimedUntil TIMESTAMP;

CREATE TABLE SchemaMigrations (
  Version     STRING(128)  NOT NULL,
  Checksum    STRING(64)   NOT NULL,
  AppliedAt   TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Version);
