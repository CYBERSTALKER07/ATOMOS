-- Allow commit timestamps on return-gate receive columns written via spanner.CommitTimestamp.

ALTER TABLE SupplierReturns ALTER COLUMN ReceivedAt SET OPTIONS (allow_commit_timestamp=true);
ALTER TABLE SupplierReturns ALTER COLUMN ResolvedAt SET OPTIONS (allow_commit_timestamp=true);
ALTER TABLE ReturnReceiveSessions ALTER COLUMN CompletedAt SET OPTIONS (allow_commit_timestamp=true);
