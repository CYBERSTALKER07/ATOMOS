import sys
import types
import unittest

try:
    import flask
    _ = flask
except ModuleNotFoundError:
    flask_stub = types.ModuleType("flask")

    class _DummyRequest:
        _json = None

        def get_json(self, silent=False):
            return self._json

    class _DummyResponse:
        def __init__(self, payload, status_code=200):
            self._payload = payload
            self.status_code = status_code

        def get_json(self):
            return self._payload

    class _DummyClient:
        def __init__(self, app_instance):
            self._app = app_instance

        def post(self, path, json=None):
            flask_stub.request._json = json
            handler = self._app._routes[path]
            res = handler()
            if isinstance(res, tuple):
                return _DummyResponse(res[0], res[1])
            return _DummyResponse(res, 200)

    class _DummyFlask:
        def __init__(self, name):
            self.name = name
            self.config = {}
            self._routes = {}

        def route(self, path, methods=None):
            def decorator(fn):
                self._routes[path] = fn
                return fn
            return decorator

        def test_client(self):
            return _DummyClient(self)

    flask_stub.Flask = _DummyFlask
    flask_stub.request = _DummyRequest()
    flask_stub.jsonify = lambda obj: obj
    sys.modules["flask"] = flask_stub

from main import app, _evaluate_credit_risk


class TestMLCreditScorer(unittest.TestCase):
    def test_volume_cap_enforces_30_percent_historical_volume_limit(self):
        # current_limit_uzs = 100_000_000 -> 25% raw_increase = 25_000_000
        # total_volume_millions = 10.0 -> 30% volume_cap_uzs = 3_000_000
        # suggested_increase_uzs must be capped at 3_000_000, NOT 25_000_000!
        body, status = _evaluate_credit_risk(
            {
                "retailer_id": "ret-1",
                "avg_days_to_pay": 4.0,
                "missed_payments": 0,
                "total_volume_millions": 10.0,
                "current_limit_uzs": 100_000_000,
                "days_past_due_max": 0,
                "active_disputes": 0,
                "completed_orders": 25,
            }
        )
        self.assertEqual(status, 200)
        self.assertTrue(body["eligible_for_increase"])
        self.assertEqual(body["risk_tier"], "LOW")
        self.assertEqual(body["suggested_increase_uzs"], 3_000_000)
        self.assertEqual(body["recommended_limit_uzs"], 103_000_000)

    def test_rejects_nan_inf_and_negative_features(self):
        for bad_payload in [
            {"retailer_id": "ret-1", "avg_days_to_pay": float("nan")},
            {"retailer_id": "ret-1", "total_volume_millions": float("inf")},
            {"retailer_id": "ret-1", "days_past_due_max": -1},
            {"retailer_id": "ret-1", "active_disputes": -2},
            {"retailer_id": "ret-1", "completed_orders": -5},
        ]:
            body, status = _evaluate_credit_risk(bad_payload)
            self.assertEqual(status, 400, f"expected 400 for {bad_payload}, got {status}: {body}")

    def test_http_endpoint_scores_and_validates(self):
        client = app.test_client()
        resp = client.post(
            "/credit/score",
            json={
                "retailer_id": "ret-high-risk",
                "avg_days_to_pay": 45.0,
                "missed_payments": 4,
                "total_volume_millions": 1.0,
                "current_limit_uzs": 5_000_000,
                "days_past_due_max": 30,
                "active_disputes": 2,
                "completed_orders": 3,
            },
        )
        self.assertEqual(resp.status_code, 200)
        data = resp.get_json()
        self.assertFalse(data["eligible_for_increase"])
        self.assertEqual(data["suggested_increase_uzs"], 0)
        self.assertEqual(data["risk_tier"], "CRITICAL")


if __name__ == "__main__":
    unittest.main()
