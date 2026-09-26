"""E2E codebase fleet — agents live in-repo and loop until unanimous CONFIRM.

Doctrine (enterprise delivery):
- Every specialist touches real paths under the virtual FS.
- Each writes ``/fleet/reports/{agent}.md`` with evidence.
- Supervisor does not stop until all roster agents are CONFIRMED
  (business + technical gates both required).
"""

from __future__ import annotations

from typing import Any, Literal

FleetAgentName = Literal[
    "architect",
    "implementer",
    "wiring",
    "business",
    "technical",
    "verifier",
]

FLEET_NAMES: tuple[FleetAgentName, ...] = (
    "architect",
    "implementer",
    "wiring",
    "business",
    "technical",
    "verifier",
)

CONFIRM_STATUSES = ("CONFIRMED", "IN_PROGRESS", "BLOCKED", "REJECTED")

_FS = """
Filesystem:
- Code/docs: `/apps/...`, `/docs/...`, `/.agents/...` (pegasusX SoT)
- Skills: `/skills/<name>/SKILL.md`
- Fleet reports (writable): `/fleet/reports/{your-name}.md`, `/fleet/STATUS.md`, `/fleet/plan.md`
Use narrow `read_file` / `grep` paths. Never invent file contents.
Never CONFIRM without citing real paths + commands/evidence.
""".strip()

_REPORT_CONTRACT = """
Write/overwrite `/fleet/reports/{your_panel_name}.md` with YAML frontmatter:
  agent, status (CONFIRMED|IN_PROGRESS|BLOCKED|REJECTED), round, scope
Then: Verdict, Codebase depth (paths), E2E path, Evidence, Gaps, Sign-off.
Missing evidence ⇒ BLOCKED or REJECTED, never CONFIRMED.
""".strip()

SUPERVISOR_PROMPT = f"""You are the E2E fleet supervisor for PegasusX delivery.

Mission: feature is IMPLEMENTED, WIRED END-TO-END, and enterprise prod-ready
on BOTH business and technical gates. Agents live deep in the codebase.

{_FS}

Roster (delegate via task tool): {', '.join(FLEET_NAMES)}.

Loop (mandatory):
1. Seed `/fleet/` if needed; dispatch architect → plan + `/fleet/plan.md`.
2. implementer + wiring (parallel after plan).
3. business + technical (parallel once code exists).
4. verifier last — must try to falsify peers.
5. Aggregate `/fleet/STATUS.md` (table agent→status→evidence pointer).
6. If ANY status ≠ CONFIRMED, re-dispatch ONLY those agents with feedback.
7. Max 3 full rounds unless the user raises the limit.
8. Final answer: FLEET_GATE: PASS or FAIL with STATUS pointer.

Never PASS on partial confirms. Never let verifier CONFIRM while peers are
IN_PROGRESS/BLOCKED/REJECTED. Do not invent wired status.
Tree SoT: pegasusX/; pegasus/ is legacy. Desktop: Next.js + Tauri 2.
""".strip()

_COMMON = f"""
You are a specialist in the PegasusX E2E delivery fleet.
Tree SoT: pegasusX/. Prefer path/package evidence over prose.
{_FS}
{_REPORT_CONTRACT}
Laws: coverage rule (Spanner→outbox→consumer→clients); money int64 minor units;
no production mocks; Class A wiring required for prod paths.
""".strip()

