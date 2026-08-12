"""Factory helpers for LangChain Deep Agents in this monorepo."""

from __future__ import annotations

import os
from collections.abc import Callable, Sequence
from typing import Any

from deepagents import create_deep_agent
from dotenv import load_dotenv
from langchain_core.language_models.chat_models import BaseChatModel
from langchain_core.tools import BaseTool
from langgraph.graph.state import CompiledStateGraph

from void_deep_agents.findings import FINDING_JSON_HINT, PANEL_NAMES
from void_deep_agents.paths import (
    deep_agents_root,
    default_memory_paths,
    default_skill_paths,
    pegasusx_root,
    surfaces_registry_path,
)
from void_deep_agents.subagents import build_subagents, panel_names

# Load .env from the deep-agents service root first, then cwd/parents.
load_dotenv(deep_agents_root() / ".env")
load_dotenv()

DEFAULT_MODEL_NAME = os.getenv("DEEP_AGENTS_MODEL", "grok-4.5")

DEFAULT_SYSTEM_PROMPT = """You are a PegasusX ecosystem quality assistant (Deep Agents).

Scope: pegasusX monorepo — backend-go, Spanner, Kafka, Redis, WS hubs, role apps,
cloud (k8s/terraform), contracts. Prefer evidence (paths, packages) over prose.

Laws:
- Coverage rule: Spanner mutation → same-txn outbox → consumer → clients.
- No production mocks. Classify Class A/B/C/D. Severity P0–P3.
- Role-row parity. Money as int64 minor units.
- Do not invent wired status without emit + consumer + client evidence.

When auditing, load skills as needed and consult surfaces.yaml + gap register.
"""

ECOSYSTEM_SYSTEM_PROMPT = f"""You are the Chief Orchestrator of the PegasusX
multi-agent ecosystem audit orchestra.

Mission: track whole-ecosystem quality — business logic, feature gaps, role
parity, code quality, architecture, money/fiscal, data-flow, security, cloud —
so features are Class A wired, not islands.

You have specialist subagents (panels). Delegate via the task tool:
{', '.join(PANEL_NAMES)}.

Method:
1. Map blast radius (roles, routes, events, clients, cloud flags).
2. Fan out to relevant panels (all panels when the user asks for a full audit).
3. Merge panel reports into one scorecard. Deduplicate; keep highest severity.
4. Never invent wired status. Never reopen surfaces.yaml resolved_gap_ids
   without regression evidence.
5. Propose minimal change sets that close emit + consumer + clients together.

Output:
1. Markdown scorecard (P0→P3 sections, panel attribution).
2. A JSON findings array at the end (fenced ```json) matching the finding schema.
{FINDING_JSON_HINT}

Desktop stack: keep Next.js + Tauri 2 (do not push Electron rewrites).
Tree: pegasusX is SoT; pegasus is legacy.
"""


def default_model(
    model_name: str | None = None,
    *,
    temperature: float = 0.2,
) -> BaseChatModel:
    """Build the default chat model (xAI / Grok via langchain-xai).

    Requires ``XAI_API_KEY`` in the environment (or a git-ignored ``.env``).
    Override the model with ``DEEP_AGENTS_MODEL`` or the ``model_name`` arg.
    """
    from langchain_xai import ChatXAI

    name = model_name or DEFAULT_MODEL_NAME
    api_key = os.getenv("XAI_API_KEY")
    if not api_key or api_key.strip() in {"", "xai-your-key-here"}:
        raise RuntimeError(
            "XAI_API_KEY is not set (or still the .env.example placeholder). "
            "Export a real key from https://console.x.ai/ or put it in "
            "pegasus/services/deep-agents/.env — then re-run ./scripts/smoke.sh"
        )

    return ChatXAI(
        model=name,
        temperature=temperature,
        api_key=api_key,
    )


