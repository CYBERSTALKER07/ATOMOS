package retailer

import (
	"slices"
	"strings"
	"testing"
)

func TestAutoOrderEnabledRetailerCursorSQL(t *testing.T) {
	t.Parallel()

	first := buildAutoOrderEnabledRetailerStmt("", 200)
	if !strings.Contains(first.SQL, "GlobalEnabled = true") {
		t.Fatalf("first page SQL missing GlobalEnabled filter: %s", first.SQL)
	}
	if !strings.Contains(first.SQL, "ORDER BY RetailerId ASC") || !strings.Contains(first.SQL, "LIMIT @lim") {
		t.Fatalf("first page SQL missing keyset ordering/limit: %s", first.SQL)
	}
	if strings.Contains(first.SQL, "@after") {
		t.Fatalf("first page SQL should not require @after: %s", first.SQL)
	}
	if got := first.Params["lim"]; got != int64(200) {
		t.Fatalf("lim=%v want 200", got)
	}

	next := buildAutoOrderEnabledRetailerStmt("ret-0200", 200)
	if !strings.Contains(next.SQL, "GlobalEnabled = true AND RetailerId > @after") {
		t.Fatalf("subsequent page SQL missing keyset predicate: %s", next.SQL)
	}
	if got := next.Params["after"]; got != "ret-0200" {
		t.Fatalf("after=%v want ret-0200", got)
	}
}

func TestStockBalanceCursorPaginationSQL(t *testing.T) {
	t.Parallel()

	first := buildStockBalancePageStmt("ret-1", "", 500)
	if !strings.Contains(first.SQL, "WHERE RetailerId = @rid") || !strings.Contains(first.SQL, "GROUP BY Sku") || !strings.Contains(first.SQL, "ORDER BY Sku ASC") {
		t.Fatalf("unexpected first stock balance SQL: %s", first.SQL)
	}
	if strings.Contains(first.SQL, "@afterSku") {
		t.Fatalf("first page SQL should not filter @afterSku: %s", first.SQL)
	}

	next := buildStockBalancePageStmt("ret-1", "SKU-0500", 500)
	if !strings.Contains(next.SQL, "WHERE RetailerId = @rid AND Sku > @afterSku") || !strings.Contains(next.SQL, "ORDER BY Sku ASC") {
		t.Fatalf("unexpected next stock balance SQL: %s", next.SQL)
	}
	if got := next.Params["afterSku"]; got != "SKU-0500" {
		t.Fatalf("afterSku=%v want SKU-0500", got)
	}
}

func TestIterateKeysetCursor_MultiPageAndRawCursorWithFilteredRows(t *testing.T) {
	t.Parallel()

	type dbRow struct {
		rawSKU string
		onHand int64
	}
	// 5 total DB rows sorted lexicographically across 3 pages (pageSize=2),
	// with a filtered "local:" SKU sitting right on page 1 boundary ('i' < 'l' < 'p').
	dataset := []dbRow{
		{rawSKU: "item-001", onHand: 5},
		{rawSKU: "local:house-brand", onHand: 10},
		{rawSKU: "prod-002", onHand: 7},
		{rawSKU: "prod-003", onHand: 9},
		{rawSKU: "prod-004", onHand: 12},
	}

	var seenCursors []string
	var collected []string
	iterateKeysetCursor(
		2,
		func(after string, limit int) ([]string, int, string, error) {
			seenCursors = append(seenCursors, after)
			var filtered []string
			rawCount := 0
			lastRaw := ""
			for _, r := range dataset {
				if after != "" && r.rawSKU <= after {
					continue
				}
				rawCount++
				lastRaw = r.rawSKU
				if !strings.HasPrefix(r.rawSKU, "local:") {
					filtered = append(filtered, r.rawSKU)
				}
				if rawCount == limit {
					break
				}
			}
			return filtered, rawCount, lastRaw, nil
		},
		func(sku string) {
			collected = append(collected, sku)
		},
		func(after string, err error) {
			t.Fatalf("unexpected error at cursor %q: %v", after, err)
		},
	)

	want := []string{"item-001", "prod-002", "prod-003", "prod-004"}
	if !slices.Equal(collected, want) {
		t.Fatalf("collected=%v want %v", collected, want)
	}
	// Notice page 1 ended on "local:house-brand" (1 filtered row < pageSize, rawCount==2):
	// raw cursor advances without truncating page 2+.
	if len(seenCursors) != 3 {
		t.Fatalf("pages read=%d cursors=%v want 3", len(seenCursors), seenCursors)
	}
}

func TestInFlightQtyStmt_CoversCanonicalPreDeliveryStatuses(t *testing.T) {
	t.Parallel()

	stmt := buildInFlightQtyStmt("ret-1", "SKU-001")
	if !strings.Contains(stmt.SQL, "UPPER(o.Status) IN UNNEST(@statuses)") {
		t.Fatalf("SQL must parameterize canonical status set: %s", stmt.SQL)
	}
	rawStatuses, ok := stmt.Params["statuses"].([]string)
	if !ok {
		t.Fatalf("statuses param type=%T want []string", stmt.Params["statuses"])
	}

	wantActive := []string{
		"PENDING",
		"SCHEDULED",
		"AUTO_ACCEPTED",
		"BACKORDERED",
		"DELAYED",
		"LOADED",
		"IN_TRANSIT",
		"ARRIVED",
		"AWAITING_PAYMENT",
		"PENDING_CASH_COLLECTION",
		"SHOP_CLOSED_PENDING",
	}
	for _, st := range wantActive {
		if !slices.Contains(rawStatuses, st) {
			t.Fatalf("missing canonical pre-delivery order status %q in %v", st, rawStatuses)
		}
	}
	for _, terminal := range []string{"COMPLETED", "CANCELLED", "DELIVERED_ON_CREDIT", "FISCALIZING"} {
		if slices.Contains(rawStatuses, terminal) {
			t.Fatalf("terminal/delivered status %q must not be counted as in-flight", terminal)
		}
	}
}

func TestDefaultRetailerID_UUIDAndUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]struct{}{}
	for i := 0; i < 64; i++ {
		id := defaultRetailerID()
		if !strings.HasPrefix(id, "ret_") || len(id) != 36 {
			t.Fatalf("unexpected retailer id format %q (len=%d)", id, len(id))
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}
