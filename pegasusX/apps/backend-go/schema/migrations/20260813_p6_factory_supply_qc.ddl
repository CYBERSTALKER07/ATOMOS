-- P6-F factory supply-request QC. Parent existence is WarehouseSupplyRequests (not Pegasus SupplyRequests).
-- Apply: go run ./cmd/apply-migration --ddl schema/migrations/20260813_p6_factory_supply_qc.ddl
-- Does not change GET /v1/warehouse/ops/crm JSON.

CREATE TABLE FactorySupplyRequestQC (
  RequestId    STRING(36) NOT NULL,
  Result       STRING(10) NOT NULL,
  Notes        STRING(MAX),
  InspectedBy  STRING(36),
  InspectedAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  CONSTRAINT CHK_FactoryQC_Result CHECK (Result IN ('PASS', 'FAIL')),
) PRIMARY KEY (RequestId);

CREATE INDEX Idx_FactorySupplyRequestQC_ByInspectedAt ON FactorySupplyRequestQC(InspectedAt DESC);