def _repo_hint_tools() -> list[Callable[..., Any]]:
    """Small tools so the agent can orient without hallucinating paths."""

    def get_pegasusx_root() -> str:
        """Absolute path to the pegasusX tree (source of truth)."""
        return str(pegasusx_root())

    def get_surfaces_registry() -> str:
        """Return path + short summary of the ecosystem surface registry."""
        p = surfaces_registry_path()
        if not p:
            return "surfaces.yaml not found under pegasusX/.agents/deep-agents/"
        text = p.read_text(encoding="utf-8")
        # Keep tool result bounded
        if len(text) > 8000:
            return f"path={p}\n\n{text[:8000]}\n...[truncated]"
        return f"path={p}\n\n{text}"

    def list_ecosystem_skills() -> str:
        """List available Deep Agent skill directories."""
        return "\n".join(default_skill_paths()) or "(no skills found)"

    def list_audit_panels() -> str:
        """List specialist audit panel names in the orchestra."""
        return "\n".join(panel_names())

    return [
        get_pegasusx_root,
        get_surfaces_registry,
        list_ecosystem_skills,
        list_audit_panels,
    ]


def create_void_deep_agent(
    *,
    model: str | BaseChatModel | None = None,
    tools: Sequence[BaseTool | Callable[..., Any] | dict[str, Any]] | None = None,
    system_prompt: str | None = None,
    skills: list[str] | None = None,
    memory: list[str] | None = None,
    name: str | None = "void-deep-agent",
    include_ecosystem_defaults: bool = True,
    subagents: list[dict[str, Any]] | None = None,
    **kwargs: Any,
) -> CompiledStateGraph:
    """Create a Deep Agent configured for this monorepo.

    Parameters
    ----------
    model:
        Provider string, a ``BaseChatModel``, or ``None`` for xAI Grok.
    tools:
        Extra tools (functions, LangChain tools, or provider tool dicts).
    system_prompt:
        Override default system prompt.
    skills:
        Skill directories; default = all under ``skills/`` when
        ``include_ecosystem_defaults``.
    memory:
        Memory files; default = pegasusX MEMORY.md + AGENTS.md.
    name:
        Agent name for tracing.
    include_ecosystem_defaults:
        When True, attach default skills, memory, and orientation tools.
    subagents:
        Optional Deep Agents subagent specs (forwarded to create_deep_agent).
    **kwargs:
        Forwarded to ``deepagents.create_deep_agent``.
    """
    resolved_model: str | BaseChatModel
    if model is None:
        resolved_model = default_model()
    else:
        resolved_model = model

    resolved_skills = skills
    resolved_memory = memory
    tool_list: list[Any] = list(tools or [])

    if include_ecosystem_defaults:
        if resolved_skills is None:
            resolved_skills = default_skill_paths() or None
        if resolved_memory is None:
            resolved_memory = default_memory_paths() or None
        tool_list = _repo_hint_tools() + tool_list

    create_kwargs: dict[str, Any] = dict(kwargs)
    if subagents is not None:
        create_kwargs["subagents"] = subagents

    return create_deep_agent(
        model=resolved_model,
        tools=tool_list or None,
        system_prompt=system_prompt or DEFAULT_SYSTEM_PROMPT,
        skills=resolved_skills,
        memory=resolved_memory,
        name=name,
        **create_kwargs,
    )


def create_ecosystem_auditor(
    *,
    model: str | BaseChatModel | None = None,
    tools: Sequence[BaseTool | Callable[..., Any] | dict[str, Any]] | None = None,
    panels: list[str] | None = None,
    **kwargs: Any,
) -> CompiledStateGraph:
    """Deep Agent orchestrator with specialist audit panels as subagents.

    Parameters
    ----------
    panels:
        Subset of panel names to attach. ``None`` = all 12 panels.
    """
    return create_void_deep_agent(
        model=model,
        tools=tools,
        system_prompt=ECOSYSTEM_SYSTEM_PROMPT,
        name="pegasusx-ecosystem-auditor",
        include_ecosystem_defaults=True,
        subagents=build_subagents(panels),
        **kwargs,
    )
