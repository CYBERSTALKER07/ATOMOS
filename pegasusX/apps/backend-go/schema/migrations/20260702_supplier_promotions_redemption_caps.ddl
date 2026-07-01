-- Promotion redemption caps used by catalog pricing and order checkout.

ALTER TABLE SupplierPromotions ADD COLUMN MaxRedemptions INT64;
ALTER TABLE SupplierPromotions ADD COLUMN CurrentRedemptions INT64 NOT NULL DEFAULT (0);
