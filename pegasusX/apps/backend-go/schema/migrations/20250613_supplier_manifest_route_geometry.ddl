-- Persist densified route overlay at manifest seal / reorder for driver map reads.
ALTER TABLE SupplierTruckManifests ADD COLUMN EncodedRoutePolyline STRING(MAX);
ALTER TABLE SupplierTruckManifests ADD COLUMN RouteGeometrySource STRING(32);
