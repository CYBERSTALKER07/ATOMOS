-- §8.2 Safety stock: observed lead time + policy service-level knobs.

ALTER TABLE FactoryInternalTransfers ADD COLUMN ReceivedAt TIMESTAMP;

ALTER TABLE ReplenishmentPolicies ADD COLUMN TargetServiceLevel FLOAT64 NOT NULL DEFAULT (0.98);
ALTER TABLE ReplenishmentPolicies ADD COLUMN LeadTimeDays INT64 NOT NULL DEFAULT (2);
ALTER TABLE ReplenishmentPolicies ADD COLUMN LeadTimeSigmaDays FLOAT64 NOT NULL DEFAULT (1.0);
