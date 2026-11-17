#!/usr/bin/env python3
"""Validate the pegasusX launch-readiness evidence bundle."""

from __future__ import annotations

import json
import re
import sys
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    print(f"launch-readiness-error: {message}", file=sys.stderr)
    raise SystemExit(1)


def read_text(relative_path: str) -> str:
    path = ROOT / relative_path
    if not path.is_file():
        fail(f"missing required file: {relative_path}")
    return path.read_text(encoding="utf-8")


def read_json(relative_path: str) -> dict:
    try:
        return json.loads(read_text(relative_path))
    except json.JSONDecodeError as error:
        fail(f"invalid JSON in {relative_path}: {error}")


def require_contains(relative_path: str, needles: list[str]) -> None:
    text = read_text(relative_path)
    for needle in needles:
        if needle not in text:
            fail(f"{relative_path} must contain {needle!r}")


def require_anchor_status(plan_text: str, anchor: str, expected_status: str) -> None:
    pattern = rf"`{re.escape(anchor)}`[^\n]*\n\s+Status: `([^`]+)`"
    match = re.search(pattern, plan_text)
    if not match:
        fail(f"context/plan.md missing anchor status for {anchor}")
    status = match.group(1)
    if status != expected_status:
        fail(f"{anchor} must be {expected_status}, got {status}")


def require_docs() -> None:
    required_docs = [
        "agents.md",
        "docs/arch-overview.md",
        "docs/svc-backend.md",
        "docs/app-fleet-features.md",
        "docs/infra-deployment.md",
        "docs/canonical-order-state-machine.md",
    ]
    for relative_path in required_docs:
        read_text(relative_path)

    require_contains(
        "docs/infra-deployment.md",
        [
            "make sandbox-infra-up",
            "infra/terraform/main.tf",
            "infra/k8s/base/",
            "scripts/money_path_gate.sh",
            "scripts/parity/role_row_contract_check_full.sh",
        ],
    )
    require_contains(
        "docs/canonical-order-state-machine.md",
        [
            "PENDING",
            "COMPLETED",
            "DELIVERED_ON_CREDIT",
        ],
    )


def require_release_gates() -> None:
    read_text("scripts/smoke_sandbox.sh")
    read_text("scripts/smoke_ssmr.sh")
    read_text("scripts/validate_ai_worker_k8s.sh")
    read_text("scripts/validate_launch_readiness.py")
    require_contains(
        "Makefile",
        [
            "validate-ai-worker-k8s:",
            "validate-launch-readiness:",
            "python3 scripts/validate_launch_readiness.py",
        ],
    )
    package = read_json("package.json")
    scripts = package.get("scripts", {})
    if scripts.get("infra:sandbox:test") != "bash scripts/smoke_sandbox.sh":
        fail("package.json scripts.infra:sandbox:test must run the sandbox smoke harness")
    if scripts.get("infra:ssmr:test") != "bash scripts/smoke_ssmr.sh":
        fail("package.json scripts.infra:ssmr:test must run the SSMR smoke harness alias")
    if scripts.get("infra:k8s:validate") != "bash scripts/validate_ai_worker_k8s.sh":
        fail("package.json scripts.infra:k8s:validate must run the ai-worker manifest validator")
    if scripts.get("infra:launch:validate") != "python3 scripts/validate_launch_readiness.py":
        fail("package.json scripts.infra:launch:validate must run the launch readiness validator")


