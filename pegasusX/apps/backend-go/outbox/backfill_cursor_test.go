package outbox

import (
	"context"
	"testing"
)

func TestBackfillSupplierIDWithCursor_NilClient(t *testing.T) {
	_, _, err := BackfillSupplierIDWithCursor(context.Background(), nil, "", 10)
	if err == nil || err.Error() != "outbox backfill: nil client" {
		t.Fatalf("expected 'outbox backfill: nil client', got %v", err)
	}
}
