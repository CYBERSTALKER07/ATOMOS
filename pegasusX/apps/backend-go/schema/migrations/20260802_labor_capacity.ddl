-- Labor Capacity Model + Driver Score
-- Tracks driver reliability scores, shift availability, and zone-level delivery capacity.

CREATE TABLE DriverScores (
  DriverId          STRING(36) NOT NULL,
  Score             FLOAT64 NOT NULL,
  OnTimeRate        FLOAT64 NOT NULL,
  CompletionRate    FLOAT64 NOT NULL,
  DamageRate        FLOAT64 NOT NULL,
  ShopClosedRate    FLOAT64 NOT NULL,
  FeedbackScore     FLOAT64 NOT NULL,
  StopsPerHour      FLOAT64 NOT NULL,
  WindowStart       DATE NOT NULL,
  WindowEnd         DATE NOT NULL,
  ComputedAt        TIMESTAMP NOT NULL,
) PRIMARY KEY (DriverId);

CREATE TABLE DriverAvailability (
  DriverId          STRING(36) NOT NULL,
  Date              DATE NOT NULL,
  AvailableHours    FLOAT64 NOT NULL,
  ZoneH3            STRING(16),
  Status            STRING(16) NOT NULL,
  UpdatedAt         TIMESTAMP NOT NULL,
) PRIMARY KEY (DriverId, Date);

CREATE TABLE ZoneCapacity (
  ZoneH3            STRING(16) NOT NULL,
  Date              DATE NOT NULL,
  TotalCapacity     FLOAT64 NOT NULL,
  UsedCapacity      FLOAT64 NOT NULL,
  ComputedAt        TIMESTAMP NOT NULL,
) PRIMARY KEY (ZoneH3, Date);
