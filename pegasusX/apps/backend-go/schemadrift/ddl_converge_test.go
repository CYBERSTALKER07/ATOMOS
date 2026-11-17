package schemadrift

import (
	"reflect"
	"testing"
)

func TestPlanDDLStatements_SkipsExistingAndCanceledIndexes(t *testing.T) {
	t.Parallel()

	live := ParseDatabaseDDL([]string{
		`CREATE TABLE Suppliers (
			SupplierId STRING(36) NOT NULL,
			Name STRING(255) NOT NULL,
			DailyOutputCapacity INT64 NOT NULL DEFAULT (700),
			UpdatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
			CONSTRAINT CHK_Suppliers_Name CHECK (Name != '')
		) PRIMARY KEY (SupplierId)`,
		`CREATE INDEX Idx_Suppliers_ByName ON Suppliers(Name)`,
	})

	candidates := []string{
		// Already exists in live catalog -> skip
		`CREATE TABLE Suppliers (SupplierId STRING(36) NOT NULL) PRIMARY KEY (SupplierId)`,
		// Existing column -> skip
		`ALTER TABLE Suppliers ADD COLUMN Name STRING(255) NOT NULL`,
		// New column -> keep, and second duplicate should be skipped via working catalog
		`ALTER TABLE Suppliers ADD COLUMN HomeCell STRING(32)`,
		`ALTER TABLE Suppliers ADD COLUMN HomeCell STRING(32)`,
		// Existing index -> skip
		`CREATE INDEX Idx_Suppliers_ByName ON Suppliers(Name)`,
		// Obsolete index created then later dropped -> skip both
		`CREATE INDEX Idx_TempHotspot ON Suppliers(UpdatedAt)`,
		`DROP INDEX Idx_TempHotspot`,
		// Already-matching ALTER COLUMN DEFAULT / OPTIONS -> skip
		`ALTER TABLE Suppliers ALTER COLUMN DailyOutputCapacity SET DEFAULT (700)`,
		`ALTER TABLE Suppliers ALTER COLUMN UpdatedAt SET OPTIONS (allow_commit_timestamp=true)`,
		// Identical DROP+ADD constraint -> skip both
		`ALTER TABLE Suppliers DROP CONSTRAINT CHK_Suppliers_Name`,
		`ALTER TABLE Suppliers ADD CONSTRAINT CHK_Suppliers_Name CHECK (Name != '')`,
		// Genuine new index -> keep
		`CREATE INDEX Idx_Suppliers_ByHomeCell ON Suppliers(HomeCell)`,
	}

	got := PlanDDLStatements(live, candidates)
	want := []string{
		`ALTER TABLE Suppliers ADD COLUMN HomeCell STRING(32)`,
		`CREATE INDEX Idx_Suppliers_ByHomeCell ON Suppliers(HomeCell)`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PlanDDLStatements mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPlanDDLStatements_KeepsChangedConstraint(t *testing.T) {
	t.Parallel()

	live := ParseDatabaseDDL([]string{
		`CREATE TABLE SupplierImportSessions (
			supplier_id STRING(36) NOT NULL,
			session_id STRING(36) NOT NULL,
			status STRING(32) NOT NULL,
			CONSTRAINT CHK_Status CHECK (status IN ('initialized', 'applied'))
		) PRIMARY KEY (supplier_id, session_id)`,
	})

	candidates := []string{
		`ALTER TABLE SupplierImportSessions DROP CONSTRAINT CHK_Status`,
		`ALTER TABLE SupplierImportSessions ADD CONSTRAINT CHK_Status CHECK (status IN ('initialized', 'discovering', 'applied'))`,
	}

	got := PlanDDLStatements(live, candidates)
	if !reflect.DeepEqual(got, candidates) {
		t.Fatalf("expected changed constraint statements to be kept, got %#v", got)
	}
}
