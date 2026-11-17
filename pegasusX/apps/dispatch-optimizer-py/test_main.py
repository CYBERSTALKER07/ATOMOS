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

    def test_predict_demand_fractional_burn_ceil(self):
        # historical_daily_sales=[1, 2] -> avg 1.5, lead_time=2 -> expected 3.0, safety=4.5 -> target 7.5.
        # current_stock=3 -> target - stock = 4.5.
        # ceil(4.5) == 5, whereas int(4.5) == 4.
        res = predict_demand(
            DemandPredictionRequest(
                sku="SKU-FRAC",
                historical_daily_sales=[1, 2],
                current_stock=3,
                lead_time_days=2,
            )
        )
        self.assertEqual(res.sku, "SKU-FRAC")
        self.assertEqual(res.predicted_daily_burn, 1.5)
        self.assertEqual(res.optimal_reorder_qty, 5)

    def test_get_fleet_route_multi_wave_capacity_and_ffd(self):
        depot = FleetLocation(id="depot", x=41.2995, y=69.2401, demand_vu=0)
        # 1 vehicle with capacity 100 serving 3 stops of 60 VU each.
        # Total demand is 180 > 100. Discrete packing requires 3 waves (60 cannot share with 60).
        stops = [
            FleetLocation(id="s1", x=41.3111, y=69.2797, demand_vu=60),
            FleetLocation(id="s2", x=41.3250, y=69.2900, demand_vu=60),
            FleetLocation(id="s3", x=41.3300, y=69.2600, demand_vu=60),
        ]
        vehicles = [FleetVehicle(id="v1", capacity_vu=100)]
        res = get_fleet_route(FleetRouteRequest(depot=depot, locations=stops, vehicles=vehicles))
        self.assertEqual(res.solver_status, "HEURISTIC")
        # Must return 3 waves since no two 60-VU stops fit in a single 100-VU capacity vehicle
        self.assertEqual(len(res.routes), 3)

        seen_stops = []
        wave_indices = []
        stop_demand_map = {s.id: s.demand_vu for s in stops}
        for route in res.routes:
            self.assertEqual(route.vehicle_id, "v1")
            wave_demand = sum(stop_demand_map[lid] for lid in route.location_ids)
            self.assertLessEqual(wave_demand, 100)
            seen_stops.extend(route.location_ids)
            wave_indices.append(route.wave_index)

        self.assertEqual(set(seen_stops), {"s1", "s2", "s3"})
        self.assertEqual(sorted(wave_indices), [1, 2, 3])

    def test_predict_demand_with_pipeline_stock_and_variance(self):
        # Without pipeline stock: burn=10.0, lead=4 -> expected=40, safety=30 -> target=70, stock=15 -> reorder=55
        # With pipeline_stock=20: effective_stock=35 -> reorder=70-35 = 35
        res_pipeline = predict_demand(
            DemandPredictionRequest(
                sku="SKU-PIPE",
                historical_daily_sales=[10, 10, 10, 10],
                current_stock=15,
                lead_time_days=4,
                pipeline_stock=20,
            )
        )
        self.assertEqual(res_pipeline.optimal_reorder_qty, 35)

        # With demand_variance=4.0 and service_level=0.95:
        # burn=10.0, expected=40, z=1.645, variance=4.0, lead=4
        # safety_stock = 1.645 * sqrt(4 * 4) = 6.58 -> target = 46.58
        # stock=15 -> reorder = ceil(46.58 - 15) = 32
        res_variance = predict_demand(
            DemandPredictionRequest(
                sku="SKU-VAR",
                historical_daily_sales=[10, 10, 10, 10],
                current_stock=15,
                lead_time_days=4,
                demand_variance=4.0,
                service_level=0.95,
            )
        )
        self.assertEqual(res_variance.optimal_reorder_qty, 32)

    def test_verify_internal_api_key_fail_closed_in_prod(self):
        import os
        from main import compute_ffd_waves, verify_internal_api_key

        old_env = {k: os.environ.get(k) for k in ("PEGASUSX_ENV", "INTERNAL_API_KEY", "OPTIMIZER_INTERNAL_API_KEY")}
        try:
            os.environ["PEGASUSX_ENV"] = "production"
            os.environ["INTERNAL_API_KEY"] = "dev-internal-key"
            os.environ.pop("OPTIMIZER_INTERNAL_API_KEY", None)
            self.assertEqual(verify_internal_api_key("dev-internal-key"), (503, "internal_api_key_unconfigured"))

            os.environ["INTERNAL_API_KEY"] = "prod-secret-token-xyz"
            self.assertEqual(verify_internal_api_key("wrong"), (401, "invalid_internal_api_key"))
            self.assertIsNone(verify_internal_api_key("prod-secret-token-xyz"))
        finally:
            for k, v in old_env.items():
                if v is None:
                    os.environ.pop(k, None)
                else:
                    os.environ[k] = v

        # Oversized infeasible stop should not append phantom wave.
        self.assertEqual(compute_ffd_waves([500], [100]), 1)


if __name__ == "__main__":
    unittest.main()
