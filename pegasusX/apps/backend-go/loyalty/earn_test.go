package loyalty

import "testing"

func TestPointsFor(t *testing.T) {
	if PointsFor(10000, 100) != 100 {
		t.Fatalf("1% of 10000 minor should be 100 points")
	}
	if PointsFor(1, 100) != 0 {
		t.Fatal("floor toward zero")
	}
	if PointsFor(0, 100) != 0 {
		t.Fatal("zero amount")
	}
}

func TestTierFor(t *testing.T) {
	cur, next := TierFor(0, DefaultTiers)
	if cur.Name != "STANDARD" || next == nil || next.Name != "SILVER" {
		t.Fatalf("got %+v %+v", cur, next)
	}
	cur, next = TierFor(50000, DefaultTiers)
	if cur.Name != "SILVER" || next == nil || next.Name != "GOLD" {
		t.Fatalf("got %+v %+v", cur, next)
	}
	cur, next = TierFor(200000, DefaultTiers)
	if cur.Name != "GOLD" || next != nil {
		t.Fatalf("got %+v %+v", cur, next)
	}
}
