-- GS-T3: KYB dual-control APPROVE pack+cell. Missing row ≠ active.
-- Nullable: empty pack is not a chosen market.

ALTER TABLE PlatformTenants ADD COLUMN MarketCode STRING(8);
ALTER TABLE PlatformTenants ADD COLUMN HomeCell STRING(32);
ALTER TABLE PlatformTenants ADD COLUMN RequestedBy STRING(128);
ALTER TABLE PlatformTenants ADD COLUMN ApprovedBy STRING(128);
