CREATE TABLE RouteTwins (
  RouteId             STRING(36) NOT NULL,
  DriverId            STRING(36) NOT NULL,
  Status              STRING(32) NOT NULL,
  CurrentLat          FLOAT64,
  CurrentLng          FLOAT64,
  CurrentH3           STRING(16),
  LocationAt          TIMESTAMP,
  RemainingStops      INT64 NOT NULL,
  CapacityUsedWeight  FLOAT64,
  CapacityUsedVolume  FLOAT64,
  LastEventAt         TIMESTAMP NOT NULL,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (RouteId);

CREATE TABLE StopTwins (
  RouteId             STRING(36) NOT NULL,
  StopId              STRING(36) NOT NULL,
  Sequence            INT64 NOT NULL,
  Status              STRING(32) NOT NULL,
  PredictedArrival    TIMESTAMP,
  WindowStart         TIMESTAMP,
  WindowEnd           TIMESTAMP,
  DeliveredGrossMinor INT64,
  RemainingGrossMinor INT64,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (RouteId, StopId),
  INTERLEAVE IN PARENT RouteTwins ON DELETE CASCADE;

CREATE TABLE VehicleInventory (
  RouteId             STRING(36) NOT NULL,
  Sku                 STRING(64) NOT NULL,
  QtyOnVehicle        INT64 NOT NULL,
  UpdatedAt           TIMESTAMP NOT NULL,
) PRIMARY KEY (RouteId, Sku),
  INTERLEAVE IN PARENT RouteTwins ON DELETE CASCADE;
