# Agent memory (shared)

| File | Who uses it |
|------|-------------|
| `GOAL.md` | **Final goal** — read first, every new session. Destination = `GLOBAL_SCALE_PROGRAM.md` + `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` |
| `WORKSPACE.md` | **All** IDEs/agents — read/write verified facts |
| `sessions/` | Ephemeral notes (gitignored) |

Grok also indexes `~/.grok/memory/` when `[memory] enabled`. Seed script syncs WORKSPACE.md into that tree when Grok creates a workspace slug dir.

Loop: `.agents/skills/graph-retrieval-memory/references/always-on.md`
