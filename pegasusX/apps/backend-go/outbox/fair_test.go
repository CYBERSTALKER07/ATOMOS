package outbox

import "testing"

func TestFairInterleaveRoundRobin(t *testing.T) {
	in := []Event{
		{EventID: "a1", SupplierID: "sup-a"},
		{EventID: "a2", SupplierID: "sup-a"},
		{EventID: "a3", SupplierID: "sup-a"},
		{EventID: "b1", SupplierID: "sup-b"},
		{EventID: "b2", SupplierID: "sup-b"},
		{EventID: "c1", SupplierID: "sup-c"},
	}
	got := FairInterleave(in, 6)
	want := []string{"a1", "b1", "c1", "a2", "b2", "a3"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), ids(got))
	}
	for i := range want {
		if got[i].EventID != want[i] {
			t.Fatalf("i=%d got=%v want=%v", i, ids(got), want)
		}
	}
}

func TestFairInterleaveSingleTenant(t *testing.T) {
	in := []Event{
		{EventID: "1", SupplierID: "sup-a"},
		{EventID: "2", SupplierID: "sup-a"},
	}
	got := FairInterleave(in, 10)
	if len(got) != 2 || got[0].EventID != "1" {
		t.Fatalf("got=%v", ids(got))
	}
}

func ids(events []Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.EventID
	}
	return out
}
