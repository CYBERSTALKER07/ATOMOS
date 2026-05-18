from __future__ import annotations

from ortools.sat.python import cp_model

from . import optimizer_core_pb2 as pb2
from .mapping import BidirectionalIndexMap

DEFAULT_CPSAT_TIME_LIMIT_MS = 30_000
MAX_CPSAT_TIME_LIMIT_MS = 300_000


def solve_cpsat(request: pb2.CPSATRequest) -> pb2.CPSATResponse:
    response = pb2.CPSATResponse()
    response.meta.CopyFrom(request.meta)

    warnings: list[str] = []

    if len(request.factory_slots) == 0:
        response.feasible = False
        response.warnings.append("no factory slots supplied")
        return response

    if len(request.manifest_requirements) == 0:
        response.feasible = True
        return response

    try:
        factory_map = BidirectionalIndexMap(slot.factory_node_uuid for slot in request.factory_slots)
        manifest_map = BidirectionalIndexMap(req.manifest_id for req in request.manifest_requirements)
    except ValueError as exc:
        response.feasible = False
        response.warnings.append(str(exc))
        return response

    model = cp_model.CpModel()

    factory_capacity = {
        slot.factory_node_uuid: max(0, slot.slot_capacity_scaled)
        for slot in request.factory_slots
    }

    decision_vars: dict[tuple[str, str], cp_model.IntVar] = {}

    for manifest in request.manifest_requirements:
        eligible_factories = list(manifest.eligible_factory_node_uuids)
        if not eligible_factories:
            eligible_factories = factory_map.ordered()

        filtered_factories = [f for f in eligible_factories if f in factory_capacity]
        if not filtered_factories:
            warnings.append(f"manifest has no eligible factory: {manifest.manifest_id}")
            continue

        vars_for_manifest: list[cp_model.IntVar] = []
        for factory_uuid in filtered_factories:
            key = (manifest.manifest_id, factory_uuid)
            decision_vars[key] = model.NewBoolVar(
                f"assign_{manifest_map.index_of(manifest.manifest_id)}_{factory_map.index_of(factory_uuid)}"
            )
            vars_for_manifest.append(decision_vars[key])

        model.Add(sum(vars_for_manifest) <= 1)

    for factory_uuid, capacity in factory_capacity.items():
        terms: list[cp_model.LinearExpr] = []
        for manifest in request.manifest_requirements:
            key = (manifest.manifest_id, factory_uuid)
            if key not in decision_vars:
                continue
            terms.append(manifest.required_capacity_scaled * decision_vars[key])
        if terms:
            model.Add(sum(terms) <= capacity)

    objective_terms: list[cp_model.LinearExpr] = []
    for manifest in request.manifest_requirements:
        for factory_uuid in factory_capacity.keys():
            key = (manifest.manifest_id, factory_uuid)
            if key not in decision_vars:
                continue
            objective_terms.append(manifest.priority_score_scaled * decision_vars[key])

    if objective_terms:
        model.Maximize(sum(objective_terms))

    solver = cp_model.CpSolver()
    time_limit_ms = request.solver_time_limit_ms or DEFAULT_CPSAT_TIME_LIMIT_MS
    time_limit_ms = min(max(1, time_limit_ms), MAX_CPSAT_TIME_LIMIT_MS)
    solver.parameters.max_time_in_seconds = float(time_limit_ms) / 1000.0

    if request.num_search_workers > 0:
        solver.parameters.num_search_workers = request.num_search_workers

    status = solver.Solve(model)

    response.feasible = status in (cp_model.OPTIMAL, cp_model.FEASIBLE)
    response.timed_out = status == cp_model.UNKNOWN

    if response.feasible:
        response.objective_score_scaled = int(solver.ObjectiveValue())

    assigned_manifest_ids: set[str] = set()

    for manifest in request.manifest_requirements:
        manifest_assigned = False
        for factory_uuid in factory_capacity.keys():
            key = (manifest.manifest_id, factory_uuid)
            chosen = False
            if response.feasible and key in decision_vars:
                chosen = solver.Value(decision_vars[key]) == 1

            response.assignments.append(
                pb2.Assignment(
                    manifest_id=manifest.manifest_id,
                    factory_node_uuid=factory_uuid,
                    assigned=chosen,
                )
            )

            if chosen:
                manifest_assigned = True
                assigned_manifest_ids.add(manifest.manifest_id)

        if not manifest_assigned:
            response.unassigned_manifest_ids.append(manifest.manifest_id)

    response.warnings.extend(warnings)
    return response
