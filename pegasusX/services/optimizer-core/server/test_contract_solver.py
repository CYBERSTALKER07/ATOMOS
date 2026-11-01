"""Smoke tests for optimizer-contract OR-Tools solver."""

from contract_solver import solve_contract


def test_single_stop_single_vehicle():
    req = {
        "v": "v1",
        "trace_id": "t1",
        "supplier_id": "s1",
        "home_node_id": "wh1",
        "departure_time": "2026-06-17T08:00:00Z",
        "stops": [
            {
                "order_id": "o1",
                "retailer_id": "r1",
                "lat": 41.31,
                "lng": 69.25,
                "volume_vu": 10.0,
            }
        ],
        "vehicles": [
            {
                "vehicle_id": "v1",
                "driver_id": "d1",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
                "avg_speed_kmph": 30.0,
            }
        ],
    }
    resp = solve_contract(req)
    assert resp["source"] == "OR_TOOLS_VRP"
    assert len(resp["routes"]) == 1
    assert resp["routes"][0]["stops"][0]["order_id"] == "o1"


def test_multi_depot_multi_vehicle():
    req = {
        "v": "v1",
        "trace_id": "t2",
        "supplier_id": "s1",
        "stops": [
            {
                "order_id": "o1",
                "retailer_id": "r1",
                "lat": 41.31,
                "lng": 69.25,
                "volume_vu": 10.0,
            },
            {
                "order_id": "o2",
                "retailer_id": "r2",
                "lat": 41.41,
                "lng": 69.35,
                "volume_vu": 10.0,
            },
        ],
        "vehicles": [
            {
                "vehicle_id": "v1",
                "driver_id": "d1",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
                "end_lat": 41.30,
                "end_lng": 69.24,
                "avg_speed_kmph": 30.0,
            },
            {
                "vehicle_id": "v2",
                "driver_id": "d2",
                "max_volume_vu": 100.0,
                "start_lat": 41.40,
                "start_lng": 69.34,
                "end_lat": 41.40,
                "end_lng": 69.34,
                "avg_speed_kmph": 30.0,
            },
        ],
    }
    resp = solve_contract(req)
    assert resp["source"] == "OR_TOOLS_VRP"
    assert resp["stats"]["stops_placed"] == 2
    assert len(resp["routes"]) >= 1


def test_cold_chain_and_hazmat_vehicle_constraints():
    # v1 is closer to o1 (cold) but has no refrigeration; v2 is further but has refrigeration
    # o2 (hazmat) requires hazmat certification; only v3 has hazmat certification
    req = {
        "v": "v1",
        "trace_id": "t-safety",
        "stops": [
            {
                "order_id": "o-cold",
                "retailer_id": "r1",
                "lat": 41.301,
                "lng": 69.241,
                "volume_vu": 10.0,
                "requires_cold_chain": True,
            },
            {
                "order_id": "o-hazmat",
                "retailer_id": "r2",
                "lat": 41.35,
                "lng": 69.25,
                "volume_vu": 10.0,
                "is_hazardous": True,
            },
            {
                "order_id": "o-dry",
                "retailer_id": "r3",
                "lat": 41.302,
                "lng": 69.242,
                "volume_vu": 10.0,
            },
        ],
        "vehicles": [
            {
                "vehicle_id": "v-dry-close",
                "driver_id": "d1",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
                "has_refrigeration": False,
                "hazmat_certified": False,
            },
            {
                "vehicle_id": "v-reefer",
                "driver_id": "d2",
                "max_volume_vu": 100.0,
                "start_lat": 41.32,
                "start_lng": 69.24,
                "has_refrigeration": True,
                "hazmat_certified": False,
            },
            {
                "vehicle_id": "v-hazmat",
                "driver_id": "d3",
                "max_volume_vu": 100.0,
                "start_lat": 41.33,
                "start_lng": 69.24,
                "has_refrigeration": False,
                "hazmat_certified": True,
            },
        ],
    }
    resp = solve_contract(req)
    assert resp["stats"]["stops_placed"] == 3

    vehicle_assignments = {}
    for route in resp["routes"]:
        vid = route["vehicle_id"]
        for stop in route["stops"]:
            vehicle_assignments[stop["order_id"]] = vid

    # o-cold MUST be on v-reefer
    assert vehicle_assignments["o-cold"] == "v-reefer"
    # o-hazmat MUST be on v-hazmat
    assert vehicle_assignments["o-hazmat"] == "v-hazmat"


