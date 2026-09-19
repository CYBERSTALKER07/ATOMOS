-- §8.3 Shadow auto-order proposal ledger + acceptance stats.
CREATE TABLE RetailerAutoOrderShadowProposals (
  RetailerId     STRING(64)  NOT NULL,
  ProposalId     STRING(64)  NOT NULL,
  Sku            STRING(64)  NOT NULL,
  SupplierId     STRING(64),
  ProposedQty    INT64       NOT NULL,
  IP             FLOAT64,
  ReorderPoint   FLOAT64,
  OrderUpTo      FLOAT64,
  Confidence     FLOAT64,
  Reason         STRING(64),
  BucketDate     DATE        NOT NULL,
  Status         STRING(32)  NOT NULL,
  RunId          STRING(64),
  MatchedOrderId STRING(64),
  MatchedQty     INT64,
  AcceptedUnmod  BOOL,
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, ProposalId);

CREATE INDEX Idx_RetailerAutoOrderShadow_ByRetailerBucket
  ON RetailerAutoOrderShadowProposals (RetailerId, BucketDate DESC);

CREATE INDEX Idx_RetailerAutoOrderShadow_ByRetailerSkuBucket
  ON RetailerAutoOrderShadowProposals (RetailerId, Sku, BucketDate DESC);
