-- Predictive ETAs
-- Stores computed arrival windows per route stop with confidence and factor breakdown.

CREATE TABLE RouteETAs (
  RouteId           STRING(36) NOT NULL,
  StopId            STRING(36) NOT NULL,
  Sequence          INT64 NOT NULL,
  PredictedArrival  TIMESTAMP NOT NULL,
  WindowStart       TIMESTAMP NOT NULL,
  WindowEnd         TIMESTAMP NOT NULL,
  Confidence        FLOAT64 NOT NULL,
  ComputedAt        TIMESTAMP NOT NULL,
  FactorsJson       JSON,
) PRIMARY KEY (RouteId, StopId);

CREATE INDEX RouteETAs_ByOrder ON RouteETAs (StopId);
