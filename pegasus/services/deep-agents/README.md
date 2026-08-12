# Deep Agents (LangChain) — PegasusX ecosystem quality

LangChain + [Deep Agents](https://docs.langchain.com/oss/python/deepagents/overview) for the **V.O.I.D / PegasusX** monorepo.

**Primary use:** ecosystem quality — code, architecture, wiring across backend, apps, Spanner, Redis, Kafka, WS, cloud.  
**Not for:** replacing production `ai-worker` or auto-shipping without review.

Deep Agents harness: planning, filesystem tools, subagents, skills, long context.  
Default LLM: **xAI Grok** via `langchain-xai` (`XAI_API_KEY`).

PegasusX instructions: `pegasusX/docs/agents/README.md` · memory: `pegasusX/.agents/deep-agents/`.

## Location

```
pegasus/services/deep-agents/
├── .env.example
├── requirements.txt
├── pyproject.toml
├── README.md
├── src/void_deep_agents/   # factory, CLI, ecosystem auditor
├── examples/
└── skills/                 # progressive SKILL.md (data-flow, kafka, redis, …)

pegasusX/.agents/deep-agents/
├── MEMORY.md               # always-on ecosystem law
└── surfaces.yaml           # surface registry to track
```

## Setup

```bash
cd pegasus/services/deep-agents
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
pip install -e .            # optional: editable install for CLI entrypoint

cp .env.example .env
# put XAI_API_KEY in .env  (never commit .env)
```

Get a key at [console.x.ai](https://console.x.ai/).

## Quickstart — Deep Agent

```bash
source .venv/bin/activate
export XAI_API_KEY=...   # or use .env
python examples/hello_deep_agent.py
```

Or programmatically:

```python
from void_deep_agents import create_void_deep_agent

def get_weather(city: str) -> str:
    """Get weather for a city (demo tool)."""
    return f"It's always sunny in {city}!"

agent = create_void_deep_agent(tools=[get_weather])
result = agent.invoke(
    {"messages": [{"role": "user", "content": "Weather in SF?"}]}
)
print(result["messages"][-1].content)
```

## Quickstart — plain LangChain agent

```bash
python examples/langchain_agent.py
```

Uses `langchain.agents.create_agent` without the Deep Agents harness (no built-in filesystem / subagents).

## CLI

After `pip install -e .`:

```bash
# Smoke (dry-run + imports; live LLM if XAI_API_KEY is set in .env)
./scripts/smoke.sh

# Smoke / general
void-deep-agent "Summarize what Deep Agents provides"

# Ecosystem auditor (loads pegasusX MEMORY + AGENTS + all skills)
void-deep-agent --dry-run --ecosystem
void-ecosystem-audit "Verify twin consumer start path and TopicWebhooks usage"
void-ecosystem-audit "Role-row parity gaps for retailer AR/HQ"
```

Programmatic auditor:

```python
from void_deep_agents import create_ecosystem_auditor

agent = create_ecosystem_auditor()
result = agent.invoke({
    "messages": [{"role": "user", "content": "Audit factory→payload Class A gaps"}]
})
print(result["messages"][-1].content)
```

## Skills (ecosystem)

| Skill | Tracks |
|-------|--------|
| `data-flow-coverage` | Coverage rule, Class A/B/C/D |
| `backend-mutations` | Spanner + outbox + tests + SSMR |
| `kafka-outbox` | Relay, topics, consumers, run modes |
| `redis-cache` | Invalidation, Pub/Sub, heartbeats |
| `role-row-clients` | Android/iOS/portal/desktop parity |
| `cloud-infra` | K8s, TF, images, secrets |
| `money-fiscal` | AR, payout, OFD, money safety |
| `void-overview` | Monorepo map |

Defaults auto-attach when `include_ecosystem_defaults=True` (default).

## Models

| Mode | How |
|------|-----|
| Default (xAI) | `create_void_deep_agent()` → `ChatXAI(model=DEEP_AGENTS_MODEL or "grok-4.5")` |
| String model | `create_void_deep_agent(model="openai:gpt-5.5")` (needs matching `*_API_KEY`) |
| Custom instance | `create_void_deep_agent(model=ChatXAI(model="grok-4"))` |

## Tracing (optional)

```bash
export LANGSMITH_TRACING=true
export LANGSMITH_API_KEY=...
```

## Docs

- Deep Agents overview: https://docs.langchain.com/oss/python/deepagents/overview
- Quickstart: https://docs.langchain.com/oss/python/deepagents/quickstart
- ChatXAI: https://docs.langchain.com/oss/python/integrations/chat/xai
- LangChain agents: https://docs.langchain.com/oss/python/langchain/agents

## Versions installed (when scaffolded)

See `requirements.txt` (pinned via `pip freeze`). Key packages:

- `deepagents`
- `langchain`
- `langchain-xai`
- `langchain-openai`
- `langgraph`
- `python-dotenv`
