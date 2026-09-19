-- Domain 5.1: dual-control for money-affecting feature flags.
-- A money-flag override is written PENDING and does not take effect until a
-- DIFFERENT PLATFORM_ADMIN approves it (ApprovedBy set, Status -> ACTIVE).
-- Non-money flags continue to apply immediately (Status ACTIVE).
ALTER TABLE FeatureFlagOverrides ADD COLUMN Status STRING(16) NOT NULL DEFAULT ('ACTIVE');
ALTER TABLE FeatureFlagOverrides ADD COLUMN ApprovedBy STRING(128);
ALTER TABLE FeatureFlagOverrides ADD COLUMN ApprovedAt TIMESTAMP;
