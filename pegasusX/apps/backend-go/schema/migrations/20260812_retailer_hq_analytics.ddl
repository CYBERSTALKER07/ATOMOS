-- Wave C2.1: Franchise HQ daily sales + stock snapshot (Spanner source of truth).
-- Writers run on POS sale/void in the same Apply as the sale ledger when Spanner is used.
-- Include local: SKUs in sales; exclude from supplier demand elsewhere.
-- Reads gated later by HQ_ANALYTICS_ENABLED (C2.2).

CREATE TABLE RetailerHqSalesDaily (
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  Day          DATE        NOT NULL,
  SkuId        STRING(128) NOT NULL,
  QtySold      INT64       NOT NULL,
  QtyVoided    INT64       NOT NULL,
  GrossMinor   INT64       NOT NULL,
  NetMinor     INT64       NOT NULL,
  Currency     STRING(8)   NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, Day, LocationId, SkuId);

CREATE INDEX Idx_RetailerHqSalesDaily_ByDay
  ON RetailerHqSalesDaily (RetailerId, Day, LocationId);

CREATE TABLE RetailerHqStockSnapshot (
  RetailerId   STRING(36)  NOT NULL,
  LocationId   STRING(36)  NOT NULL,
  SkuId        STRING(128) NOT NULL,
  OnHand       INT64       NOT NULL,
  Reserved     INT64       NOT NULL,
  AsOf         TIMESTAMP   NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocationId, SkuId);
