"""OR-Tools VRP solver for pegasusX optimizer-contract JSON payloads.

Multi-depot layout: one start node per vehicle (optional distinct end), then
customer stops. Optional distance_matrix_m (meters) from Go/OSRM; else haversine.
"""

from __future__ import annotations

import math
import re
from datetime import datetime
from typing import Any

from ortools.constraint_solver import pywrapcp
from ortools.constraint_solver import routing_enums_pb2

CONTRACT_V = "v1"
SOURCE_OR_TOOLS = "OR_TOOLS_VRP"

DEFAULT_TETRIS_BUFFER = 0.95
DEFAULT_SERVICE_MINUTES = 5
DEFAULT_AVG_SPEED_KMPH = 30.0
DEFAULT_MAX_STOPS = 25
DEFAULT_TIME_LIMIT_MS = 5_000
MAX_TIME_LIMIT_MS = 60_000
SCALE_FACTOR = 10_000

_HH_MM = re.compile(r"^(\d{1,2}):(\d{2})$")


class SolverError(Exception):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message


def haversine_km(lat1: float, lng1: float, lat2: float, lng2: float) -> float:
    earth_r = 6371.0
    rad = math.pi / 180.0
    d_lat = (lat2 - lat1) * rad
    d_lng = (lng2 - lng1) * rad
    a = (
        math.sin(d_lat / 2) ** 2
        + math.cos(lat1 * rad) * math.cos(lat2 * rad) * math.sin(d_lng / 2) ** 2
    )
    return earth_r * 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))


def _parse_hh_mm(value: str) -> int | None:
    raw = (value or "").strip()
    if not raw:
        return None
    match = _HH_MM.match(raw)
    if not match:
        return None
    hours = int(match.group(1))
    minutes = int(match.group(2))
    if hours > 23 or minutes > 59:
        return None
    return hours * 60 + minutes


def _resolve_tunables(raw: dict[str, Any] | None) -> dict[str, float | int]:
    out: dict[str, float | int] = {
        "tetris_buffer": DEFAULT_TETRIS_BUFFER,
        "max_stops_per_route": DEFAULT_MAX_STOPS,
        "time_limit_ms": DEFAULT_TIME_LIMIT_MS,
    }
    if not raw:
        return out
    buffer = float(raw.get("tetris_buffer") or 0)
    if 0 < buffer <= 1:
        out["tetris_buffer"] = buffer
    max_stops = int(raw.get("max_stops_per_route") or 0)
    if max_stops > 0:
        out["max_stops_per_route"] = max_stops
    time_limit = int(raw.get("time_limit_ms") or 0)
    if time_limit > 0:
        out["time_limit_ms"] = min(max(1, time_limit), MAX_TIME_LIMIT_MS)
    return out


def _scaled_distance_m(lat1: float, lng1: float, lat2: float, lng2: float) -> int:
    return max(1, int(haversine_km(lat1, lng1, lat2, lng2) * 1000))


def _travel_minutes(distance_m: int, speed_kmph: float) -> int:
    speed = speed_kmph if speed_kmph > 0 else DEFAULT_AVG_SPEED_KMPH
    return max(1, int((distance_m / 1000.0) / speed * 60))


def _vehicle_end_coords(vehicle: dict[str, Any]) -> tuple[float, float]:
    start_lat = float(vehicle.get("start_lat") or 0)
    start_lng = float(vehicle.get("start_lng") or 0)
    end_lat = float(vehicle.get("end_lat") or 0)
    end_lng = float(vehicle.get("end_lng") or 0)
    if end_lat == 0 and end_lng == 0:
        return start_lat, start_lng
    return end_lat, end_lng


def _route_duration_cap_minutes(vehicle: dict[str, Any]) -> int | None:
    explicit = int(vehicle.get("max_route_minutes") or 0)
    if explicit > 0:
        return explicit
    start = _parse_hh_mm(str(vehicle.get("shift_start") or ""))
    end = _parse_hh_mm(str(vehicle.get("shift_end") or ""))
    if start is not None and end is not None and end > start:
        return end - start
    return None


def _matrix_ok(matrix: Any, size: int) -> bool:
    if not isinstance(matrix, list) or len(matrix) != size:
        return False
    for row in matrix:
        if not isinstance(row, list) or len(row) != size:
            return False
    return True


