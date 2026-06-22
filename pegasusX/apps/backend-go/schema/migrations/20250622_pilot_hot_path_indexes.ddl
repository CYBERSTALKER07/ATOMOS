-- Pilot hot-path indexes (P1 Spanner review): supplier order list, dispatch preview.

CREATE INDEX Idx_Orders_BySupplierUpdated ON Orders(SupplierId, UpdatedAt DESC);
CREATE INDEX Idx_Orders_BySupplierStatusUpdated ON Orders(SupplierId, Status, UpdatedAt DESC);
