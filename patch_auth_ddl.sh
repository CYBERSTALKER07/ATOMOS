ed pegasusX/apps/backend-go/schema/spanner.ddl << 'ED'
/CREATE TABLE PaymentSessions/
i
CREATE TABLE SupplierUsers (
  UserId               STRING(36)  NOT NULL,
  SupplierId           STRING(36)  NOT NULL,
  Email                STRING(MAX),
  Phone                STRING(32),
  Name                 STRING(MAX) NOT NULL,
  PasswordHash         STRING(MAX) NOT NULL,
  SupplierRole         STRING(30)  NOT NULL,
  AssignedWarehouseId  STRING(36),
  AssignedFactoryId    STRING(36),
  IsActive             BOOL        NOT NULL,
  FirebaseUid          STRING(128),
  CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UserId);

CREATE INDEX Idx_SupplierUsers_ByPhone ON SupplierUsers(Phone);

.
w
ED
