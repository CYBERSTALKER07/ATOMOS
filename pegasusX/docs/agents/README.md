# PegasusX — LangChain / Deep Agents ecosystem audit orchestra

## Should we use this?

**Yes — as a development and audit harness**, not as production business AI.

| Use | Yes / No |
|-----|----------|
| Audit business logic, feature gaps, role parity | **Yes** |
| Audit code quality, architecture, data-flow | **Yes** |
| Track Spanner → outbox → Kafka → WS → clients | **Yes** |
| One orchestrator + ~12 specialist panels | **Yes** |
| Deploy tens of agent microservices on GKE | **No** |
| Replace `ai-worker` / OR-Tools / production LLM | **No** |
| Auto-merge PRs without human review | **No** |

Production AI remains `apps/ai-worker`. Deep Agents is a **companion auditor mesh** for humans and coding agents.

## Orchestra architecture

One **Chief Orchestrator** (`create_ecosystem_auditor`) + **12 specialist subagents** connected via Deep Agents `task` tool:

| Panel | Tracks |
|-------|--------|
| `data_flow` | Coverage rule Class A/B/C/D |
| `business_logic` | Order/credit/fiscal state machines |
| `role_parity` | Android/iOS/portal/desktop/terminal row |
| `money_fiscal` | AR, payout, OFD, EHF, reconciler |
| `kafka_outbox` | Topics, run-mode, twin, location bus |
| `redis_cache` | Invalidation, heartbeat, never SoT |
| `code_quality` | Ownership, tests, dead paths |
| `architecture` | Blast radius, anti-islands, stack |
| `security_tenancy` | Tenant, IDOR, JWT denylist |
| `cloud_infra` | Images, overlays, flags, secrets |
| `client_contracts` | types / schema vs apps |
| `gap_register_sync` | Docs vs code; resolved_gap_ids |

Findings schema: `pegasus/services/deep-agents/schemas/finding.schema.json`.

## Layout

| Piece | Path |
|-------|------|
| Runtime (Python) | `pegasus/services/deep-agents/` |
| Memory (always on) | `pegasusX/.agents/deep-agents/MEMORY.md` |
| Ecosystem law | `pegasusX/.agents/AGENTS.md` |
| Surface registry | `pegasusX/.agents/deep-agents/surfaces.yaml` (`audit_panels`) |
| Skills | `pegasus/services/deep-agents/skills/*` |
| Gap SoT | `docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md` |

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
void-ecosystem-audit --dry-run   # lists memory, 12+ skills, 12 panels
```

## When to run

1. **Before coding a gap** — `--panel` for the blast radius (e.g. `money_fiscal,role_parity`).
2. **Full scorecard** — `--full --json-out /tmp/audit.json` after a P0/P1 batch or weekly.
3. **Docs vs code** — `--panel gap_register_sync`.
4. Never as a substitute for landing Class A wiring yourself.

## Run commands

```bash
source .venv/bin/activate

# Subset
void-ecosystem-audit --panel business_logic,security_tenancy \
  "Invalid order transitions and open IDORs"

# Full orchestra
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
