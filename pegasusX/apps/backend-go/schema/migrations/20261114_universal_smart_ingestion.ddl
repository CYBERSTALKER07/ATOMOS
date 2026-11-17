-- Universal Smart Ingestion & Cutover System (M1.1)
-- Lockstep with apps/backend-go/schema/spanner.ddl

-- 1. Projects (orchestrates dependency order across packs; CUTOVER only in this release)
CREATE TABLE MigrationProjects (
    supplier_id           STRING(36)  NOT NULL,
    project_id            STRING(36)  NOT NULL,
    name                  STRING(255) NOT NULL,
    mode                  STRING(24)  NOT NULL DEFAULT ('CUTOVER'),
    status                STRING(32)  NOT NULL DEFAULT ('DRAFT'),
    checklist_json        JSON,
    default_warehouse_id  STRING(36),
    created_by            STRING(64)  NOT NULL,
    cutover_at            TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    created_at            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_MigrationProjectMode CHECK (mode IN ('CUTOVER')),
    CONSTRAINT CHK_MigrationProjectStatus CHECK (
        status IN ('DRAFT', 'STAGING', 'DRY_RUN_READY', 'READY_TO_CUTOVER', 'CUTOVER_COMPLETE', 'FAILED')
    )
) PRIMARY KEY (supplier_id, project_id);

CREATE INDEX Idx_MigrationProjects_BySupplierUpdated
    ON MigrationProjects(supplier_id, updated_at DESC);

-- 2. Intake Bundles (one user drop = 1..N uploaded files auto-sorted into child sessions)
CREATE TABLE MigrationBundles (
    supplier_id           STRING(36)  NOT NULL,
    project_id            STRING(36)  NOT NULL,
    bundle_id             STRING(36)  NOT NULL,
    status                STRING(32)  NOT NULL DEFAULT ('CREATED'),
    sort_summary_json     JSON,
    error_message         STRING(MAX),
    created_by            STRING(64)  NOT NULL,
    created_at            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_MigrationBundleStatus CHECK (
        status IN ('CREATED', 'UPLOADING', 'CLASSIFYING', 'SORTED', 'NEEDS_REVIEW', 'FAILED')
    )
) PRIMARY KEY (supplier_id, project_id, bundle_id),
  INTERLEAVE IN PARENT MigrationProjects ON DELETE CASCADE;

-- 3. Uploaded Raw Artifacts inside a Bundle
CREATE TABLE MigrationArtifacts (
    supplier_id      STRING(36)   NOT NULL,
    project_id       STRING(36)   NOT NULL,
    bundle_id        STRING(36)   NOT NULL,
    artifact_id      STRING(36)   NOT NULL,
    file_name        STRING(255)  NOT NULL,
    byte_size        INT64        NOT NULL,
    declared_mime    STRING(128),
    detected_mime    STRING(128),
    content_sha256   STRING(64)   NOT NULL,
    encoding         STRING(32),
    gcs_path         STRING(1024) NOT NULL,
    streams_json     JSON,
    created_at       TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at       TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, project_id, bundle_id, artifact_id),
  INTERLEAVE IN PARENT MigrationBundles ON DELETE CASCADE;

-- 4. Widen existing SupplierImportSessions & SupplierImportStagedRows
ALTER TABLE SupplierImportSessions DROP CONSTRAINT CHK_SupplierImportSessionStatus;
ALTER TABLE SupplierImportSessions ADD COLUMN project_id STRING(36);
ALTER TABLE SupplierImportSessions ADD COLUMN bundle_id STRING(36);
ALTER TABLE SupplierImportSessions ADD COLUMN artifact_id STRING(36);
ALTER TABLE SupplierImportSessions ADD COLUMN stream_key STRING(255);
ALTER TABLE SupplierImportSessions ADD COLUMN stream_split_key STRING(64);
ALTER TABLE SupplierImportSessions ADD COLUMN pack STRING(32);
ALTER TABLE SupplierImportSessions ADD COLUMN sheet_name STRING(255);
ALTER TABLE SupplierImportSessions ADD COLUMN gcs_path STRING(1024);
ALTER TABLE SupplierImportSessions ADD COLUMN content_sha256 STRING(64);
ALTER TABLE SupplierImportSessions ADD COLUMN detected_mime STRING(128);
ALTER TABLE SupplierImportSessions ADD COLUMN encoding STRING(32);
ALTER TABLE SupplierImportSessions ADD COLUMN ai_model STRING(64);
ALTER TABLE SupplierImportSessions ADD COLUMN ai_status STRING(32);
ALTER TABLE SupplierImportSessions ADD COLUMN dry_run_report_id STRING(36);
ALTER TABLE SupplierImportSessions ADD COLUMN dry_run_fingerprint STRING(64);
ALTER TABLE SupplierImportSessions ADD COLUMN apply_mode STRING(24);
ALTER TABLE SupplierImportSessions ADD COLUMN set_on_hand_confirmed BOOL;
ALTER TABLE SupplierImportSessions ADD COLUMN resume_row_index INT64;
ALTER TABLE SupplierImportSessions ADD COLUMN last_chunk_id STRING(64);
ALTER TABLE SupplierImportSessions ADD COLUMN rollback_until TIMESTAMP;