def require_platform_evidence() -> None:
    required_files = [
        "infra/k8s/ai-worker/configmap.yaml",
        "infra/k8s/ai-worker/deployment.yaml",
        "infra/k8s/ai-worker/service.yaml",
        "infra/k8s/backend-go/deployment.yaml",
        "infra/k8s/backend-go/configmap.yaml",
        "infra/k8s/backend-go/service.yaml",
        "infra/k8s/backend-go/pdb.yaml",
        "infra/k8s/backend-go/hpa.yaml",
        "infra/k8s/backend-go-worker/deployment.yaml",
        "infra/k8s/backend-go-worker/service.yaml",
        "infra/k8s/ingress/ingress.yaml",
        "infra/k8s/ingress/backendconfig.yaml",
        "infra/k8s/base/kustomization.yaml",
        "infra/k8s/overlays/prod/kustomization.yaml",
        "infra/k8s/osrm/deployment.yaml",
        "infra/k8s/osrm/service.yaml",
        "infra/k8s/namespace.yaml",
        "infra/k8s/serviceaccount.yaml",
        "scripts/p0_launch_preflight.sh",
        "scripts/validate_production_profile.sh",
        "scripts/wire_ready.sh",
        "scripts/p1_pilot_weekly.sh",
        "apps/backend-go/Dockerfile",
        "apps/ai-worker/Dockerfile",
        "scripts/docker_build.sh",
        "scripts/render_k8s_manifests.sh",
        "scripts/validate_backend_k8s.sh",
        "infra/terraform/main.tf",
        "infra/terraform/variables.tf",
        "infra/terraform/modules/compute/main.tf",
        "infra/terraform/modules/monitoring/main.tf",
        "infra/terraform/legacy/observability.tf",
        "infra/terraform/legacy/observability_pilot.tf",
        "infra/docker-compose.sandbox.yml",
        "infra/docker-compose.ssmr.yml",
        "apps/backend-go/platform/handlers.go",
        "scripts/parity/role_row_contract_check.sh",
        "scripts/parity/role_row_contract_check_full.sh",
        "scripts/parity/gap_hunter_gate.sh",
    ]
    for relative_path in required_files:
        read_text(relative_path)

    require_contains(
        "infra/terraform/modules/monitoring/main.tf",
        [
            "google_monitoring_alert_policy",
            "spanner.googleapis.com/instance/cpu/utilization",
        ],
    )
    require_contains(
        "infra/terraform/legacy/observability.tf",
        [
            "void_ai_worker_up",
            "void_ai_worker_ready",
            "void_kafka_consumer_lag_seconds",
            "google_monitoring_dashboard",
        ],
    )
    require_contains(
        "infra/terraform/legacy/observability_pilot.tf",
        [
            "void_ws_connections",
            "void_http_requests_total",
            "void_outbox_unpublished_count",
            "spanner.googleapis.com/instance/cpu/utilization",
            "pegasusX — Pilot Launch (P1)",
        ],
    )
    require_contains(
        "infra/k8s/backend-go/configmap.yaml",
        [
            'HTTP_PORT: "8080"',
            "ROUTING_OSRM_URL",
            "GLOBAL_PAY_ENV",
            "KAFKA_TOPIC_FREEZE_LOCKS",
            "OPTIMIZER_BASE_URL",
            "optimizer-core",
            "PEGASUSX_RUN_MODE",
            "REQUIRE_INFRA_ADAPTERS",
        ],
    )
    require_contains("infra/docker-compose.ssmr.yml", ["8181", "ai-worker"])


def require_context_sync() -> None:
    inventory = read_json("context/technology-inventory.json")
    backend = inventory.get("backend", {})
    if "launchReadinessGate" not in backend:
        fail("context/technology-inventory.json missing backend.launchReadinessGate")
    if "launchReadinessRunbook" not in backend:
        fail("context/technology-inventory.json missing backend.launchReadinessRunbook")

    graph = read_json("context/architecture-graph.json")
    notes = graph.get("runtimeNotes", [])
    if not any("launch readiness guard" in note.get("summary", "").lower() for note in notes):
        fail("context/architecture-graph.json missing launch readiness runtime note")


def run_make_target(target: str) -> None:
    print(f"Running make {target}...", file=sys.stderr)
    result = subprocess.run(["make", target], cwd=ROOT)
    if result.returncode != 0:
        fail(f"make {target} failed")

def main() -> None:
    require_docs()
    require_release_gates()
    require_platform_evidence()
    require_context_sync()

    anti_theatre = subprocess.run(["bash", "scripts/ci_anti_theatre_gate.sh"], cwd=ROOT)
    if anti_theatre.returncode != 0:
        fail("scripts/ci_anti_theatre_gate.sh failed")

    run_make_target("parity-contract-full")
    run_make_target("validate-ai-worker-k8s")
    run_make_target("validate-backend-k8s")

    print("launch-readiness-ok")


if __name__ == "__main__":
    main()