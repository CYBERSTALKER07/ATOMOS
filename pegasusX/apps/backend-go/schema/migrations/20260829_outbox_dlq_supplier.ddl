ALTER TABLE OutboxDeadLetters ADD COLUMN SupplierId STRING(36);
CREATE INDEX Idx_OutboxDeadLetters_Supplier ON OutboxDeadLetters(SupplierId);
