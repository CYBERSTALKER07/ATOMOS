-- Gate 5 / §8.10 Phase 2: ParentOrders + Orders.ParentOrderId
-- Mixed-supplier cart checkout creates one parent and N tenant-scoped child Orders.

CREATE TABLE ParentOrders (
  ParentOrderId  STRING(36)    NOT NULL,
  RetailerId     STRING(36)    NOT NULL,
  Status         STRING(32)    NOT NULL,
  Currency       STRING(3)     NOT NULL,
  TotalMinor     INT64         NOT NULL DEFAULT (0),
  ChildCount     INT64         NOT NULL DEFAULT (0),
  CreatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ParentOrderId);

CREATE INDEX Idx_ParentOrders_ByRetailerCreated ON ParentOrders(RetailerId, CreatedAt DESC);

ALTER TABLE Orders ADD COLUMN ParentOrderId STRING(36);

CREATE INDEX Idx_Orders_ByParentOrder ON Orders(ParentOrderId, CreatedAt DESC);
