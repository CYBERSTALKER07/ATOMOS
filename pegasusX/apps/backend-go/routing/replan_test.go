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
