-- Gate-3 Wave 2C: GS1 GLN on orgs + ManifestShipUnits (SSCC) + supplier company prefix.

ALTER TABLE SupplierProfiles ADD COLUMN Gln STRING(13);
ALTER TABLE SupplierProfiles ADD COLUMN Gs1CompanyPrefix STRING(10);

ALTER TABLE Warehouses ADD COLUMN Gln STRING(13);

ALTER TABLE Retailers ADD COLUMN Gln STRING(13);

ALTER TABLE RetailerLocations ADD COLUMN Gln STRING(13);

CREATE TABLE ManifestShipUnits (
  ManifestId  STRING(36)  NOT NULL,
  ShipUnitId  STRING(36)  NOT NULL,
  Sscc        STRING(18)  NOT NULL,
  OrderId     STRING(36)  NOT NULL,
  Sequence    INT64       NOT NULL DEFAULT (0),
  Gtin        STRING(14),
  CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ManifestId, ShipUnitId),
  INTERLEAVE IN PARENT SupplierTruckManifests ON DELETE CASCADE;

CREATE UNIQUE INDEX UQ_ManifestShipUnits_Sscc ON ManifestShipUnits(Sscc);

CREATE UNIQUE INDEX UQ_ManifestShipUnits_ByOrder
  ON ManifestShipUnits(ManifestId, OrderId);
