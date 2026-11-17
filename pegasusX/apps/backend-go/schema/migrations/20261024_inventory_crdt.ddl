-- Migration: 20261024_inventory_crdt.ddl
-- Support LWW offline CRDT inventory adjustments in warehouseops

CREATE TABLE InventoryCRDT (
    ProductID STRING(64) NOT NULL,
    Quantity INT64 NOT NULL,
    WriteClock INT64 NOT NULL,
    WorkerNodeID STRING(64) NOT NULL,
    UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ProductID);
