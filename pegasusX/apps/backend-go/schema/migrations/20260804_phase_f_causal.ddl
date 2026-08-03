CREATE TABLE EventDensitySignals (
  ZoneH3 STRING(15) NOT NULL,
  Date DATE NOT NULL,
  DensityScore FLOAT64,
  EventsJson JSON,
  ComputedAt TIMESTAMP,
) PRIMARY KEY (ZoneH3, Date);
