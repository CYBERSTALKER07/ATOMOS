patch -p0 << 'DIFF'
--- pegasusX/apps/backend-go/schema/spanner.ddl
+++ pegasusX/apps/backend-go/schema/spanner.ddl
@@ -52,6 +52,38 @@
 CREATE INDEX Idx_Orders_ByRetailerCreated ON Orders(RetailerId, CreatedAt DESC);
 CREATE INDEX Idx_Orders_BySupplierCreated ON Orders(SupplierId, CreatedAt DESC);
 
+CREATE TABLE Drivers (
+  DriverId        STRING(36)    NOT NULL,
+  Name            STRING(255)   NOT NULL,
+  Phone           STRING(32)    NOT NULL,
+  PinHash         STRING(MAX),
+  SupplierId      STRING(36)    NOT NULL,
+  HomeNodeType    STRING(20)    NOT NULL,
+  HomeNodeId      STRING(36)    NOT NULL,
+  VehicleId       STRING(36),
+  IsActive        BOOL          NOT NULL,
+  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
+  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
+) PRIMARY KEY (DriverId);
+
+CREATE INDEX Idx_Drivers_BySupplierPhone ON Drivers(SupplierId, Phone);
+CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId, IsActive);
+
+CREATE TABLE Vehicles (
+  VehicleId       STRING(36)    NOT NULL,
+  Label           STRING(100),
+  LicensePlate    STRING(32)    NOT NULL,
+  SupplierId      STRING(36)    NOT NULL,
+  HomeNodeType    STRING(20)    NOT NULL,
+  HomeNodeId      STRING(36)    NOT NULL,
+  IsActive        BOOL          NOT NULL,
+  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
+  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
+) PRIMARY KEY (VehicleId);
+
+CREATE INDEX Idx_Vehicles_BySupplierPlate ON Vehicles(SupplierId, LicensePlate);
+CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId, IsActive);
+
 CREATE TABLE PaymentSessions (
   SessionId         STRING(36)    NOT NULL,
   OrderId           STRING(36)    NOT NULL,
DIFF
