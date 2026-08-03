-- Offline POS: durable client_sale_id uniqueness for offline cash sync.
ALTER TABLE RetailerPosSales ADD COLUMN ClientSaleId STRING(36);
ALTER TABLE RetailerPosSales ADD COLUMN Origin STRING(16);
ALTER TABLE RetailerPosSales ADD COLUMN ClientCreatedAt TIMESTAMP;

CREATE UNIQUE NULL_FILTERED INDEX UQ_RetailerPosSales_ClientSale
  ON RetailerPosSales(RetailerId, ClientSaleId);
