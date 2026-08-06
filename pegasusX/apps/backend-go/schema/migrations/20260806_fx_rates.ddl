-- Theatre #13 Wave 1: FX rate table (integer scaled rates; no silent 1:1 conversion).

CREATE TABLE FxRates (
  RateId         STRING(36)  NOT NULL,
  BaseCurrency   STRING(3)   NOT NULL,
  QuoteCurrency  STRING(3)   NOT NULL,
  RateScaled     INT64       NOT NULL,
  Scale          INT64       NOT NULL DEFAULT (100000000),
  EffectiveAt    TIMESTAMP   NOT NULL,
  Source         STRING(32)  NOT NULL,
  CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RateId);

CREATE UNIQUE INDEX Idx_FxRates_PairEffective
  ON FxRates (BaseCurrency, QuoteCurrency, EffectiveAt);

CREATE INDEX Idx_FxRates_PairEffectiveDesc
  ON FxRates (BaseCurrency, QuoteCurrency, EffectiveAt DESC);
