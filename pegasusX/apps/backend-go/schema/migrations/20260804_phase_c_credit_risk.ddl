CREATE TABLE RetailerCreditScores (
  RetailerId STRING(MAX) NOT NULL,
  Score INT64 NOT NULL,
  RiskTier STRING(64) NOT NULL,
  SuggestedLimitMinor INT64 NOT NULL,
  FactorsJson JSON,
  WindowStart TIMESTAMP NOT NULL,
  WindowEnd TIMESTAMP NOT NULL,
  ComputedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(RetailerId);
