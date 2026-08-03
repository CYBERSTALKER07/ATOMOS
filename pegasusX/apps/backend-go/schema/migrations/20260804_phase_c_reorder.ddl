CREATE TABLE ReorderSuggestions (
  RetailerId      STRING(36) NOT NULL,
  Sku             STRING(64) NOT NULL,
  SuggestedQty    INT64 NOT NULL,
  AdjustedDemand  FLOAT64 NOT NULL,
  CurrentStock    INT64 NOT NULL,
  InFlightQty     INT64 NOT NULL,
  SafetyStock     FLOAT64 NOT NULL,
  SuggestedByDate DATE NOT NULL,
  ComputedAt      TIMESTAMP NOT NULL,
  Status          STRING(16) NOT NULL,
) PRIMARY KEY (RetailerId, Sku);
