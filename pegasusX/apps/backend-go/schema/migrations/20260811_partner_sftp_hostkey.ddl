-- Phase 2: SFTP host-key pinning for partner SFTP destinations.
ALTER TABLE PartnerSftpConfigs ADD COLUMN HostKeySha256 STRING(128);
