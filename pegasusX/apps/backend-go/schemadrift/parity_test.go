package schemadrift

import "testing"

func TestMigrationTableParity_spannerDDL(t *testing.T) {
	mig, ddl, err := FindSchemaPaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := AssertMigrationTableParity(mig, ddl); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredProductTables_nonEmpty(t *testing.T) {
	if len(RequiredProductTables) < 14 {
		t.Fatalf("expected >=14 required tables, got %d", len(RequiredProductTables))
	}
	seen := map[string]bool{}
	for _, tname := range RequiredProductTables {
		if tname == "" || seen[tname] {
			t.Fatalf("bad/duplicate table %q", tname)
		}
		seen[tname] = true
	}
}

func TestCreateTableRe_skipsIF(t *testing.T) {
	raw := []byte("CREATE TABLE IF NOT EXISTS PlatformTenants (\n  X INT64\n);\nCREATE TABLE Foo (Y INT64);")
	got := extractCreateTables(raw)
	if len(got) != 2 || got[0] != "PlatformTenants" || got[1] != "Foo" {
		t.Fatalf("got %#v", got)
	}
}
