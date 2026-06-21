package warehouse

import "testing"

func TestFilterProposedRoutesByOrderIDs(t *testing.T) {
	routes := []map[string]any{
		{
			"driver_id": "d1",
			"order_ids": []any{"a", "b"},
			"stops": []any{
				map[string]any{"order_id": "a", "volume_vu": 2.0},
				map[string]any{"order_id": "b", "volume_vu": 3.0},
			},
			"max_volume_vu": 20.0,
		},
		{
			"driver_id": "d2",
			"order_ids": []any{"c"},
			"stops": []any{
				map[string]any{"order_id": "c", "volume_vu": 4.0},
			},
			"max_volume_vu": 20.0,
		},
	}

	filtered := filterProposedRoutesByOrderIDs(routes, []string{"a", "c"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(filtered))
	}
	if filtered[0]["driver_id"] != "d1" {
		t.Fatalf("expected d1 route first, got %v", filtered[0]["driver_id"])
	}
	if filtered[0]["volume_vu"] != 2.0 {
		t.Fatalf("expected loaded 2.0 VU, got %v", filtered[0]["volume_vu"])
	}
}
