import math
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List
from ortools.constraint_solver import routing_enums_pb2
from ortools.constraint_solver import pywrapcp

app = FastAPI(title="PegasusX Optimizer Sidecar")

class Location(BaseModel):
    id: str
    x: float
    y: float

class PickPathRequest(BaseModel):
    warehouse_id: str
    locations: List[Location]

class PickPathResponse(BaseModel):
    ordered_location_ids: List[str]

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

    # Setting first solution heuristic.
    search_parameters = pywrapcp.DefaultRoutingSearchParameters()
    search_parameters.first_solution_strategy = (
        routing_enums_pb2.FirstSolutionStrategy.PATH_CHEAPEST_ARC)

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
    try:
        ordered_ids = solve_tsp(req.locations)
        return PickPathResponse(ordered_location_ids=ordered_ids)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

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
    location_ids: List[str]

class FleetRouteResponse(BaseModel):
    routes: List[VehicleRoute]

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
                dist = math.hypot(loc1.x - loc2.x, loc1.y - loc2.y)
                row.append(int(dist * 1000))
        distance_matrix.append(row)
        
    data['distance_matrix'] = distance_matrix
    data['demands'] = [loc.demand_vu for loc in all_locations]
    
    # MULTI-WAVE LOGIC:
    # If total demand > total capacity, we create multiple virtual vehicles for each physical vehicle
    # to allow multiple trips (waves).
    total_demand = sum(data['demands'])
    total_capacity = sum(v.capacity_vu for v in req.vehicles)
    
    multiplier = 1
    if total_capacity > 0 and total_demand > total_capacity:
        multiplier = math.ceil(total_demand / total_capacity)
    
    vehicle_capacities = []
    vehicle_ids = []
    
    for _ in range(multiplier):
        for v in req.vehicles:
            vehicle_capacities.append(v.capacity_vu)
            vehicle_ids.append(v.id)
            
    data['vehicle_capacities'] = vehicle_capacities
    data['vehicle_ids'] = vehicle_ids
    data['num_vehicles'] = len(vehicle_capacities)
    data['depot'] = 0
    return data

@app.post("/fleet-route", response_model=FleetRouteResponse)
def get_fleet_route(req: FleetRouteRequest):
    try:
        if not req.locations or not req.vehicles:
            return FleetRouteResponse(routes=[])
            
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
            raise Exception("No solution found by OR-Tools")

        routes_map = {}
        for vehicle_idx in range(data['num_vehicles']):
            real_vehicle_id = data['vehicle_ids'][vehicle_idx]
            if real_vehicle_id not in routes_map:
                routes_map[real_vehicle_id] = []
                
            index = routing.Start(vehicle_idx)
            wave_route = []
            while not routing.IsEnd(index):
                node_index = manager.IndexToNode(index)
                if node_index != data['depot']:
                    # Sub offset 1 because locations array was [depot] + locations
                    wave_route.append(req.locations[node_index - 1].id)
                index = solution.Value(routing.NextVar(index))
                
            if wave_route:
                routes_map[real_vehicle_id].extend(wave_route)
                
        final_routes = [VehicleRoute(vehicle_id=vid, location_ids=lids) 
                        for vid, lids in routes_map.items() if lids]
        
        return FleetRouteResponse(routes=final_routes)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

class DemandPredictionRequest(BaseModel):
    sku: str
    historical_daily_sales: List[int]
    current_stock: int
    lead_time_days: int

class DemandPredictionResponse(BaseModel):
    sku: str
    optimal_reorder_qty: int
    predicted_daily_burn: float

@app.post("/predict-demand", response_model=DemandPredictionResponse)
def predict_demand(req: DemandPredictionRequest):
    try:
        if not req.historical_daily_sales:
            return DemandPredictionResponse(sku=req.sku, optimal_reorder_qty=0, predicted_daily_burn=0.0)
            
        # Simple Moving Average (SMA) heuristic for burn rate
        avg_burn = sum(req.historical_daily_sales) / len(req.historical_daily_sales)
        
        # Expected stock during lead time
        expected_burn = avg_burn * req.lead_time_days
        
        # We want to reorder enough to cover lead time + safety stock (e.g. 3 days extra)
        safety_stock = avg_burn * 3
        target_stock = expected_burn + safety_stock
        
        reorder_qty = max(0, int(target_stock - req.current_stock))
        
        return DemandPredictionResponse(
            sku=req.sku,
            optimal_reorder_qty=reorder_qty,
            predicted_daily_burn=avg_burn
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
