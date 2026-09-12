# Gate-0 Track A — Blast radius (preflight)

**Date:** 2026-08-05  
**Tree:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`  
**SoTs read:** `.agents/AGENTS.md`, `docs/SUBSTANCE_GATE.md`, `docs/ROLE_ROW_PARITY_MATRIX.md`, `context/current_status.md`

| Fix | Roles | Route owner | Downstream | Clients |
|-----|-------|-------------|------------|---------|
| Fiscalize payment bypass | driver, supplier, retailer | `order/supplier_ops` confirm-bypass; fiscal worker | Outbox ORDER_STATUS → Kafka/WS; fiscal receipt | driver Android/iOS; supplier portal |
| Early-complete fiscal | supplier, driver | `order/supplier_ops` approve-early-complete | Same fiscal spine | supplier portal |
| Ops PIN bcrypt | warehouse, driver | `warehouse/ops_fleet_handlers` | Driver login verify | WH portal/Android/iOS |
| Nil Spanner fail-loud | all | `spannerutils` | Any RW mis-bootstrap | N/A (API) |
| Migration integrity | ops | `cmd/apply-migration` | Schema drift all roles | N/A |
| Outbox UUID + lease | all | `outbox/*` | Kafka/WS fanout | all |
| Idempotency hash/scope | retailer (+ mutating APIs) | `idempotency/*`, `packages/api-client` | Checkout duplicates | web + native keys |
| Theatre confidence/seasonality | supplier, AI | synthesis + replenishment + planning | Preorders / touchless | portal honesty |
| HPA/OSRM/backup/hygiene | ops | infra | Availability / RPO | N/A |

**Cold-verified open:** bypass→COMPLETED, PinHash `4321`, nil client silent OK, FailedPrecondition benign, evt_UnixNano, no outbox lease, FNV-32, unused MinConfidenceScore, seasonality unread, KMEANS label, no Spanner backup TF.
