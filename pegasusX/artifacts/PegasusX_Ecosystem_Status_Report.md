# PegasusX Ecosystem Status Report

**Audit date:** 2026-07-31  
**Closure pass:** 2026-08-01 (no-mocks + SSMR marker proof)  
**Ops refresh:** 2026-08-02 (`nomock4`, DNS/TLS Active, Firebase clients in-tree)  
**Method:** Code-backed. Docs are hints only.  
**Inventory SoT:** `docs/ECOSYSTEM_FEATURES_BY_ROLE.md`  
**Intentional divergences:** `context/parity-ledger.md`  
**Cloud checklist:** `artifacts/GAP_CLOSURE_CLOUD_CHECKLIST_2026-07-31.md`

---

## Executive verdict

Core commerce/logistics spine and role-row clients are **WIRED_E2E** for the vast majority of catalogued features. SSMR cloud marker proof is **green** (2026-08-01). Ops refresh 2026-08-02: image **`ssmr-gap-closure-nomock4`**, ManagedCert **Active** on `https://api-ssmr.pegasusx.app`, Firebase iOS/Android client configs applied. Remaining ops: **Global Pay SUCCESS** merchant password (+ SMS/SHA-1 for real OTP). Negotiations + Soliq OFD stay product-deferred.

| Severity | Finding | Status after closure |
|----------|---------|----------------------|
| **P0** Shop-closed cancel without inventory release | **FIXED** — `ReleaseReservationsInTxn` on all CANCELLED/RETURN paths |
| **P1** CLAIM_* dispatcher no-op | **FIXED** — fans via `handleDriverEdgeEvent` + inbox formatters |
| **P1** gen-contracts strict (LogisticsException) | **FIXED** — gate green |
| **P1** Retailer credit mobile | **FIXED** — Android + iOS |
| **P2** Supplier finance mobile | **FIXED** — accept/issue/credit-profiles |
| **P2** Driver rescue/earnings/scan-qr; WH rescue URLs | **FIXED** |
| **P2** BillingTierWorker / Analytics dummy | **FIXED** — worker wired; dummy removed |
| **P2** Factory staff write / exception resolve | **FIXED** — backend + 3 clients |
| **Ops** Cloud staging | **PARTIAL** — image `nomock4` live; DNS/TLS **Active**; Firebase clients applied; e2e/marker **PASS**; GP SUCCESS still open |

**Gates (post-closure):**

| Gate | Result |
|------|--------|
| `role_row_contract_check(_full).sh` | PASS (baseline) |
| `gen_contracts_gate.sh` | **PASS** |
| `go test ./order ./kafka ./notifications ./events` | **PASS** |
| `ssmr_ecosystem_marker_gate.sh` | **PASS** (`ssmr-e2e.log`, 2026-08-01) |

---

## Role → client matrix

| Role | Desktop/Portal | Android | iOS |
|------|----------------|---------|-----|
| Supplier (`ADMIN`) | `supplier-portal` | `supplier-app-android` | `supplier-app-ios` |
| Retailer | `retailer-app-desktop` | `retailer-app-android` | `retailer-app-ios` |
| Warehouse | `warehouse-portal` | `warehouse-app-android` | `warehouse-app-ios` |
| Factory | `factory-portal` | `factory-app-android` | `factory-app-ios` |
| Driver | — | `driver-app-android` | `driver-app-ios` |
| Payload | `payload-terminal` | `payload-app-android` | `payload-app-ios` |

---

## Spine (post-fix)

| ID | Hop | Status |
|----|-----|--------|
| X-1 | Checkout → reserve → create | **WIRED_E2E** |
| X-2 | Dispatch → LOADED + manifest | **WIRED_E2E** (vet deprecated but dispatch is path) |
| X-3 | Seal → depart → IN_TRANSIT | **WIRED_E2E** |
| X-4 | Arrive → scan-qr → cash → fiscal → COMPLETED | **WIRED_E2E** (clients on `scan-qr`) |
| X-5 | Claim → approve → chargeback + WS | **WIRED_E2E** |
| X-6 | Cancel/reject/shop-closed release | **WIRED_E2E** |

---

## Role scorecards (post-fix)

| Role | WIRED_E2E | Notes |
|------|-----------|-------|
| Retailer | 9/9 | Credit profile on all three |
| Supplier | 12/13 | Negotiations **product-deferred**; finance mutations wired |
| Warehouse | 8/8 | Reverse-logistics + exceptions + claims + rescues on all three; control tower uses JWT supplier |
| Factory | 7/7 | Staff create + exception resolve |
| Payload | 7/7 | Unchanged strongest row |
| Driver | 9/9 | Offline flush orchestrator wired (Room/SwiftData + Sync Queue + BGTask) |

---

## Tech inventory (post-fix)

| Tech | Judgment |
|------|----------|
| Outbox → Kafka → consumers | WIRED_CODE (+ BillingTierWorker) |
| AnalyticsStreamProcessor | **Removed** (was dummy) |
| KAFKA_TOPIC_SPATIAL | **Removed** from env examples |
| WS hubs + FCM | WIRED_CODE / ENV_DEPENDENT |
| Fiscal / Soliq | Pegasus branded HTML/PDF receipts Wired (`FISCAL_PROVIDER=PEGASUS`); Soliq OFD deferred |
| Cloud SSMR | Live pods `nomock4`; DNS/TLS Active; Firebase clients in-tree; GP SUCCESS pending |

---

## Remaining deferred (parity-ledger)

- Quantity negotiations (product)
- Soliq OFD legal tax receipts (platform Pegasus branded receipts already ship)
- GP SUCCESS merchant password (cash fallback proven; DNS/ManagedCert Active)

---

## Living ROLE_ROW

See [`docs/ROLE_ROW_PARITY_MATRIX.md`](../docs/ROLE_ROW_PARITY_MATRIX.md) (rebuilt 2026-07-31).
