"""HTTP adapter for pegasusX optimizer-contract (POST /v1/optimizer/solve)."""

from __future__ import annotations

import hmac
import json
import os
import signal
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from contract_solver import CONTRACT_V, SolverError, solve_contract

AUTH_HEADER = "X-Internal-Api-Key"
SOLVE_PATH = "/v1/optimizer/solve"
DEFAULT_TIMEOUT_SEC = 8.0
DEFAULT_SOCKET_TIMEOUT_SEC = 10.0
DEFAULT_MAX_CONCURRENCY = 4
MAX_BODY_BYTES = 4 * 1024 * 1024  # 4 MiB


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
    timeout = DEFAULT_SOCKET_TIMEOUT_SEC
    solver_semaphore = threading.BoundedSemaphore(DEFAULT_MAX_CONCURRENCY)
    is_shutting_down = False
    in_flight_count = 0
    _in_flight_lock = threading.Lock()
    _in_flight_cond = threading.Condition(_in_flight_lock)

    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/healthz":
            _write_json(self, 200, {"status": "ok"})
            return
        if self.path == "/ready":
            if self.is_shutting_down:
                _write_json(self, 503, {"status": "shutting_down"})
                return
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

        if self.is_shutting_down:
            _write_json(
                self,
                503,
                {
                    "v": CONTRACT_V,
                    "trace_id": trace_id,
                    "code": "SHUTTING_DOWN",
                    "message": "server is shutting down",
                },
                headers=resp_headers,
            )
            METRICS.record_request(503, time.time() - start_time)
            return

        # Constant-time API key check (CG-8)
        provided_key = self.headers.get(AUTH_HEADER) or ""
        if not self.api_key or not provided_key or not hmac.compare_digest(provided_key, self.api_key):
            _write_json(
                self,
                401,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "UNAUTHORIZED", "message": "missing or invalid internal api key"},
                headers=resp_headers,
            )
            METRICS.record_request(401, time.time() - start_time)
            return

        # Content-Length check & Max body size capped at 4 MiB (413 if exceeded)
        content_len_hdr = self.headers.get("Content-Length")
        try:
            length = int(content_len_hdr) if content_len_hdr is not None else 0
        except ValueError:
            _write_json(
                self,
                400,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "BAD_REQUEST", "message": "invalid Content-Length"},
                headers=resp_headers,
            )
            METRICS.record_request(400, time.time() - start_time)
            return

        if length < 0:
            _write_json(
                self,
                400,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "BAD_REQUEST", "message": "negative Content-Length"},
                headers=resp_headers,
            )
            METRICS.record_request(400, time.time() - start_time)
            return

        if length > MAX_BODY_BYTES:
            _write_json(
                self,
                413,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "PAYLOAD_TOO_LARGE", "message": f"payload exceeds maximum limit of {MAX_BODY_BYTES} bytes"},
                headers=resp_headers,
            )
            METRICS.record_request(413, time.time() - start_time)
            return

        raw = self.rfile.read(length) if length > 0 else b"{}"
        if len(raw) > MAX_BODY_BYTES:
            _write_json(
                self,
                413,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "PAYLOAD_TOO_LARGE", "message": f"payload exceeds maximum limit of {MAX_BODY_BYTES} bytes"},
                headers=resp_headers,
            )
            METRICS.record_request(413, time.time() - start_time)
            return

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

        # Time limit clamping: min(requested, soft_timeout_ms) (UA-9)
        tunables = req.get("tunables") if isinstance(req.get("tunables"), dict) else {}
        requested_time_limit_ms = 2000
        if "time_limit_ms" in tunables:
            try:
                t_val = int(tunables["time_limit_ms"])
                if t_val > 0:
                    requested_time_limit_ms = t_val
            except (ValueError, TypeError):
                pass

        soft_timeout_ms = max(1, int(self.soft_timeout_sec * 1000))
        applied_time_limit_ms = min(requested_time_limit_ms, soft_timeout_ms)
        is_clamped = requested_time_limit_ms > applied_time_limit_ms

        if not isinstance(req.get("tunables"), dict):
            req["tunables"] = {}
        req["tunables"]["time_limit_ms"] = applied_time_limit_ms
        req["_time_limit_ms_requested"] = requested_time_limit_ms
        req["_time_limit_ms_applied"] = applied_time_limit_ms

        if is_clamped:
            resp_headers["X-Solver-Time-Clamped"] = "true"

        # Concurrency bound via BoundedSemaphore (SO-11, CG-8)
        if not self.solver_semaphore.acquire(blocking=False):
            _write_json(
                self,
                429,
                {"v": CONTRACT_V, "trace_id": trace_id, "code": "OVERLOADED", "message": "solver concurrency limit reached"},
                headers={**resp_headers, "Retry-After": "1"},
            )
            METRICS.record_request(429, time.time() - start_time)
            return

        result: dict[str, Any] = {}
        error: SolverError | None = None
        done = threading.Event()

        def run_solver() -> None:
            nonlocal result, error
            with OptimizerHandler._in_flight_cond:
                OptimizerHandler.in_flight_count += 1
            try:
                result = solve_contract(req)
            except SolverError as exc:
                error = exc
            except Exception as exc:  # pragma: no cover - defensive
                error = SolverError("INTERNAL", str(exc))
            finally:
                try:
                    self.solver_semaphore.release()
                except ValueError:
                    pass
                with OptimizerHandler._in_flight_cond:
                    OptimizerHandler.in_flight_count -= 1
                    OptimizerHandler._in_flight_cond.notify_all()
                done.set()

        worker_started = False
        try:
            worker = threading.Thread(target=run_solver, daemon=True)
            worker.start()
            worker_started = True
        finally:
            if not worker_started:
                try:
                    self.solver_semaphore.release()
                except ValueError:
                    pass

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

        # Ensure response metadata includes requested and applied time limits (UA-9)
        if "stats" in result and isinstance(result["stats"], dict):
            result["stats"]["solver_time_limit_ms_requested"] = requested_time_limit_ms
            result["stats"]["solver_time_limit_ms_applied"] = applied_time_limit_ms
        result["solver_metadata"] = {
            "solver_time_limit_ms_requested": requested_time_limit_ms,
            "solver_time_limit_ms_applied": applied_time_limit_ms,
            "time_limit_requested_ms": requested_time_limit_ms,
            "time_limit_applied_ms": applied_time_limit_ms,
            "clamped": is_clamped,
        }

        _write_json(self, 200, result, headers=resp_headers)
        METRICS.record_request(200, time.time() - start_time)


