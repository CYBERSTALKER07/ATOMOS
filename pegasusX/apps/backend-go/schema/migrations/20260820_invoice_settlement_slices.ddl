CREATE TABLE InvoiceSettlementSlices (
  SliceId            STRING(36)  NOT NULL,
  OrderId            STRING(36)  NOT NULL,
  SupplierId         STRING(36)  NOT NULL,
  CapturedLegId      STRING(36)  NOT NULL,
  GrossMinor         INT64       NOT NULL,
  CommissionMinor    INT64       NOT NULL,
  NetPayoutMinor     INT64       NOT NULL,
  Currency           STRING(8)   NOT NULL,
  PayoutBatchId      STRING(36),
  Status             STRING(16)  NOT NULL,  -- UNSETTLED | BATCHED
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SliceId);

CREATE INDEX IDX_InvoiceSettlementSlices_Supplier_Status ON InvoiceSettlementSlices(SupplierId, Status);
CREATE INDEX IDX_InvoiceSettlementSlices_PayoutBatch ON InvoiceSettlementSlices(PayoutBatchId);
