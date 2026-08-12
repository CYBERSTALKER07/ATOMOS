"""Resolve monorepo roots for PegasusX Deep Agents."""

from __future__ import annotations

import os
from pathlib import Path


def deep_agents_root() -> Path:
    """Package root: .../pegasus/services/deep-agents."""
    return Path(__file__).resolve().parents[2]


def void_root() -> Path:
    """V.O.I.D monorepo root (parent of pegasus/ and pegasusX/)."""
    env = os.getenv("VOID_ROOT")
    if env:
        return Path(env).expanduser().resolve()
    # deep-agents -> services -> pegasus -> V.O.I.D
    return deep_agents_root().parents[2]


def pegasusx_root() -> Path:
    env = os.getenv("PEGASUSX_ROOT")
    if env:
        return Path(env).expanduser().resolve()
    return void_root() / "pegasusX"


def default_memory_paths() -> list[str]:
    """AGENTS.md-style memory files always loaded by Deep Agents."""
    px = pegasusx_root()
    candidates = [
        px / ".agents" / "deep-agents" / "MEMORY.md",
        px / ".agents" / "AGENTS.md",
    ]
    return [str(p) for p in candidates if p.is_file()]


def default_skill_paths() -> list[str]:
    """Skill directories containing SKILL.md."""
    skills_root = deep_agents_root() / "skills"
    if not skills_root.is_dir():
        return []
    out: list[str] = []
    for child in sorted(skills_root.iterdir()):
        if child.is_dir() and (child / "SKILL.md").is_file():
            out.append(str(child))
    return out


def surfaces_registry_path() -> Path | None:
    p = pegasusx_root() / ".agents" / "deep-agents" / "surfaces.yaml"
    return p if p.is_file() else None


def gap_register_glob() -> Path:
    return pegasusx_root() / "docs" / "session-2026-08-07"
