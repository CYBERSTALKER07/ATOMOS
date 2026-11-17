import hmac
import math
import os
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
from ortools.constraint_solver import routing_enums_pb2
from ortools.constraint_solver import pywrapcp

app = FastAPI(title="PegasusX Optimizer Sidecar")

_INSECURE_DEFAULT_KEYS = {"", "dev-internal-key", "changeme", "secret"}


def _expected_internal_api_key() -> str:
    return (
        os.environ.get("OPTIMIZER_INTERNAL_API_KEY")
        or os.environ.get("INTERNAL_API_KEY")
        or ""
    ).strip()


def _is_production_env() -> bool:
    env = (os.environ.get("PEGASUSX_ENV") or os.environ.get("ENVIRONMENT") or "").strip().lower()
    return env in {"production", "prod", "staging"}


def verify_internal_api_key(provided_key: Optional[str]) -> Optional[tuple]:
    """Returns None when authorized, else (status_code, error_detail)."""
    expected = _expected_internal_api_key()
    if _is_production_env() and expected in _INSECURE_DEFAULT_KEYS:
        return (503, "internal_api_key_unconfigured")
    if not expected:
        return None
    supplied = (provided_key or "").strip()
    if not supplied or not hmac.compare_digest(supplied, expected):
        return (401, "invalid_internal_api_key")
    return None


if hasattr(app, "middleware"):
    from fastapi.responses import JSONResponse

    @app.middleware("http")
    async def internal_api_key_middleware(request, call_next):
        if request.url.path == "/healthz":
            return await call_next(request)
        err = verify_internal_api_key(request.headers.get("X-Internal-Api-Key"))
        if err is not None:
            status_code, detail = err
            return JSONResponse(status_code=status_code, content={"detail": detail})
        return await call_next(request)

@app.get("/healthz")
def healthz():
    return {"status": "ok"}

MAX_LOCATIONS = 500


class Location(BaseModel):
    id: str
    x: float
    y: float

class PickPathRequest(BaseModel):
    warehouse_id: str
    locations: List[Location]

class PickPathResponse(BaseModel):
    ordered_location_ids: List[str]
    solver_status: str = "HEURISTIC"

def create_data_model(locations: List[Location]):
    """Stores the data for the problem."""
    data = {}
    
    # Compute distance matrix (Euclidean distances scaled to integers)
    distance_matrix = []
    for i, loc1 in enumerate(locations):
        row = []
        for j, loc2 in enumerate(locations):
            if i == j:
                row.append(0)
            else:
                dist = math.hypot(loc1.x - loc2.x, loc1.y - loc2.y)
                row.append(int(dist * 1000)) # Scale up to avoid float truncation
        distance_matrix.append(row)
        
    data['distance_matrix'] = distance_matrix
    data['num_vehicles'] = 1
    data['depot'] = 0 # Assume the first location is the depot or start point
    return data

def solve_tsp(locations: List[Location]) -> List[str]:
    if not locations:
        return []
    if len(locations) == 1:
        return [locations[0].id]

    data = create_data_model(locations)

    # Create the routing index manager.
    manager = pywrapcp.RoutingIndexManager(
        len(data['distance_matrix']), data['num_vehicles'], data['depot'])

    # Create Routing Model.
    routing = pywrapcp.RoutingModel(manager)

    def distance_callback(from_index, to_index):
        """Returns the distance between the two nodes."""
        # Convert from routing variable Index to distance matrix NodeIndex.
        from_node = manager.IndexToNode(from_index)
        to_node = manager.IndexToNode(to_index)
        return data['distance_matrix'][from_node][to_node]

    transit_callback_index = routing.RegisterTransitCallback(distance_callback)

    # Define cost of each arc.
    routing.SetArcCostEvaluatorOfAllVehicles(transit_callback_index)

    # Setting first solution heuristic and solver deadline.
    search_parameters = pywrapcp.DefaultRoutingSearchParameters()
    search_parameters.first_solution_strategy = (
        routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC)
    search_parameters.time_limit.FromSeconds(2)

    # Solve the problem.
    solution = routing.SolveWithParameters(search_parameters)

    if solution:
        index = routing.Start(0)
        ordered_ids = []
        while not routing.IsEnd(index):
            node_index = manager.IndexToNode(index)
            ordered_ids.append(locations[node_index].id)
            index = solution.Value(routing.NextVar(index))
        return ordered_ids
    else:
        return [loc.id for loc in locations] # Fallback to original order


