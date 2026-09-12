# Honest code gate — reference

## Verdict labels

| Label | Meaning |
|-------|---------|
| **REAL** | Durable SoT + live client(s) on the path + fail-closed errors |
| **PARTIAL** | Durable but missing clients, outbox, edges, or a downstream role |
| **THEATRE** | Interface implies capability; persist/decision path missing (`{status:ok}`, always `[]`, in-memory sold as DB) |
| **GONE** | Explicit 410/403/501 (product removal) — OK if docs do not sell it as live |
| **DOC DRIFT** | Docs/matrix disagree with code |
| **NOT READY** | Must not start Layer B (cloud keys / prod APIs) |
| **READY FOR LAYER B** | Code+tests+data-flow proved; remaining work is secrets/env/IAM only |

## Theatre checklist

Flag immediately:

- Handler returns success without Spanner/Redis write
- List endpoint hard-codes `[]` / nil while UI shows a feature
- Duplicate alias route returns dummy; real route unused by clients
- `Apply()` for multi-row money/stock mutations
- Outbox outside the RW transaction (or `emit=nil`)
- In-memory maps for staff/history/session that production replicas cannot share
- 501/410 still linked from nav as a working action
- Test suite asserts the fake 200

## Impact trace (every edit)

Walk this list. Skip a row only with a one-line reason.

| Surface | What to grep / open |
|---------|---------------------|
| Route owner | `*routes/routes.go`, `main.go` mounts |
| Schema | `schema/spanner.ddl`, migrations |
| Contracts | `packages/types`, `packages/api-client`, `contracts/events.schema.json` |
| Events | outbox emit, topic routing, Kafka consumer, WS hub |
| Role row | every web/Android/iOS (or terminal) client for that role |
| Other features | who reads this table/status next |
| Infra | `.env*.example`, ConfigMap, ExternalSecret, Terraform, GSM |
| Tests | package `*_test.go`, SSMR markers, CI workflow for the slice |

## pegasusX paths (canonical tree)

Do not plan from `pegasus/` or frozen `.docx`.

| Resource | Path |
|----------|------|
| Docs vs code inventory | `pegasusX/docs/ROLE_FEATURES_DOCS_VS_CODE.md` (evidence, **re-verify**) |
| Substance Gate | `pegasusX/docs/SUBSTANCE_GATE.md` |
| Residuals order | `pegasusX/docs/PROD_READINESS_SEQUENCE.md` |
| Doc map | `pegasusX/docs/DOCS_SOURCE_OF_TRUTH.md` |
| Alignment rule | `pegasusX` via `Desktop/V.O.I.D/.cursor/rules/pegasusx-ecosystem-alignment.mdc` |
| Pre-cloud hunt | skill `gap-hunter` |
| Feature shape | skill `pegasus-doctrine` |

**Not certificates:** matrix **Wired**, FEATURES lists, AGENTS.md "runtime additive notes", previous chat claims.

## Tests to run (minimum)

After a backend slice:

```bash
cd pegasusX/apps/backend-go && go test ./<touched>/... -count=1
```

After contracts:

```bash
# from pegasusX — use repo Makefile targets that exist
make gen-contracts-gate   # if present
```

After ecosystem-visible behavior: SSMR marker or documented why UI-only.

If you cannot run tests in this environment, verdict is **unverified**, not done.

## Reply examples

**User:** "Can we start wiring cloud / PSP / GKE?"

```
VERDICT: NOT READY
EVIDENCE: factory dispatch stamps ≤2 CREATED transfers (factory/...); warehouse treasury invoices return []
DOCS vs CODE: ROLE_ROW may say Wired for nearby screens — those rows are not Layer B proof
BLOCKERS: 1) factory dispatch engine 2) honest treasury read 3) …
NEXT: Phase P3/P2 in ROLE_FEATURES_DOCS_VS_CODE — code, not Terraform
```

**User:** "Did we implement the plan?"

Re-read edits + live path + tests. If a test failed or a file still has `emit=nil`:

```
VERDICT: PARTIAL — plan not closed
EVIDENCE: transfer create still apply(..., emit=nil) at factory/service.go:N
NEXT: emit outbox in the same RW txn, then re-test; do not start the next module
```


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
