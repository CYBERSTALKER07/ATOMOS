from __future__ import annotations

import logging

import grpc

from . import optimizer_core_pb2 as pb2
from . import optimizer_core_pb2_grpc as pb2_grpc
from .cpsat_adapter import solve_cpsat
from .vrp_adapter import solve_vrp


class OptimizerCoreService(pb2_grpc.OptimizerCoreServiceServicer):
    def CalculateRoute(self, request: pb2.VRPRequest, context: grpc.ServicerContext) -> pb2.VRPResponse:
        try:
            return solve_vrp(request)
        except Exception as exc:  # pragma: no cover - defensive boundary
            logging.exception("CalculateRoute failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            response = pb2.VRPResponse(feasible=False, timed_out=False)
            response.meta.CopyFrom(request.meta)
            response.warnings.append(f"internal error: {exc}")
            return response

    def ResolveConstraint(
        self,
        request: pb2.CPSATRequest,
        context: grpc.ServicerContext,
    ) -> pb2.CPSATResponse:
        try:
            return solve_cpsat(request)
        except Exception as exc:  # pragma: no cover - defensive boundary
            logging.exception("ResolveConstraint failed")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(exc))
            response = pb2.CPSATResponse(feasible=False, timed_out=False)
            response.meta.CopyFrom(request.meta)
            response.warnings.append(f"internal error: {exc}")
            return response

    def HealthCheck(
        self,
        request: pb2.HealthCheckRequest,
        context: grpc.ServicerContext,
    ) -> pb2.HealthCheckResponse:
        del request, context
        return pb2.HealthCheckResponse(ok=True, service="optimizer-core")
