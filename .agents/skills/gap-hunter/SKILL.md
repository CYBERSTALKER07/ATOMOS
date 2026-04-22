---
name: gap-hunter
description: Systematically finds and fixes "silent failures" in the V.O.I.D. monorepo — contract drift (event/DTO shape mismatches between producers and consumers), dead code (defined but never called), unwired features (UI calls endpoint that doesn't exist, or endpoint emits events nobody subscribes to), schema drift (Spanner DDL ≠ Go struct ≠ TS type ≠ Swift/Kotlin model), missing enforcement (mutations without outbox.Emit, cache.Invalidate, RequireRole, signature verification, idempotency guard), and nonsense naming / role confusion. Use when the user asks to "audit", "find inconsistencies", "sanity check", "find gaps", "find dead code", "find nonsense", "what's broken but silently", "contract mismatch", "why doesn't this notification arrive", "why is this cache stale", or after any cross-cutting refactor.
version: 1.0.0
---

# Gap Hunter — The Silent-Failure Detective

Ghost entities, zombie types, broken notification chains, and stale caches do not raise alarms. They accrete until an operator at 3 AM can't explain why a driver's phone shows one route and the admin portal shows another. This skill is the systematic sweep that finds them before the operator does.

## When to Use

Invoke when any of these trigger:
- User asks to "audit", "sanity-check", "find gaps / nonsense / dead code / contract drift / silent failures".
- Finishing a feature that crosses the **backend → Kafka → consumer → WS → mobile** seam.
- Before declaring a phase complete (Wave B, Phase 3, etc.).
- After any rename, type extraction, or package split.
- When a user reports a flaky symptom with no obvious cause ("notification didn't arrive", "cache won't clear", "driver assignment shows wrong name").

## The Seven Classes of Gap

Every hunt scans for these, in this order (cheapest to expensive):

### 1. Contract Drift — Same Name, Different Shape
Two packages define a type with the **same name** but **different JSON fields**. Producer writes shape A, consumer decodes shape B, `json.Unmarshal` succeeds silently with zero-valued fields, and the feature never works.

Search:
```bash
# Go: duplicate type names
grep -rnE '^type [A-Z][A-Za-z]+Event ' --include='*.go' . | awk -F: '{print $NF}' | sort | uniq -c | sort -rn | awk '$1>1'
# Then diff the struct bodies side-by-side.
```
Also check:
- `type X request` in handler vs. `type X` in DTO package.
- Swift `Codable` struct vs. Kotlin `@Serializable` vs. TS `type` — all should match the backend JSON tags.

### 2. Dead Code — Defined but Never Called
A function, hub method, or helper exists but no caller invokes it. Either the wiring was never completed or the call site was deleted.

Search:
```bash
# For a symbol "Foo" inside package "bar":
grep -rnE '\bbar\.Foo\b' --include='*.go' . | grep -v '_test.go'
# If zero callers outside bar/ itself, it is dead or orphaned.
```
Before deleting, confirm it is not part of a public API exported for a future ticket; if uncertain, ask.

### 3. Unwired Features — Producer Without Consumer (or Vice Versa)
- Kafka event `EventX` produced by handler A but no `case EventX:` in any consumer switch.
- HTTP endpoint mounted but no frontend / mobile client calls it.
- WebSocket room broadcast but no client subscribes.
- Spanner column written but never read (or read but never written).

Search:
```bash
# Event produced but not consumed:
grep -rnE 'EventX|"X_EVENT"' --include='*.go' . | grep -v _test.go
# Column written but not read:
grep -rnE '"ColumnName"' --include='*.go' .
```

### 4. Schema Drift — DDL ↔ Struct ↔ Wire Type
Spanner DDL declares column `HomeNodeId STRING(36)`. The Go struct reading that row must list it; the INSERT must write it; the DTO serialising it must carry it; the mobile / frontend model must parse it. Any missing layer = partial migration.

Search:
```bash
# For every column in spanner.ddl, grep the Go reader/writer:
grep -nE 'CREATE TABLE|STRING\(|INT64|TIMESTAMP' schema/spanner.ddl
grep -rnE '"ColumnName"' --include='*.go' .
```

### 5. Enforcement Gaps — Mutation Without Its Guard
Every state-changing handler must pair mutation with four guards:

| Guard | Required when | Grep for its absence |
| --- | --- | --- |
| `outbox.EmitJSON` | state transition emitted as Kafka event | mutation inside RW txn with no `outbox.` call |
| `cache.Invalidate` | cached aggregate mutated | `txn.Commit`/`Apply` with no `cache.Invalidate` after |
| `auth.RequireRole` / signature check | HTTP entry point | `func.*http.HandlerFunc` with no auth wrapper in registration |
| `idempotency.Guard` | webhook / external trigger | `webhookroutes/` handler with no idempotency lookup |

### 6. Role / Scope Violations
- `supplier_id`, `factory_id`, `warehouse_id` read from `r.Body` instead of JWT claims → role-spoofing P0.
- `RequireWarehouseScope` missing on a warehouse-scoped handler.
- A factory admin allowed to read all warehouses because the default branch doesn't pin scope.

### 7. Nonsense Naming / Role Confusion
- "admin" used to mean something other than the Supplier-Portal user.
- `Customer` referring to a Retailer.
- `DriverId` vs `RouteId` used interchangeably when they are actually the same value (pre-existing convention debt — flag, don't auto-fix).

## The Hunt Protocol

1. **Define scope** — which packages, which apps, which time window of recent changes? Narrow if possible.
2. **Run cheap grep sweeps** for Classes 1–3 first. Collect candidates.
3. **Triangulate each candidate** — confirm it is a real gap via `codebase-retrieval` + `view`.
4. **Rank by blast radius**:
   - P0: money / auth / data integrity (ghost entity, missing outbox, body-trusted supplier_id).
   - P1: user-visible silent failure (notification never arrives, stale cache in prod).
   - P2: dead code / naming debt (no runtime impact).
5. **Propose fix per gap** before writing code — the user may want to defer some (e.g., pre-existing payload-shape mismatches that would expand scope).
6. **Fix smallest-blast-radius-first**; build after each fix; never batch 5 fixes into one commit.
7. **Write a test** that would have caught the gap, where the test surface exists.

## Fix Patterns by Class

| Class | Canonical Fix |
| --- | --- |
| Contract drift | Collapse to ONE shared type in the owning domain package; add a JSON round-trip test. |
| Dead code | Delete OR wire to its missing caller; never leave "someone will use this later". |
| Unwired feature | Either wire it end-to-end in the same PR OR remove the half-wired piece. |
| Schema drift | Migrate struct + DTO + mobile model in one commit; add a backfill if needed. |
| Missing guard | Add the guard + retrofit the test; file a follow-up for peers that share the omission. |
| Role violation | Drop the body field; resolve from `auth.*` claims; add a security regression test. |
| Naming debt | Flag in "Known Gaps" of `gemini-instructions.md`; fix only when touched by a feature PR. |

## Output Format

Every hunt report ends with:
1. **Findings** — table of `(class, severity, location, description)`.
2. **Fixed now** — bullet list of files touched with one-line rationale.
3. **Flagged for follow-up** — deferred items with suggested owner / ticket.
4. **Regression tests added** — file paths + what they assert.

No decorative prose, no "great question" openers, no summary of what the user already knows. Operator-grade output only.