@app.post("/pick-path", response_model=PickPathResponse)
def get_pick_path(req: PickPathRequest):
    if len(req.locations) > MAX_LOCATIONS:
        raise HTTPException(status_code=400, detail=f"locations count exceeds {MAX_LOCATIONS}")
    for loc in req.locations:
        if not math.isfinite(loc.x) or not math.isfinite(loc.y):
            raise HTTPException(status_code=400, detail="location coordinates must be finite numbers")
    try:
        ordered_ids = solve_tsp(req.locations)
        return PickPathResponse(ordered_location_ids=ordered_ids, solver_status="HEURISTIC")
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

class FleetLocation(BaseModel):
    id: str
    x: float
    y: float
    demand_vu: int

class FleetVehicle(BaseModel):
    id: str
    capacity_vu: int

class FleetRouteRequest(BaseModel):
    depot: FleetLocation
    locations: List[FleetLocation]
    vehicles: List[FleetVehicle]

class VehicleRoute(BaseModel):
    vehicle_id: str
    wave_index: int = 1
    location_ids: List[str]

class FleetRouteResponse(BaseModel):
    routes: List[VehicleRoute]
    solver_status: str = "HEURISTIC"

def haversine_distance_meters(lat1: float, lon1: float, lat2: float, lon2: float) -> int:
    r = 6371000.0  # Earth radius in meters
    phi1, phi2 = math.radians(lat1), math.radians(lat2)
    dphi = math.radians(lat2 - lat1)
    dlambda = math.radians(lon2 - lon1)
    a = math.sin(dphi / 2.0) ** 2 + math.cos(phi1) * math.cos(phi2) * math.sin(dlambda / 2.0) ** 2
    a = min(1.0, max(0.0, a))
    return int(2.0 * r * math.atan2(math.sqrt(a), math.sqrt(max(0.0, 1.0 - a))))

def fleet_distance_metric(loc1: FleetLocation, loc2: FleetLocation) -> int:
    if -90.0 <= loc1.x <= 90.0 and -90.0 <= loc2.x <= 90.0 and -180.0 <= loc1.y <= 180.0 and -180.0 <= loc2.y <= 180.0:
        return haversine_distance_meters(loc1.x, loc1.y, loc2.x, loc2.y)
    return int(math.hypot(loc1.x - loc2.x, loc1.y - loc2.y) * 1000)

def compute_ffd_waves(demands: List[int], vehicle_capacities: List[int]) -> int:
    """Computes a First-Fit Decreasing (FFD) lower bound on the number of fleet waves needed."""
    positive_demands = sorted([d for d in demands if d > 0], reverse=True)
    positive_caps = [c for c in vehicle_capacities if c > 0]
    if not positive_demands or not positive_caps:
        return 1

    max_cap = max(positive_caps)
    bin_remainders: List[int] = []
    for item in positive_demands:
        if item > max_cap:
            continue
        placed = False
        for i in range(len(bin_remainders)):
            if bin_remainders[i] >= item:
                bin_remainders[i] -= item
                placed = True
                break
        while not placed:
            wave_start_idx = len(bin_remainders)
            bin_remainders.extend(positive_caps)
            for i in range(wave_start_idx, len(bin_remainders)):
                if bin_remainders[i] >= item:
                    bin_remainders[i] -= item
                    placed = True
                    break

    if not bin_remainders:
        return 1
    num_bins_used = len(bin_remainders)
    num_veh = len(positive_caps)
    return max(1, math.ceil(num_bins_used / num_veh))

