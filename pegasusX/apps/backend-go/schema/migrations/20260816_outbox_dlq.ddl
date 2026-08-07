-- Outbox dead-lettering (Phase 0): events that exhaust publish attempts are moved
-- to OutboxDeadLetters instead of retrying forever invisibly. Ops requeues after
-- fixing the cause; nothing is silently dropped.
CREATE TABLE OutboxDeadLetters (
  EventId        STRING(36)   NOT NULL,
  AggregateType  STRING(64)   NOT NULL,
  AggregateId    STRING(64)   NOT NULL,
  TopicName      STRING(128)  NOT NULL,
  Payload        BYTES(MAX)   NOT NULL,
  CreatedAt      TIMESTAMP    NOT NULL,
  DeadLetteredAt TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  Attempts       INT64        NOT NULL,
  LastError      STRING(MAX),
) PRIMARY KEY (EventId);

ALTER TABLE OutboxEvents ADD COLUMN PublishAttempts INT64 NOT NULL DEFAULT (0);