def drain_in_flight(timeout: float = 20.0) -> None:
    deadline = time.time() + timeout
    with OptimizerHandler._in_flight_cond:
        while OptimizerHandler.in_flight_count > 0:
            remaining = deadline - time.time()
            if remaining <= 0:
                break
            OptimizerHandler._in_flight_cond.wait(timeout=min(remaining, 0.5))


def serve() -> None:
    api_key = os.getenv("INTERNAL_API_KEY", "")
    port = _env_int("OPTIMIZER_HTTP_PORT", 8082)
    timeout_sec = _env_float("OPTIMIZER_SOFT_TIMEOUT_SEC", DEFAULT_TIMEOUT_SEC)
    max_concurrency = _env_int("OPTIMIZER_MAX_CONCURRENCY", DEFAULT_MAX_CONCURRENCY)
    drain_timeout = _env_float("OPTIMIZER_DRAIN_TIMEOUT_SEC", 20.0)

    OptimizerHandler.api_key = api_key
    OptimizerHandler.soft_timeout_sec = timeout_sec
    OptimizerHandler.timeout = DEFAULT_SOCKET_TIMEOUT_SEC
    OptimizerHandler.solver_semaphore = threading.BoundedSemaphore(max_concurrency)
    OptimizerHandler.is_shutting_down = False

    server = ThreadingHTTPServer(("0.0.0.0", port), OptimizerHandler)
    server.timeout = DEFAULT_SOCKET_TIMEOUT_SEC
    print(f"optimizer-core http listening on :{port}", flush=True)

    shutdown_event = threading.Event()

    def handle_shutdown(signum: int, frame: Any) -> None:
        print(f"optimizer-core received signal {signum}, initiating graceful shutdown...", flush=True)
        OptimizerHandler.is_shutting_down = True
        shutdown_event.set()

    signal.signal(signal.SIGTERM, handle_shutdown)
    signal.signal(signal.SIGINT, handle_shutdown)

    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()

    shutdown_event.wait()

    # Stop accepting new connections
    server.shutdown()
    server.server_close()

    # Drain in-flight solver operations
    drain_in_flight(timeout=drain_timeout)
    print("optimizer-core graceful shutdown completed", flush=True)


if __name__ == "__main__":
    serve()