def create_cvrp_data_model(req: FleetRouteRequest):
    data = {}
    all_locations = [req.depot] + req.locations
    
    distance_matrix = []
    for i, loc1 in enumerate(all_locations):
        row = []
        for j, loc2 in enumerate(all_locations):
            if i == j:
                row.append(0)
            else:
                row.append(fleet_distance_metric(loc1, loc2))
        distance_matrix.append(row)
        
    data['distance_matrix'] = distance_matrix
    data['demands'] = [loc.demand_vu for loc in all_locations]
    
    # MULTI-WAVE LOGIC:
    # If total demand > total capacity, or if discrete stop demands cannot pack
    # into math.ceil(total_demand / total_capacity) trips under FFD, we create
    # multiple virtual vehicles for each physical vehicle to allow multiple trips (waves).
    total_demand = sum(data['demands'])
    total_capacity = sum(v.capacity_vu for v in req.vehicles)
    
    multiplier = 1
    if total_capacity > 0 and total_demand > total_capacity:
        multiplier = math.ceil(total_demand / total_capacity)
    
    ffd_waves = compute_ffd_waves([loc.demand_vu for loc in req.locations], [v.capacity_vu for v in req.vehicles])
    multiplier = max(multiplier, ffd_waves)
    
    vehicle_capacities = []
    vehicle_ids = []
    wave_indices = []
    
    for wave_idx in range(1, multiplier + 1):
        for v in req.vehicles:
            vehicle_capacities.append(v.capacity_vu)
            vehicle_ids.append(v.id)
            wave_indices.append(wave_idx)
            
    data['vehicle_capacities'] = vehicle_capacities
    data['vehicle_ids'] = vehicle_ids
    data['wave_indices'] = wave_indices
    data['num_vehicles'] = len(vehicle_capacities)
    data['depot'] = 0
    return data

@app.post("/fleet-route", response_model=FleetRouteResponse)
def get_fleet_route(req: FleetRouteRequest):
    if len(req.locations) > MAX_LOCATIONS:
        raise HTTPException(status_code=400, detail=f"locations count exceeds {MAX_LOCATIONS}")
    for loc in [req.depot] + req.locations:
        if not math.isfinite(loc.x) or not math.isfinite(loc.y):
            raise HTTPException(status_code=400, detail="location coordinates must be finite numbers")
        if loc.demand_vu < 0:
            raise HTTPException(status_code=400, detail="demand_vu must be non-negative")
    for v in req.vehicles:
        if v.capacity_vu <= 0:
            raise HTTPException(status_code=400, detail="vehicle capacity_vu must be positive")
    if req.locations and req.vehicles:
        max_cap = max(v.capacity_vu for v in req.vehicles)
        if any(loc.demand_vu > max_cap for loc in req.locations):
            raise HTTPException(status_code=422, detail="stop demand exceeds maximum single vehicle capacity")

    try:
        if not req.locations or not req.vehicles:
            return FleetRouteResponse(routes=[], solver_status="HEURISTIC")
            
        data = create_cvrp_data_model(req)
        manager = pywrapcp.RoutingIndexManager(
            len(data['distance_matrix']), data['num_vehicles'], data['depot'])
        routing = pywrapcp.RoutingModel(manager)

        def distance_callback(from_index, to_index):
            from_node = manager.IndexToNode(from_index)
            to_node = manager.IndexToNode(to_index)
            return data['distance_matrix'][from_node][to_node]

        transit_callback_index = routing.RegisterTransitCallback(distance_callback)
        routing.SetArcCostEvaluatorOfAllVehicles(transit_callback_index)

        def demand_callback(from_index):
            from_node = manager.IndexToNode(from_index)
            return data['demands'][from_node]

        demand_callback_index = routing.RegisterUnaryTransitCallback(demand_callback)
        routing.AddDimensionWithVehicleCapacity(
            demand_callback_index,
            0,  # null capacity slack
            data['vehicle_capacities'],  # vehicle maximum capacities
            True,  # start cumul to zero
            'Capacity')

        search_parameters = pywrapcp.DefaultRoutingSearchParameters()
        search_parameters.first_solution_strategy = (
            routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC)
        search_parameters.local_search_metaheuristic = (
            routing_enums_pb2.LocalSearchMetaheuristic.GUIDED_LOCAL_SEARCH)
        search_parameters.time_limit.FromSeconds(2) # Prevent hanging on huge datasets

        solution = routing.SolveWithParameters(search_parameters)
        
        if not solution:
            raise HTTPException(status_code=422, detail="No feasible solution found by OR-Tools")

        final_routes = []
        for vehicle_idx in range(data['num_vehicles']):
            real_vehicle_id = data['vehicle_ids'][vehicle_idx]
            wave_idx = data['wave_indices'][vehicle_idx]
                
            index = routing.Start(vehicle_idx)
            wave_route = []
            while not routing.IsEnd(index):
                node_index = manager.IndexToNode(index)
                if node_index != data['depot']:
                    # Sub offset 1 because locations array was [depot] + locations
                    wave_route.append(req.locations[node_index - 1].id)
                index = solution.Value(routing.NextVar(index))
                
            if wave_route:
                final_routes.append(VehicleRoute(
                    vehicle_id=real_vehicle_id,
                    wave_index=wave_idx,
                    location_ids=wave_route
                ))
        
        return FleetRouteResponse(routes=final_routes, solver_status="HEURISTIC")
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

