-- 20261105_remediate_monotonic_index_hotspots.ddl
-- Cloud Spanner Monotonic Index Key Hotspot Remediation (M4-06 / Target 2)
--
-- Drops secondary indexes with leading monotonic timestamp keys that concentrate
-- writes onto single Spanner splits, replacing them with entity-prefixed covering
-- indexes that distribute writes across Paxos splits and preserve STORING clauses.

-- 1) Remediate SupplierProfiles UpdatedAt hotspot
DROP INDEX Idx_SupplierProfiles_ByUpdatedAt;
CREATE INDEX Idx_SupplierProfiles_BySupplierUpdated
  ON SupplierProfiles(SupplierId, UpdatedAt DESC);

-- 2) Remediate SupplierPricingRules UpdatedAt hotspot
DROP INDEX Idx_SupplierPricingRules_ByUpdatedAt;
CREATE INDEX Idx_SupplierPricingRules_BySupplierUpdated
  ON SupplierPricingRules(SupplierId, UpdatedAt DESC);

-- 3) Remediate OrderStatusTransitions CreatedAt hotspot (High-Throughput Write Path)
DROP INDEX Idx_OrderStatusTransitions_ByCreatedAt;
CREATE INDEX Idx_OrderStatusTransitions_ByOrderTime
  ON OrderStatusTransitions(OrderId, CreatedAt DESC)
  STORING (NewStatus, Reason, EventKind, PreviousStatus, ActorRole, ActorId);

-- 4) Remediate PlatformAdminAudit CreatedAt hotspot
DROP INDEX Idx_PlatformAdminAudit_ByCreated;
CREATE INDEX Idx_PlatformAdminAudit_ByActorCreated
  ON PlatformAdminAudit(ActorSubject, CreatedAt DESC)
  STORING (Action, TenantType, TenantId);

-- 5) Remediate ReplenishmentLocks ExpiresAt hotspot
DROP INDEX Idx_ReplenishmentLocks_ByExpiry;
CREATE INDEX Idx_ReplenishmentLocks_BySupplierExpiry
  ON ReplenishmentLocks(SupplierId, ExpiresAt DESC)
  STORING (AcquiredBy, Priority);

-- 6) Remediate FactorySupplyRequestQC InspectedAt hotspot
DROP INDEX Idx_FactorySupplyRequestQC_ByInspectedAt;
CREATE INDEX Idx_FactorySupplyRequestQC_ByRequestInspected
  ON FactorySupplyRequestQC(RequestId, InspectedAt DESC)
  STORING (Result, InspectedBy);

-- 7) Remediate OutboxEvents PublishedAt monotonic hotspot
-- Replaced by NULL_FILTERED Idx_OutboxEvents_Pending_BySupplier in 20261105_outbox_pending_sentinel_index.ddl
DROP INDEX Idx_OutboxEvents_Unpublished;
