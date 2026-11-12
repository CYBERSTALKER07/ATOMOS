import sys
import types
import unittest

try:
    import fastapi
    import pydantic
    _ = (fastapi, pydantic)
except ModuleNotFoundError:
    fastapi_stub = types.ModuleType("fastapi")

    class HTTPException(Exception):
        def __init__(self, status_code: int, detail: str):
            super().__init__(detail)
            self.status_code = status_code
            self.detail = detail

    class FastAPI:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

        def get(self, path, **kwargs):
            def decorator(fn):
                return fn
            return decorator

        def post(self, path, **kwargs):
            def decorator(fn):
                return fn
            return decorator

    fastapi_stub.FastAPI = FastAPI
    fastapi_stub.HTTPException = HTTPException
    sys.modules["fastapi"] = fastapi_stub

    pydantic_stub = types.ModuleType("pydantic")

    class BaseModel:
        def __init__(self, **kwargs):
            for k, v in getattr(self.__class__, "__annotations__", {}).items():
                if hasattr(self.__class__, k):
                    setattr(self, k, getattr(self.__class__, k))
            for k, v in kwargs.items():
                setattr(self, k, v)

    pydantic_stub.BaseModel = BaseModel
    sys.modules["pydantic"] = pydantic_stub

from main import (
    DemandPredictionRequest,
    FleetLocation,
    FleetRouteRequest,
    FleetVehicle,
    HTTPException,
    Location,
    PickPathRequest,
    get_fleet_route,
    get_pick_path,
    predict_demand,
)


class TestDispatchOptimizerSidecar(unittest.TestCase):
    def test_pick_path_returns_heuristic_and_rejects_nan(self):
        req = PickPathRequest(
            warehouse_id="wh-1",
            locations=[
                Location(id="A", x=0.0, y=0.0),
                Location(id="B", x=3.0, y=4.0),
                Location(id="C", x=1.0, y=1.0),
            ],
        )
        res = get_pick_path(req)
        self.assertEqual(res.solver_status, "HEURISTIC")
        self.assertEqual(set(res.ordered_location_ids), {"A", "B", "C"})

        with self.assertRaises(HTTPException) as ctx:
            get_pick_path(
                PickPathRequest(
                    warehouse_id="wh-1",
                    locations=[Location(id="bad", x=float("nan"), y=1.0)],
                )
            )
        self.assertEqual(ctx.exception.status_code, 400)

    def test_fleet_route_enforces_capacity_and_returns_heuristic(self):
        depot = FleetLocation(id="depot", x=41.2995, y=69.2401, demand_vu=0)
        stops = [
            FleetLocation(id="s1", x=41.3111, y=69.2797, demand_vu=40),
            FleetLocation(id="s2", x=41.3250, y=69.2900, demand_vu=50),
        ]
        vehicles = [FleetVehicle(id="v1", capacity_vu=100)]
        res = get_fleet_route(FleetRouteRequest(depot=depot, locations=stops, vehicles=vehicles))
        self.assertEqual(res.solver_status, "HEURISTIC")
        self.assertEqual(len(res.routes), 1)
        self.assertEqual(set(res.routes[0].location_ids), {"s1", "s2"})

        # Stop demand exceeding single vehicle capacity must fail with 422
        with self.assertRaises(HTTPException) as ctx:
            get_fleet_route(
                FleetRouteRequest(
                    depot=depot,
                    locations=[FleetLocation(id="s-huge", x=41.31, y=69.28, demand_vu=250)],
                    vehicles=[FleetVehicle(id="v1", capacity_vu=100)],
                )
            )
        self.assertEqual(ctx.exception.status_code, 422)

    def test_predict_demand_reachable_and_validates_bounds(self):
        res = predict_demand(
            DemandPredictionRequest(
                sku="SKU-1",
                historical_daily_sales=[10, 12, 8, 10],
                current_stock=15,
                lead_time_days=4,
            )
        )
        self.assertEqual(res.sku, "SKU-1")
        self.assertEqual(res.predicted_daily_burn, 10.0)
        self.assertEqual(res.optimal_reorder_qty, 55)

        with self.assertRaises(HTTPException) as ctx:
            predict_demand(
                DemandPredictionRequest(
                    sku="SKU-1",
                    historical_daily_sales=[10, -2],
                    current_stock=5,
                    lead_time_days=2,
                )
            )
        self.assertEqual(ctx.exception.status_code, 400)


if __name__ == "__main__":
    unittest.main()
