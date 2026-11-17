package routing

import "testing"

func TestDispatchLocalSearchSolver_Reorders(t *testing.T) {
	solver := &DispatchLocalSearchSolver{DepotLat: 41.31, DepotLng: 69.22}
	seq, err := solver.Solve(ReplanProblem{
		RemainingStops: []StopContext{
			{OrderID: "a", Lat: 41.30, Lng: 69.20},
			{OrderID: "c", Lat: 41.32, Lng: 69.24},
			{OrderID: "b", Lat: 41.30, Lng: 69.24},
			{OrderID: "d", Lat: 41.32, Lng: 69.20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seq) != 4 {
		t.Fatalf("len=%d", len(seq))
	}
	seen := map[string]bool{}
	for _, id := range seq {
		seen[id] = true
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if !seen[id] {
			t.Fatalf("missing %s", id)
		}
	}
}

func TestSolveAnchoredReplan_PreservesAnchoredPrefix(t *testing.T) {
	anchored := []string{"arrived-1"}
	candidates := []StopContext{
		{OrderID: "cand-a", Lat: 41.30, Lng: 69.20},
		{OrderID: "cand-c", Lat: 41.32, Lng: 69.24},
		{OrderID: "cand-b", Lat: 41.30, Lng: 69.24},
		{OrderID: "cand-d", Lat: 41.32, Lng: 69.20},
	}
	startLat, startLng := 41.31, 69.22
	res, err := SolveAnchoredReplan(anchored, candidates, startLat, startLng, nil)
	if err != nil {
		t.Fatalf("SolveAnchoredReplan error: %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("expected 5 stops, got %d", len(res))
	}
	if res[0] != "arrived-1" {
		t.Fatalf("expected arrived-1 to remain anchored at index 0, got %s", res[0])
	}
	// Verify all candidate orders are present
	seen := map[string]bool{}
	for _, id := range res[1:] {
		seen[id] = true
	}
	for _, id := range []string{"cand-a", "cand-b", "cand-c", "cand-d"} {
		if !seen[id] {
			t.Fatalf("missing candidate %s in replanned sequence", id)
		}
	}

	// Single candidate test
	singleRes, err := SolveAnchoredReplan(anchored, candidates[:1], startLat, startLng, nil)
	if err != nil {
		t.Fatalf("single candidate error: %v", err)
	}
	if len(singleRes) != 2 || singleRes[0] != "arrived-1" || singleRes[1] != "cand-a" {
		t.Fatalf("unexpected single candidate res: %v", singleRes)
	}
}
