-- Fleet availability: warehouse vehicle holds + driver shift/offline reasons.

ALTER TABLE Vehicles ADD COLUMN UnavailableReason STRING(64);
ALTER TABLE Vehicles ADD COLUMN UnavailableNote STRING(255);

ALTER TABLE Drivers ADD COLUMN OnShift BOOL NOT NULL DEFAULT (true);
ALTER TABLE Drivers ADD COLUMN UnavailableReason STRING(64);
ALTER TABLE Drivers ADD COLUMN UnavailableNote STRING(255);
