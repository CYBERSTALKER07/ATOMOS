-- Security & cold-chain hardening:
-- 1. DriverCommands issuance ledger so POST /v1/ws/ack verifies issued commands.
-- 2. Restore Products.RequiresColdChain and FleetDVIRInspections.ReeferTempCelsius.

CREATE TABLE DriverCommands (
  DriverId   STRING(36)  NOT NULL,
  CommandId  STRING(128) NOT NULL,
  EventType  STRING(64)  NOT NULL,
  TraceId    STRING(128),
  IssuedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DriverId, CommandId);

CREATE INDEX Idx_DriverCommands_ByIssued
  ON DriverCommands(DriverId, IssuedAt DESC);

ALTER TABLE Products ADD COLUMN RequiresColdChain BOOL NOT NULL DEFAULT (FALSE);
ALTER TABLE FleetDVIRInspections ADD COLUMN ReeferTempCelsius FLOAT64;
