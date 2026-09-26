-- Analytics column tenancy wave 2: demand sensing tables + shadow supplier index.
-- DemandSignals / DemandAdjustments gain SupplierId; writers stamp PreferTenant or _platform.
-- RetailerAutoOrderShadowProposals already has SupplierId — add supplier bucket index.

ALTER TABLE DemandSignals ADD COLUMN SupplierId STRING(36);

CREATE NULL_FILTERED INDEX Idx_DemandSignals_BySupplierCreated
  ON DemandSignals(SupplierId, CreatedAt DESC);

ALTER TABLE DemandAdjustments ADD COLUMN SupplierId STRING(36);

CREATE NULL_FILTERED INDEX Idx_DemandAdjustments_BySupplierDate
  ON DemandAdjustments(SupplierId, Date DESC);

CREATE NULL_FILTERED INDEX Idx_RetailerAutoOrderShadow_BySupplierBucket
  ON RetailerAutoOrderShadowProposals(SupplierId, BucketDate DESC);