ALTER TABLE SupplierImportSessions ADD CONSTRAINT CHK_SupplierImportSessionStatus CHECK (
    status IN (
        'initialized', 'uploaded', 'discovering', 'discovered', 'mapping_required', 'approved', 'applying', 'applied', 'failed',
        'INITIALIZED', 'UPLOADED', 'DISCOVERING', 'DISCOVERED', 'MAPPING_REQUIRED', 'MAPPING_PROPOSED',
        'VALIDATING', 'DRY_RUN_READY', 'DRY_RUNNING', 'APPROVED', 'SNAPSHOTTING', 'APPLYING', 'APPLIED',
        'ROLLING_BACK', 'ROLLED_BACK', 'CUTOVER_COMPLETE', 'FAILED'
    )
);

CREATE INDEX Idx_SupplierImportSessions_ByProjectPack
    ON SupplierImportSessions(supplier_id, project_id, pack, created_at DESC);

ALTER TABLE SupplierImportStagedRows ADD COLUMN external_id STRING(128);
ALTER TABLE SupplierImportStagedRows ADD COLUMN stream_split_key STRING(64);
ALTER TABLE SupplierImportStagedRows ADD COLUMN apply_action STRING(16);
ALTER TABLE SupplierImportStagedRows ADD COLUMN apply_chunk INT64;
ALTER TABLE SupplierImportStagedRows ADD COLUMN is_new_entity BOOL;

-- 5. Dry-run & Apply Reports (totals stored as INT64 minor units, never floats)
CREATE TABLE MigrationReports (
    supplier_id          STRING(36)  NOT NULL,
    report_id            STRING(36)  NOT NULL,
    project_id           STRING(36),
    session_id           STRING(36),
    report_kind          STRING(24)  NOT NULL,
    content_fingerprint  STRING(64)  NOT NULL,
    rows_total           INT64       NOT NULL DEFAULT (0),
    rows_valid           INT64       NOT NULL DEFAULT (0),
    rows_invalid         INT64       NOT NULL DEFAULT (0),
    would_create         INT64       NOT NULL DEFAULT (0),
    would_update         INT64       NOT NULL DEFAULT (0),
    would_skip           INT64       NOT NULL DEFAULT (0),
    money_impact_minor   INT64       NOT NULL DEFAULT (0),
    currency             STRING(3),
    blocking_count       INT64       NOT NULL DEFAULT (0),
    warning_count        INT64       NOT NULL DEFAULT (0),
    summary_json         JSON        NOT NULL,
    expires_at           TIMESTAMP   NOT NULL,
    created_at           TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, report_id);

CREATE INDEX Idx_MigrationReports_ByProject
    ON MigrationReports(supplier_id, project_id, created_at DESC);

-- 6. Per-row Before-Images for Rollback + Per-(Supplier, Warehouse) Apply Locks
CREATE TABLE MigrationSnapshots (
    supplier_id      STRING(36)   NOT NULL,
    session_id       STRING(36)   NOT NULL,
    snapshot_seq     INT64        NOT NULL,
    target_table     STRING(64)   NOT NULL,
    row_key_json     STRING(512)  NOT NULL,
    op_kind          STRING(16)   NOT NULL,
    before_json      JSON,
    expires_at       TIMESTAMP    NOT NULL,
    created_at       TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, session_id, snapshot_seq),
  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE,
  ROW DELETION POLICY (OLDER_THAN(expires_at, INTERVAL 30 DAY));

CREATE TABLE MigrationWarehouseLocks (
    supplier_id      STRING(36)  NOT NULL,
    warehouse_id     STRING(36)  NOT NULL,
    session_id       STRING(36)  NOT NULL,
    acquired_at      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    expires_at       TIMESTAMP   NOT NULL
) PRIMARY KEY (supplier_id, warehouse_id);

