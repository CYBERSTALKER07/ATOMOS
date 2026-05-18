from __future__ import annotations

from ortools.constraint_solver import pywrapcp
from ortools.constraint_solver import routing_enums_pb2

from . import optimizer_core_pb2 as pb2
from .mapping import BidirectionalIndexMap

DEFAULT_VRP_TIME_LIMIT_MS = 2_000
MAX_VRP_TIME_LIMIT_MS = 60_000


def solve_vrp(request: pb2.VRPRequest) -> pb2.VRPResponse:
    response = pb2.VRPResponse()
    response.meta.CopyFrom(request.meta)

    warnings: list[str] = []

    try:
        ordered_nodes = [request.depot_node_uuid] + list(request.drop_off_node_uuids)
        node_map = BidirectionalIndexMap(ordered_nodes)
    except ValueError as exc:
        response.feasible = False
        response.timed_out = False
        response.warnings.append(str(exc))
        return response

    if node_map.size <= 1:
        response.feasible = True
        return response

    if len(request.vehicles) == 0:
        response.feasible = False
        response.warnings.append("no vehicles supplied")
        return response

    try:
        distance_matrix = _matrix_from_request(request.distance_matrix_scaled, node_map.size)
    except ValueError as exc:
        response.feasible = False
        response.warnings.append(str(exc))
        return response

    manager = pywrapcp.RoutingIndexManager(node_map.size, len(request.vehicles), 0)
    routing = pywrapcp.RoutingModel(manager)

    demand_by_uuid = {d.node_uuid: max(0, d.demand_scaled) for d in request.node_demands}
    demands = [0]
    for uuid in request.drop_off_node_uuids:
        demands.append(demand_by_uuid.get(uuid, 0))

    def transit_callback(from_index: int, to_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        to_node = manager.IndexToNode(to_index)
        return distance_matrix[from_node][to_node]

    transit_callback_index = routing.RegisterTransitCallback(transit_callback)
    routing.SetArcCostEvaluatorOfAllVehicles(transit_callback_index)

    def demand_callback(from_index: int) -> int:
        from_node = manager.IndexToNode(from_index)
        return demands[from_node]

    demand_callback_index = routing.RegisterUnaryTransitCallback(demand_callback)
    vehicle_capacities = [max(0, v.capacity_scaled) for v in request.vehicles]
    routing.AddDimensionWithVehicleCapacity(
        demand_callback_index,
        0,
        vehicle_capacities,
        True,
        "Capacity",
    )

    time_window_by_uuid = {
        tw.node_uuid: (tw.start_time_window_scaled, tw.end_time_window_scaled)
        for tw in request.node_time_windows
    }
    has_time_windows = bool(time_window_by_uuid)

    if has_time_windows:
        max_horizon = _resolve_max_horizon(request, time_window_by_uuid)
        routing.AddDimension(
            transit_callback_index,
            0,
            max_horizon,
            False,
            "Time",
        )
        time_dimension = routing.GetDimensionOrDie("Time")

        for vehicle_index, vehicle in enumerate(request.vehicles):
            start = routing.Start(vehicle_index)
            end = routing.End(vehicle_index)
            window_start = max(0, vehicle.start_time_window_scaled)
            window_end = max(window_start, vehicle.end_time_window_scaled)
            if window_end == 0:
                window_end = max_horizon
            time_dimension.CumulVar(start).SetRange(window_start, window_end)
            time_dimension.CumulVar(end).SetRange(window_start, window_end)

        for node_uuid in request.drop_off_node_uuids:
            if node_uuid not in time_window_by_uuid:
                continue
            node_index = node_map.index_of(node_uuid)
            manager_index = manager.NodeToIndex(node_index)
            if manager_index < 0:
                continue
            start_window, end_window = time_window_by_uuid[node_uuid]
            start_window = max(0, start_window)
            end_window = max(start_window, end_window)
            time_dimension.CumulVar(manager_index).SetRange(start_window, end_window)

    search_parameters = pywrapcp.DefaultRoutingSearchParameters()
    search_parameters.first_solution_strategy = routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC
    search_parameters.local_search_metaheuristic = routing_enums_pb2.LocalSearchMetaheuristic.GUIDED_LOCAL_SEARCH

    time_limit_ms = request.solver_time_limit_ms or DEFAULT_VRP_TIME_LIMIT_MS
    time_limit_ms = min(max(1, time_limit_ms), MAX_VRP_TIME_LIMIT_MS)
    search_parameters.time_limit.FromMilliseconds(time_limit_ms)

    solution = routing.SolveWithParameters(search_parameters)

    timeout_status = getattr(pywrapcp.RoutingModel, "ROUTING_FAIL_TIMEOUT", -1)
    response.timed_out = routing.status() == timeout_status

    if solution is None:
        response.feasible = False
        if response.timed_out:
            response.warnings.append("solver timed out without feasible solution")
        else:
            response.warnings.append("no feasible solution found")
        return response

    response.feasible = True
    response.objective_cost_scaled = int(solution.ObjectiveValue())

    assigned_nodes: set[str] = set()
    for vehicle_index, vehicle in enumerate(request.vehicles):
        index = routing.Start(vehicle_index)
        route_cost = 0
        route_load = 0
        ordered_dropoffs: list[str] = []

        while not routing.IsEnd(index):
            node_index = manager.IndexToNode(index)
            next_index = solution.Value(routing.NextVar(index))
            next_node_index = manager.IndexToNode(next_index)
            route_cost += distance_matrix[node_index][next_node_index]

            if node_index != 0:
                node_uuid = node_map.uuid_of(node_index)
                ordered_dropoffs.append(node_uuid)
                assigned_nodes.add(node_uuid)
                route_load += demand_by_uuid.get(node_uuid, 0)

            index = next_index

        if ordered_dropoffs:
            response.routes.append(
                pb2.VehicleRoute(
                    vehicle_uuid=vehicle.vehicle_uuid,
                    driver_uuid=vehicle.driver_uuid,
                    ordered_node_uuids=ordered_dropoffs,
                    load_scaled=route_load,
                    route_cost_scaled=route_cost,
                )
            )

    response.unassigned_node_uuids.extend(
        [uuid for uuid in request.drop_off_node_uuids if uuid not in assigned_nodes]
    )

    response.warnings.extend(warnings)
    return response


def _matrix_from_request(rows: list[pb2.Int64Row], expected_size: int) -> list[list[int]]:
    matrix: list[list[int]] = [list(row.values) for row in rows]
    if len(matrix) != expected_size:
        raise ValueError(
            f"distance matrix row count mismatch: got {len(matrix)} expected {expected_size}"
        )
    for row in matrix:
        if len(row) != expected_size:
            raise ValueError(
                f"distance matrix column count mismatch: got {len(row)} expected {expected_size}"
            )
    return matrix


def _resolve_max_horizon(
    request: pb2.VRPRequest,
    time_window_by_uuid: dict[str, tuple[int, int]],
) -> int:
    max_end = 0
    for _, end_window in time_window_by_uuid.values():
        max_end = max(max_end, end_window)
    for vehicle in request.vehicles:
        max_end = max(max_end, vehicle.end_time_window_scaled)
    if max_end <= 0:
        return 24 * 60 * 60 * request.meta.scale_factor
    return max_end
