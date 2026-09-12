"""HTTP adapter for pegasusX optimizer-contract (POST /v1/optimizer/solve)."""

from __future__ import annotations

import json
import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any

from contract_solver import CONTRACT_V, SolverError, solve_contract

AUTH_HEADER = "X-Internal-Api-Key"
SOLVE_PATH = "/v1/optimizer/solve"
DEFAULT_TIMEOUT_SEC = 8.0


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


def _write_json(handler: BaseHTTPRequestHandler, status: int, payload: dict[str, Any]) -> None:
    body = json.dumps(payload).encode("utf-8")
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(body)))
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
        _write_json(self, 404, {"error": "not_found"})

    def do_POST(self) -> None:  # noqa: N802
        if self.path != SOLVE_PATH:
            _write_json(self, 404, {"error": "not_found"})
            return
        if self.headers.get(AUTH_HEADER) != self.api_key:
            _write_json(
                self,
                401,
                {"v": CONTRACT_V, "trace_id": "", "code": "UNAUTHORIZED", "message": "missing or invalid internal api key"},
            )
            return
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length > 0 else b"{}"
        try:
            req = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            _write_json(
                self,
                400,
                {"v": CONTRACT_V, "trace_id": "", "code": "BAD_REQUEST", "message": "invalid JSON body"},
            )
            return
        if req.get("v") != CONTRACT_V:
            _write_json(
                self,
                400,
                {
                    "v": CONTRACT_V,
                    "trace_id": str(req.get("trace_id") or ""),
                    "code": "VERSION_MISMATCH",
                    "message": f"contract version mismatch: server expects {CONTRACT_V}",
                },
            )
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
            _write_json(
                self,
                504,
                {
                    "v": CONTRACT_V,
                    "trace_id": str(req.get("trace_id") or ""),
                    "code": "TIMEOUT",
                    "message": "solver exceeded timeout budget",
                },
            )
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
                    "trace_id": str(req.get("trace_id") or ""),
                    "code": error.code,
                    "message": error.message,
                },
            )
            return
        _write_json(self, 200, result)


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
