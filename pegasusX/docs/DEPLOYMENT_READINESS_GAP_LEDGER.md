# Deployment Readiness Gap Ledger (PX-12)

> **Canonical ecosystem spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md)

Last updated: 2026-06-05. Tracks production v1 closure — not full Pegasus reference parity.

## Priority legend

| Priority | Meaning |
|----------|---------|
| **P0** | Shipped client calls endpoint; backend missing or returns 501/404 — blocks deployment |
| **P1** | Backend exists; not all role-row clients wired |
| **P2** | Intentional delta vs Pegasus; documented in `context/parity-ledger.md` |

## Resolved — P0 contract (PX12-B)

| ID | Role | Path | Owner phase |
|----|------|------|-------------|
| P0-01 | DRIVER | `POST /v1/fleet/route/reorder` | `order/driver_edges.go` |
| P0-02 | DRIVER | `POST /v1/delivery/bypass-offload` | shop-closed bypass |
| P0-03 | DRIVER | `POST /v1/delivery/credit-delivery` | driver edges |
| P0-04 | DRIVER | `POST /v1/delivery/missing-items` | driver edges |
| P0-05 | DRIVER | `POST /v1/delivery/split-payment` | driver edges |
| P0-06 | RETAILER | `GET /v1/catalog/categories/{id}/suppliers` | `catalog` |
| P0-07 | RETAILER | auto-order settings scaffold | `retailer/auto_order.go` |
| P0-08 | DRIVER | FCM `POST /v1/user/device-token` | driver Android + iOS login hook |

## P1 — role-row UI / API client (PX12-F–K)

| ID | Role | Gap | Phase |
|----|------|-----|-------|
| P1-01 | SUPPLIER | Native apps: ops slice vs portal depth | **Closed** — ops facade + dedicated More-hub panels on Android+iOS: exceptions, shop-closed, negotiations, manifests, dispatch preview, activity, fleet orders, ledger, replenishment trigger. Portal remains primary for broadcast/payment-bypass/empathy (v1 portal-only). |
| P1-02 | RETAILER | Mobile catalog/checkout thinner than desktop | **Closed** — checkout unified + card/cash; mobile API layer: `searchSuppliers` + profile PUT; backend profile DTO emits `id`/`company`/`status`. **Catalog UI depth (2026-06-05):** Android+iOS catalog browse chips (Categories \| All products \| Suppliers), flat product grid + supplier/category/name search; My Suppliers connect-vendor sheet (`GET /v1/catalog/suppliers/search` + add/remove `/v1/retailer/suppliers/{id}/*`) parity with desktop procurement. **Receiving window (2026-06-05):** desktop Settings + Android `AccountProfileScreen` + iOS `AccountProfileView` edit `receiving_window_open/close`; backend `Retailers.ReceivingWindowOpen/Close` + `GET/PUT /v1/retailer/profile`; mobile registration path persists windows. Portal-only v1 deferrals: broadcast, payment-bypass, empathy adoption. |
| P1-03 | WAREHOUSE | Portal depth vs Pegasus reference | **Closed** — backend + `packages/api-client` (9 routes). Portal: `lib/warehouse-api.ts` + `lib/warehouse-ops.ts`; order detail mutation panel at `/orders/[id]`; orders list drill-down. Native: `WarehouseOperationsRepository` / `WarehouseOperationsService`; order detail mutations (Android+iOS); transfer action panel on More hub (Android+iOS). SSMR: `PX_E2E_WAREHOUSE_ORDER_MUTATION_OK`, `PX_E2E_WAREHOUSE_TRANSFER_ACTIONS_OK`. |
| P1-04 | FACTORY | Inter-hub transfers in-memory L1 | **Closed** — `FactoryInternalTransfers` Spanner table + `manifest.Store` batch commit; `HandleTransferTransition` uses `apply()` |
| P1-05 | PAYLOAD | Expo used `/v1/supplier/manifests/*`; tablet native mixed; umbrella SSMR only | **Closed** — Expo + `payload-app-android` + `payload-app-ios` canonical `/v1/payloader/manifests/*` (repository layer only; UI freeze). SSMR: `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK`, `PX_E2E_PAYLOAD_REASSIGN_OK`, `PX_E2E_PAYLOAD_DRIVER_GATE_OK`, `PX_E2E_PAYLOAD_DEVICE_TOKEN_OK` under umbrella `PX_E2E_PAYLOAD_OK`. |

## Wave B — realtime fanout (2026-06-05)

| ID | Item | Status |
|----|------|--------|
| WB-01 | Kafka dispatcher: `ROUTE_REORDERED`, `ROUTE_CREATED`, `MISSING_ITEMS_REPORTED`, `SPLIT_PAYMENT_CREATED` | **Closed** — `kafka/notification_dispatcher.go` |
| WB-02 | Kafka dispatcher: `DRIVER_AVAILABILITY_CHANGED` (supplier + driver + home-node warehouse/factory) | **Closed** |
| WB-03 | Kafka dispatcher: `AI_RECOMMENDATION_*`, `DELIVERY_SESSION_UPDATED`, `FACTORY_CREATED`, `SUPPLIER_CREATED` | **Closed** |
| WB-04 | Pegasus-scale dispatcher parity (~90 handlers) | **Closed** — `notification_dispatcher_parity.go` routes order/finance/transfer/supply/lock/replenishment/payload/pre-order/import/optimization parity + explicit telemetry/sync/command/platform handlers; unknown types no-op |
| WB-05 | Factory inter-hub transfer Spanner persistence (P1-04) | **Closed** — `schema/spanner.ddl` `FactoryInternalTransfers`; hydrate/seed via `factory/repository_spanner.go`; snapshot includes transfers |

## P2 — intentional (document only)

| ID | Item |
|----|------|
| P2-01 | Full Pegasus supplier-portal ~59 routes |
| P2-02 | Rust optimizer sidecar |
| P2-03 | Payme/Click production SDK depth (GlobalPay scaffold acceptable for v1) |
| P2-04 | Multi-supplier beyond `MAX_SUPPLIERS` policy |

## Verification

```bash
cd pegasusX
bash scripts/parity/role_row_contract_check_full.sh   # role-row-contract-full-ok
make parity-contract-full
make gap-hunter-gate                                  # gap-hunter-gate-ok
make validate-launch-readiness                        # launch-readiness-ok
make test-ssmr-infra
```

**2026-06-05 gates:** `parity-contract-full`, `gap-hunter-gate`, `validate-launch-readiness` — pass. P1-01 supplier native: iOS `xcodebuild` + Android `:app:compileDebugKotlin` — pass (foojay toolchain plugin removed from pegasusX `settings.gradle.kts` — local JDK; `plugins-artifacts.gradle.org` DNS blocked auto-download).

**QA prep:** `make px12-preflight` + manual runbook [`docs/qa/PX12_MANUAL_QA_RUNBOOK.md`](qa/PX12_MANUAL_QA_RUNBOOK.md); Boss sign-off sheet [`docs/qa/PX12_ROLE_ROW_QA.md`](qa/PX12_ROLE_ROW_QA.md).

P0 rows moved to **Resolved** (2026-06-05). P1 integration layer **closed** across all role rows.
