# pegasusX ⇄ Pegasus Parity Ledger

Maps each pegasusX surface to the Pegasus reference and tracks intentional divergence.

## Backend Route Families
| pegasusX | Pegasus reference | Divergence |
|---|---|---|
| `authroutes/` | `pegasus/apps/backend-go/authroutes` | None planned. |
| `supplierroutes/` | `pegasus/apps/backend-go/supplierroutes` | Single seeded supplier; same DTOs. |
| `supplierplanningroutes/` | `pegasus/apps/backend-go/supplierplanningroutes` | Same; topology bootstrap surfaces wired during onboarding. |
| `supplierinsightsroutes/` | `pegasus/apps/backend-go/supplierinsightsroutes` | Same. |
| `supplierlogisticsroutes/` | `pegasus/apps/backend-go/supplierlogisticsroutes` | Same. |
| `supplieroperationsroutes/` | `pegasus/apps/backend-go/supplieroperationsroutes` | Same. |
| `suppliercoreroutes/` | `pegasus/apps/backend-go/suppliercoreroutes` | Same. |
| `suppliercatalogroutes/` | `pegasus/apps/backend-go/suppliercatalogroutes` | Same. |
| `retailerroutes/` | `pegasus/apps/backend-go/retailerroutes` | Supplier discovery returns the seeded supplier. |
| `driverroutes/` | `pegasus/apps/backend-go/driverroutes` | Same. |
| `warehouseroutes/` | `pegasus/apps/backend-go/warehouseroutes` | Same. |
| `factoryroutes/` | `pegasus/apps/backend-go/factoryroutes` | Advanced lifecycle mounted additively (`start-loading`/`seal`/`dispatch`/`complete`, rebalance/cancel/cancel-transfer, exception queue) with scaffold in-memory state; outbox/event flow is active in scaffold runtime while Spanner-backed durability remains pending. |
| `payloaderroutes/` | `pegasus/apps/backend-go/payloaderroutes` | Advanced payload lifecycle/exception/reassignment mounted additively (manifests list/detail/start-loading/inject/seal, exception queue, recommendation + apply reassignment) with scaffold in-memory state; websocket relay path is active via typed fanout envelopes while production Kafka/Redis dependency hard guarantees remain pending. |
| `orderroutes/` | `pegasus/apps/backend-go/orderroutes` | Same. |
| `paymentroutes/` | `pegasus/apps/backend-go/paymentroutes` | Additive scaffold parity: checkout + chargeback/reversal + deprecated global-pay initiate are mounted with idempotency replay support and outbox events, backed by in-memory payment repository seams. |
| `webhookroutes/` | `pegasus/apps/backend-go/webhookroutes` | Signature-first HMAC scaffold handling with transaction-id idempotency and minimal provider payload contracts pending full provider SDK wiring. |
| `telemetryroutes/` | `pegasus/apps/backend-go/telemetryroutes` | Same. |

## Client Surfaces
| pegasusX | Pegasus reference | Divergence |
|---|---|---|
| `apps/supplier-portal` | `pegasus/apps/admin-portal` | Renamed for clarity; same role (SUPPLIER). |
| `apps/retailer-app-android` | `pegasus/apps/retailer-app-android` | Same role row. |
| `apps/retailer-app-ios` | `pegasus/apps/retailer-app-ios` | Same. |
| `apps/retailer-app-desktop` | `pegasus/apps/retailer-app-desktop` | Same. |
| `apps/driver-app-android` | `pegasus/apps/driver-app-android` | Same. |
| `apps/driver-app-ios` | `pegasus/apps/driverappios` | Folder renamed to `driver-app-ios` for consistency. |
| `apps/warehouse-portal` | `pegasus/apps/warehouse-portal` | Same. |
| `apps/warehouse-app-android` | `pegasus/apps/warehouse-app-android` | Same. |
| `apps/warehouse-app-ios` | `pegasus/apps/warehouse-app-ios` | Same. |
| `apps/factory-portal` | `pegasus/apps/factory-portal` | Same. |
| `apps/factory-app-android` | `pegasus/apps/factory-app-android` | Same. |
| `apps/factory-app-ios` | `pegasus/apps/factory-app-ios` | Same. |
| `apps/payload-terminal` | `pegasus/apps/payload-terminal` | Same. |
| `apps/payload-app-ios` | `pegasus/apps/payload-app-ios` | Same. |
| `apps/payload-app-android` | `pegasus/apps/payload-app-android` | Same. |

