-- T9: Migrate DraftAdjustments and DraftCreditNotes ProposedAmount NUMERIC to
-- INT64 minor units (UZS Tiyin, Invariant 1) via ProposedAmountMinor.
-- Greenfield DBs already contain ProposedAmountMinor (skipped by schemadrift);
-- legacy DBs add ProposedAmountMinor first and drop NUMERIC ProposedAmount.

ALTER TABLE DraftAdjustments ADD COLUMN ProposedAmountMinor INT64 NOT NULL DEFAULT (0);
ALTER TABLE DraftAdjustments DROP COLUMN ProposedAmount;
ALTER TABLE DraftAdjustments ALTER COLUMN ReviewedAt SET OPTIONS (allow_commit_timestamp=true);

ALTER TABLE DraftCreditNotes ADD COLUMN ProposedAmountMinor INT64 NOT NULL DEFAULT (0);
ALTER TABLE DraftCreditNotes DROP COLUMN ProposedAmount;
ALTER TABLE DraftCreditNotes ALTER COLUMN ReviewedAt SET OPTIONS (allow_commit_timestamp=true);
