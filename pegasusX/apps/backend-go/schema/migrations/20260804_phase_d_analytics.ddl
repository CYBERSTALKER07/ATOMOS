CREATE TABLE RoutePerformanceAnalytics (
  RouteId STRING(36) NOT NULL,
  DriverId STRING(36),
  PlannedStops INT64,
  ActualStops INT64,
  PlannedDurationSec INT64,
  ActualDurationSec INT64,
  ReplanCount INT64,
  ComputedAt TIMESTAMP,
) PRIMARY KEY (RouteId);
