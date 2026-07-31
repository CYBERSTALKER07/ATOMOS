# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-07-31 (gap-closure)  
**SoT for feature inventory:** [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)  
**Divergences:** [`../context/parity-ledger.md`](../context/parity-ledger.md)  
**Full status report:** [`../artifacts/PegasusX_Ecosystem_Status_Report.md`](../artifacts/PegasusX_Ecosystem_Status_Report.md)

## Summary

| Role | Clients | Backend | UI parity | E2E / notes |
|------|---------|---------|-----------|-------------|
| SUPPLIER | portal, Android, iOS | supplierroutes + finance/claims/pulse | **Wired** | Negotiations product-deferred |
| RETAILER | desktop, Android, iOS | retailerroutes, order, payment, credit | **Wired** | Credit profile on all three |
| DRIVER | Android, iOS | driverroutes, delivery, telemetry | **Wired** | scan-qr doorstep; rescue+earnings bound |
| WAREHOUSE | portal, Android, iOS | warehouseroutes | **Wired** | Rescue preview/propose; reverse-log portal-deferred |
| FACTORY | portal, Android, iOS | factoryroutes | **Wired** | Staff POST + exception resolve |
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

## Deferred (explicit)

| Item | Owner |
|------|-------|
| Quantity negotiations | Product |
| WH reverse-logistics / exceptions mobile | Ops / mobile |
| Driver offline flush orchestrator | Driver mobile |
| Soliq OFD | Tax / ops |
| SSMR DNS + GP SUCCESS + marker log | Cloud ops |

Historical snapshot (pre-delete, not current truth): [`../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md`](../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md)
