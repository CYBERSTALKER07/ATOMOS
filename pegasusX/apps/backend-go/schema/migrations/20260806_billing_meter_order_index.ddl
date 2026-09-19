-- Billing meter idempotency lookup by OrderId (MeterWorker ProcessOrderFinalized).
CREATE INDEX Idx_BillingMeterEvents_ByOrderId ON BillingMeterEvents(OrderId);
