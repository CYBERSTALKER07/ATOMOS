"""HTTP adapter for pegasusX optimizer-contract (POST /v1/optimizer/solve)."""

from __future__ import annotations

import json
import os
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from contract_solver import CONTRACT_V, SolverError, solve_contract

AUTH_HEADER = "X-Internal-Api-Key"
SOLVE_PATH = "/v1/optimizer/solve"
DEFAULT_TIMEOUT_SEC = 8.0


class MetricsTracker:
    """Thread-safe in-memory Prometheus metrics accumulator for optimizer-core."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self.requests_total: dict[int, int] = {}
        self.total_duration_sec: float = 0.0
        self.solve_count: int = 0
        self.timeouts_total: int = 0

    def record_request(self, status: int, duration_sec: float) -> None:
        with self._lock:
            self.requests_total[status] = self.requests_total.get(status, 0) + 1
            self.total_duration_sec += duration_sec
            self.solve_count += 1

    def record_timeout(self) -> None:
        with self._lock:
            self.timeouts_total += 1

    def render_prometheus(self) -> str:
        with self._lock:
            lines = [
                "# HELP pegasusx_optimizer_up 1 if the optimizer server process is alive.",
                "# TYPE pegasusx_optimizer_up gauge",
                "pegasusx_optimizer_up 1",
                "# HELP pegasusx_optimizer_requests_total Total number of solve requests by HTTP status.",
                "# TYPE pegasusx_optimizer_requests_total counter",
            ]
            if not self.requests_total:
                lines.append('pegasusx_optimizer_requests_total{status="200"} 0')
            else:
                for status, count in sorted(self.requests_total.items()):
                    lines.append(f'pegasusx_optimizer_requests_total{{status="{status}"}} {count}')

            lines.extend([
                "# HELP pegasusx_optimizer_solve_duration_seconds_total Total duration of solves in seconds.",
                "# TYPE pegasusx_optimizer_solve_duration_seconds_total counter",
                f"pegasusx_optimizer_solve_duration_seconds_total {self.total_duration_sec:.6f}",
                "# HELP pegasusx_optimizer_solve_count_total Total count of solver runs.",
                "# TYPE pegasusx_optimizer_solve_count_total counter",
                f"pegasusx_optimizer_solve_count_total {self.solve_count}",
                "# HELP pegasusx_optimizer_timeouts_total Total number of solver timeouts.",
                "# TYPE pegasusx_optimizer_timeouts_total counter",
                f"pegasusx_optimizer_timeouts_total {self.timeouts_total}",
            ])
            return "\n".join(lines) + "\n"


METRICS = MetricsTracker()


def _env_float(name: str, fallback: float) -> float:
    raw = os.getenv(name)
    if raw is None:
        return fallback
    try:
        return float(raw)
    except ValueError:
        return fallback


def _env_int(name: str, fallback: int) -> int:
    raw = os.getenv(name)
    if raw is None:
        return fallback
    try:
        return int(raw)
    except ValueError:
        return fallback


def _write_json(
    handler: BaseHTTPRequestHandler,
    status: int,
    payload: dict[str, Any],
    headers: dict[str, str] | None = None,
) -> None:
    body = json.dumps(payload).encode("utf-8")
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(body)))
    if headers:
        for k, v in headers.items():
            if v:
                handler.send_header(k, v)
    handler.end_headers()
    handler.wfile.write(body)


class OptimizerHandler(BaseHTTPRequestHandler):
    api_key = ""
    soft_timeout_sec = DEFAULT_TIMEOUT_SEC

    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def do_GET(self) -> None:  # noqa: N802
        if self.path in ("/healthz", "/ready"):
            _write_json(self, 200, {"status": "ok"})
            return
        if self.path == "/metrics":
            body = METRICS.render_prometheus().encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        _write_json(self, 404, {"error": "not_found"})

    def do_POST(self) -> None:  # noqa: N802
        start_time = time.time()
        if self.path != SOLVE_PATH:
            _write_json(self, 404, {"error": "not_found"})
            METRICS.record_request(404, time.time() - start_time)
            return

        trace_id = self.headers.get("X-Trace-Id") or ""
        traceparent = self.headers.get("traceparent") or ""

        resp_headers: dict[str, str] = {}
        if trace_id:
            resp_headers["X-Trace-Id"] = trace_id
        if traceparent:
            resp_headers["traceparent"] = traceparent

        if self.headers.get(AUTH_HEADER) != self.api_key:
            _write_json(
                self,
                401,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "UNAUTHORIZED", "message": "missing or invalid internal api key"},
                headers=resp_headers,
            )
            METRICS.record_request(401, time.time() - start_time)
            return

        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length > 0 else b"{}"
        try:
            req = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            _write_json(
                self,
                400,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "BAD_REQUEST", "message": "invalid JSON body"},
                headers=resp_headers,
            )
            METRICS.record_request(400, time.time() - start_time)
            return

        if not trace_id and req.get("trace_id"):
            trace_id = str(req.get("trace_id"))
            resp_headers["X-Trace-Id"] = trace_id

        if req.get("v") != CONTRACT_V:
            _write_json(
                self,
                400,
                {
                    "v": CONTRACT_V,
                    "trace_id": trace_id,
                    "code": "VERSION_MISMATCH",
                    "message": f"contract version mismatch: server expects {CONTRACT_V}",
                },
                headers=resp_headers,
            )
            METRICS.record_request(400, time.time() - start_time)
            return

        result: dict[str, Any] = {}
        error: SolverError | None = None
        done = threading.Event()

        def run_solver() -> None:
            nonlocal result, error
            try:
                result = solve_contract(req)
            except SolverError as exc:
                error = exc
            except Exception as exc:  # pragma: no cover - defensive
                error = SolverError("INTERNAL", str(exc))
            finally:
                done.set()

        worker = threading.Thread(target=run_solver, daemon=True)
        worker.start()
        if not done.wait(timeout=self.soft_timeout_sec):
            METRICS.record_timeout()
            _write_json(
                self,
                504,
                {
                    "v": CONTRACT_V,
                    "trace_id": trace_id,
                    "code": "TIMEOUT",
                    "message": "solver exceeded timeout budget",
                },
                headers=resp_headers,
            )
            METRICS.record_request(504, time.time() - start_time)
            return

        if error is not None:
            status = 400
            if error.code == "INTERNAL":
                status = 500
            _write_json(
                self,
                status,
                {
                    "v": CONTRACT_V,
                    "trace_id": trace_id,
                    "code": error.code,
                    "message": error.message,
                },
                headers=resp_headers,
            )
            METRICS.record_request(status, time.time() - start_time)
            return

        _write_json(self, 200, result, headers=resp_headers)
        METRICS.record_request(200, time.time() - start_time)


def serve() -> None:
    api_key = os.getenv("INTERNAL_API_KEY", "")
    port = _env_int("OPTIMIZER_HTTP_PORT", 8082)
    timeout_sec = _env_float("OPTIMIZER_SOFT_TIMEOUT_SEC", DEFAULT_TIMEOUT_SEC)
    OptimizerHandler.api_key = api_key
    OptimizerHandler.soft_timeout_sec = timeout_sec
    server = ThreadingHTTPServer(("0.0.0.0", port), OptimizerHandler)
    print(f"optimizer-core http listening on :{port}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    serve()
