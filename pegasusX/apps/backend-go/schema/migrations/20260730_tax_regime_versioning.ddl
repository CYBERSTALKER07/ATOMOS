-- Tax regime versions (immutable once published)
CREATE TABLE TaxRegimeVersions (
  Id              STRING(36) NOT NULL,          -- UUID
  CountryCode     STRING(2) NOT NULL,           -- 'UZ'
  EffectiveFrom   TIMESTAMP NOT NULL,
  EffectiveTo     TIMESTAMP,                    -- NULL = currently active
  Currency        STRING(3) NOT NULL,           -- 'UZS'
  VatRateBps      INT64 NOT NULL,               -- e.g. 1200 = 12.00%
  Simplified      BOOL NOT NULL DEFAULT false,
  RulesJson       JSON,                         -- thresholds, exemptions, etc.
  CreatedAt       TIMESTAMP NOT NULL,
  CreatedBy       STRING(64) NOT NULL,
  UpdatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (Id);

CREATE INDEX TaxRegimeVersions_ByCountryEffective
ON TaxRegimeVersions (CountryCode, EffectiveFrom DESC);

-- Snapshot stamped on every order line at the moment of COMPLETED
CREATE TABLE OrderLineFiscalSnapshots (
  OrderId         STRING(36) NOT NULL,
  OrderLineId     STRING(36) NOT NULL,
  RegimeId        STRING(36) NOT NULL,          -- FK → TaxRegimeVersions.Id
  VatRateBps      INT64 NOT NULL,               -- denormalised for fast audit
  NetMinor        INT64 NOT NULL,               -- line net in tiyin
  VatMinor        INT64 NOT NULL,
  GrossMinor      INT64 NOT NULL,
  SnapshotAt      TIMESTAMP NOT NULL,           -- = order completion time
  CreatedAt       TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderId, OrderLineId),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;

CREATE INDEX OrderLineFiscalSnapshots_ByRegime
ON OrderLineFiscalSnapshots (RegimeId);
