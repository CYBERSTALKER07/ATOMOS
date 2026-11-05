"""OR-Tools VRP solver for pegasusX optimizer-contract JSON payloads."""

from __future__ import annotations

import math
import re
from datetime import datetime, timezone
from typing import Any

from ortools.constraint_solver import pywrapcp
from ortools.constraint_solver import routing_enums_pb2

CONTRACT_V = "v1"
SOURCE_OR_TOOLS = "OR_TOOLS_VRP"

DEFAULT_TETRIS_BUFFER = 0.95
DEFAULT_SERVICE_MINUTES = 5
DEFAULT_AVG_SPEED_KMPH = 30.0
DEFAULT_MAX_STOPS = 25
DEFAULT_TIME_LIMIT_MS = 2_000
MAX_TIME_LIMIT_MS = 60_000
DEFAULT_VEHICLE_FIXED_COST_M = 40_000
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
        "vehicle_fixed_cost_m": DEFAULT_VEHICLE_FIXED_COST_M,
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
        out["time_limit_ms"] = min(time_limit, MAX_TIME_LIMIT_MS)
    if "vehicle_fixed_cost_m" in raw and raw.get("vehicle_fixed_cost_m") is not None:
        fixed_cost = int(raw.get("vehicle_fixed_cost_m") or 0)
        if fixed_cost >= 0:
            out["vehicle_fixed_cost_m"] = fixed_cost
    return out


def _scaled_distance_m(lat1: float, lng1: float, lat2: float, lng2: float) -> int:
    return max(1, int(haversine_km(lat1, lng1, lat2, lng2) * 1000))


def _travel_minutes(distance_m: int, speed_kmph: float) -> int:
    speed = speed_kmph if speed_kmph > 0 else DEFAULT_AVG_SPEED_KMPH
    return max(1, int((distance_m / 1000.0) / speed * 60))


