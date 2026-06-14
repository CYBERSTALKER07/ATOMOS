# pegasusX SUPPLIER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Reference:** pegasus `admin-portal` (read-only)  
**Parent plan:** `VEGETABLE_PLAN.md` §2.1  
**Last updated:** 2026-06-14

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase 0 — Onboarding gate integrity (P0)

| ID | Feature | Backend | supplier-portal | Android | iOS | Status | Proof |
|----|---------|---------|---------------|---------|-----|--------|-------|
| SP0-01 | JWT `is_registered` claim | `auth/claims.go`, `auth/jwt.go` | `middleware.ts` reads claim | cookie via login | cookie via login | **E2E_SSMR_GREEN** | unit tests |
| SP0-02 | Business setup contract | `supplier/setup.go` `CompleteBusinessSetup` | `/setup/business` | `setupBusiness` API | `setupBusiness` API | **E2E_SSMR_GREEN** | `setup_test.go` |
| SP0-03 | Register → business redirect | `Register` sets `is_registered=false` for minimal wizard | `/auth/register` → `/setup/business` | register flow | register flow | **WIRED** | manual + tests |
| SP0-04 | Login next-step routing | `Login` returns `is_registered` + correct `next_step` | login page | login | login | **WIRED** | `setup_test.go` |
| SP0-05 | Shared types | — | — | — | — | **WIRED** | `packages/types`, `api-client` |

**Exit:** New supplier can register → complete business → billing gate → portal without 400 on business setup.

---

## Phase 1 — Network topology CRUD (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP1-01 | Topology editor (`PUT /v1/supplier/topology`) | exists + JSON wire tags | `/topology` editor wired | Android + iOS edit/save | **WIRED** |
| SP1-02 | Warehouse/factory create UI | `PUT topology` | add/remove nodes in topology editor | native edit forms | **WIRED** |
| SP1-03 | Delivery zones + supply lanes CRUD | partial GET (derived from topology) | read-only + topology drives coverage | read-only + handoff | **WIRED** (topology-owned) |
| SP1-04 | `getSupplierInventory()` in api-client | GET exists | raw fetch today (optional migrate) | — | **WIRED** |

**Dependency:** Phase 0 complete. **Blocks:** warehouse/factory ops for new tenants.

**Exit:** Supplier can add/edit warehouse + factory nodes on portal and native; `GET /v1/supplier/topology` returns snake_case JSON aligned with shared types.

---

## Phase 2 — Dispatch & manifest oversight (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP2-01 | Dispatch preview + AUTO execute | `supplierroutes` | `/dispatch` partial | preview screens | **WIRED** |
| SP2-02 | Manifest lifecycle actions | `payloaderroutes` supplier manifests | `/manifests/[id]` start-loading / inject / seal | Android + iOS detail actions | **WIRED** |
| SP2-03 | Manifest exceptions inbox | `GET /v1/supplier/manifest-exceptions` | `/manifest-exceptions` | Android + iOS gate screens | **WIRED** |
| SP2-04 | Fleet live map oversight | `GET /v1/supplier/fleet/live-map` | dashboard + `/fleet` | MapLibre/MapKit | **E2E_SSMR_GREEN** |

**Cross-sync:** warehouse dispatch execute, payload seal, driver assign detection.

**Exit:** Supplier can inspect manifest detail, run loading lifecycle actions, and triage manifest gate exceptions on portal and native.

---

## Phase 3 — Staff, org, treasury depth (P1)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP3-01 | Org member lifecycle (edit/deactivate) | PATCH/DELETE `/v1/supplier/org/members/{id}` | `/org-fleet` edit + deactivate | Android + iOS deactivate | **WIRED** |
| SP3-02 | Returns resolution | orders `filter=RETURNS` | `/returns` order handoff + context | — | **WIRED** |
| SP3-03 | Payment config / gateway vault | billing setup | `/setup/billing` | billing screen | **WIRED** |
| SP3-04 | Notification inbox recipient fix | `notifications.RecipientIDFromClaims` | `useNotifications.ts` (unchanged path) | — | **WIRED** |

---

## Phase 4 — Intelligence & catalog depth (P2 for single-tenant)

| ID | Feature | Backend | Portal | Native | Status |
|----|---------|---------|--------|--------|--------|
| SP4-01 | Analytics / demand forecast | `/v1/supplier/analytics/*` | `/analytics` + `/analytics/demand` | — | **E2E_SSMR_GREEN** |
| SP4-02 | Retailer pricing overrides | pegasus-only | — | — | **DEFERRED** |
| SP4-03 | Inventory CSV import | `POST /v1/supplier/inventory/import` | `/inventory/import` | — | **E2E_SSMR_GREEN** |
| SP4-04 | Products vs catalog canonical | `/v1/catalog/products` | `/catalog` | — | **WIRED** |

---

## Intentional single-tenant deltas (do not close)

- Platform control center, DLQ, KYC, country config (pegasus multi-tenant admin)
- Full ~59-route pegasus admin parity (portal ~36 routes is v1 target)
- Broadcast / payment-bypass / empathy: portal-primary; native handoff OK

---

## Verification (supplier row)

```bash
cd pegasusX/apps/backend-go && go test ./supplier/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_ORDER_OK umbrella
cd pegasusX && make parity-contract-full
```

---

## Next execution batch (Boss-approved start: supplier)

1. ~~Phase 0 onboarding~~ — **done**
2. ~~Phase 1 topology editor~~ — **done**
3. ~~Phase 2 manifest actions + manifest-exceptions~~ — **done**
4. ~~Phase 3 staff/org lifecycle + notification inbox fix~~ — **done this session**
5. ~~Phase 4 intelligence & catalog depth (analytics, CSV import)~~ — **E2E_SSMR_GREEN** (`PX_E2E_SUPPLIER_ANALYTICS_OK`, `PX_E2E_SUPPLIER_INVENTORY_IMPORT_OK`)
6. ~~SSMR / parity verification~~ — **green** (`make test-ssmr-infra`, `make parity-contract-full`)
7. **Cross-role next** — factory/warehouse analytics native depth, or Boss-picked role row per `VEGETABLE_PLAN.md` §3 (warehouse WH1–WH3 **WIRED**; notification inbox persistence **WIRED** — `PX_E2E_NOTIFICATION_INBOX_OK`)

---

## Cross-role — Notification inbox persistence (WIRED)

| ID | Feature | Backend | Portal / clients | Status |
|----|---------|---------|------------------|--------|
| NI-01 | Dispatcher → Spanner inbox | `kafka/notification_inbox.go` `persistInbox`; `ShouldPersistInboxEvent` skips telemetry | — | **WIRED** |
| NI-02 | Supplier inbox routes | `supplierroutes` GET/POST `/v1/user/notifications` | `useNotifications.ts` | **WIRED** |
| NI-03 | Retailer / driver / payload read paths | existing `retailerroutes` / `driverroutes` / `payloaderroutes` | native inbox screens | **WIRED** (read; rows now populate) |
| NI-04 | SSMR | `runNotificationInboxE2E` polls supplier + retailer inbox after order flow | — | **E2E_SSMR_GREEN** (`PX_E2E_NOTIFICATION_INBOX_OK`) |
