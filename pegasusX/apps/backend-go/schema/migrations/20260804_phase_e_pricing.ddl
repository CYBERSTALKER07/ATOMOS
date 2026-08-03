CREATE TABLE PriceLists (
  PriceListId     STRING(36) NOT NULL,
  SupplierId      STRING(36) NOT NULL,
  Name            STRING(128) NOT NULL,
  EffectiveFrom   TIMESTAMP NOT NULL,
  EffectiveTo     TIMESTAMP,
) PRIMARY KEY (PriceListId);

CREATE TABLE PriceListItems (
  PriceListId     STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  UnitPriceMinor  INT64 NOT NULL,
  MinQty          INT64,
) PRIMARY KEY (PriceListId, Sku),
  INTERLEAVE IN PARENT PriceLists ON DELETE CASCADE;
