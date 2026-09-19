package analytics

import (
	"strings"
	"testing"
)

func TestRoutePerfListStmt_FiltersBySupplier(t *testing.T) {
	stmt := routePerfListStmt("sup-a", 25)
	if !strings.Contains(stmt.SQL, "WHERE SupplierId = @supplierId") {
		t.Fatalf("expected tenant filter, sql=%s", stmt.SQL)
	}
	if stmt.Params["supplierId"] != "sup-a" {
		t.Fatalf("params=%v", stmt.Params)
	}
	if stmt.Params["limit"] != 25 {
		t.Fatalf("limit=%v", stmt.Params["limit"])
	}
}

func TestRoutePerfListStmt_FailClosedEmptySupplier(t *testing.T) {
	stmt := routePerfListStmt("  ", 10)
	if !strings.Contains(stmt.SQL, "WHERE FALSE") {
		t.Fatalf("expected fail-closed empty query, sql=%s", stmt.SQL)
	}
	if _, ok := stmt.Params["supplierId"]; ok {
		t.Fatalf("unexpected supplierId param: %v", stmt.Params)
	}
}