## Onboarding
| Concept | Pegasus | pegasusX |
|---|---|---|
| Supplier signup | Open registration | Single-tenant company bootstrap (one supplier seeded) |
| Step 2 | Single warehouse address + lat/lng on supplier row | Topology builder creates real `Factories` + `Warehouses` |
| Employment / staffing questions | Inferred via `/supplier/org` | Out of scope (intentionally removed) |
| Billing gate | `/setup/billing` | `/setup/billing` (identical) |

## Divergence Log
_Add an entry whenever pegasusX intentionally drifts from Pegasus behavior._
- 2026-05-21: Phase-1 SSMR physical sandbox baseline now lives in pegasusX with dedicated `docker-compose.ssmr.yml`, `.env.ssmr.example`, `apps/backend-go/cmd/setup`, tenant-scoped terraform resource/secret naming, and env-resolved `KAFKA_TOPIC_MAIN` for outbox topic isolation. Divergence remains explicit: the compose stack intentionally stops short of a Rust optimizer sidecar because pegasusX does not yet carry a concrete implementation.
- 2026-05-17: Firebase bearer verification is implemented as optional route-level middleware in pegasusX (`FIREBASE_AUTH_ENABLED`) while supplier onboarding keeps cookie JWT as canonical auth path.
- 2026-05-17: Supplier and retailer backend operational route coverage was expanded additively in pegasusX route composers. Retailer protected endpoints are Firebase-role-gated when enabled; local development fallback remains open when Firebase wiring is disabled.
- 2026-05-17: Driver and warehouse backend operational route coverage was expanded additively in pegasusX route composers. Warehouse local-scaffold mode currently uses cookie `ADMIN` fallback auth when Firebase verifier wiring is disabled.
- 2026-05-17: Factory and payload backend operational route coverage was expanded additively in pegasusX route composers with in-memory scaffold handlers. Factory (`FACTORY_ADMIN|ADMIN`) and payload (`PAYLOAD|ADMIN`) protected endpoints are Firebase-role-gated when enabled, with cookie `ADMIN` fallback auth in local scaffold mode.
- 2026-05-17: Payment and webhook backend operational route coverage was expanded additively in pegasusX route composers. Payment checkout/mutation endpoints now use idempotency replay guards and outbox emission, while webhook endpoints use signature-first HMAC verification and transaction-id idempotency with simplified provider payload contracts.
- 2026-05-17: Factory/payload advanced workflow parity moved beyond scaffold lists: manifest lifecycle transitions, exception handling, and reassignment depth are now available through additive endpoints. Current implementation persists in-memory only in pegasusX services and intentionally diverges from Pegasus transactional outbox + websocket fanout paths until persistence wiring is added.
- 2026-05-17: Shared event contracts are extended additively for advanced factory/payload workflow (`MANIFEST_ORDER_INJECTED`, `MANIFEST_ORDER_EXCEPTION`, `MANIFEST_DLQ_ESCALATION`, `MANIFEST_REBALANCED`, `MANIFEST_CANCELLED`) to preserve cross-client discriminator compatibility ahead of full outbox/websocket emission parity.
- 2026-05-17: P1 adapter bridge now attempts Redis cache backend and Kafka outbox publisher with fail-open fallback to in-memory/logging seams; websocket hubs now relay typed `ws:<hub>:fanout` envelopes with source-instance suppression. Additive upgrade: relay read/mark authority now binds to Spanner `OutboxEvents` through `outbox.SpannerStore` when Spanner is reachable, with in-memory fallback when unavailable.
- 2026-05-17: P1 strict reliability mode is additive in pegasusX: `REQUIRE_INFRA_ADAPTERS=true` enforces fail-fast startup if Redis or Kafka adapters fail to initialize, and bootstrap tests now cover strict fail-fast and healthy-adapter startup paths. Divergence remains: most domain repositories are still scaffold in-memory (no end-to-end Spanner ReadWriteTransaction domain persistence parity yet).
- 2026-05-17: Payment execution routing is now additive in pegasusX (`payment/execution.go`) with bounded retry/backoff+jitter and typed gateway policy errors, but provider SDK depth remains partial versus Pegasus; AIRWALLEX direct execution is feature-gated via `AIRWALLEX_DIRECT_EXECUTION_ENABLED` and defaults off in scaffold mode.
- 2026-05-18: Checkout attempt execution metadata persistence is now additive in pegasusX (`payment/service.go` + repository `SaveAttempt`) and checkout responses/events now expose `attempt_id`, `execution_action`, `execution_mode`, and `provider_reference`. Divergence remains: current scaffold persistence stores attempts in-memory; Spanner-backed `PaymentAttempts` durability is still pending.
