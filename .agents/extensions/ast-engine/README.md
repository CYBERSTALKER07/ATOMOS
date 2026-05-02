# Agent AST Engine

Local context engine that gives Copilot/Gemini a deterministic symbol graph before code generation.

## What It Does

- Scans TypeScript, JavaScript, and Go source files.
- Indexes definitions (function, class, interface, type, const/var) and import edges.
- Supports definition lookup, usage lookup, and impacted-file graph queries.

## Commands

Run from repository root:

```bash
node .agents/extensions/ast-engine/engine.mjs index --root pegasus --index .agents/extensions/ast-engine/.cache/pegasus-index.json
node .agents/extensions/ast-engine/engine.mjs definition --root pegasus --index .agents/extensions/ast-engine/.cache/pegasus-index.json --symbol RegisterRoutes
node .agents/extensions/ast-engine/engine.mjs usages --root pegasus --symbol RegisterRoutes --limit 30
node .agents/extensions/ast-engine/engine.mjs graph --root pegasus --symbol RegisterRoutes --limit 30
```

Run from the extension folder:

```bash
cd .agents/extensions/ast-engine
npm run index
npm run definition -- --symbol RegisterRoutes
npm run usages -- --symbol RegisterRoutes --limit 30
npm run graph -- --symbol RegisterRoutes --limit 30
```

## MCP Native Tools (Copilot)

Workspace MCP server config is in `.vscode/mcp.json` under server id `void-ast-engine`.
Repository template for team sync is `.github/mcp.vscode.example.json` because `.vscode/` is gitignored.

Native MCP tool names exposed by this server:

- `void_ast_index`
- `void_ast_definition`
- `void_ast_usages`
- `void_ast_graph`

Server entrypoint:

```bash
node .agents/extensions/ast-engine/mcp-server.mjs
```

## Expected Agent Flow

1. Build or refresh index.
2. Query target symbol definition.
3. Query usages to compute blast radius.
4. Query graph for imports/impacted files.
5. Read docs + code, then edit.
6. Re-index after edits and update architecture docs.

## ACT Policy Sync

- The engine is governed by `.github/ACT.md`.
- For technical requests, ACT is always-on and requires native MCP AST retrieval (fallback to scripts only if MCP is unavailable) plus architecture doc checks before execution.
