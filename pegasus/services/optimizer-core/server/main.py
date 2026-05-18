from __future__ import annotations

import logging
import os
from concurrent import futures

import grpc

from . import optimizer_core_pb2_grpc as pb2_grpc
from .service import OptimizerCoreService


def _env_int(name: str, fallback: int) -> int:
    raw = os.getenv(name)
    if raw is None:
        return fallback
    try:
        return int(raw)
    except ValueError:
        return fallback


def serve() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )

    port = str(_env_int("OPTIMIZER_CORE_GRPC_PORT", 50055))
    max_workers = _env_int("OPTIMIZER_CORE_MAX_WORKERS", 8)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=max_workers))
    pb2_grpc.add_OptimizerCoreServiceServicer_to_server(OptimizerCoreService(), server)
    server.add_insecure_port(f"[::]:{port}")

    logging.info("optimizer-core gRPC server starting on :%s", port)
    server.start()

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logging.info("optimizer-core shutdown requested")
        server.stop(grace=5)


if __name__ == "__main__":
    serve()
