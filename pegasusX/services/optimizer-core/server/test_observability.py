"""Unit tests for optimizer-core observability: Prometheus /metrics and W3C trace headers."""

import json
import threading
import time
import unittest
from http.client import HTTPConnection
from http.server import ThreadingHTTPServer

import http_main
from contract_solver import CONTRACT_V


class TestOptimizerObservability(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.api_key = "test-internal-api-key"
        http_main.OptimizerHandler.api_key = cls.api_key
        http_main.OptimizerHandler.soft_timeout_sec = 8.0
        # Start server on dynamic port
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), http_main.OptimizerHandler)
        cls.port = cls.server.server_address[1]
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()

    def test_metrics_endpoint_returns_prometheus_format(self):
        conn = HTTPConnection("127.0.0.1", self.port)
        conn.request("GET", "/metrics")
        resp = conn.getresponse()
        self.assertEqual(resp.status, 200)
        content_type = resp.getheader("Content-Type")
        self.assertIn("text/plain", content_type)
        body = resp.read().decode("utf-8")
        self.assertIn("pegasusx_optimizer_up 1", body)
        self.assertIn("pegasusx_optimizer_requests_total", body)
        self.assertIn("pegasusx_optimizer_solve_duration_seconds_total", body)
        conn.close()

    def test_healthz_and_ready_endpoints(self):
        for path in ("/healthz", "/ready"):
            conn = HTTPConnection("127.0.0.1", self.port)
            conn.request("GET", path)
            resp = conn.getresponse()
            self.assertEqual(resp.status, 200)
            data = json.loads(resp.read().decode("utf-8"))
            self.assertEqual(data.get("status"), "ok")
            conn.close()

    def test_trace_headers_reflected_in_solve_response(self):
        conn = HTTPConnection("127.0.0.1", self.port)
        test_trace_id = "test-trace-abc-123"
        test_traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

        headers = {
            "Content-Type": "application/json",
            "X-Internal-Api-Key": self.api_key,
            "X-Trace-Id": test_trace_id,
            "traceparent": test_traceparent,
        }
        body = json.dumps({
            "v": CONTRACT_V,
            "trace_id": test_trace_id,
            "supplier_id": "sup-1",
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
            "tunables": {
                "time_limit_ms": 500,
            },
        })

        conn.request("POST", "/v1/optimizer/solve", body=body, headers=headers)
        resp = conn.getresponse()
        self.assertEqual(resp.status, 200)

        # Assert trace headers are present on response
        self.assertEqual(resp.getheader("X-Trace-Id"), test_trace_id)
        self.assertEqual(resp.getheader("traceparent"), test_traceparent)

        data = json.loads(resp.read().decode("utf-8"))
        self.assertEqual(data.get("v"), CONTRACT_V)
        self.assertEqual(data.get("trace_id"), test_trace_id)
        conn.close()

    def test_unauthorized_request_still_reflects_trace_headers(self):
        conn = HTTPConnection("127.0.0.1", self.port)
        test_trace_id = "test-unauth-trace"
        test_traceparent = "00-11111111111111111111111111111111-2222222222222222-01"

        headers = {
            "Content-Type": "application/json",
            "X-Internal-Api-Key": "wrong-key",
            "X-Trace-Id": test_trace_id,
            "traceparent": test_traceparent,
        }
        conn.request("POST", "/v1/optimizer/solve", body=b"{}", headers=headers)
        resp = conn.getresponse()
        self.assertEqual(resp.status, 401)
        self.assertEqual(resp.getheader("X-Trace-Id"), test_trace_id)
        self.assertEqual(resp.getheader("traceparent"), test_traceparent)
        conn.close()


if __name__ == "__main__":
    unittest.main()
