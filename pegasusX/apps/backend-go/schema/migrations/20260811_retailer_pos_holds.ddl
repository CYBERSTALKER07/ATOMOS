-- Wave C3.1: Parked POS carts (holds). Never touch stock / OnHand.
-- Resume restricted to same LocationId. Default TTL 24h (enforced in app + ExpiresAt).
-- Flag: POS_HOLDS_ENABLED (default off).

CREATE TABLE RetailerPosHolds (
  HoldId        STRING(36)  NOT NULL,
  RetailerId    STRING(36)  NOT NULL,
  LocationId    STRING(36)  NOT NULL,
  RegisterId    STRING(36),
  UserId        STRING(36)  NOT NULL,
  Status        STRING(16)  NOT NULL,
  CartJson      STRING(MAX) NOT NULL,
  Note          STRING(512),
  ExpiresAt     TIMESTAMP   NOT NULL,
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ResumedAt     TIMESTAMP,
  VoidedAt      TIMESTAMP
) PRIMARY KEY (RetailerId, HoldId);

CREATE INDEX Idx_RetailerPosHolds_ByLocationStatus
  ON RetailerPosHolds (RetailerId, LocationId, Status, ExpiresAt);

CREATE INDEX Idx_RetailerPosHolds_ByExpires
  ON RetailerPosHolds (Status, ExpiresAt);
