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
        "docs/SUPPLIER_ONBOARDING_SOP.md",
        "docs/BILLING_RECOVERY_SCRIPT.md",
        "docs/TOPOLOGY_ENTRY_SUPPORT_GUIDE.md",
        "docs/RETAILER_ONBOARDING_SUPPORT_FLOWS.md",
        "docs/PRICING_AUTHORITY_RULES.md",
        "docs/ZONE_MISS_COMMUNICATION_POLICY.md",
        "docs/PAYMENT_EXCEPTION_SOP.md",
        "docs/FINANCE_SUPPORT_WORKFLOW.md",
        "docs/DISPUTE_CLASSIFICATION_VOCABULARY.md",
        "docs/WAREHOUSE_EXCEPTION_SOP.md",
        "docs/REASSIGNMENT_SUPPORT_PLAYBOOK.md",
        "docs/TRANSFER_CANCELLATION_RUNBOOK.md",
        "docs/DRIVER_SUPPORT_PLAYBOOK.md",
        "docs/LIVE_TRACKING_EXPECTATIONS.md",
        "docs/DELIVERY_ESCALATION_POLICY.md",
        "docs/AI_WORKER_LAUNCH_RUNBOOK.md",
        "docs/LAUNCH_READINESS_RUNBOOK.md",
        "docs/P0_LAUNCH_CHECKLIST.md",
        "docs/DEPLOYMENT_AND_DISTRIBUTION_PLAN.md",
        "docs/INCIDENT_RESPONSE_RUNBOOK.md",
        "docs/RELEASE_TRAIN.md",
        "docs/DEPLOYMENT_READINESS_GAP_LEDGER.md",
    ]
    for relative_path in required_docs:
        read_text(relative_path)

    require_contains(
        "docs/LAUNCH_READINESS_RUNBOOK.md",
        [
            "make test-ssmr-infra",
            "make validate-ai-worker-k8s",
            "make validate-launch-readiness",
            "rollback",
            "launch support",
        ],
    )


def require_release_gates() -> None:
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
    if scripts.get("infra:ssmr:test") != "bash scripts/smoke_ssmr.sh":
        fail("package.json scripts.infra:ssmr:test must run the SSMR smoke harness")
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
        "scripts/p1_pilot_weekly.sh",
        "docs/P0_LAUNCH_CHECKLIST.md",
        "docs/P1_PILOT_CHECKLIST.md",
        "docs/SPANNER_HOT_PATH_REVIEW.md",
        "docs/P2_SCALE_ROADMAP.md",
        "apps/backend-go/Dockerfile",
        "apps/ai-worker/Dockerfile",
        "scripts/docker_build.sh",
        "scripts/render_k8s_manifests.sh",
        "scripts/validate_backend_k8s.sh",
        "infra/terraform/observability.tf",
        "infra/terraform/observability_pilot.tf",
        "infra/terraform/main.tf",
        "infra/terraform/gke.tf",
        "infra/terraform/variables.tf",
        "infra/docker-compose.ssmr.yml",
        "apps/backend-go/platform/handlers.go",
        "scripts/parity/role_row_contract_check.sh",
        "scripts/parity/role_row_contract_check_full.sh",
        "scripts/parity/gap_hunter_gate.sh",
    ]
    for relative_path in required_files:
        read_text(relative_path)

    require_contains(
        "infra/terraform/observability.tf",
        [
            "void_ai_worker_up",
            "void_ai_worker_ready",
            "void_kafka_consumer_lag_seconds",
            "google_monitoring_dashboard",
        ],
    )
    require_contains(
        "infra/terraform/observability_pilot.tf",
        [
            "void_ws_connections",
            "void_http_requests_total",
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
            "PEGASUSX_RUN_MODE",
            "REQUIRE_INFRA_ADAPTERS",
        ],
    )
    require_contains(
        "docs/CLOUD_CREDENTIALS_CHECKLIST.md",
        [
            "GLOBAL_PAY_USERNAME",
            "GLOBAL_PAY_PASSWORD",
            "Maps SDK for Android",
        ],
    )
    require_contains("infra/docker-compose.ssmr.yml", ["8181", "ai-worker"])


def require_context_sync() -> None:
    plan_text = read_text("context/plan.md")
    require_anchor_status(plan_text, "PX0-A5", "implemented")
    require_anchor_status(plan_text, "PX7-A3", "implemented")
    if "`PX11-C1`" not in plan_text:
        fail("context/plan.md missing PX-11 section")
    if "`PX12-A1`" not in plan_text:
        fail("context/plan.md missing PX-12 section")
    require_contains(
        "context/plan.md",
        [
            "scripts/validate_launch_readiness.py",
            "docs/LAUNCH_READINESS_RUNBOOK.md",
        ],
    )

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
    
    run_make_target("parity-contract-full")
    run_make_target("validate-ai-worker-k8s")
    run_make_target("validate-backend-k8s")

    print("launch-readiness-ok")


if __name__ == "__main__":
    main()