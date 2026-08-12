# PegasusX — LangChain / Deep Agents for ecosystem quality

## Should we use this?

**Yes — as a development and audit harness**, not as production business AI.

| Use | Yes / No |
|-----|----------|
| Audit wiring: Spanner → outbox → Kafka → WS → clients | **Yes** |
| Track role-row parity, gap register, Class A/B/C/D | **Yes** |
| Improve code/architecture quality checklists | **Yes** |
| Replace `ai-worker` / OR-Tools / production LLM mapping | **No** |
| Auto-merge PRs without human review | **No** |

Production AI remains `apps/ai-worker` (Kafka consumers). Deep Agents help **humans and coding agents** enforce ecosystem law across backend, apps, Redis, Kafka, Spanner, and cloud.

## Layout

| Piece | Path |
|-------|------|
| Runtime (Python) | `pegasus/services/deep-agents/` |
| Memory (always on) | `pegasusX/.agents/deep-agents/MEMORY.md` |
| Ecosystem law | `pegasusX/.agents/AGENTS.md` |
| Surface registry | `pegasusX/.agents/deep-agents/surfaces.yaml` |
| Skills (progressive) | `pegasus/services/deep-agents/skills/*` |
| Gap / alignment SoT | `docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md`, `MASTER_ALIGNMENT_DATAFLOW_*.md` |

## Setup

```bash
cd pegasus/services/deep-agents   # from monorepo root V.O.I.D or adjust path
# if cwd is pegasusX: cd ../pegasus/services/deep-agents
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
pip install -e .
cp .env.example .env              # set XAI_API_KEY (https://console.x.ai/)
```

Optional env:

```bash
export PEGASUSX_ROOT=/Users/shakhzod/Desktop/V.O.I.D/pegasusX
export VOID_ROOT=/Users/shakhzod/Desktop/V.O.I.D
export XAI_API_KEY=...
export DEEP_AGENTS_MODEL=grok-4.5
```

## Smoke (paths + imports; live if key set)

```bash
./scripts/smoke.sh
# or
void-ecosystem-audit --dry-run
```

## When to run Deep Agents

Use the auditor **before and after** closing a gap cluster, not as a substitute for coding:

1. **Before coding a gap** — map blast radius (roles, emit, consumer, clients, cloud flags) against `surfaces.yaml` + gap register Part 4 sequence.
2. **After a P0/P1 batch** — confirm MEMORY/`resolved_gap_ids` and register ✅ rows agree; catch regressions that re-open Class A wiring.
3. **When docs disagree with code** — force evidence (`path:line`) instead of stale wiki claims.

Gap register Part 4 order (data-flow first): P0 money/tenant → P1-9/10 run-mode + location bus → cross-role client loops → security → certification.

## Run ecosystem auditor

```bash
source .venv/bin/activate
void-ecosystem-audit "List open P1 gaps from ECOSYSTEM_GAP_REGISTER and classify by surface"
# or
void-deep-agent --ecosystem "Verify twin consumer start path and TopicWebhooks usage"
```

Programmatic:

```python
from void_deep_agents import create_ecosystem_auditor

agent = create_ecosystem_auditor()
result = agent.invoke({
    "messages": [{"role": "user", "content": "Audit factory→payload Class A gaps"}]
})
print(result["messages"][-1].content)
```

## What the agent loads

1. **Memory:** `MEMORY.md` + `AGENTS.md` (coverage rule, role rows, DoD, recently resolved IDs)
2. **Skills (on demand):** data-flow, backend mutations, kafka/outbox, redis, cloud, clients, money
3. **Filesystem:** repo tree (read tools) — agent can open `surfaces.yaml`, gap register, source

## Quality workflow (recommended)

1. Pick a feature or gap ID (e.g. P1-18 factory→payload).
2. Run auditor: map blast radius → emit? consumer? clients? cloud flags?
3. Classify Class A/B/C/D.
4. Land code + contracts + clients + tests in one batch.
5. Update gap register status + `surfaces.yaml` `resolved_gap_ids` in the same session.

## Companion skills (Grok / Cursor, not Python)

Human coding agents also use monorepo skills: `void-guardian`, `kafka-event-contracts`, `cache-redis-correctness`, `gap-hunter`, `pegasus-doctrine` / efficient-code. Deep Agents skills **mirror the same laws** so LangChain runs stay consistent with IDE agents.

## Anti-patterns

- Do not point Deep Agents at production secrets.
- Do not let the agent “fix” a gap without code evidence.
- Do not grow skills into a second outdated wiki — link gap register + code paths.
- Do not use this to invent RFQ/marketplace features not in the plan.
- Do not re-report `resolved_gap_ids` / register ✅ rows as open without re-verification.
