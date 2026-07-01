-- Pegasus multi-supplier planning federation (P1-P4) — optional tenant workspace mirror.
-- Apply via cmd/setup or cmd/apply-migration idempotent convergence.
-- Does not modify pegasusX Spanner tables.

CREATE TABLE PlanningTenantWorkspace (
  SupplierId             STRING(36) NOT NULL,
  BaselineSnapshotJson   STRING(MAX),
  UpdatedAt              TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);
