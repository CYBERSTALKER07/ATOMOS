-- Gate 5 / Enterprise Phase 5 soak: OutboxEvents.SupplierId NOT NULL.
-- Precondition: writers stamp ResolveSupplierID (incl. _platform); backfill
-- stamps remaining NULL/empty rows. Apply backfill before this DDL on live DBs.

ALTER TABLE OutboxEvents ALTER COLUMN SupplierId STRING(64) NOT NULL;
