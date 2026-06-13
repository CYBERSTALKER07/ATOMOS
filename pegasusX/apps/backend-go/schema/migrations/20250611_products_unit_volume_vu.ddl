-- Ops migration: add product volumetric unit for dispatch capacity (qty × VU).
-- Safe to run on existing Spanner instances; default 1.0 preserves legacy behavior.

ALTER TABLE Products ADD COLUMN UnitVolumeVU FLOAT64 NOT NULL DEFAULT (1.0);
