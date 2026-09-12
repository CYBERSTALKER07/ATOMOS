CREATE TABLE EvidenceDossiers (
  DossierId          STRING(36)  NOT NULL,
  TargetId           STRING(36)  NOT NULL,
  TargetType         STRING(32)  NOT NULL,
  Status             STRING(16)  NOT NULL,
  SealedAt           TIMESTAMP,
  SealedHash         STRING(128),
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DossierId);

CREATE INDEX IDX_EvidenceDossiers_Target ON EvidenceDossiers(TargetType, TargetId);

CREATE TABLE EvidenceItems (
  DossierId          STRING(36)  NOT NULL,
  ItemId             STRING(36)  NOT NULL,
  ItemType           STRING(32)  NOT NULL,
  StorageUri         STRING(256) NOT NULL,
  FileHash           STRING(128) NOT NULL,
  MimeType           STRING(128) NOT NULL,
  SizeBytes          INT64       NOT NULL,
  UploaderUserId     STRING(36)  NOT NULL,
  CapturedAt         TIMESTAMP,
  Latitude           FLOAT64,
  Longitude          FLOAT64,
  CreatedAt          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DossierId, ItemId),
  INTERLEAVE IN PARENT EvidenceDossiers ON DELETE CASCADE;
