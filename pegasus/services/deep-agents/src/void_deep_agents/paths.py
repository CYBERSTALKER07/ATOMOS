"""Resolve monorepo roots for PegasusX Deep Agents."""

from __future__ import annotations

import os
from pathlib import Path

from deepagents.backends import CompositeBackend, FilesystemBackend, StateBackend
from deepagents.backends.protocol import BackendProtocol
from deepagents.middleware.filesystem import FilesystemPermission


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


def to_virtual_path(path: str | Path) -> str:
    """Map a host path to the composite virtual FS.

    - Under ``pegasusX/`` → ``/apps/...``, ``/docs/...``, ``/.agents/...``
    - Under ``deep-agents/skills/`` → ``/skills/...``
    """
    p = Path(path).expanduser().resolve()
    skills = (deep_agents_root() / "skills").resolve()
    px = pegasusx_root()
    try:
        rel = p.relative_to(skills)
        return "/skills/" + rel.as_posix()
    except ValueError:
        pass
    try:
        rel = p.relative_to(px)
        return "/" + rel.as_posix()
    except ValueError as exc:
        raise ValueError(
            f"Path {p} is outside pegasusX ({px}) and skills ({skills})"
        ) from exc


def default_memory_paths(*, virtual: bool = False) -> list[str]:
    """AGENTS.md-style memory files always loaded by Deep Agents."""
    px = pegasusx_root()
    candidates = [
        px / ".agents" / "deep-agents" / "MEMORY.md",
        px / ".agents" / "AGENTS.md",
    ]
    out = [p for p in candidates if p.is_file()]
    if virtual:
        return [to_virtual_path(p) for p in out]
    return [str(p) for p in out]


def default_skill_paths(*, virtual: bool = False) -> list[str]:
    """Skill directories containing SKILL.md."""
    skills_root = deep_agents_root() / "skills"
    if not skills_root.is_dir():
        return []
    out: list[Path] = []
    for child in sorted(skills_root.iterdir()):
        if child.is_dir() and (child / "SKILL.md").is_file():
            out.append(child)
    if virtual:
        return [to_virtual_path(p) for p in out]
    return [str(p) for p in out]


def surfaces_registry_path() -> Path | None:
    p = pegasusx_root() / ".agents" / "deep-agents" / "surfaces.yaml"
    return p if p.is_file() else None


def gap_register_glob() -> Path:
    return pegasusx_root() / "docs" / "session-2026-08-07"


def default_filesystem_backend() -> CompositeBackend:
    """Composite host FS: pegasusX at ``/`` + skills at ``/skills/``.

    Deep Agents defaults to in-memory ``StateBackend``. Rooting the whole
    VOID tree caused 5s glob timeouts; pegasusX-only default keeps reads fast.
    """
    code = FilesystemBackend(root_dir=pegasusx_root(), virtual_mode=True)
    skills = FilesystemBackend(
        root_dir=deep_agents_root() / "skills",
        virtual_mode=True,
    )
    return CompositeBackend(
        default=code,
        routes={"/skills/": skills},
    )


def _secret_read_denies() -> list[FilesystemPermission]:
    return [
        FilesystemPermission(
            operations=["read"],
            paths=[
                "/**/.env",
                "/**/.env.*",
                "/**/credentials.json",
                "/**/service-account*.json",
            ],
            mode="deny",
        ),
    ]


def audit_filesystem_permissions() -> list[FilesystemPermission]:
    """Read-mostly permissions: block writes + secret-looking paths."""
    return [
        FilesystemPermission(
            operations=["write"],
            paths=["/**"],
            mode="deny",
        ),
        *_secret_read_denies(),
    ]


def fleet_filesystem_backend() -> CompositeBackend:
    """Code + skills + thread-local writable ``/fleet/`` for confirmation reports.

    Implementation edits still go through host FS under pegasusX; ``/fleet/``
    stays in ``StateBackend`` so report loops do not litter the tree unless
    the operator opts into a host fleet dir via ``FLEET_HOST_DIR``.
    """
    code = FilesystemBackend(root_dir=pegasusx_root(), virtual_mode=True)
    skills = FilesystemBackend(
        root_dir=deep_agents_root() / "skills",
        virtual_mode=True,
    )
    fleet_host = os.getenv("FLEET_HOST_DIR", "").strip()
    fleet: BackendProtocol
    if fleet_host:
        host = Path(fleet_host).expanduser().resolve()
        host.mkdir(parents=True, exist_ok=True)
        fleet = FilesystemBackend(root_dir=host, virtual_mode=True)
    else:
        fleet = StateBackend()
    return CompositeBackend(
        default=code,
        routes={
            "/skills/": skills,
            "/fleet/": fleet,
        },
    )


def fleet_filesystem_permissions(*, allow_code_writes: bool = True) -> list[FilesystemPermission]:
    """Fleet permissions: always allow ``/fleet/**`` writes; optionally code writes."""
    rules: list[FilesystemPermission] = [
        FilesystemPermission(
            operations=["read", "write"],
            paths=["/fleet/**"],
            mode="allow",
        ),
        *_secret_read_denies(),
    ]
    if not allow_code_writes:
        rules.append(
            FilesystemPermission(
                operations=["write"],
                paths=["/**"],
                mode="deny",
            )
        )
    return rules


def probe_filesystem_backend(backend: BackendProtocol | None = None) -> dict[str, str]:
    """Smoke-check that pegasusX code + skills are readable via the agent backend."""
    be = backend or default_filesystem_backend()
    checks: dict[str, str] = {
        "pegasusx_root": str(pegasusx_root()),
        "skills_route": "/skills/",
    }
    samples = [
        "/apps/backend-go/order/state_machine.go",
        "/docs/ORDER_FLOW_AND_EDGE_CASES.md",
        "/.agents/deep-agents/surfaces.yaml",
        "/skills/business-logic/SKILL.md",
    ]
    for path in samples:
        result = be.read(path, offset=0, limit=1)
        err = getattr(result, "error", None)
        checks[path] = "ok" if err is None else f"ERR:{err}"
    return checks
