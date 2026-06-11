## Summary

<!-- 1–3 bullets: what changed and why -->

## Parity matrix

- [ ] Updated `context/parity-ledger.md` if role-row contract or intentional divergence changed
- [ ] Cross-checked `docs/ROLE_ROW_PARITY_MATRIX.md` for the affected role row
- [ ] Every client in the role row updated OR partial rollout documented with feature flag + deadline

## Gap-hunter sweep

Run mentally (or via `.agents/skills/gap-hunter/SKILL.md`) before requesting review:

- [ ] **Contract drift** — producer JSON matches every consumer (Kafka, WS, mobile, portal)
- [ ] **Unwired features** — no UI calling missing endpoints; no events without consumers
- [ ] **Schema drift** — Spanner DDL ↔ Go struct ↔ `@pegasusx/types` ↔ native models aligned
- [ ] **Missing enforcement** — mutating handlers use outbox + cache invalidation + auth scope
- [ ] **Role scope** — no `supplier_id` / `warehouse_id` / `factory_id` trusted from request bodies

## Test plan

- [ ] `cd pegasusX && make backend-build`
- [ ] `cd pegasusX && make test-ssmr-infra` (when backend/routes touched)
- [ ] Portal smoke: affected app `npm run build` or `tsc --noEmit`
- [ ] Role-specific SSMR markers exercised (see parity-ledger phase table)

## SSMR subchecks (when applicable)

| Path | Command |
|---|---|
| Payment | `go run ./cmd/ssmr-smokecheck payment` |
| Shop closed | `go run ./cmd/ssmr-smokecheck shop-closed` |
| Manifest seal | `go run ./cmd/ssmr-smokecheck manifest-seal` |
| Full stack | `go run ./cmd/ssmr-smokecheck e2e` |
