-- Allow commit timestamps on MigrationProjects.cutover_at written via spanner.CommitTimestamp.

ALTER TABLE MigrationProjects ALTER COLUMN cutover_at SET OPTIONS (allow_commit_timestamp=true);
