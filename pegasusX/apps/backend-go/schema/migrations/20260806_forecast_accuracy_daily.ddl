-- §8.4 Forecast accuracy monitoring — baseline vs completed SKU-day units.

CREATE TABLE ForecastAccuracyDaily (
  SupplierId      STRING(36)  NOT NULL,
  ForecastDate    DATE        NOT NULL,
  WarehouseId     STRING(36)  NOT NULL,
  ProductId       STRING(36)  NOT NULL,
  ForecastQty     INT64       NOT NULL,
  ActualQty       INT64       NOT NULL,
  AbsError        INT64       NOT NULL,
  SignedError     INT64       NOT NULL,
  Wape7           FLOAT64,
  Wape28          FLOAT64,
  Bias7           FLOAT64,
  Bias28          FLOAT64,
  TrackingSignal  FLOAT64,
  SampleDays7     INT64       NOT NULL DEFAULT (0),
  SampleDays28    INT64       NOT NULL DEFAULT (0),
  AlertTs         BOOL        NOT NULL DEFAULT (FALSE),
  ComputedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ForecastDate, WarehouseId, ProductId);

CREATE INDEX Idx_ForecastAccuracyDaily_BySupplierDate
  ON ForecastAccuracyDaily(SupplierId, ForecastDate DESC);