-- 7. Fallback Identity Map & Reusable Header Mapping Library
CREATE TABLE MigrationIdentityMap (
    supplier_id         STRING(36)  NOT NULL,
    pack                STRING(32)  NOT NULL,
    external_id         STRING(128) NOT NULL,
    internal_id         STRING(64)  NOT NULL,
    created_by_session  STRING(36)  NOT NULL,
    created_at          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, pack, external_id);

CREATE TABLE SupplierImportMappingLibrary (
    supplier_id         STRING(36)  NOT NULL,
    pack                STRING(32)  NOT NULL,
    header_fingerprint  STRING(64)  NOT NULL,
    mapping_json        JSON        NOT NULL,
    confirmed_by        STRING(64)  NOT NULL,
    updated_at          TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, pack, header_fingerprint);

-- 8. Legal-entity Tax ID / INN Cache / Duplicate-INN Join Requests
ALTER TABLE Suppliers ADD COLUMN NormalizedTaxId STRING(32);
ALTER TABLE Suppliers ADD COLUMN InnVerifiedAt TIMESTAMP;
CREATE UNIQUE NULL_FILTERED INDEX UQ_Suppliers_CellNormalizedTaxId
    ON Suppliers(HomeCell, NormalizedTaxId);

ALTER TABLE Retailers ADD COLUMN TaxId STRING(64);
ALTER TABLE Retailers ADD COLUMN NormalizedTaxId STRING(32);
ALTER TABLE Retailers ADD COLUMN HomeCell STRING(32);
ALTER TABLE Retailers ADD COLUMN LegalEntityStatus STRING(24) NOT NULL DEFAULT ('INFORMAL_PHONE');
ALTER TABLE Retailers ADD COLUMN InnVerifiedAt TIMESTAMP;
CREATE UNIQUE NULL_FILTERED INDEX UQ_Retailers_CellNormalizedTaxId
    ON Retailers(HomeCell, NormalizedTaxId);

CREATE TABLE InnRegistryCache (
    home_cell     STRING(32)   NOT NULL,
    inn           STRING(32)   NOT NULL,
    status        STRING(16)   NOT NULL,
    legal_name    STRING(512),
    address       STRING(MAX),
    raw_json      JSON,
    fetched_at    TIMESTAMP    NOT NULL,
    expires_at    TIMESTAMP    NOT NULL
) PRIMARY KEY (home_cell, inn),
  ROW DELETION POLICY (OLDER_THAN(expires_at, INTERVAL 7 DAY));

CREATE TABLE JoinRequests (
    request_id         STRING(36)  NOT NULL,
    home_cell          STRING(32)  NOT NULL,
    target_tenant_type STRING(16)  NOT NULL,
    target_tenant_id   STRING(36)  NOT NULL,
    normalized_inn     STRING(32)  NOT NULL,
    requester_phone    STRING(32)  NOT NULL,
    requester_name     STRING(255),
    requester_email    STRING(320),
    role_sought        STRING(32)  NOT NULL,
    status             STRING(16)  NOT NULL DEFAULT ('PENDING'),
    decided_by         STRING(64),
    decided_at         TIMESTAMP,
    decision_reason    STRING(512),
    created_at         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (request_id);

CREATE INDEX Idx_JoinRequests_ByTargetStatus
    ON JoinRequests(target_tenant_type, target_tenant_id, status, created_at DESC);

-- 9. Historical & Held Orders + Migrated AR Acknowledgement
ALTER TABLE Orders ADD COLUMN ExternalOrderNo STRING(128);
ALTER TABLE Orders ADD COLUMN ExternalFiscalRef STRING(128);
ALTER TABLE Orders ADD COLUMN SupersededByRollback BOOL NOT NULL DEFAULT (FALSE);
CREATE NULL_FILTERED INDEX Idx_Orders_BySupplierExternalOrderNo
    ON Orders(SupplierId, ExternalOrderNo);

ALTER TABLE ArInvoices ADD COLUMN ExternalId STRING(128);
ALTER TABLE ArInvoices ADD COLUMN Source STRING(24);
ALTER TABLE ArInvoices ADD COLUMN AcknowledgedAt TIMESTAMP;
ALTER TABLE ArInvoices ADD COLUMN DisputedAt TIMESTAMP;
ALTER TABLE ArInvoices ADD COLUMN DisputeReason STRING(512);
CREATE NULL_FILTERED INDEX Idx_ArInvoices_BySupplierExternalId
    ON ArInvoices(SupplierId, ExternalId);