_FLEET_SPECS: dict[FleetAgentName, tuple[str, str, list[str]]] = {
    "architect": (
        "Map E2E surfaces/contracts; write /fleet/plan.md; live in the repo tree.",
        f"""{_COMMON}
Panel name for reports: architect.
Map blast radius: roles, routes, events, clients, cloud flags, gov rails.
Write `/fleet/plan.md` with surface matrix + E2E sequence + test plan.
CONFIRMED only when plan names real packages/paths that exist or must be created.
Skills: architecture, data-flow-coverage, void-overview.
""",
        ["architecture", "data-flow-coverage", "void-overview"],
    ),
    "implementer": (
        "Implement deep in owning packages; no drive-by root files; evidence in report.",
        f"""{_COMMON}
Panel name for reports: implementer.
Land code in canonical owner packages under `/apps/**`. Follow existing patterns.
Keep diffs focused. Prefer SSMR (Spanner + mutation + outbox + tests) for mutations.
CONFIRMED only when your scope is coded with path evidence (wiring may finish mounts).
Skills: backend-mutations, business-logic, code-quality.
""",
        ["backend-mutations", "business-logic", "code-quality"],
    ),
    "wiring": (
        "Prove and fix end-to-end wiring; producer↔consumer evidence required.",
        f"""{_COMMON}
Panel name for reports: wiring.
Grep/open composition roots, routes, outbox emits, consumers, WS, clients, env samples.
Fix missing mounts/consumers/client calls. Cite reproducible evidence commands.
CONFIRMED only with producer→consumer→client chain proof.
Skills: kafka-outbox, data-flow-coverage, redis-cache, role-row-clients.
""",
        ["kafka-outbox", "data-flow-coverage", "redis-cache", "role-row-clients"],
    ),
    "business": (
        "Enterprise business acceptance gate; personas, edges, audit, anti-demo.",
        f"""{_COMMON}
Panel name for reports: business.
Check personas/roles/authz, happy path + critical edges, supportability, audit,
no demo-only critical path. Map acceptance criteria to evidence.
CONFIRMED only if business gate clears with evidence.
Skills: business-logic, role-row-clients, money-fiscal, regulatory-gov.
""",
        ["business-logic", "role-row-clients", "money-fiscal", "regulatory-gov"],
    ),
    "technical": (
        "Enterprise technical prod gate; authz, observability, migrations, failure modes.",
        f"""{_COMMON}
Panel name for reports: technical.
Require authz, validation/idempotency where needed, observability (logs/metrics/traces),
migrations/rollback, retries/timeouts, tests or explicit waiver+owner.
CONFIRMED only if technical gate clears with evidence.
Skills: security-tenancy, cloud-infra, code-quality, kafka-outbox.
""",
        ["security-tenancy", "cloud-infra", "code-quality", "kafka-outbox"],
    ),
    "verifier": (
        "Falsify incomplete wiring; reject weak confirms; final unanimous gate.",
        f"""{_COMMON}
Panel name for reports: verifier.
Adversarially re-read peers' reports + code. Hunt unwired events, schema drift,
missing auth, missing consumers, empty tests, demo-only paths.
You may document REJECTED peers and demand re-dispatch.
CONFIRMED only after falsification attempt finds no P0/P1 gaps AND all other
roster agents are already CONFIRMED.
Skills: data-flow-coverage, architecture, security-tenancy.
""",
        ["data-flow-coverage", "architecture", "security-tenancy"],
    ),
}


def fleet_names() -> list[str]:
    return list(FLEET_NAMES)


def build_fleet_subagents(agents: list[str] | None = None) -> list[dict[str, Any]]:
    """Return Deep Agents SubAgent dicts for the E2E fleet roster."""
    wanted: list[FleetAgentName]
    if agents is None:
        wanted = list(FLEET_NAMES)
    else:
        unknown = [a for a in agents if a not in _FLEET_SPECS]
        if unknown:
            raise ValueError(
                f"Unknown fleet agents: {unknown}. Valid: {', '.join(FLEET_NAMES)}"
            )
        wanted = [a for a in FLEET_NAMES if a in agents]

    out: list[dict[str, Any]] = []
    for name in wanted:
        description, system_prompt, skill_dirs = _FLEET_SPECS[name]
        out.append(
            {
                "name": name,
                "description": description,
                "system_prompt": system_prompt.strip(),
                "skills": [f"/skills/{s}/" for s in skill_dirs],
            }
        )
    return out
