-- Human-readable addresses for topology + retailer delivery (coords remain for dispatch).
ALTER TABLE Warehouses ADD COLUMN Address STRING(MAX);
ALTER TABLE Warehouses ADD COLUMN PlaceId STRING(128);

ALTER TABLE Factories ADD COLUMN Address STRING(MAX);
ALTER TABLE Factories ADD COLUMN PlaceId STRING(128);

ALTER TABLE Retailers ADD COLUMN DeliveryAddress STRING(MAX);
ALTER TABLE Retailers ADD COLUMN PlaceId STRING(128);
