ed pegasusX/apps/backend-go/schema/spanner.ddl << 'ED'
/CREATE TABLE PaymentSessions/
i
CREATE TABLE Warehouses (
  WarehouseId      STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  CoverageRadiusKm FLOAT64       NOT NULL,
  IsActive         BOOL          NOT NULL,
  IsOnShift        BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (WarehouseId);

CREATE INDEX Idx_Warehouses_BySupplier ON Warehouses(SupplierId);

CREATE TABLE Factories (
  FactoryId        STRING(36)    NOT NULL,
  SupplierId       STRING(36)    NOT NULL,
  Name             STRING(255)   NOT NULL,
  Lat              FLOAT64,
  Lng              FLOAT64,
  IsActive         BOOL          NOT NULL,
  CreatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (FactoryId);

CREATE INDEX Idx_Factories_BySupplier ON Factories(SupplierId);

.
w
ED
