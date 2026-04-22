---
name: efficient-code
description: Produce efficient, context-aware code in the V.O.I.D. monorepo — ground every line in the existing doctrine (backend topology, enterprise algorithm patterns, clean-code standards, cross-role sync, Wave-Based Extraction), reuse existing primitives instead of reinventing them, pick the lowest-cost data path (index-backed reads, batched writes, cache hits, bounded goroutines), and ship code that an Augment agent one year from now can read, extend, and refactor without archaeology. Use when the user asks to "write", "implement", "add", "build", "refactor", "optimize", "clean up", "make this faster", "make this readable", "do this the right way", or any time code is about to be authored or modified in this repo.
version: 1.0.0
---

# Efficient Code — Context-Aware Authoring in V.O.I.D.

Efficient code in this repo is not "fewer lines" or "clever tricks". It is code that (a) reuses the primitives already solved, (b) chooses the lowest-cost data path, (c) matches the existing shape so review is cheap, and (d) survives the next refactor without silent breakage. Writing without context is how duplicate `OrderReassignedEvent` types, two `cache.Get` implementations, and a fourth bespoke idempotency layer are born.

## When to Use

Invoke at the start of every code-authoring task:
- User says "implement / add / build / write / refactor / optimize / fix / clean up".
- You are about to create a new file, function, struct, event, or endpoint.
- You are touching an existing file for anything beyond a single-line fix.
- You are tempted to "just add a quick helper" — that helper likely already exists.

## The Five Phases

### Phase I — Ground Context (before typing)
Answer these five questions. If any answer is "I don't know", search the codebase before writing.
1. **Which domain package owns this logic?** Map to one of the 33 domain packages listed in `.github/gemini-instructions.md` § Backend Package Topology. If "none obviously", the answer is not "create a new package" — re-read the list.
2. **Is there an existing primitive for what I'm about to write?** Search BEFORE writing: `cache.Invalidate`, `outbox.EmitJSON`, `idempotency.Guard`, `auth.RequireRole`, `h3.CellOf`, `circuit.Breaker`, `priorityGuard`, `notifications/formatter`, `kafka/events.go` structs. If a primitive exists, use it — do not wrap it, do not reinvent it.
3. **Which role(s) consume this?** Map to the Cross-Role Sync matrix (SUPPLIER / DRIVER / RETAILER / PAYLOAD / FACTORY_ADMIN / WAREHOUSE_ADMIN). Identify every client surface for that role before designing the response shape.
4. **What is the existing file-shape convention?** Open two sibling files in the target package. Match their imports order, handler signature, error style, comment density. Deviation without reason is noise.
5. **Is this a mutating path?** If yes, the six-step mutating-handler shape is mandatory: auth gate → method gate → `ReadWriteTransaction` → `outbox.EmitJSON` inside the txn → `cache.Invalidate` after commit → structured log with `trace_id`. Decide now which aggregate, topic, and event type you will use.

### Phase II — Choose the Lowest-Cost Data Path
Efficiency in V.O.I.D. is algorithmic + infrastructural, not micro-optimization.
- **Reads**: index-backed or don't ship. Every `WHERE` clause must hit a declared secondary index. For large lists, `spanner.TimestampBound{StaleRead: 15s}` is the default — strong reads only when correctness demands.
- **Geo queries**: `h3.GridDisk(cell, k)` + `WHERE H3Cell IN UNNEST(@cells)` against `Idx_*_ByH3Cell`. Never `ST_Distance` full-scan.
- **Writes**: batch under 1000 mutations per `ReadWriteTransaction`. `InsertOrUpdateMap` for bulk. Never `Apply` for multi-row.
- **Caches**: `cache.Get` before Spanner; `cache.Invalidate` after every mutation. TTL is the safety net, not the correctness mechanism.
- **Kafka reads**: one goroutine per partition, bounded by `GOMAXPROCS`. Version-gate every event against stored `Version` — stale replays ACK-and-skip.
- **External calls**: wrap in `pkg/circuit.Breaker`. Payme/Click/Stripe/FCM/Telegram/Firebase are breaker-mandatory.
- **Loops**: if iterating > 100 items with I/O inside, use `errgroup` with `SetLimit`. Never unbounded `go fn()`.

