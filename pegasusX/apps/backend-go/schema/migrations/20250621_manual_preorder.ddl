-- Manual pre-order lifecycle + delivery intent (Option A state machine).

ALTER TABLE Orders ADD COLUMN DeliverBefore TIMESTAMP;
ALTER TABLE Orders ADD COLUMN DeliveryPriority STRING(16) NOT NULL DEFAULT ('STANDARD');
ALTER TABLE Orders ADD COLUMN WarehouseNotes STRING(MAX);
ALTER TABLE Orders ADD COLUMN PreorderReminderSentAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN NudgeNotifiedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN ConfirmationNotifiedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN CancelLockedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN CancelLockReason STRING(MAX);
ALTER TABLE Orders ADD COLUMN DeliveryFeeMinor INT64 NOT NULL DEFAULT (0);

CREATE INDEX Idx_Orders_ByWarehouseStatusDelivery ON Orders(WarehouseId, Status, RequestedDeliveryDate DESC);
