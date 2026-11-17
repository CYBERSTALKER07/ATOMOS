-- B2B Operating Model End-to-End Closure Schema (Workstreams W0-W5)
-- Authoritative lockstep with apps/backend-go/schema/spanner.ddl

-- W0: Multi-pod distributed worker leasing (replaces fail-open Redis heartbeat fallback)
CREATE TABLE WorkerLeases (
  LeaseName  STRING(128) NOT NULL,
  HolderId   STRING(128) NOT NULL,
  ExpiresAt  TIMESTAMP   NOT NULL,
  UpdatedAt  TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (LeaseName);

-- W0: Spanner persistence for Payme Merchant JSON-RPC transactions (eliminates replica RAM)
CREATE TABLE PaymeMerchantTransactions (
  PaymeId      STRING(128) NOT NULL,
  OrderId      STRING(36)  NOT NULL,
  TargetType   STRING(32)  NOT NULL DEFAULT ('ORDER'),
  AmountMinor  INT64       NOT NULL,
  State        INT64       NOT NULL,
  CreateTime   INT64       NOT NULL,
  PerformTime  INT64       NOT NULL DEFAULT (0),
  CancelTime   INT64       NOT NULL DEFAULT (0),
  Reason       INT64,
  MerchantTx   STRING(64)  NOT NULL,
  UpdatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PaymeId);

CREATE INDEX Idx_PaymeMerchantTx_ByOrder ON PaymeMerchantTransactions(OrderId, CreateTime DESC);
CREATE INDEX Idx_PaymeMerchantTx_ByCreateTime ON PaymeMerchantTransactions(CreateTime);

-- W0, W2, W5: Orders idempotency key, delivery completion timestamp, opt-in auto-confirm & query indexes
ALTER TABLE Orders ADD COLUMN IdempotencyKey STRING(128);
ALTER TABLE Orders ADD COLUMN CompletedAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN AutoConfirmOptIn BOOL DEFAULT (false);

CREATE UNIQUE NULL_FILTERED INDEX UQ_Orders_ByRetailerIdempotency ON Orders(RetailerId, SupplierId, IdempotencyKey);
CREATE INDEX Idx_Orders_ByStatusSourceDeliveryDate ON Orders(Status, OrderSource, RequestedDeliveryDate DESC);
CREATE INDEX Idx_Orders_ByRetailerStatusCreated ON Orders(RetailerId, Status, CreatedAt DESC) STORING (SupplierId, TotalMinor, Currency, RequestedDeliveryDate, ConfirmationStatus);

-- W0: Fix missing RetailerAutoOrderSettings.GlobalEnabled column queried by auto-order sweepers
ALTER TABLE RetailerAutoOrderSettings ADD COLUMN GlobalEnabled BOOL DEFAULT (false);
CREATE INDEX Idx_RetailerAutoOrderSettings_ByEnabled ON RetailerAutoOrderSettings(GlobalEnabled, RetailerId);

-- W2: Seller mandate daily cap counter (Usage -> CreditProfile -> Orders lock order) & owner approvals
CREATE TABLE RetailerSellerDailyUsage (
  RetailerId       STRING(36) NOT NULL,
  UserId           STRING(64) NOT NULL,
  UsageDate        DATE       NOT NULL,
  SpentCreditMinor INT64      NOT NULL DEFAULT (0),
  SpentCashMinor   INT64      NOT NULL DEFAULT (0),
  OrderCount       INT64      NOT NULL DEFAULT (0),
  UpdatedAt        TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, UserId, UsageDate);

CREATE TABLE RetailerApprovalRequests (
  RequestId        STRING(36)  NOT NULL,
  RetailerId       STRING(36)  NOT NULL,
  SupplierId       STRING(36),
  RequestedByUser  STRING(64)  NOT NULL,
  ClientOrderId    STRING(128) NOT NULL,
  AmountMinor      INT64       NOT NULL,
  TrippedRule      STRING(64)  NOT NULL,
  Status           STRING(24)  NOT NULL DEFAULT ('PENDING_OWNER'),
  OrderPayloadJson STRING(MAX) NOT NULL,
  ApprovedByUser   STRING(64),
  DecisionReason   STRING(512),
  ExpiresAt        TIMESTAMP   NOT NULL,
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RequestId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_RetailerApprovals_ClientOrder ON RetailerApprovalRequests(RetailerId, ClientOrderId);
CREATE INDEX Idx_RetailerApprovals_ByRetailerStatus ON RetailerApprovalRequests(RetailerId, Status, CreatedAt DESC);

-- W2: Claim evidence GCS generation pinning + nullable SHA-256 duplicate prevention (NULL_FILTERED avoids hotspots)
ALTER TABLE ClaimEvidences ADD COLUMN ContentSha256 STRING(64);
ALTER TABLE ClaimEvidences ADD COLUMN UriSha256 STRING(64);
ALTER TABLE ClaimEvidences ADD COLUMN ObjectGeneration INT64;

CREATE NULL_FILTERED INDEX Idx_ClaimEvidences_ByContentSha256 ON ClaimEvidences(ContentSha256);
CREATE UNIQUE NULL_FILTERED INDEX UQ_ClaimEvidences_ByUriSha256 ON ClaimEvidences(UriSha256);

-- W2, W3: Prepay downgrade flag (replaces invalid status) + nullable quarterly promise uniqueness
ALTER TABLE RetailerCreditProfiles ADD COLUMN PrepayOnlyLocked BOOL DEFAULT (false);
ALTER TABLE RetailerCreditProfiles ADD COLUMN ActiveQuarterKey STRING(16);

CREATE UNIQUE NULL_FILTERED INDEX UQ_RetailerCreditProfiles_ActiveQuarter ON RetailerCreditProfiles(RetailerId, SupplierId, ActiveQuarterKey);

-- W3: AR payment promises & advance balance ledger
CREATE TABLE ArPaymentPromises (
  PromiseId       STRING(36) NOT NULL,
  RetailerId      STRING(36) NOT NULL,
  SupplierId      STRING(36) NOT NULL,
  InvoiceId       STRING(36),
  QuarterKey      STRING(16) NOT NULL,
  PromisedAt      TIMESTAMP  NOT NULL,
  DueAt           TIMESTAMP  NOT NULL,
  PromisedMinor   INT64      NOT NULL,
  Status          STRING(24) NOT NULL DEFAULT ('ACTIVE'),
  ApprovedBy      STRING(64),
  CreatedAt       TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PromiseId);

CREATE INDEX Idx_ArPaymentPromises_ByRetailerQuarter ON ArPaymentPromises(RetailerId, SupplierId, QuarterKey, Status);
CREATE INDEX Idx_ArPaymentPromises_ByStatusDue ON ArPaymentPromises(Status, DueAt);

CREATE TABLE ArAdvanceBalances (
  RetailerId    STRING(36) NOT NULL,
  SupplierId    STRING(36) NOT NULL,
  BalanceMinor  INT64      NOT NULL DEFAULT (0),
  Currency      STRING(3)  NOT NULL DEFAULT ('UZS'),
  UpdatedAt     TIMESTAMP  NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, SupplierId);

-- W4: Idempotent 1C EnterpriseData pull/ack/reject ledger bridge (replaces replica RAM onecSeen)
CREATE TABLE PartnerOnecPostings (
  PostingId       STRING(36)  NOT NULL,
  SupplierId      STRING(36)  NOT NULL,
  DocumentKey     STRING(128) NOT NULL,
  DocumentType    STRING(32)  NOT NULL,
  ExportJobId     STRING(36),
  PayloadJson     STRING(MAX) NOT NULL,
  ExposureMinor   INT64       NOT NULL DEFAULT (0),
  Status          STRING(24)  NOT NULL DEFAULT ('PENDING'),
  OnecRef         STRING(128),
  RejectReason    STRING(512),
  Attempts        INT64       NOT NULL DEFAULT (0),
  AckedAt         TIMESTAMP,
  RejectedAt      TIMESTAMP,
  CreatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (PostingId);

CREATE UNIQUE INDEX UQ_PartnerOnecPostings_DocKey ON PartnerOnecPostings(SupplierId, DocumentKey);
CREATE INDEX Idx_PartnerOnecPostings_BySupplierStatus ON PartnerOnecPostings(SupplierId, Status, CreatedAt ASC);

-- W4: Tenant-safe supplier-scoped retailer links for deterministic TIN + phone dedup
CREATE TABLE SupplierRetailerLinks (
  SupplierId        STRING(36)  NOT NULL,
  RetailerId        STRING(36)  NOT NULL,
  TaxIdTIN          STRING(32),
  NormalizedPhone   STRING(32),
  LegalName         STRING(255),
  DedupClusterId    STRING(36),
  DedupStatus       STRING(24)  NOT NULL DEFAULT ('CLEAN'),
  MergedIntoId      STRING(36),
  CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, RetailerId);

CREATE NULL_FILTERED INDEX Idx_SupplierRetailerLinks_ByTIN ON SupplierRetailerLinks(SupplierId, TaxIdTIN);
CREATE NULL_FILTERED INDEX Idx_SupplierRetailerLinks_ByPhone ON SupplierRetailerLinks(SupplierId, NormalizedPhone);
CREATE INDEX Idx_SupplierRetailerLinks_ByDedup ON SupplierRetailerLinks(SupplierId, DedupStatus, UpdatedAt DESC);

-- W5 (D-11): One-tap OTP / phone-share Telegram binding & locale preference
CREATE TABLE RetailerTelegramLinks (
  RetailerId       STRING(36)  NOT NULL,
  UserId           STRING(64)  NOT NULL,
  PhoneNumber      STRING(32),
  TelegramChatId   STRING(64),
  OtpHash          STRING(128),
  Locale           STRING(8)   NOT NULL DEFAULT ('uz'),
  VerifiedAt       TIMESTAMP,
  ExpiresAt        TIMESTAMP,
  UpdatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (RetailerId, UserId);

CREATE UNIQUE NULL_FILTERED INDEX UQ_RetailerTelegramLinks_ChatId ON RetailerTelegramLinks(TelegramChatId);
CREATE NULL_FILTERED INDEX Idx_RetailerTelegramLinks_Phone ON RetailerTelegramLinks(PhoneNumber);
CREATE INDEX Idx_RetailerTelegramLinks_ByRetailerVerified ON RetailerTelegramLinks(RetailerId, VerifiedAt DESC);