def test_hazmat_orphan_when_no_certified_vehicle():
    req = {
        "v": "v1",
        "trace_id": "t-orphan",
        "stops": [
            {
                "order_id": "o-hazmat",
                "retailer_id": "r1",
                "lat": 41.31,
                "lng": 69.25,
                "volume_vu": 10.0,
                "is_hazardous": True,
            }
        ],
        "vehicles": [
            {
                "vehicle_id": "v-regular",
                "driver_id": "d1",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
                "hazmat_certified": False,
            }
        ],
    }
    resp = solve_contract(req)
    assert resp["stats"]["stops_placed"] == 0
    assert any(o["order_id"] == "o-hazmat" for o in resp["orphans"])


def test_distance_matrix_and_tunables_override():
    # 1 vehicle, 2 stops.
    # By haversine, stop 1 is at 41.301 (closer) and stop 2 is at 41.350 (farther).
    # But distance_matrix_m gives road distance to stop 1 (5,000m) and road distance to stop 2 (500m).
    # Total nodes: 1 vehicle start + 2 stops = 3 nodes.
    dist_m = [
        [0, 5000, 500],
        [5000, 0, 500],
        [500, 500, 0],
    ]
    req = {
        "v": "v1",
        "trace_id": "t-matrix",
        "stops": [
            {"order_id": "o1", "lat": 41.301, "lng": 69.24, "volume_vu": 10.0},
            {"order_id": "o2", "lat": 41.350, "lng": 69.24, "volume_vu": 10.0},
        ],
        "vehicles": [
            {"vehicle_id": "v1", "max_volume_vu": 100.0, "start_lat": 41.30, "start_lng": 69.24}
        ],
        "distance_matrix_m": dist_m,
        "tunables": {"time_limit_ms": 1500},
    }
    resp = solve_contract(req)
    assert resp["stats"]["stops_placed"] == 2
    # First stop visited from depot should be o2 because of road distance in distance_matrix_m
    assert resp["routes"][0]["stops"][0]["order_id"] == "o2"
    assert resp["stats"]["solver_time_limit_ms_requested"] == 1500
    assert resp["stats"]["solver_time_limit_ms_applied"] == 1500
    assert resp["solver_metadata"]["clamped"] is False


def test_all_stops_zero_volume_early_return_datetime():
    # Tests the early return when all stops have non-positive volume.
    # Verifies that datetime.now(timezone.utc) is used without raising TypeError (CG-8).
    req = {
        "v": "v1",
        "trace_id": "t-zero-vol",
        "stops": [
            {"order_id": "o-zero", "lat": 41.301, "lng": 69.24, "volume_vu": 0.0},
            {"order_id": "o-neg", "lat": 41.350, "lng": 69.24, "volume_vu": -5.0},
        ],
        "vehicles": [
            {"vehicle_id": "v1", "max_volume_vu": 100.0, "start_lat": 41.30, "start_lng": 69.24}
        ],
        "tunables": {"time_limit_ms": 3000},
    }
    resp = solve_contract(req)
    assert resp["routes"] == []
    assert len(resp["orphans"]) == 2
    assert resp["stats"]["stops_placed"] == 0
    assert resp["stats"]["stops_orphaned"] == 2
    assert resp["stats"]["solver_time_limit_ms_requested"] == 3000
    assert resp["stats"]["solver_time_limit_ms_applied"] == 3000
    assert "elapsed_ms" in resp["stats"]
    assert resp["solver_metadata"]["clamped"] is False


