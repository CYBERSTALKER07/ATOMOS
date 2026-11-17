-- Two-Phase Distributed Inventory Reservation: Soft-Hold & Expiration System.
-- Holds allow cart checkout, B2B wholesale reservation, and allocation locks with
-- strict expiration TTLs and automated background sweeps without stuck allocations.

CREATE TABLE InventoryHolds (
  HoldId          STRING(36)    NOT NULL,
  SupplierId      STRING(36)    NOT NULL,
  WarehouseId     STRING(36)    NOT NULL,
  ReferenceType   STRING(32)    NOT NULL,
  ReferenceId     STRING(36)    NOT NULL,
  Status          STRING(20)    NOT NULL,
  HoldLinesJson   BYTES(MAX)    NOT NULL,
  ExpiresAt       TIMESTAMP     NOT NULL,
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (HoldId);

CREATE INDEX Idx_InventoryHolds_ByExpires ON InventoryHolds(Status, ExpiresAt);

CREATE INDEX Idx_InventoryHolds_ByRef ON InventoryHolds(ReferenceType, ReferenceId);