### Phase III — Apply the Clean-Code Standards (per every line written)
The non-negotiable rules from `.github/gemini-instructions.md` § Clean Code Standards:
- Function ≤ 60 lines. Parameters ≤ 5 (else `Deps` struct).
- Guard clauses over nested ifs. ≤ 3 nesting levels.
- Errors wrap with `fmt.Errorf("...: %w", err)`. `errors.Is` / `errors.As` only.
- No `float64` for money. Typed IDs (`type OrderID string`) at package boundaries.
- No package-level mutable vars. Singletons on `*bootstrap.App`.
- `context.Context` always first arg of I/O functions.
- JSON tags match DDL snake_case EXACTLY.
- Exported symbols have doc comments starting with the symbol name.
- No commented-out code, no `TODO` without an issue link, no debug prints.

### Phase IV — Verify the Cross-Seam Integrity
Before declaring done, confirm each seam your change crosses:
- **Backend → Kafka**: producer struct and consumer struct are the SAME type from `kafka/events.go` (never redeclared). Run gap-hunter if uncertain.
- **Backend → Cache**: every mutated aggregate has a matching `cache.Invalidate` call post-commit.
- **Backend → WebSocket**: if the event is user-visible, a `Hub.BroadcastX` call exists and is reachable from the mutation path (directly or via consumer).
- **Backend → Mobile/Web**: for every response field added, walk the Cross-Role Sync protocol (9 steps). Do not claim "the mobile app can add it later" — that is partial-rollout violation.
- **Go → go.sum**: no new dependency added without `go mod tidy` and a line in PR description explaining why the stdlib + existing deps are insufficient.

### Phase V — Reduce, Then Stop
After the code compiles and tests pass, do one pass of reduction:
- Delete any `Deps` field you ended up not using.
- Collapse two helpers that share > 70% of their body into one.
- Inline a single-use helper if its name is not more informative than its body.
- Confirm `go build ./...`, `go vet ./...`, and `go test ./<pkg>/...` are all clean.
- Confirm the mechanical review checklist (`.github/gemini-instructions.md` § Clean Code Standards rule 10) passes for every touched file.

Do NOT keep reducing past this. Further "clever" condensation is negative-value: it makes the next reader slower.

## Anti-Patterns This Skill Prevents

| Anti-pattern | Symptom | The V.O.I.D. fix |
|---|---|---|
| Redeclaring an event struct in a new package | Producer and consumer see different fields; silent zero-values. | Use the canonical struct in `kafka/events.go`. Add new fields additively. |
| Writing a bespoke cache wrapper | TTL-only, no Pub/Sub invalidation; stale reads across pods. | Use `cache.Cache` from `bootstrap.App`. |
| Direct `writer.WriteMessages` in a mutation handler | Ghost entity: Spanner commits, Kafka fails, no one notices. | `outbox.EmitJSON` inside the same `ReadWriteTransaction`. |
| `float64` for money | Reconciliation mismatches at 0.01 UZS scale. | `int64` minor units + `Currency` field. |
| Package-level `var spannerClient *spanner.Client` | Test isolation broken, init-order bugs. | Hang on `*bootstrap.App`, pass through `Deps`. |
| 400-line handler inline in `main.go` | Unreviewable PRs, route discovery via grep. | Wave-Based Extraction Playbook — lift to `*routes` package. |
| Reinventing `idempotency.Guard` for a new webhook | Duplicate charges on gateway retry. | Use the existing guard, keyed on the gateway's transaction id. |
| Unbounded `for ... { go fn() }` | Goroutine explosion on spike load. | `errgroup.WithContext` + `SetLimit`. |
| Spanner query without an index | p99 latency blows up at scale. | Declare the index in `migrations/`; confirm with `EXPLAIN`. |
| Raw string role checks (`if claims.Role == "admin"`) | Role-name drift silently opens the door. | `auth.RequireRole(...)` / `claims.ResolveSupplierID()`. |

## Mental Model

Before every line:
> "What does this repo already provide that does 80% of this job? I will use that, add the missing 20% in the canonical shape, and leave nothing behind that a future reader would have to decode."

When in doubt, run `gap-hunter` on the surrounding area before writing. When a new primitive genuinely does not exist, name it precisely, document its WHY, and put it in the package that owns the concern — never a utility bucket.

## Companion Skills

- `gap-hunter` — run before and after non-trivial changes to catch contract drift, dead code, and unwired features.
- `test-with-spanner` — for tests that touch Spanner, run via the emulator harness.
- `swiftui-pro` — when the change crosses into iOS client code.

## Source of Truth

Every rule above is a direct reference to `.github/gemini-instructions.md`. If the instructions file and this skill ever disagree, the instructions file wins — update this skill to match.
