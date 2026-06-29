CREATE TABLE WarehouseBroadcastTemplates (
  WarehouseId   STRING(36)  NOT NULL,
  TemplateId    STRING(36)  NOT NULL,
  SupplierId    STRING(36)  NOT NULL,
  Title         STRING(255) NOT NULL,
  Body          STRING(MAX) NOT NULL,
  DefaultRole   STRING(32)  NOT NULL,
  Category      STRING(64),
  CreatedBy     STRING(36),
  CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId, TemplateId);

CREATE INDEX Idx_WarehouseBroadcastTemplates_ByWarehouseUpdated
  ON WarehouseBroadcastTemplates(WarehouseId, UpdatedAt DESC);
