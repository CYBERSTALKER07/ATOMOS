DROP INDEX UQ_PayoutBatches_SupplierPeriod;
CREATE UNIQUE INDEX UQ_PayoutBatches_SupplierPeriodCurrency ON PayoutBatches(SupplierId, PeriodStart, PeriodEnd, Currency);
