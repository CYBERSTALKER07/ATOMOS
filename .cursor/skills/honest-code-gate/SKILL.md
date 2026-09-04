---
name: honest-code-gate
description: >-
  Forces honesty from real code (not docs) for status, readiness, wiring, and
  "are we done" questions. Production/cloud/API/infra/deploy gate: YES only if
  backend, apps, and data flow actually work. Phased verify, blast-radius on
  every edit, tests before done. Use when the user asks if something is wired,
  implemented, production-ready, or whether to connect cloud, GCP, APIs, infra,
  Terraform, deploy; after implementing a plan; when editing any app/backend/UI/infra.
---

# Honest code gate

**Law:** Current source code is the only status SoT. Docs and matrices are compared **to** code, never used **as** code.

Read [reference.md](reference.md) for the verdict template, theatre checklist, and pegasusX paths.

## When this applies

- Any question about **status**, **wired**, **done**, **ready**, **production**, **cloud**, **deploy**, **infra**, **APIs**
- After **planning** and after **implementation**
- Any **create/edit** on apps, backend, clients, UI, infra, cloud

## Phase 0 — Answer shape (before coding)

Every status/readiness reply uses this skeleton:

```
VERDICT: REAL | PARTIAL | THEATRE | GONE | NOT READY | READY FOR LAYER B
EVIDENCE: file:line (opened this session)
DOCS vs CODE: match | drift (quote both)
BLOCKERS: ranked, or none
NEXT: one phase, or Layer B secrets-only
```

Do **not** say the system can start connecting cloud unless VERDICT is **READY FOR LAYER B**.

## Phase 1 — Trace the live path

For the claimed capability:

1. Find the **route mount** (not only a function that exists).
2. Auth/claims — body IDs are not authority.
3. Mutation: durable write **or** honest 4xx/5xx/410. `{status:ok}` with no persist = THEATRE.
4. Same-transaction outbox if it is a state change others must see.
5. Consumer + WS + **each client in the role row** that should call it.
6. Tests that prove the behavior (not tests that freeze theatre).

If you did not open those files this session, you do not have a verdict.

## Phase 2 — Cloud / infra / API wiring

**READY FOR LAYER B** only if:

- Product spine in this repo is REAL on the claimed path (all shipped clients or explicit deferral with owner)
- Data flow is durable (DB + outbox + consumers), not in-memory theatre
- Unit + integration/CI-equivalent for that spine were run and passed **after** the latest edits
- Remaining work is credentials, env, DNS, IAM — **not** new business logic

Otherwise **NOT READY**. Do not start Terraform/GKE/PSP/OFD key wiring as a substitute for missing code.

When in `pegasusX/`: also run the `gap-hunter` skill and Substance Gate (`docs/SUBSTANCE_GATE.md`). Matrix "Wired" ≠ Layer B.

## Phase 3 — Execute (if building)

1. One slice per phase. Deep-dive that slice.
2. **Impact trace** before edit: other files, other clients, cloud config, downstream features (see reference.md).
3. Prefer existing skills + official docs + web + proven OSS/big-tech algorithms (cite + adapt, honor license). If none fit, invent tested in-house logic — do not name-drop an algorithm you did not implement.
4. Keep the tree production-shaped: fail closed, no new theatre, no silent `{status:ok}`.

## Phase 4 — Verify (mandatory, even if the plan "succeeded")

The plan is not the result.

1. Re-read **every file you edited**.
2. Re-trace the live path (Phase 1) on the result.
3. Run unit tests on touched packages; run integration/CI-equivalent for the slice.
4. If behavior is wrong, incomplete, or untested → **not done**. Replan. Close gaps. Do not claim the phase complete.

## Agents

Use extra agents for parallel traces (one role row or one package each). The parent issues a single honest VERDICT. Extra agents are optional.

## Anti-patterns

- Answering from `ROLE_ROW_PARITY_MATRIX`, `FEATURES_BY_APP_ROLE`, or chat history
- "Implemented the plan" without re-read + tests
- Starting cloud because remaining items are "just infra" while handlers are stubs
- Porting from `pegasus/` as if it were the product (pegasusX is the living tree)


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