def solve_contract(req: dict[str, Any]) -> dict[str, Any]:
    started = datetime.now(timezone.utc)
    trace_id = str(req.get("trace_id") or "")
    stops_in = list(req.get("stops") or [])
    vehicles_in = list(req.get("vehicles") or [])
    tunables = _resolve_tunables(req.get("tunables"))

    raw_tunables = req.get("tunables") if isinstance(req.get("tunables"), dict) else {}
    req_time_limit = int(raw_tunables.get("time_limit_ms") or DEFAULT_TIME_LIMIT_MS)
    if req_time_limit <= 0:
        req_time_limit = DEFAULT_TIME_LIMIT_MS
    time_limit_requested = int(req.get("_time_limit_ms_requested") or req_time_limit)
    time_limit_applied = int(req.get("_time_limit_ms_applied") or int(tunables["time_limit_ms"]))

    if not vehicles_in:
        raise SolverError("EMPTY_FLEET", "fleet is empty")
    if not stops_in:
        raise SolverError("BAD_REQUEST", "stops slice is empty")

    depot_nodes: list[tuple[float, float]] = []
    depot_map: dict[tuple[float, float], int] = {}

    def get_or_add_depot(lat: float, lng: float) -> int:
        coord = (round(lat, 6), round(lng, 6))
        if coord not in depot_map:
            idx = len(depot_nodes)
            depot_map[coord] = idx
            depot_nodes.append((lat, lng))
        return depot_map[coord]

    starts: list[int] = []
    ends: list[int] = []
    for v in vehicles_in:
        s_lat = float(v.get("start_lat") or 0)
        s_lng = float(v.get("start_lng") or 0)
        e_lat = float(v.get("end_lat") or s_lat)
        e_lng = float(v.get("end_lng") or s_lng)
        starts.append(get_or_add_depot(s_lat, s_lng))
        ends.append(get_or_add_depot(e_lat, e_lng))

    num_depots = len(depot_nodes)

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
        elapsed_ms = int((datetime.now(timezone.utc) - started).total_seconds() * 1000)
        return _empty_response(
            trace_id,
            len(stops_in),
            orphans,
            elapsed_ms,
            time_limit_requested,
            time_limit_applied,
        )

    node_coords = list(depot_nodes)
    for stop in stops:
        node_coords.append((float(stop["lat"]), float(stop["lng"])))

    size = len(node_coords)
    raw_matrix = req.get("distance_matrix_m")
    matrix_loaded = False
    if isinstance(raw_matrix, list) and len(raw_matrix) > 0 and all(isinstance(row, list) for row in raw_matrix):
        # Case A: matrix matches node_coords (size x size)
        if len(raw_matrix) == size and all(len(row) == size for row in raw_matrix):
            distance_matrix = [
                [max(1, int(raw_matrix[i][j])) if i != j else 0 for j in range(size)]
                for i in range(size)
            ]
            matrix_loaded = True
        # Case B: matrix matches (len(vehicles_in) + len(stops_in)) per optimizer-contract specification
        elif len(raw_matrix) == len(vehicles_in) + len(stops_in) and all(len(row) == len(vehicles_in) + len(stops_in) for row in raw_matrix):
            depot_to_v: dict[int, int] = {}
            for v_idx, d_idx in enumerate(starts):
                if d_idx not in depot_to_v:
                    depot_to_v[d_idx] = v_idx

            node_to_client: list[int] = []
            for d in range(num_depots):
                node_to_client.append(depot_to_v.get(d, 0))
            for stop in stops:
                orig_idx = stops_in.index(stop) if stop in stops_in else 0
                node_to_client.append(len(vehicles_in) + orig_idx)

            distance_matrix = [
                [
                    max(1, int(raw_matrix[node_to_client[i]][node_to_client[j]])) if i != j else 0
                    for j in range(size)
                ]
                for i in range(size)
            ]
            matrix_loaded = True

    if not matrix_loaded:
        distance_matrix = [
            [
                _scaled_distance_m(node_coords[i][0], node_coords[i][1], node_coords[j][0], node_coords[j][1])
                for j in range(size)
            ]
            for i in range(size)
        ]

    demands = [0] * num_depots
    for stop in stops:
        demands.append(max(1, int(stop["volume_vu"] * SCALE_FACTOR)))

    vehicle_caps = [
        max(1, int(float(v.get("max_volume_vu") or 0) * float(tunables["tetris_buffer"]) * SCALE_FACTOR))
        for v in vehicles_in
    ]
    vehicle_speeds = [
        float(v.get("avg_speed_kmph") or 0) or DEFAULT_AVG_SPEED_KMPH for v in vehicles_in
    ]

    manager = pywrapcp.RoutingIndexManager(size, len(vehicles_in), starts, ends)
    routing = pywrapcp.RoutingModel(manager)

    def distance_callback(from_index: int, to_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        to_node = manager.IndexToNode(to_index)
        return distance_matrix[from_node][to_node]

    distance_idx = routing.RegisterTransitCallback(distance_callback)
    routing.SetArcCostEvaluatorOfAllVehicles(distance_idx)
    # Priority 1: Minimize active vehicle count (< drop_penalty = 100_000 so stops are never dropped just to save a vehicle).
    routing.SetFixedCostOfAllVehicles(int(tunables["vehicle_fixed_cost_m"]))

    def demand_callback(from_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        return demands[from_node]

    demand_idx = routing.RegisterUnaryTransitCallback(demand_callback)
    routing.AddDimensionWithVehicleCapacity(demand_idx, 0, vehicle_caps, True, "Capacity")

    max_stops = int(tunables["max_stops_per_route"])

    def stop_count_callback(from_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        return 1 if from_node >= num_depots else 0

    stop_count_idx = routing.RegisterUnaryTransitCallback(stop_count_callback)
    routing.AddDimensionWithVehicleCapacity(
        stop_count_idx,
        0,
        [max(1, max_stops)] * len(vehicles_in),
        True,
        "StopCount",
    )

    # Per-vehicle travel time in minutes (must not reuse the meters cost callback).
    time_callback_indices: list[int] = []
    for vehicle_index, speed in enumerate(vehicle_speeds):
        def make_time_cb(veh_idx: int = vehicle_index, spd: float = speed):
            def time_callback(from_index: int, to_index: int) -> int:
                from_node = manager.IndexToNode(from_index)
                to_node = manager.IndexToNode(to_index)
                travel = _travel_minutes(distance_matrix[from_node][to_node], spd)
                # Service time is paid when leaving a customer node.
                if from_node >= num_depots:
                    travel += int(stops[from_node - num_depots]["service_minutes"])
                return travel

            return time_callback

        time_callback_indices.append(routing.RegisterTransitCallback(make_time_cb()))

    time_windows: list[tuple[int, int] | None] = [None] * num_depots
    has_time_windows = False
    for stop in stops:
        open_min = _parse_hh_mm(str(stop.get("window_open") or ""))
        close_min = _parse_hh_mm(str(stop.get("window_close") or ""))
        if open_min is not None and close_min is not None and close_min >= open_min:
            time_windows.append((open_min, close_min))
            has_time_windows = True
        else:
            time_windows.append(None)

    if has_time_windows:
        max_horizon = 24 * 60
        for tw in time_windows:
            if tw is not None:
                max_horizon = max(max_horizon, tw[1] + DEFAULT_SERVICE_MINUTES)
        routing.AddDimensionWithVehicleTransits(
            time_callback_indices, 30, max_horizon, False, "Time"
        )
        time_dim = routing.GetDimensionOrDie("Time")
        # Priority 2: Balanced time-window adherence and compact route span.
        time_dim.SetSpanCostCoefficientForAllVehicles(10)
        for node_index in range(num_depots, size):
            tw = time_windows[node_index]
            if tw is None:
                continue
            manager_index = manager.NodeToIndex(node_index)
            service = int(stops[node_index - num_depots]["service_minutes"])
            hard_upper = max(tw[0], tw[1] - service)
            time_dim.CumulVar(manager_index).SetRange(tw[0], hard_upper)
            soft_upper = max(tw[0], hard_upper - min(10, (hard_upper - tw[0]) // 4))
            if soft_upper < hard_upper:
                time_dim.SetCumulVarSoftUpperBound(manager_index, soft_upper, 50)

    # Drop infeasible stops individually instead of collapsing the whole model.
    # Penalty high enough that assignment is preferred when feasible.
    drop_penalty = 100_000
    for node_index in range(num_depots, size):
        mgr_idx = manager.NodeToIndex(node_index)
        routing.AddDisjunction([mgr_idx], drop_penalty)

        stop = stops[node_index - num_depots]
        req_cold = bool(stop.get("requires_cold_chain")) or (stop.get("handling_class") == "COLD_CHAIN")
        req_hazmat = bool(stop.get("is_hazardous")) or (stop.get("handling_class") == "HAZMAT")

        if req_cold or req_hazmat:
            allowed_vehicles: list[int] = []
            for v_idx, v in enumerate(vehicles_in):
                if req_cold and not bool(v.get("has_refrigeration")):
                    continue
                if req_hazmat and not bool(v.get("hazmat_certified")):
                    continue
                allowed_vehicles.append(v_idx)
            # Allowed values include -1 so node can be dropped via disjunction if no feasible vehicle
            routing.VehicleVar(mgr_idx).SetValues([-1] + allowed_vehicles)

    search = pywrapcp.DefaultRoutingSearchParameters()
    search.first_solution_strategy = (
        routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC
    )
    search.local_search_metaheuristic = (
        routing_enums_pb2.LocalSearchMetaheuristic.GUIDED_LOCAL_SEARCH
    )
    time_limit_ms = min(max(1, int(tunables["time_limit_ms"])), MAX_TIME_LIMIT_MS)
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
                if node_index >= num_depots:
                    stop = stops[node_index - num_depots]
                    ordered_stops.append(stop)
                    total_vu += float(stop["volume_vu"])
                    duration_min += int(stop["service_minutes"])
                index = next_index
            if not ordered_stops:
                continue
            if len(ordered_stops) > max_stops:
                for stop in ordered_stops[max_stops:]:
                    orphans.append(
                        {"order_id": str(stop.get("order_id") or ""), "reason": "max_stops_per_route"}
                    )
                ordered_stops = ordered_stops[:max_stops]
                total_vu = sum(float(s["volume_vu"]) for s in ordered_stops)
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
                    "requires_cold_chain": bool(s.get("requires_cold_chain")),
                    "is_hazardous": bool(s.get("is_hazardous")),
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
    stop_cap_saturated = (
        bool(routes)
        and len(routes) == len(vehicles_in)
        and all(len(r["stops"]) >= max_stops for r in routes)
    )
    for stop in stops:
        oid = str(stop.get("order_id") or "")
        if oid and oid not in assigned_ids and not any(o["order_id"] == oid for o in orphans):
            orphans.append(
                {
                    "order_id": oid,
                    "reason": "max_stops_per_route" if stop_cap_saturated else "unassigned",
                }
            )

    elapsed_ms = int((datetime.now(timezone.utc) - started).total_seconds() * 1000)
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
            "solver_time_limit_ms_requested": time_limit_requested,
            "solver_time_limit_ms_applied": time_limit_applied,
        },
        "solver_metadata": {
            "solver_time_limit_ms_requested": time_limit_requested,
            "solver_time_limit_ms_applied": time_limit_applied,
            "time_limit_requested_ms": time_limit_requested,
            "time_limit_applied_ms": time_limit_applied,
            "clamped": time_limit_requested > time_limit_applied,
        },
    }


def _empty_response(
    trace_id: str,
    considered: int,
    orphans: list[dict[str, str]],
    elapsed_ms: int,
    time_limit_requested: int = DEFAULT_TIME_LIMIT_MS,
    time_limit_applied: int = DEFAULT_TIME_LIMIT_MS,
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
            "solver_time_limit_ms_requested": time_limit_requested,
            "solver_time_limit_ms_applied": time_limit_applied,
        },
        "solver_metadata": {
            "solver_time_limit_ms_requested": time_limit_requested,
            "solver_time_limit_ms_applied": time_limit_applied,
            "time_limit_requested_ms": time_limit_requested,
            "time_limit_applied_ms": time_limit_applied,
            "clamped": time_limit_requested > time_limit_applied,
        },
    }
