-- Store hashed onboarding credentials on pending JoinRequests so approved
-- joiners receive a login-capable SupplierUsers / RetailerUsers row.
-- Lockstep with apps/backend-go/schema/spanner.ddl.

ALTER TABLE JoinRequests ADD COLUMN requester_password_hash STRING(255);
