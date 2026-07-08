# pegasusX ecosystem alignment (required on every change)

When you edit backend code or add a feature, **trace every surface the change touches** and update them in the same batch. Do not land a partial slice that leaves role rows, contracts, or cross-role flows inconsistent.

## 1. Map the blast radius first

Before coding, identify:
- **Role(s)** affected (supplier, retailer, driver, warehouse, factory, payload)
- **Route owner** (`*routes/routes.go` under `apps/backend-go`)
- **Cross-role consumers** (who reads this state next in the order/dispatch/payment chain)
- **Realtime path** (outbox event → Kafka → WS hub → client inbox)

Reference: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`, `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`.

## 2. Backend mutation checklist

For any state-changing backend work, include in the same change set:
- Spanner schema/migration if columns or indexes change (`schema/spanner.ddl`)
- Repository + service in the **canonical owner package** (not a duplicate path)
- Outbox emit in the **same RW transaction** as the row write
- Post-commit **cache invalidation** keys (supplier/retailer/catalog/inventory as applicable)
- WebSocket fanout envelopes for roles that must react live
- Focused `*_test.go` in the touched package
- **SSMR marker** in `cmd/ssmr-smokecheck/e2e_check.go` when behavior is user-visible or cross-role; register in `contracts/ssmr_ecosystem_markers.json` if ecosystem-gated

Cancel/side-effect paths: if a new code path sets terminal state (cancel, reject, vet reject), verify **inventory release**, payment state, and notification fanout — not only the happy-path `UpdateStatus`.

## 3. Role-row client parity

A feature for a role must land on **all clients in that role row** unless explicitly deferred in context docs:

| Role | Clients |
|---|---|
| Supplier | portal, Android, iOS |
| Retailer | desktop, Android, iOS |
| Driver | Android, iOS |
| Warehouse | portal, Android, iOS |
| Factory | portal, Android, iOS |
| Payload | terminal, Android, iOS |

Shared contracts first: `packages/types`, `packages/api-client`, then each client. Match existing patterns (silent WS refresh, idempotency keys, claims-scoped API calls).

## 4. Contracts & events

When API shapes or events change:
- `packages/types` + `packages/api-client`
- `contracts/events.schema.json` via `go run ./cmd/gen-contracts` (CI: `make gen-contracts-gate`)
- Regenerate native `Generated/` stubs where apps wire Quicktype (Android Gradle, iOS build phases)

## 5. Infra & config (when env or secrets change)

- `.env.ssmr.example`, `.env.example`, K8s configmap/externalsecret, Terraform GSM if new secrets
- `docs/CLOUD_CREDENTIALS_CHECKLIST.md` when a new external service is introduced

## 6. Context docs (same PR / session)

- Update `context/*_PHASE.md` or `context/plan.md` anchor status when closing or opening work
- `context/parity-ledger.md` if behavior intentionally diverges from Pegasus reference
- `docs/ROLE_ROW_PARITY_MATRIX.md` row status when a screen/API moves from partial → wired

## 7. Definition of done

A feature is not done until:
1. All touched role-row clients compile and use the same contract
2. Cross-role downstream effects are handled (or explicitly documented as deferred)
3. `go test` on touched backend packages passes
4. New ecosystem behavior has an SSMR assertion or a documented reason it is UI-only / manual QA