def _build_haversine_matrix(node_coords: list[tuple[float, float]]) -> list[list[int]]:
    size = len(node_coords)
    return [
        [
            _scaled_distance_m(
                node_coords[i][0], node_coords[i][1], node_coords[j][0], node_coords[j][1]
            )
            for j in range(size)
        ]
        for i in range(size)
    ]


def solve_contract(req: dict[str, Any]) -> dict[str, Any]:
    started = datetime.utcnow()
    trace_id = str(req.get("trace_id") or "")
    stops_in = list(req.get("stops") or [])
    vehicles_in = list(req.get("vehicles") or [])
    tunables = _resolve_tunables(req.get("tunables"))

    if not vehicles_in:
        raise SolverError("EMPTY_FLEET", "fleet is empty")
    if not stops_in:
        raise SolverError("BAD_REQUEST", "stops slice is empty")

    stops: list[dict[str, Any]] = []
    orphans: list[dict[str, str]] = []
    for stop in stops_in:
        volume = float(stop.get("volume_vu") or 0)
        if volume <= 0:
            orphans.append({"order_id": str(stop.get("order_id") or ""), "reason": "non-positive volume"})
            continue
        service = int(stop.get("service_minutes") or 0)
        if service <= 0:
            service = DEFAULT_SERVICE_MINUTES
        stops.append({**stop, "volume_vu": volume, "service_minutes": service})

    if not stops:
        elapsed_ms = int((datetime.utcnow() - started).total_seconds() * 1000)
        return _empty_response(trace_id, len(stops_in), orphans, elapsed_ms)

    num_vehicles = len(vehicles_in)
    node_coords: list[tuple[float, float]] = []
    starts: list[int] = []
    ends: list[int] = []
    for vehicle in vehicles_in:
        start_lat = float(vehicle.get("start_lat") or 0)
        start_lng = float(vehicle.get("start_lng") or 0)
        end_lat, end_lng = _vehicle_end_coords(vehicle)
        start_idx = len(node_coords)
        node_coords.append((start_lat, start_lng))
        starts.append(start_idx)
        if abs(end_lat - start_lat) > 1e-9 or abs(end_lng - start_lng) > 1e-9:
            end_idx = len(node_coords)
            node_coords.append((end_lat, end_lng))
            ends.append(end_idx)
        else:
            ends.append(start_idx)

    customer_offset = len(node_coords)
    node_to_stop: dict[int, int] = {}
    for i, stop in enumerate(stops):
        idx = len(node_coords)
        node_coords.append((float(stop["lat"]), float(stop["lng"])))
        node_to_stop[idx] = i

    size = len(node_coords)
    provided = req.get("distance_matrix_m")
    if _matrix_ok(provided, size):
        distance_matrix = [[max(0, int(v)) for v in row] for row in provided]
        # Self-loops and zero arcs still need a positive cost for OR-Tools stability.
        for i in range(size):
            for j in range(size):
                if i != j and distance_matrix[i][j] <= 0:
                    distance_matrix[i][j] = _scaled_distance_m(
                        node_coords[i][0], node_coords[i][1],
                        node_coords[j][0], node_coords[j][1],
                    )
                if i == j:
                    distance_matrix[i][j] = 0
    else:
        distance_matrix = _build_haversine_matrix(node_coords)

    demands = [0] * size
    for node_idx, stop_idx in node_to_stop.items():
        demands[node_idx] = max(1, int(stops[stop_idx]["volume_vu"] * SCALE_FACTOR))

    vehicle_caps = [
        max(1, int(float(v.get("max_volume_vu") or 0) * float(tunables["tetris_buffer"]) * SCALE_FACTOR))
        for v in vehicles_in
    ]
    vehicle_speeds = [
        float(v.get("avg_speed_kmph") or 0) or DEFAULT_AVG_SPEED_KMPH for v in vehicles_in
    ]

    manager = pywrapcp.RoutingIndexManager(size, num_vehicles, starts, ends)
    routing = pywrapcp.RoutingModel(manager)

    def distance_callback(from_index: int, to_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        to_node = manager.IndexToNode(to_index)
        return distance_matrix[from_node][to_node]

    distance_idx = routing.RegisterTransitCallback(distance_callback)
    routing.SetArcCostEvaluatorOfAllVehicles(distance_idx)

    def demand_callback(from_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        return demands[from_node]

    demand_idx = routing.RegisterUnaryTransitCallback(demand_callback)
    routing.AddDimensionWithVehicleCapacity(demand_idx, 0, vehicle_caps, True, "Capacity")

    # Count dimension: each customer consumes 1 stop slot; capacity = max_stops.
    max_stops = int(tunables["max_stops_per_route"])

    def stop_count_callback(from_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        return 1 if from_node in node_to_stop else 0

    stop_count_idx = routing.RegisterUnaryTransitCallback(stop_count_callback)
    routing.AddDimensionWithVehicleCapacity(
        stop_count_idx,
        0,
        [max_stops] * num_vehicles,
        True,
        "StopCount",
    )

    # Per-vehicle travel time in minutes.
    time_callback_indices: list[int] = []
    for vehicle_index, speed in enumerate(vehicle_speeds):
        def make_time_cb(veh_idx: int = vehicle_index, spd: float = speed):
            def time_callback(from_index: int, to_index: int) -> int:
                from_node = manager.IndexToNode(from_index)
                to_node = manager.IndexToNode(to_index)
                travel = _travel_minutes(distance_matrix[from_node][to_node], spd)
                if from_node in node_to_stop:
                    travel += int(stops[node_to_stop[from_node]]["service_minutes"])
                return travel

            return time_callback

        time_callback_indices.append(routing.RegisterTransitCallback(make_time_cb()))

    time_windows: dict[int, tuple[int, int]] = {}
    has_time_windows = False
    for node_idx, stop_idx in node_to_stop.items():
        stop = stops[stop_idx]
        open_min = _parse_hh_mm(str(stop.get("window_open") or ""))
        close_min = _parse_hh_mm(str(stop.get("window_close") or ""))
        if open_min is not None and close_min is not None and close_min >= open_min:
            time_windows[node_idx] = (open_min, close_min)
            has_time_windows = True

    duration_caps = [_route_duration_cap_minutes(v) for v in vehicles_in]
    has_duration_caps = any(c is not None for c in duration_caps)
    max_horizon = 24 * 60
    for tw in time_windows.values():
        max_horizon = max(max_horizon, tw[1] + DEFAULT_SERVICE_MINUTES)
    for cap in duration_caps:
        if cap is not None:
            max_horizon = max(max_horizon, cap)

    if has_time_windows or has_duration_caps:
        routing.AddDimensionWithVehicleTransits(
            time_callback_indices, 30, max_horizon, False, "Time"
        )
        time_dim = routing.GetDimensionOrDie("Time")
        for node_idx, tw in time_windows.items():
            manager_index = manager.NodeToIndex(node_idx)
            service = int(stops[node_to_stop[node_idx]]["service_minutes"])
            time_dim.CumulVar(manager_index).SetRange(tw[0], max(tw[0], tw[1] - service))
        for vehicle_index, cap in enumerate(duration_caps):
            if cap is None:
                continue
            # RouteDuration: end cumul - start cumul ≤ cap.
            time_dim.SetSpanUpperBoundForVehicle(cap, vehicle_index)

    # Cold-chain / hazmat vehicle eligibility.
    reefer_vehicles = [
        i for i, v in enumerate(vehicles_in) if bool(v.get("has_refrigeration"))
    ]
    hazmat_vehicles = [
        i for i, v in enumerate(vehicles_in) if bool(v.get("hazmat_certified"))
    ]
    for node_idx, stop_idx in node_to_stop.items():
        stop = stops[stop_idx]
        manager_index = manager.NodeToIndex(node_idx)
        needs_cold = bool(stop.get("requires_cold_chain"))
        needs_hazmat = bool(stop.get("is_hazardous"))
        allowed: set[int] | None = None
        if needs_cold:
            allowed = set(reefer_vehicles)
        if needs_hazmat:
            haz = set(hazmat_vehicles)
            allowed = haz if allowed is None else allowed & haz
        if allowed is not None:
            # VehicleVar.SetValues is the portable SWIG binding for vehicle filters.
            routing.VehicleVar(manager_index).SetValues(sorted(int(v) for v in allowed))

    # Drop infeasible stops individually instead of collapsing the whole model.
    drop_penalty = 100_000
    for node_idx in node_to_stop:
        routing.AddDisjunction([manager.NodeToIndex(node_idx)], drop_penalty)

    search = pywrapcp.DefaultRoutingSearchParameters()
    search.first_solution_strategy = (
        routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC
    )
    search.local_search_metaheuristic = (
        routing_enums_pb2.LocalSearchMetaheuristic.GUIDED_LOCAL_SEARCH
    )
    time_limit_ms = int(tunables["time_limit_ms"])
    search.time_limit.FromMilliseconds(time_limit_ms)

    solution = routing.SolveWithParameters(search)

    routes: list[dict[str, Any]] = []
    placed = 0
    if solution is not None:
        for vehicle_index, vehicle in enumerate(vehicles_in):
            index = routing.Start(vehicle_index)
            ordered_stops: list[dict[str, Any]] = []
            total_vu = 0.0
            distance_m = 0
            duration_min = 0
            while not routing.IsEnd(index):
                node_index = manager.IndexToNode(index)
                next_index = solution.Value(routing.NextVar(index))
                next_node = manager.IndexToNode(next_index)
                distance_m += distance_matrix[node_index][next_node]
                duration_min += _travel_minutes(
                    distance_matrix[node_index][next_node], vehicle_speeds[vehicle_index]
                )
                if node_index in node_to_stop:
                    stop = stops[node_to_stop[node_index]]
                    ordered_stops.append(stop)
                    total_vu += float(stop["volume_vu"])
                    duration_min += int(stop["service_minutes"])
                index = next_index
            if not ordered_stops:
                continue
            cap = float(vehicle.get("max_volume_vu") or 0)
            util_pct = (total_vu / cap * 100.0) if cap > 0 else 0.0
            wire_stops = [
                {
                    "order_id": s.get("order_id"),
                    "retailer_id": s.get("retailer_id"),
                    "lat": s.get("lat"),
                    "lng": s.get("lng"),
                    "h3_cell": s.get("h3_cell") or "",
                    "volume_vu": s.get("volume_vu"),
                    "window_open": s.get("window_open") or "",
                    "window_close": s.get("window_close") or "",
                    "service_minutes": s.get("service_minutes"),
                    "priority": int(s.get("priority") or 0),
                    "handling_class": s.get("handling_class") or "",
                    "requires_cold_chain": bool(s.get("requires_cold_chain")),
                    "is_hazardous": bool(s.get("is_hazardous")),
                    "access_restriction": s.get("access_restriction") or "",
                }
                for s in ordered_stops
            ]
            placed += len(wire_stops)
            routes.append(
                {
                    "vehicle_id": str(vehicle.get("vehicle_id") or ""),
                    "driver_id": str(vehicle.get("driver_id") or ""),
                    "stops": wire_stops,
                    "total_vu": total_vu,
                    "util_pct": util_pct,
                    "distance_km": distance_m / 1000.0,
                    "duration_min": duration_min,
                }
            )
    else:
        for stop in stops:
            orphans.append(
                {"order_id": str(stop.get("order_id") or ""), "reason": "no feasible solution"}
            )

    assigned_ids = {s.get("order_id") for r in routes for s in r["stops"]}
    for stop in stops:
        oid = str(stop.get("order_id") or "")
        if oid and oid not in assigned_ids and not any(o["order_id"] == oid for o in orphans):
            reason = "unassigned"
            if bool(stop.get("requires_cold_chain")) and not reefer_vehicles:
                reason = "no_refrigerated_vehicle"
            elif bool(stop.get("is_hazardous")) and not hazmat_vehicles:
                reason = "no_hazmat_vehicle"
            orphans.append({"order_id": oid, "reason": reason})

    elapsed_ms = int((datetime.utcnow() - started).total_seconds() * 1000)
    util_sum = sum(float(r["util_pct"]) for r in routes)
    avg_util = util_sum / len(routes) if routes else 0.0
    return {
        "v": CONTRACT_V,
        "trace_id": trace_id,
        "source": SOURCE_OR_TOOLS,
        "routes": routes,
        "orphans": orphans,
        "stats": {
            "elapsed_ms": elapsed_ms,
            "stops_considered": len(stops_in),
            "stops_placed": placed,
            "stops_orphaned": len(orphans),
            "vehicles_used": len(routes),
            "avg_utilisation_pct": avg_util,
            "two_opt_improvement_pct": 0.0,
        },
    }


def _empty_response(
    trace_id: str, considered: int, orphans: list[dict[str, str]], elapsed_ms: int
) -> dict[str, Any]:
    return {
        "v": CONTRACT_V,
        "trace_id": trace_id,
        "source": SOURCE_OR_TOOLS,
        "routes": [],
        "orphans": orphans,
        "stats": {
            "elapsed_ms": elapsed_ms,
            "stops_considered": considered,
            "stops_placed": 0,
            "stops_orphaned": len(orphans),
            "vehicles_used": 0,
            "avg_utilisation_pct": 0.0,
            "two_opt_improvement_pct": 0.0,
        },
    }
