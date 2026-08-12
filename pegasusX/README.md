# PegasusX Platform

PegasusX is the wire-ready B2B logistics and retail operating system (SoT tree for product code and docs).

## Documentation SoT

Start here — do **not** plan from frozen Reality Reports or empty SOP stubs:

| Doc | Role |
|-----|------|
| [`docs/DOCS_SOURCE_OF_TRUTH.md`](docs/DOCS_SOURCE_OF_TRUTH.md) | Living vs frozen index |
| [`docs/PROD_ECOSYSTEM_GOAL.md`](docs/PROD_ECOSYSTEM_GOAL.md) | North-star prod goal |
| [`docs/PROD_READINESS_SEQUENCE.md`](docs/PROD_READINESS_SEQUENCE.md) | Ordered residuals R0–R6 after W0–W5 |
| [`docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_2026-08-12.md) | Evidence backlog |
| [`context/current_status.md`](context/current_status.md) | SSMR / ops snapshot |
| [`context/architecture.md`](context/architecture.md) | Architecture overview |

## Current project state (2026-08-12 re-verify)

- **Local + SSMR:** Spanner migrations applied; GKE SSMR live; ManagedCert Active on `api-ssmr.pegasusx.app`.
- **In-tree waves W0–W5:** closed in code (see gap register). Residuals are **ops/owner keys** and a short Class A/client list — not Spanner DDL quota.

### Active ops blockers (not Spanner schema)

1. **Owner keys:** Global Pay merchant password → card SUCCESS; Firebase real SMS / device trust — [`docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md`](docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md)
2. **Legal fiscal (optional markets):** Soliq / E-IMZO PKCS#12 — [`docs/FISCAL_EDS_PROOF.md`](docs/FISCAL_EDS_PROOF.md)
3. **Prod optimizer:** AR image publish + raise `overlays/prod` optimizer-core **replicas 0 → ≥1** — [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) (SSMR overlay already patches **replicas: 1**)
4. **Observability:** set `enable_observability_resources=true` + alert channels — [`docs/PLATFORM_SLOS.md`](docs/PLATFORM_SLOS.md)
5. **Optional residual cloud quota:** staging GKE SSD/IP in asia-south1 — [`artifacts/GCP_SUPPORT_CASE_QUOTA.md`](artifacts/GCP_SUPPORT_CASE_QUOTA.md) (not a Spanner migration pause)

Do **not** set `PEGASUSX_ENV=production` until GP SUCCESS + `ValidateProductionProfile` pass.

## Core infrastructure stack

| Layer | Technology |
|-------|------------|
| Database | Google Cloud Spanner |
| Cache / fan-out | Redis (invalidation + WS relay) |
| Events | Kafka (outbox relay → consumers) |
| Compute | GKE (API, worker, ai-worker, optimizer-core) |
| Secrets | GSM + External Secrets Operator |
| Auth / push | Firebase Auth (OTP) + FCM |

## Architecture kernel (as-built)

```
Clients → backend-go → Spanner + same-txn outbox
  → relay → Kafka → consumers (WS / FCM / webhooks / mutators)
  → role clients
```

Coverage rule: every Spanner mutation is evented; every event has a consumer; cross-role loops are Class A on the platforms that role uses. See [`docs/PROD_ECOSYSTEM_GOAL.md`](docs/PROD_ECOSYSTEM_GOAL.md).
