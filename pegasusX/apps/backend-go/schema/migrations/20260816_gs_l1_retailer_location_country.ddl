-- GS-L1: store matching needs pack country on RetailerLocations.
-- Not a tenant key. Empty is geography_incomplete at resolve time.

ALTER TABLE RetailerLocations ADD COLUMN CountryCode STRING(2);
