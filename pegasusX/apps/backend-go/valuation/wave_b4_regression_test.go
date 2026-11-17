package valuation

import "testing"

func TestComputeProportionalCost_BigIntNoOverflow(t *testing.T) {
	// 5_000_000_000_000 * 3_000_000 exceeds math.MaxInt64 (~9.22e18); big.Int avoids overflow.
	totalVal := int64(5_000_000_000_000)
	avail := int64(6_000_000)
	issue := int64(3_000_000)
	got := computeProportionalCost(totalVal, avail, issue)
	want := int64(2_500_000_000_000)
	if got != want {
		t.Fatalf("computeProportionalCost=%d, want %d", got, want)
	}
}
