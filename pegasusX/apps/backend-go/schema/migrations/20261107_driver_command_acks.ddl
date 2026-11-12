CREATE TABLE DriverCommandAcks (
  DriverId   STRING(36)  NOT NULL,
  CommandId  STRING(128) NOT NULL,
  TraceId    STRING(128),
  EventType  STRING(64)  NOT NULL,
  AckedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DriverId, CommandId);
