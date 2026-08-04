-- Align RetailerCreditProfiles with schema/spanner.ddl ReservedMinor (reservation holds).
ALTER TABLE RetailerCreditProfiles ADD COLUMN ReservedMinor INT64 NOT NULL DEFAULT (0);
