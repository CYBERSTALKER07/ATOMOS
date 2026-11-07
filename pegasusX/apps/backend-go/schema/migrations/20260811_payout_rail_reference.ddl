-- Domain 1.4 live payout rail: track the settlement reference a live rail
-- returns on dispatch, reconciled by the settlement webhook. Nullable — file
-- (bank-file) rail batches leave it empty.
ALTER TABLE PayoutBatches ADD COLUMN RailReference STRING(128);
