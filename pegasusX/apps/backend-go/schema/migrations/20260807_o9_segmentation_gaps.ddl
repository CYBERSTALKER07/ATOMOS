ALTER TABLE AllocationDecisions ADD COLUMN ConstraintReason STRING(64);
ALTER TABLE AllocationDecisions ADD COLUMN RequestedQty INT64;
ALTER TABLE AllocationDecisions ADD COLUMN AllocatedQty INT64;

CREATE INDEX AllocationDecisions_ByRetailer ON AllocationDecisions(RetailerId, CreatedAt DESC);
