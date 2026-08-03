-- Retailer-owned local/manual SKUs for POS between wholesales (never supplier demand).
CREATE TABLE RetailerLocalCatalog (
  RetailerId         STRING(64) NOT NULL,
  LocalSkuId         STRING(128) NOT NULL,
  Barcode            STRING(64),
  Name               STRING(256) NOT NULL,
  Unit               STRING(32),
  DefaultPriceMinor  INT64 NOT NULL DEFAULT (0),
  Currency           STRING(8),
  SectionId          STRING(64),
  IsActive           BOOL NOT NULL DEFAULT (true),
  CreatedAt          TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt          TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (RetailerId, LocalSkuId);

CREATE INDEX Idx_RetailerLocalCatalog_Barcode
  ON RetailerLocalCatalog (RetailerId, Barcode);
