# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-08-05 (Client Parity Closure)  
**SoT for feature inventory:** [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)  
**Divergences:** [`../context/parity-ledger.md`](../context/parity-ledger.md)  
**Full status report:** [`../artifacts/PegasusX_Ecosystem_Status_Report.md`](../artifacts/PegasusX_Ecosystem_Status_Report.md)  
**Client walks:** [`../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md`](../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md)  
**Retail OS gate:** [`RETAILER_OS_PRODUCTION_GATE.md`](./RETAILER_OS_PRODUCTION_GATE.md) · [`RETAILER_OS_E2E_MATRIX.md`](./RETAILER_OS_E2E_MATRIX.md)

## Summary

| Role | Clients | Backend | UI parity | E2E / notes |
|------|---------|---------|-----------|-------------|
| SUPPLIER | portal, Android, iOS | supplierroutes + finance/claims/pulse + return-policy | **Wired** | Return-policy settings on all 3; negotiations product-deferred |
| RETAILER | desktop, Android, iOS | retailerroutes, order, payment, credit + Retail OS packs 0–6 | **Wired** | AUTHORIZE_BYPASS photo wired; CT pulse honest empty/live |
| DRIVER | Android, iOS | driverroutes, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave |
| WAREHOUSE | portal, Android, iOS | warehouseroutes + return-policy | **Wired** | Returns & reverse SLA settings on all 3 |
| FACTORY | portal, Android, iOS | factoryroutes | **Wired** | Staff POST + exception resolve; empty forecast chart unmounted |
| PAYLOAD | terminal, Android, iOS | payloaderroutes | **Wired** | Seal/inject/reassign/returns |

## Cross-role spine status

| Hop | Status |
|-----|--------|
| Checkout → reserve → create | Wired |
| Dispatch → manifest LOADED | Wired |
| Seal → depart IN_TRANSIT | Wired |
| scan-qr → collect-cash → fiscal → COMPLETED | Wired |
| Claim file → approve → chargeback + WS | Wired |
| Shop-closed cancel inventory release | Wired (2026-07-31) |

## Platform / realtime

| Surface | Status |
|---------|--------|
| Outbox → Kafka → notification dispatcher | Wired |
| CLAIM_* fanout | Wired |
| FCM / device-token | Env-dependent |
| BillingTierWorker on ORDER_FINALIZED | Wired |
| gen-contracts strict | Green |

## Retail OS capability packs (retailer role-row)

| Pack | Backend | Desktop | Android | iOS |
|------|---------|---------|---------|-----|
| CORE | Wired | Wired | Wired | Wired |
| TEAM | Wired | Wired | Wired | Wired |
| LOCATIONS | Wired | Wired | Wired | Wired |
| STORE_STOCK | Wired | Wired | Wired | Wired |
| POS | Wired | Wired | Wired | Wired |
| SHIFTS | Wired | Wired | Wired | Wired |
| SECTIONS | Wired | Wired | Wired | Wired |
| REPORTS_PRO | Wired | Wired | Wired | Wired |
| CUSTOMER_ASSIST | Wired | Wired | Wired | Wired |
| CT pulse (ops digest) | Wired | Wired | Wired | Wired |

## Deferred (explicit — product / ops flags)

| Item | Owner | Exit criteria |
|------|-------|----------------|
| Quantity negotiations | Product | `QUANTITY_NEGOTIATION_ENABLED=true` + client un-stub |
| Soliq OFD (state tax; Pegasus branded receipts Wired) | Tax / ops | `FISCAL_PROVIDER=SOLIQ` + sandbox readiness |
| Offline POS | Product | Offline Retail OS flag; online-required v1 until then |
| GP card SUCCESS + Firebase SMS | Cloud ops | Merchant password + SHA-1 / Blaze |
| Interactive UI walk PASS cells | Operator | Flip READY_FOR_WALK → PASS in client signoff |

Historical snapshot (pre-delete, not current truth): [`../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md`](../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md)
