-- Supplier return disposition workflow (write-off vs restock).

ALTER TABLE SupplierReturns ADD COLUMN Status STRING(32) NOT NULL DEFAULT ('PENDING');
ALTER TABLE SupplierReturns ADD COLUMN ResolvedAt TIMESTAMP;
ALTER TABLE SupplierReturns ADD COLUMN ResolutionNotes STRING(MAX);

CREATE INDEX Idx_SupplierReturns_ByStatus ON SupplierReturns(Status, CreatedAt DESC);
