#!/usr/bin/env bash
# Idempotent: symlink skill + copy always-on rules into every IDE discovery path.
set -euo pipefail
REPO="$(cd "$(dirname "$0")/../../../.." && pwd)"
SKILL="$REPO/.agents/skills/graph-retrieval-memory"
ALWAYS="$SKILL/references/always-on.md"

link_skill() {
  local dest="$1"
  mkdir -p "$(dirname "$dest")"
  if [ -e "$dest" ] || [ -L "$dest" ]; then
    rm -rf "$dest"
  fi
  ln -s "$SKILL" "$dest"
  echo "skill -> $dest"
}

copy_rule() {
  local dest="$1"
  mkdir -p "$(dirname "$dest")"
  cp "$ALWAYS" "$dest"
  echo "rule  -> $dest"
}

# Repo skill aliases
link_skill "$REPO/.cursor/skills/graph-retrieval-memory"
link_skill "$REPO/.claude/skills/graph-retrieval-memory"
link_skill "$REPO/.github/skills/graph-retrieval-memory"
link_skill "$REPO/.gemini/skills/graph-retrieval-memory"
link_skill "$REPO/.continue/skills/graph-retrieval-memory"

# User-level skill aliases (this machine)
link_skill "$HOME/.grok/skills/graph-retrieval-memory"
link_skill "$HOME/.claude/skills/graph-retrieval-memory"
link_skill "$HOME/.cursor/skills/graph-retrieval-memory"
link_skill "$HOME/.codex/skills/graph-retrieval-memory"
link_skill "$HOME/.continue/skills/graph-retrieval-memory"

# User-level always-on rules
copy_rule "$HOME/.grok/rules/graph-retrieval-memory.md"
copy_rule "$HOME/.claude/rules/graph-retrieval-memory.md"
copy_rule "$HOME/.cursor/rules/graph-retrieval-memory.mdc"
copy_rule "$HOME/.windsurf/rules/graph-retrieval-memory.md"
copy_rule "$HOME/.continue/rules/graph-retrieval-memory.md"

# Cursor user .mdc needs frontmatter for alwaysApply
if ! grep -q 'alwaysApply' "$HOME/.cursor/rules/graph-retrieval-memory.mdc" 2>/dev/null; then
  tmp="$(mktemp)"
  printf '%s\n' '---' 'description: Graph retrieval + shared WORKSPACE.md memory' 'alwaysApply: true' '---' '' > "$tmp"
  cat "$HOME/.cursor/rules/graph-retrieval-memory.mdc" >> "$tmp"
  mv "$tmp" "$HOME/.cursor/rules/graph-retrieval-memory.mdc"
fi

# Cursor CLI: slash command + sessionStart hook (merge; keep existing hooks)
mkdir -p "$HOME/.cursor/commands" "$HOME/.cursor/hooks" \
  "$REPO/.cursor/hooks" "$REPO/.cursor/commands"
chmod +x "$SKILL/scripts/cursor_cli_session_hook.sh" "$SKILL/scripts/cursor_cli_memory.py"
ln -sfn "$SKILL/scripts/cursor_cli_session_hook.sh" "$HOME/.cursor/hooks/graph-retrieval-session.sh"
ln -sfn "$SKILL/scripts/cursor_cli_session_hook.sh" "$REPO/.cursor/hooks/graph-retrieval-session.sh"
cat > "$HOME/.cursor/commands/graph-retrieve.md" <<'EOF'
# Graph retrieve (Cursor CLI)

Hits are **paths**, not status. `generatedAt` is null. Code wins.

1. Read `$HOME/Desktop/V.O.I.D/.agents/memory/GOAL.md` then `WORKSPACE.md`.
2. Run (any cwd):

```bash
python3 "$HOME/.cursor/skills/graph-retrieval-memory/scripts/graph_retrieve.py" -q "$ARGUMENTS" --hops 2
```

3. Open every returned path this session. Grep if the graph misses (e.g. `fiscal retry`).
4. Persist only after file:line verify to `$HOME/Desktop/V.O.I.D/.agents/memory/WORKSPACE.md`.
EOF
echo "command -> $HOME/.cursor/commands/graph-retrieve.md"
echo "hook    -> $HOME/.cursor/hooks/graph-retrieval-session.sh"
echo "project hook -> $REPO/.cursor/hooks/graph-retrieval-session.sh"

python3 - <<PY
import json
from pathlib import Path

def merge(path: Path, command: str) -> None:
    data = {"version": 1, "hooks": {}}
    if path.is_file():
        data = json.loads(path.read_text())
    hooks = data.setdefault("hooks", {})
    existing = hooks.get("sessionStart") or []
    if not any(isinstance(h, dict) and h.get("command") == command for h in existing):
        existing.append({"command": command})
    hooks["sessionStart"] = existing
    data["version"] = data.get("version", 1)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2) + "\n")
    print(f"hooks.json -> {path}")

merge(Path.home() / ".cursor" / "hooks.json", "./hooks/graph-retrieval-session.sh")
merge(Path("$REPO") / ".cursor" / "hooks.json", ".cursor/hooks/graph-retrieval-session.sh")
PY

echo "done $REPO"
