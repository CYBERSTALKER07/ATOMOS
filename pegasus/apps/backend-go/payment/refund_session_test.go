package payment

import (
	"strings"
	"testing"
)

func TestLookupHighestSettledSessionSQLPrefersLargestPaidAmount(t *testing.T) {
	if !strings.Contains(settledSessionLookupSQL, "ORDER BY COALESCE(PaidAmount, LockedAmount) DESC") {
		t.Fatalf("expected settled session lookup to order by paid amount desc, got: %s", settledSessionLookupSQL)
	}
	if !strings.Contains(settledSessionLookupSQL, "SettledAt DESC") {
		t.Fatalf("expected settled session lookup to break ties by SettledAt, got: %s", settledSessionLookupSQL)
	}
}
