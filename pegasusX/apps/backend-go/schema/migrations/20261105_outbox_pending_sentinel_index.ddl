-- 20261105_outbox_pending_sentinel_index.ddl
-- Cloud Spanner Outbox Index Fix (M4-01):
-- In Cloud Spanner, NULL_FILTERED indexes physically exclude rows where ANY indexed column is NULL.
-- In an outbox pattern where PublishedAt is NULL for pending events, an index on PublishedAt cannot be NULL_FILTERED.
-- By introducing a sentinel column UnpublishedAt (set on insert, cleared to NULL on publication),
-- the pending index Idx_OutboxEvents_Pending_BySupplier can be NULL_FILTERED on (SupplierId, UnpublishedAt, CreatedAt).
-- This ensures the index physically stores ONLY pending events (O(pending) storage), completely excluding
-- millions of historical published events and preventing index bloat and read degradation.

ALTER TABLE OutboxEvents ADD COLUMN UnpublishedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true);

DROP INDEX Idx_OutboxEvents_Unpublished_BySupplier;

CREATE NULL_FILTERED INDEX Idx_OutboxEvents_Pending_BySupplier
  ON OutboxEvents(SupplierId, UnpublishedAt, CreatedAt)
  STORING (ClaimedUntil, AggregateType, AggregateId, TopicName);
