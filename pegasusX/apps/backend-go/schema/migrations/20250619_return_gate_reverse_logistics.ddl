-- Reverse logistics: retail barcode on products + physical return gate workflow.

ALTER TABLE Products ADD COLUMN Barcode STRING(32);
CREATE NULL_FILTERED INDEX Idx_Products_BySupplierBarcode ON Products(SupplierId, Barcode);

ALTER TABLE SupplierReturns ADD COLUMN ManifestId STRING(36);
ALTER TABLE SupplierReturns ADD COLUMN DriverId STRING(36);
ALTER TABLE SupplierReturns ADD COLUMN WarehouseId STRING(36);
ALTER TABLE SupplierReturns ADD COLUMN ExpectedQty INT64;
ALTER TABLE SupplierReturns ADD COLUMN ReceivedQty INT64 NOT NULL DEFAULT (0);
ALTER TABLE SupplierReturns ADD COLUMN PhysicalStatus STRING(32) NOT NULL DEFAULT ('PENDING');
ALTER TABLE SupplierReturns ADD COLUMN ReceivedAt TIMESTAMP;
ALTER TABLE SupplierReturns ADD COLUMN ReceivedBy STRING(36);
ALTER TABLE SupplierReturns ADD COLUMN ReceiveSessionId STRING(36);

CREATE INDEX Idx_SupplierReturns_ByPhysicalStatus ON SupplierReturns(PhysicalStatus, CreatedAt DESC);
CREATE INDEX Idx_SupplierReturns_ByManifest ON SupplierReturns(ManifestId, PhysicalStatus);
CREATE INDEX Idx_SupplierReturns_ByWarehousePhysical ON SupplierReturns(WarehouseId, PhysicalStatus, CreatedAt DESC);
CREATE INDEX Idx_SupplierReturns_ByDriverPhysical ON SupplierReturns(DriverId, PhysicalStatus, CreatedAt DESC);

CREATE TABLE ReturnReceiveSessions (
  SessionId     STRING(36)  NOT NULL,
  WarehouseId   STRING(36)  NOT NULL,
  ManifestId    STRING(36),
  DriverId      STRING(36),
  OperatorId    STRING(36)  NOT NULL,
  OperatorRole  STRING(32)  NOT NULL,
  Status        STRING(32)  NOT NULL DEFAULT ('OPEN'),
  StartedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  CompletedAt   TIMESTAMP,
) PRIMARY KEY (SessionId);

CREATE INDEX Idx_ReturnReceiveSessions_ByWarehouse ON ReturnReceiveSessions(WarehouseId, Status, StartedAt DESC);
