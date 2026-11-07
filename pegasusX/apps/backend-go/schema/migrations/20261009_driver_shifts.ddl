-- Driver Shifts: durable driver shift state machine tracking for labor capacity & worker scoring.
CREATE TABLE DriverShifts (
  DriverId        STRING(36)    NOT NULL,
  ShiftId         STRING(36)    NOT NULL,
  SupplierId      STRING(36)    NOT NULL,
  WarehouseId     STRING(36),
  VehicleId       STRING(36),
  Status          STRING(20)    NOT NULL,
  StartAt         TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  EndAt           TIMESTAMP,
  DurationHours   FLOAT64       NOT NULL DEFAULT (0),
  StartLat        FLOAT64,
  StartLng        FLOAT64,
  EndLat          FLOAT64,
  EndLng          FLOAT64,
  Notes           STRING(255),
  CreatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt       TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (DriverId, ShiftId),
  INTERLEAVE IN PARENT Drivers ON DELETE CASCADE;

CREATE INDEX Idx_DriverShifts_ByStartAt ON DriverShifts(DriverId, StartAt DESC);
CREATE INDEX Idx_DriverShifts_BySupplier ON DriverShifts(SupplierId, Status, StartAt DESC);
