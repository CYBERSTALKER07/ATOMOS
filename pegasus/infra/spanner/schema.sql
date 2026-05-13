-- Supplier inventory import sandbox substrate (Phase 2 compatibility)
-- Canonical runtime schema remains in apps/backend-go/schema/spanner.ddl.

CREATE TABLE SupplierImportSessions (
    supplier_id    STRING(36)  NOT NULL,
    session_id     STRING(36)  NOT NULL,
    status         STRING(30)  NOT NULL,
    file_name      STRING(255) NOT NULL,
    total_rows     INT64       NOT NULL DEFAULT (0),
    error_summary  JSON,
    created_at     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at     TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
    CONSTRAINT CHK_SupplierImportSessionStatus CHECK (
        status IN ('uploaded', 'discovering', 'mapping_required', 'approved', 'applying', 'applied', 'failed')
    )
) PRIMARY KEY (supplier_id, session_id);

CREATE INDEX Idx_SupplierImportSessions_BySupplierUpdated
    ON SupplierImportSessions(supplier_id, updated_at DESC);
CREATE INDEX Idx_SupplierImportSessions_BySupplierStatus
    ON SupplierImportSessions(supplier_id, status, updated_at DESC);

CREATE TABLE SupplierImportStagedRows (
    supplier_id         STRING(36)       NOT NULL,
    session_id          STRING(36)       NOT NULL,
    row_index           INT64            NOT NULL,
    raw_data            JSON,
    cleaned_data        JSON,
    validation_errors   ARRAY<STRING(MAX)>,
    is_new_product      BOOL             NOT NULL DEFAULT (false),
    created_at          TIMESTAMP        NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at          TIMESTAMP        OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, session_id, row_index),
  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE;

CREATE INDEX Idx_SupplierImportStagedRows_BySession
    ON SupplierImportStagedRows(supplier_id, session_id, row_index);

CREATE TABLE SupplierImportMapping (
    supplier_id    STRING(36)  NOT NULL,
    session_id     STRING(36)  NOT NULL,
    mapping_json   JSON,
    created_at     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at     TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (supplier_id, session_id),
  INTERLEAVE IN PARENT SupplierImportSessions ON DELETE CASCADE;
