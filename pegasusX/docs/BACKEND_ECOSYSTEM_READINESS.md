# Pegasus X — Backend Ecosystem Readiness

> **Canonical ecosystem spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md) · **Cost model:** [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md)

This document is the handoff checklist for Boss: backend-first readiness before cloud cutover and frontend wiring. Canonical tree: `pegasusX/`.

## Architecture (runtime)

| Layer | Component | Role |
| --- | --- | --- |
| HTTP | `chi` router in `apps/backend-go/main.go` | REST for all roles; `SessionAuth` (cookie + Bearer JWT) |
| Persistence | Spanner (prod) / in-memory scaffold (SSMR) | Orders, suppliers, retailers, fleet, manifests |
| Cache | Redis + Pub/Sub invalidation | List caches, last driver location, WS relay |
| Events | Transactional outbox → Kafka `KAFKA_TOPIC_MAIN` | Durable state transitions |
| Real-time | 7 WebSocket hubs + Redis cross-pod relay | Per-role rooms; telemetry on dedicated hub |
| Notifications | `kafka.NotificationDispatcher` | Kafka → WS fan-out (multi-pod) |
| Webhooks | `webhookroutes` (signature-first) | GlobalPay and gateway settlement |
| Telemetry | `POST /v1/telemetry/location` | Driver JWT → `telemetry:driver:*` + `telemetry:supplier:*` |

## Role → API → Real-time matrix

| Role | JWT `role` | Primary REST prefix | WebSocket rooms (on `/v1/ws`) |
| --- | --- | --- | --- |
| SUPPLIER (Admin Portal) | `ADMIN` | `/v1/supplier/*`, `/v1/auth/supplier/*` | `supplier:{supplier_id}`, `telemetry:supplier:{supplier_id}` |
| RETAILER | `RETAILER` | `/v1/retailer/*`, `/v1/auth/retailer/*` | `retailer:{retailer_id}` |
| DRIVER | `DRIVER` | `/v1/driver/*`, `/v1/telemetry/location` | `driver:{driver_id}`, `telemetry:driver:{driver_id}` |
| WAREHOUSE_ADMIN | `ADMIN` + `supplier_role=WAREHOUSE_ADMIN` | `/v1/warehouse/*` | `warehouse:{home_node_id}`, `supplier:{supplier_id}`, telemetry supplier room |
| FACTORY_ADMIN | `ADMIN` + `supplier_role=FACTORY_ADMIN` | `/v1/factory/*` | `factory:{home_node_id}` **and** `factory:{supplier_id}` (scaffold), `supplier:*`, telemetry |
| PAYLOAD | `PAYLOAD` | `/v1/payload/*` | `payload:{subject}` **and** `payload:{supplier_id}` |

Order create and payment paths also perform **best-effort local WS broadcast** in-process; cross-pod delivery uses outbox → Kafka → notification dispatcher.

## Kafka events consumed by notification dispatcher

| Event family | WS targets |
| --- | --- |
| `ORDER_*` (created, status, finalized, assigned, reassigned) | supplier, retailer, driver (+ warehouse on assign) |
| `PAYMENT_*`, `SETTLEMENT_*`, `DELIVERY_DISPUTED` | supplier, retailer |
| `RETAILER_REGISTERED` | supplier, retailer |
| `SUPPLIER_UPDATED`, `SUPPLIER_BILLING_CONFIGURED` | supplier |
| `DRIVER_CREATED`, `VEHICLE_CREATED` | supplier, warehouse (if home warehouse), driver |
| `WAREHOUSE_*` | supplier, warehouse |
| `MANIFEST_*` | supplier, factory, warehouse, driver, payload |
| `SHOP_CLOSED`, `SHOP_CLOSED_RESPONSE`, `SHOP_CLOSED_ESCALATED`, `SHOP_CLOSED_RESOLVED` | supplier, retailer, driver |

Telemetry (`DRIVER_LOCATION_UPDATED`) is **HTTP → TelemetryHub only** (not Kafka) by design (loss-tolerant, high volume).

## Verification commands

```bash
# Unit tests (backend-go)
cd pegasusX/apps/backend-go && go test ./...

# Full SSMR stack + cross-role E2E (requires Docker)
cd pegasusX && make test-ssmr-infra
```

E2E markers when green: `PX_E2E_ORDER_OK`, `PX_E2E_PAYMENT_OK`, `PX_E2E_WAREHOUSE_OK`, `PX_E2E_FACTORY_OK`, `PX_E2E_DELIVERY_OK`, `PX_E2E_TELEMETRY_OK`, `PX_E2E_PAYLOAD_OK` (umbrella; sub-markers `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`, `PX_E2E_PAYLOAD_REASSIGN_OK`, `PX_E2E_PAYLOAD_DRIVER_GATE_OK`, `PX_E2E_PAYLOAD_DEVICE_TOKEN_OK` — payloader login → manifest lifecycle → driver gate/detail → recommend/apply reassign → fleet reassign → manifest-exceptions → payloader device-token), `PX_E2E_SHOP_CLOSED_OK` (assign → ARRIVED → driver report → retailer `OPEN_NOW`), `PX_E2E_REPLENISH_OK`, `PX_E2E_REPLENISH_COLOCATE_OK`, `__SSMR_OK__`.

## Environment (SSMR / local)

Copy `pegasusX/.env.ssmr.example` → `pegasusX/.env.ssmr`. Key flags:

- `REQUIRE_INFRA_ADAPTERS=true` — Spanner emulator, Redis, Kafka
- `KAFKA_TOPIC_MAIN` — main notification topic (default `pegasusx-main`)
- `GLOBAL_PAY_WEBHOOK_SECRET` — webhook replay in smoke (dev `dev-*` values allowed when `PEGASUSX_ENV` is not `production`)
- `SHOP_CLOSED_GRACE_MINUTES` — escalation timer before `SHOP_CLOSED_ESCALATED` (default `5`)
- `ALLOW_DRIVER_DEMO_FALLBACK=true` — optional dev-only demo fleet orders when Spanner list query is unavailable
- `JWT_SECRET` / `JWT_ISSUER` — all role tokens

Production: set `PEGASUSX_ENV=production` and non-`dev-*` webhook secrets or bootstrap refuses to start.

Cloud credentials: see `docs/CLOUD_CREDENTIALS_CHECKLIST.md` and `docs/CLOUD_CUTOVER_RUNBOOK.md`.

## Known progressive gaps (do not block SSMR)
- Inline-Kafka audit closed (PX11-D4): all state transitions emit via `outbox.EmitJSON`; the only direct producers are the outbox relay, DLQ writer, CLI tools, and the loss-tolerant planning signal-ingest publisher (by design, like telemetry).
- Factory portal / admin portal UI parity deferred until backend sign-off.
- Load cert gate: `make load-cert` / `make load-cert-ssmr` (`scripts/load/`); report template filled by `generate_report.py` after each run.

## Warehouse Admin iOS (`apps/warehouse-app-ios`)

Full SwiftUI parity with `pegasus/apps/warehouse-portal` (17 routes via tabs + **More** hub). Demo login: `POST /v1/auth/warehouse/login` with phone `+998901000088` / PIN `1234`. Backend port **8180** in debug.

## Boss sign-off criteria

1. `make test-ssmr-infra` passes on CI or local Docker.
2. Mobile/web clients can use documented REST + WS room keys against `PUBLIC_BASE_URL`.
3. Cloud cutover applies Terraform + real Spanner/Redis/Kafka with same env contract.
