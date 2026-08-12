"""Specialist subagent specs for the PegasusX ecosystem audit orchestra."""

from __future__ import annotations

from typing import Any

from void_deep_agents.findings import FINDING_JSON_HINT, PANEL_NAMES, PanelName

# Shared laws injected into every specialist.
_COMMON = f"""
You are a specialist panel in the PegasusX multi-agent audit orchestra.
Tree SoT: pegasusX/ (pegasus/ is legacy). Prefer path/package evidence.
{FINDING_JSON_HINT}
Money: int64 minor units only. Desktop stack: Next.js + Tauri 2 (no Electron push).
Consult pegasusX/.agents/deep-agents/MEMORY.md, surfaces.yaml, and
docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md.
""".strip()

# name -> (description, system_prompt, skill_hint)
_PANEL_SPECS: dict[PanelName, tuple[str, str, str]] = {
    "data_flow": (
        "Audit Spanner→outbox→Kafka→WS/push/webhook→client coverage; Class A/B/C/D.",
        f"""{_COMMON}
Panel: data_flow. Skill: data-flow-coverage.
Enforce coverage rule: every Spanner mutation emits in-txn outbox; every event
has a consumer; no cross-role loop ends at an API with no client.
Classify Class A/B/C/D. Cite outbox.EmitJSON / SpannerTxnBuffer / consumers.
""",
        "data-flow-coverage",
    ),
    "business_logic": (
        "Audit order/credit/payment/fiscal state machines, ADR-009, invalid transitions.",
        f"""{_COMMON}
Panel: business_logic. Skill: business-logic.
Check order Status transitions (state_machine), ADR-009 fiscal gate,
credit leave / cash collect / AR pay-down, buyer EHF vs COMPLETED,
invalid happy-path-only cancel paths. Flag business-rule bugs as P0/P1.
""",
        "business-logic",
    ),
    "role_parity": (
        "Audit role-row client parity across Android/iOS/portal/desktop/terminal.",
        f"""{_COMMON}
Panel: role_parity. Skill: role-row-clients.
A feature for a role must land on all clients in that row unless deferred
in the gap register. Cite shells, navigation, FEATURES_BY_APP_ROLE.
Known holes: retailer AR/HQ mobile; supplier planning web; factory↔payload.
""",
        "role-row-clients",
    ),
    "money_fiscal": (
        "Audit AR, payout, OFD, buyer EHF, webhook reconciler, live-rail fail-closed.",
        f"""{_COMMON}
Panel: money_fiscal. Skill: money-fiscal.
Verify AR/payout outbox, CollectCash pay-down, WebhookReconciler running,
BuyerAcceptance PENDING on MySoliq, credit-note default ON, ErrNoLiveRail.
Do not claim Soliq EDS live without proof (P1-7).
""",
        "money-fiscal",
    ),
    "kafka_outbox": (
        "Audit topics, run-mode parity, twin, DLQ, throttled driver location bus.",
        f"""{_COMMON}
Panel: kafka_outbox. Skill: kafka-outbox.
Check relay on worker tier, worker heartbeat + api safety net (P1-9),
throttled DRIVER_LOCATION_UPDATED on TopicRealtime (P1-10), twin start (P2-11),
TopicWebhooks unused?, DLQ/dedup patterns.
""",
        "kafka-outbox",
    ),
    "redis_cache": (
        "Audit Redis cache invalidation, WS Pub/Sub, worker heartbeat; never SoT.",
        f"""{_COMMON}
Panel: redis_cache. Skill: redis-cache.
Spanner is SoT. Check post-mutation invalidation, cross-pod WS relay,
pegasusx:runtime:worker:heartbeat TTL/interval. Flag money cached without proof.
""",
        "redis-cache",
    ),
    "code_quality": (
        "Audit Go ownership, tests, dead paths, duplicate writers, package hygiene.",
        f"""{_COMMON}
Panel: code_quality. Skill: code-quality.
Canonical owner packages only; focused *_test.go; no silent dead consumers;
no duplicate write paths; bootstrap composition root discipline.
""",
        "code-quality",
    ),
    "architecture": (
        "Audit layering, blast-radius checklist, anti-islands, stack decisions.",
        f"""{_COMMON}
Panel: architecture. Skill: architecture.
Blast radius: roles, routes, events, clients, cloud flags in one change set.
Ban Class C UI islands for prod. Keep Tauri+Next; no Electron rewrite advocacy.
""",
        "architecture",
    ),
    "security_tenancy": (
        "Audit tenant scope, IDOR endpoints, PLATFORM_ADMIN, JWT denylist gaps.",
        f"""{_COMMON}
Panel: security_tenancy. Skill: security-tenancy.
RequireTenant + PreferTenantSupplierID; detail IDORs from gap register;
PLATFORM_ADMIN exempt is intentional (P0-3); JWT session revocation still open (P1-11).
""",
        "security-tenancy",
    ),
    "cloud_infra": (
        "Audit k8s images, overlays, autonomy flags, secrets, RUN_MODE deploy pair.",
        f"""{_COMMON}
Panel: cloud_infra. Skill: cloud-infra.
backend-go api vs backend-go-worker; optimizer prod image not backend digest;
flags for soak/forecast; ESO/GSM secrets; no placeholder images in prod overlays.
""",
        "cloud-infra",
    ),
    "client_contracts": (
        "Audit packages/types and events.schema vs app/SDK usage drift.",
        f"""{_COMMON}
Panel: client_contracts.
When events/API change: contracts/events.schema.json + packages/types +
api-client + native Generated stubs. Flag clients on stale shapes.
""",
        "role-row-clients",
    ),
    "gap_register_sync": (
        "Audit docs vs code; never reopen resolved_gap_ids without regression proof.",
        f"""{_COMMON}
Panel: gap_register_sync.
Compare ECOSYSTEM_GAP_REGISTER ✅ rows and surfaces.yaml resolved_gap_ids
to live code. Flag stale docs claiming stubs that are fixed. Do not invent
new gap IDs; propose register updates with evidence.
""",
        "data-flow-coverage",
    ),
}


def panel_names() -> list[str]:
    return list(PANEL_NAMES)


def build_subagents(panels: list[str] | None = None) -> list[dict[str, Any]]:
    """Return Deep Agents SubAgent dicts for the requested panels (default: all)."""
    wanted: list[PanelName]
    if panels is None:
        wanted = list(PANEL_NAMES)
    else:
        unknown = [p for p in panels if p not in _PANEL_SPECS]
        if unknown:
            raise ValueError(
                f"Unknown panels: {unknown}. Valid: {', '.join(PANEL_NAMES)}"
            )
        wanted = [p for p in PANEL_NAMES if p in panels]  # stable order

    out: list[dict[str, Any]] = []
    for name in wanted:
        description, system_prompt, skill = _PANEL_SPECS[name]
        out.append(
            {
                "name": name,
                "description": description,
                "system_prompt": system_prompt.strip()
                + f"\nLoad skill directory named `{skill}` when available.",
            }
        )
    return out