class DemandPredictionRequest(BaseModel):
    sku: str
    historical_daily_sales: List[int]
    current_stock: int
    lead_time_days: int
    pipeline_stock: Optional[int] = 0
    demand_variance: Optional[float] = None
    service_level: Optional[float] = None

class DemandPredictionResponse(BaseModel):
    sku: str
    optimal_reorder_qty: int
    predicted_daily_burn: float

@app.post("/predict-demand", response_model=DemandPredictionResponse)
def predict_demand(req: DemandPredictionRequest):
    if req.current_stock < 0 or req.lead_time_days < 0:
        raise HTTPException(status_code=400, detail="current_stock and lead_time_days must be non-negative")
    if req.pipeline_stock is not None and req.pipeline_stock < 0:
        raise HTTPException(status_code=400, detail="pipeline_stock must be non-negative")
    if req.demand_variance is not None and req.demand_variance < 0:
        raise HTTPException(status_code=400, detail="demand_variance must be non-negative")
    if any(sale < 0 for sale in req.historical_daily_sales):
        raise HTTPException(status_code=400, detail="historical_daily_sales entries must be non-negative")
    try:
        if not req.historical_daily_sales:
            return DemandPredictionResponse(sku=req.sku, optimal_reorder_qty=0, predicted_daily_burn=0.0)

        # Simple Moving Average (SMA) heuristic for burn rate
        avg_burn = sum(req.historical_daily_sales) / len(req.historical_daily_sales)

        # Expected stock during lead time
        expected_burn = avg_burn * req.lead_time_days

        # Safety stock: analytical if demand_variance provided, otherwise 3 days of avg burn
        if req.demand_variance is not None and req.demand_variance > 0:
            sl = req.service_level if (req.service_level and 0.5 < req.service_level < 1.0) else 0.95
            if sl >= 0.99:
                z = 2.326
            elif sl >= 0.98:
                z = 2.054
            elif sl >= 0.95:
                z = 1.645
            elif sl >= 0.90:
                z = 1.282
            else:
                z = 1.0
            safety_stock = z * math.sqrt(float(req.lead_time_days) * req.demand_variance)
        else:
            safety_stock = avg_burn * 3

        target_stock = expected_burn + safety_stock
        effective_stock = req.current_stock + (req.pipeline_stock or 0)

        reorder_qty = max(0, math.ceil(target_stock - effective_stock))

        return DemandPredictionResponse(
            sku=req.sku,
            optimal_reorder_qty=reorder_qty,
            predicted_daily_burn=avg_burn
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
