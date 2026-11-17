package seed

import (
	"context"
	"testing"
)

func TestSeedUzbekistanHolidays_NilClientSafe(t *testing.T) {
	t.Parallel()
	n, err := SeedUzbekistanHolidays(context.Background(), nil, "", nil)
	if err != nil || n != 0 {
		t.Fatalf("expected 0, nil for nil Spanner client, got n=%d err=%v", n, err)
	}
}
