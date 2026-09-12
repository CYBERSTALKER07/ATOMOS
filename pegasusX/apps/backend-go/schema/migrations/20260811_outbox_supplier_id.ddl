-- Gate 5 / Phase 5 Week 9: soft-partition OutboxEvents by SupplierId.
-- Nullable during backfill; EmitJSON stamps supplier_id from payload when present.
-- Follow-up: 20260819_outbox_supplier_id_not_null.ddl tightens to NOT NULL.

ALTER TABLE OutboxEvents ADD COLUMN SupplierId STRING(64);

CREATE NULL_FILTERED INDEX Idx_OutboxEvents_Unpublished_BySupplier
  ON OutboxEvents(SupplierId, PublishedAt, CreatedAt);
