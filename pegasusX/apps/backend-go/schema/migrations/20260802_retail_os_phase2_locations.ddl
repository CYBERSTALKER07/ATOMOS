-- Retail OS Phase 2: multi-location stores + staff scope
CREATE TABLE RetailerLocations (
  LocationId             STRING(36)  NOT NULL,
  RetailerId             STRING(36)  NOT NULL,
  Name                   STRING(255) NOT NULL,
  DeliveryAddress        STRING(MAX),
  PlaceId                STRING(128),
  Lat                    FLOAT64,
  Lng                    FLOAT64,
  H3Cell                 STRING(15),
  ReceivingWindowOpen    STRING(10),
  ReceivingWindowClose   STRING(10),
  IsPrimary              BOOL        NOT NULL,
  IsActive               BOOL        NOT NULL,
  CreatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt              TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LocationId);

CREATE INDEX Idx_RetailerLocations_ByRetailer ON RetailerLocations(RetailerId, IsActive, IsPrimary DESC, UpdatedAt DESC);

CREATE TABLE RetailerUserLocations (
  UserId      STRING(36) NOT NULL,
  LocationId  STRING(36) NOT NULL,
  RetailerId  STRING(36) NOT NULL,
  CreatedAt   TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId, LocationId);

CREATE INDEX Idx_RetailerUserLocations_ByRetailerUser ON RetailerUserLocations(RetailerId, UserId);
CREATE INDEX Idx_RetailerUserLocations_ByLocation ON RetailerUserLocations(LocationId);
