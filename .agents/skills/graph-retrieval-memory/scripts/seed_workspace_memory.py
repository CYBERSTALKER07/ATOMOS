#!/usr/bin/env python3
"""Seed Grok workspace MEMORY.md once. Does not overwrite curated text.

Grok creates ~/.grok/memory/<slug>-<hash8>/ on first memory-enabled session.
This script finds those dirs (or the global file) and injects the VOID seed
only if our marker is missing.
"""

from __future__ import annotations

from pathlib import Path

MARKER = "<!-- VOID-GRAPH-MEMORY-SEED -->"
REPO_MEMORY = Path(__file__).resolve().parents[3] / "memory" / "WORKSPACE.md"

WORKSPACE_SEED = """# Workspace Memory (V.O.I.D. / pegasusX)

<!-- VOID-GRAPH-MEMORY-SEED -->

## Project Context

- Living product tree: `pegasusX/`. `pegasus/` is legacy port source, not SoT.
- Tenant key: `SupplierId` only. Market pack + home cell are attributes, not a second RLS key.
- Dual planes: `FactoryTruckManifests` vs `SupplierTruckManifests` — do not merge.
- Factory planning / auto-order **place**: flags default off. Do not flip globally.
- Money: integer minor units. Fiscal hard-gate (ADR-009). Pay-at-delivery (ADR-001).

## Verified 2026-08-15 (agent RAG configure)

- Grok `[memory] enabled = true` in `~/.grok/config.toml`. Before this day `~/.grok/memory/` did not exist.
- Unique ecosystem product features: **250** `BF-*` IDs in `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md` (re-count rows if the file changes).
- MarketPack is advertised (`GET /v1/auth/session`, `GET /v1/platform/market-packs`) with `checkout_reads_this: false`. Checkout/fiscal/proximity do not read the pack until GS-M.
- `POST /v1/platform/tenants/register` is **not mounted**. Companies cannot self-register.
- Architecture graph: `pegasusX/context/architecture-graph.json` — 88 nodes, 160 edges, `generatedAt: null`. Use as routing index only.
- Retrieval loop skill: `.agents/skills/graph-retrieval-memory/`. Walker: `scripts/graph_retrieve.py`.

## Do not persist

- Docs labeled Wired / matrices / prior chat as “done”
- Planned EU/US/KZ packs as live markets
- Cloud-ready / Layer B without live-path proof this session
"""

GLOBAL_HINT = MARKER  # global file is authored separately


def patch(path: Path, body: str) -> str:
    if not path.exists():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(body)
        return f"created {path}"
    text = path.read_text()
    if MARKER in text:
        return f"skip (seed present) {path}"
    # Template-only files from Grok: prepend seed after title if empty-ish
    if len(text.strip()) < 80 or "Add project-specific knowledge here" in text:
        path.write_text(body)
        return f"replaced template {path}"
    path.write_text(text.rstrip() + "\n\n" + body)
    return f"appended {path}"


def main() -> int:
    if REPO_MEMORY.is_file() and MARKER not in REPO_MEMORY.read_text():
        print(patch(REPO_MEMORY, WORKSPACE_SEED))
    elif REPO_MEMORY.is_file():
        print(f"ok {REPO_MEMORY}")
    else:
        print(patch(REPO_MEMORY, WORKSPACE_SEED))

    root = Path.home() / ".grok" / "memory"
    if not root.is_dir():
        print("grok memory dir missing; [memory] off or Grok not started — repo WORKSPACE.md still applies")
        return 0
    print(f"ok {root / 'MEMORY.md'}")
    for child in sorted(root.iterdir()):
        if not child.is_dir():
            continue
        name = child.name.lower()
        if any(k in name for k in ("atomos", "void", "pegasus", "cyberstalker")):
            print(patch(child / "MEMORY.md", WORKSPACE_SEED))
            continue
        # Newly created unknown workspace: only seed if it looks like this repo
        mem = child / "MEMORY.md"
        if mem.exists() and "pegasusX" in mem.read_text():
            print(patch(mem, WORKSPACE_SEED))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
