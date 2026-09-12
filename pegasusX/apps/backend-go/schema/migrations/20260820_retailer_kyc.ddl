-- Phase 5.1 KYC
CREATE TABLE RetailerKycDocuments (
  DocumentId      STRING(36)  NOT NULL,
  RetailerId      STRING(36)  NOT NULL,
  Status          STRING(32)  NOT NULL,
  DocumentType    STRING(64)  NOT NULL,
  DocumentUrl     STRING(MAX) NOT NULL,
  SubmittedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ReviewedAt      TIMESTAMP,
  ReviewedBy      STRING(36),
  RejectionReason STRING(MAX),
) PRIMARY KEY (DocumentId);

CREATE INDEX Idx_RetailerKycDocuments_ByRetailer ON RetailerKycDocuments(RetailerId, SubmittedAt DESC);
CREATE INDEX Idx_RetailerKycDocuments_ByStatus ON RetailerKycDocuments(Status, SubmittedAt ASC);
