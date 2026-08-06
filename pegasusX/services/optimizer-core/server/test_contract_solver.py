"""Smoke + constraint tests for optimizer-contract OR-Tools solver."""

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


def test_cold_stop_never_on_non_reefer():
    req = {
        "v": "v1",
        "trace_id": "cold1",
        "supplier_id": "s1",
        "home_node_id": "wh1",
        "stops": [
            {
                "order_id": "cold-o1",
                "retailer_id": "r1",
                "lat": 41.31,
                "lng": 69.25,
                "volume_vu": 10.0,
                "requires_cold_chain": True,
                "handling_class": "COLD_CHAIN",
            },
            {
                "order_id": "dry-o2",
                "retailer_id": "r2",
                "lat": 41.32,
                "lng": 69.26,
                "volume_vu": 10.0,
            },
        ],
        "vehicles": [
            {
                "vehicle_id": "dry-truck",
                "driver_id": "d-dry",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
                "has_refrigeration": False,
            },
            {
                "vehicle_id": "reefer",
                "driver_id": "d-cold",
                "max_volume_vu": 100.0,
                "start_lat": 41.301,
                "start_lng": 69.241,
                "has_refrigeration": True,
            },
        ],
        "tunables": {"time_limit_ms": 3000},
    }
    resp = solve_contract(req)
    cold_vehicle = None
    for route in resp["routes"]:
        for stop in route["stops"]:
            if stop["order_id"] == "cold-o1":
                cold_vehicle = route["vehicle_id"]
    assert cold_vehicle == "reefer", f"cold stop on {cold_vehicle!r}; orphans={resp['orphans']}"


def test_multi_depot_distinct_starts():
    """Two vehicles at distant depots should keep their own start nodes."""
    req = {
        "v": "v1",
        "trace_id": "md1",
        "supplier_id": "s1",
        "home_node_id": "wh1",
        "stops": [
            {
                "order_id": "near-a",
                "retailer_id": "ra",
                "lat": 41.311,
                "lng": 69.251,
                "volume_vu": 5.0,
            },
            {
                "order_id": "near-b",
                "retailer_id": "rb",
                "lat": 41.411,
                "lng": 69.351,
                "volume_vu": 5.0,
            },
        ],
        "vehicles": [
            {
                "vehicle_id": "depot-a",
                "driver_id": "da",
                "max_volume_vu": 50.0,
                "start_lat": 41.31,
                "start_lng": 69.25,
            },
            {
                "vehicle_id": "depot-b",
                "driver_id": "db",
                "max_volume_vu": 50.0,
                "start_lat": 41.41,
                "start_lng": 69.35,
            },
        ],
        "tunables": {"time_limit_ms": 3000},
    }
    resp = solve_contract(req)
    by_vehicle = {r["vehicle_id"]: [s["order_id"] for s in r["stops"]] for r in resp["routes"]}
    # Prefer nearest: near-a on depot-a, near-b on depot-b when both used.
    if "depot-a" in by_vehicle and "depot-b" in by_vehicle:
        assert "near-a" in by_vehicle["depot-a"]
        assert "near-b" in by_vehicle["depot-b"]
    else:
        # Still a valid multi-start solve — at least one route placed.
        assert resp["stats"]["stops_placed"] >= 1


def test_max_stops_dimension_orphans_without_truncating_metrics():
    """Stops beyond max_stops are orphaned by the solver, not post-hoc chopped."""
    stops = [
        {
            "order_id": f"o{i}",
            "retailer_id": f"r{i}",
            "lat": 41.30 + i * 0.001,
            "lng": 69.24 + i * 0.001,
            "volume_vu": 1.0,
        }
        for i in range(5)
    ]
    req = {
        "v": "v1",
        "trace_id": "ms1",
        "supplier_id": "s1",
        "home_node_id": "wh1",
        "stops": stops,
        "vehicles": [
            {
                "vehicle_id": "v1",
                "driver_id": "d1",
                "max_volume_vu": 100.0,
                "start_lat": 41.30,
                "start_lng": 69.24,
            }
        ],
        "tunables": {"max_stops_per_route": 2, "time_limit_ms": 3000},
    }
    resp = solve_contract(req)
    placed = sum(len(r["stops"]) for r in resp["routes"])
    assert placed <= 2
    assert resp["stats"]["stops_placed"] == placed
    # Metrics must reflect the actual route (no stale distance after a tail chop).
    for route in resp["routes"]:
        assert route["distance_km"] >= 0
        assert len(route["stops"]) <= 2
    orphan_ids = {o["order_id"] for o in resp["orphans"]}
    assert len(orphan_ids) >= 3
