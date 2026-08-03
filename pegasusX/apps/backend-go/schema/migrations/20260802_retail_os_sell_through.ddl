-- L3 sell-through flywheel: daily POS/store velocity per retailer location SKU.
CREATE TABLE RetailerSellThroughDaily (
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  SkuId        STRING(64)  NOT NULL,
  Day          DATE        NOT NULL,
  QtySold      INT64       NOT NULL,
  QtyVoided    INT64       NOT NULL,
  QtyOnHandEod INT64,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, LocationId, SkuId, Day);

CREATE INDEX Idx_RetailerSellThroughDaily_ByRetailerDay
  ON RetailerSellThroughDaily(RetailerId, Day DESC);
