-- Phase 5.2 Returns
CREATE TABLE RetailerReturnRequests (
  RequestId       STRING(36)  NOT NULL,
  RetailerId      STRING(36)  NOT NULL,
  OrderId         STRING(36)  NOT NULL,
  Status          STRING(32)  NOT NULL,
  LinesJson       JSON        NOT NULL,
  Reason          STRING(MAX) NOT NULL,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP,
) PRIMARY KEY (RequestId);

CREATE INDEX Idx_RetailerReturnRequests_ByRetailer ON RetailerReturnRequests(RetailerId, CreatedAt DESC);
CREATE INDEX Idx_RetailerReturnRequests_ByOrder ON RetailerReturnRequests(OrderId);
