-- Phase 2 / HARDEN-02: Outbox-Backed Distributed Saga Coordinator & Crash Recovery
-- ParentOrders saga tracking for resilient multi-supplier checkout.

ALTER TABLE ParentOrders ADD COLUMN SagaState STRING(32);
ALTER TABLE ParentOrders ADD COLUMN ExpectedChildCount INT64;
ALTER TABLE ParentOrders ADD COLUMN CreatedChildOrderIds ARRAY<STRING(64)>;
ALTER TABLE ParentOrders ADD COLUMN LeaseExpiresAt TIMESTAMP;

CREATE INDEX Idx_ParentOrders_SagaRecovery ON ParentOrders(SagaState, LeaseExpiresAt);
