# PegasusX — LangChain / Deep Agents ecosystem audit orchestra

## Should we use this?

**Yes — as a development and audit harness**, not as production business AI.

| Use | Yes / No |
|-----|----------|
| Audit business logic, feature gaps, role parity | **Yes** |
| Track doorstep/order **edge cases** per role | **Yes** |
| Track **gov/regulatory** rails (Soliq OFD/EHF, GS1, AS2) | **Yes** |
| Audit per-role **features + app parity** | **Yes** |
| Audit code quality, architecture, data-flow | **Yes** |
| Track Spanner → outbox → Kafka → WS → clients | **Yes** |
| One orchestrator + ~12 specialist panels | **Yes** |
| Deploy tens of agent microservices on GKE | **No** |
| Replace `ai-worker` / OR-Tools / production LLM | **No** |
| Auto-merge PRs without human review | **No** |

Production AI remains `apps/ai-worker`. Deep Agents is a **companion auditor mesh** for humans and coding agents.

**Docs SoT map:** [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) — living vs frozen markdown (no `.docx` in repo).

## Orchestra architecture

One **Chief Orchestrator** (`create_ecosystem_auditor`) + **12 specialist subagents** connected via Deep Agents `task` tool:

| Panel | Tracks |
|-------|--------|
| `data_flow` | Coverage rule Class A/B/C/D |
| `business_logic` | State machines + **edge cases** + per-role business duties |
| `role_parity` | Per-role features vs Android/iOS/portal/desktop/terminal |
| `money_fiscal` | AR, payout, **Soliq OFD/EHF**, PSP reconciler (+ skill `regulatory-gov`) |
| `kafka_outbox` | Topics, run-mode, twin, location bus |
| `redis_cache` | Invalidation, heartbeat, never SoT |
| `code_quality` | Ownership, tests, dead paths |
| `architecture` | Blast radius, anti-islands, stack |
| `security_tenancy` | Tenant, IDOR, JWT denylist |
| `cloud_infra` | Images, overlays, flags, secrets |
| `client_contracts` | types / schema vs apps |
| `gap_register_sync` | Docs vs code; resolved_gap_ids |

`surfaces.yaml` → `orchestra_tracks:` binds **business_edges**, **role_features_parity**, **regulatory_gov** to panels.

Findings schema: `pegasus/services/deep-agents/schemas/finding.schema.json`.

## Layout

| Piece | Path |
|-------|------|
| Runtime (Python) | `pegasus/services/deep-agents/` |
| Memory (always on) | `pegasusX/.agents/deep-agents/MEMORY.md` |
| Ecosystem law | `pegasusX/.agents/AGENTS.md` |
| Surface registry | `pegasusX/.agents/deep-agents/surfaces.yaml` (`audit_panels`, `orchestra_tracks`) |
| Skills | `pegasus/services/deep-agents/skills/*` (incl. `regulatory-gov`) |
| Gap SoT | `docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md` |
| Business edges SoT | `docs/ORDER_FLOW_AND_EDGE_CASES.md` |
| Role features SoT | `docs/FEATURES_BY_APP_ROLE.md`, `docs/ROLE_ROW_PARITY_MATRIX.md` |

## Setup

```bash
cd pegasus/services/deep-agents
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt && pip install -e .
cp .env.example .env   # set XAI_API_KEY from https://console.x.ai/
```

## Smoke

```bash
./scripts/smoke.sh
void-ecosystem-audit --dry-run   # lists memory, skills, 12 panels + fs_probe_ok
```

Filesystem: audits use `CompositeBackend` (pegasusX at `/`, skills at `/skills/`)
with virtual paths like `/apps/backend-go/...`. Prefer narrow greps; do not glob `/`.
Do not remove the backend wiring in `factory.py` / `paths.py`.

## When to run

1. **Before coding a gap** — `--panel` for the blast radius (e.g. `money_fiscal,role_parity`).
2. **Business + edges + gov + parity** — `--panel business_logic,role_parity,money_fiscal`.
3. **Full scorecard** — `--full --json-out /tmp/audit.json` after a P0/P1 batch or weekly.
4. **Docs vs code** — `--panel gap_register_sync`.
5. Never as a substitute for landing Class A wiring yourself.

## Run commands

```bash
source .venv/bin/activate

# Business edges + Soliq/gov + per-role features/parity
void-ecosystem-audit --panel business_logic,role_parity,money_fiscal \
  "Edge cases, Soliq OFD/EHF, GS1/AS2, and role app parity"

# Subset
void-ecosystem-audit --panel business_logic,security_tenancy \
  "Invalid order transitions and open IDORs"

# Full orchestra (scorecard must include Business/edges · Regulatory · Role parity)
void-ecosystem-audit --full --json-out /tmp/pegasusx-audit.json

# Via void-deep-agent wrapper
void-deep-agent --ecosystem --panel role_parity "Retailer AR/HQ mobile gaps"
```

Programmatic:

```python
from void_deep_agents import create_ecosystem_auditor

agent = create_ecosystem_auditor(panels=["data_flow", "architecture"])
result = agent.invoke({
    "messages": [{"role": "user", "content": "Factory→payload Class A gaps"}]
})
print(result["messages"][-1].content)
```

## Anti-patterns

- Do not deploy one GKE service per panel.
- Do not point agents at production secrets.
- Do not “fix” gaps without code evidence.
- Do not reopen `resolved_gap_ids` without regression proof.
- Do not grow skills into a second outdated wiki — link the gap register.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
