CREATE TABLE DemandSignals (
  SignalId        STRING(36) NOT NULL,
  Type            STRING(32) NOT NULL,
  Scope           STRING(64) NOT NULL,
  Sku             STRING(64),
  StartAt         TIMESTAMP NOT NULL,
  EndAt           TIMESTAMP NOT NULL,
  Multiplier      FLOAT64 NOT NULL,
  Meta            JSON,
  CreatedAt       TIMESTAMP NOT NULL,
  CreatedBy       STRING(64) NOT NULL,
) PRIMARY KEY (SignalId);

CREATE INDEX DemandSignals_ByScopeTime ON DemandSignals (Scope, StartAt, EndAt);

CREATE TABLE DemandAdjustments (
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  Date            DATE NOT NULL,
  BaseVelocity    FLOAT64 NOT NULL,
  Adjustment      FLOAT64 NOT NULL,
  AdjustedDemand  FLOAT64 NOT NULL,
  FactorsJson     JSON,
  ComputedAt      TIMESTAMP NOT NULL,
) PRIMARY KEY (RetailerId, Sku, Date);
