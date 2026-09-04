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
