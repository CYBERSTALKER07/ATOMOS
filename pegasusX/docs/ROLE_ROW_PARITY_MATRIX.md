# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-08-13 (G7: factory SLA board + admin OutboxDeadLetters; G1–G6 honesty residuals)  
**Primary inventory:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md)  
**Narrative secondary:** [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)  
**Optimizer + maps runtime:** [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md)  
**Partner API:** [`PARTNER_API.md`](./PARTNER_API.md) · [`../contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml) · **JWT core OpenAPI:** [`JWT_CORE_OPENAPI.md`](./JWT_CORE_OPENAPI.md) · [`../contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml)  
**Divergences:** [`../context/parity-ledger.md`](../context/parity-ledger.md)  
**Doc map:** [`DOCS_SOURCE_OF_TRUTH.md`](./DOCS_SOURCE_OF_TRUTH.md)  
**Client walks:** [`../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md`](../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md)  
**Retail OS gate:** [`RETAILER_OS_PRODUCTION_GATE.md`](./RETAILER_OS_PRODUCTION_GATE.md) · [`RETAILER_OS_E2E_MATRIX.md`](./RETAILER_OS_E2E_MATRIX.md)

## Summary

| Role | Clients | Backend | UI parity | E2E / notes |
|------|---------|---------|-----------|-------------|
| SUPPLIER | portal (Tauri desktop), Android, iOS | supplierroutes + finance/claims/pulse + return-policy + planning | **Wired** | Desktop = `supplier-portal` Tauri; `/planning` web; negotiations product-deferred |
| RETAILER | desktop, Android, iOS | retailerroutes, order, payment, credit + Retail OS packs 0–6 | **Wired** | HQ / Credit-AR / CT on all 3; AUTHORIZE_BYPASS photo wired |
| DRIVER | Android, iOS | driverroutes, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave; §8.8 kit |
| WAREHOUSE | portal, Android, iOS | warehouseroutes + WMS + return-policy | **Wired** | Portal: bins/pick-waves/cycle/cold/labor; mobile pick/cycle under Transfer Actions; CT portal-primary |
| FACTORY | portal, Android, iOS | factoryroutes | **Wired** | Staff POST + exception resolve; loading bay ↔ payload Class A; **G7 SLA board + badges on portal** (mobile shows supply list `sla_*` when present) |
| PAYLOAD | Expo terminal + Android + iOS | payloaderoutes + factory manifests bridge | **Wired** | Seal/inject/reassign/returns; factory loading-bay APIs on all three |
| PLATFORM_ADMIN | `admin-portal` (web only) | platformadmin + featureflags + partner admin | **Wired** | Login+MFA; tenants/flags dual-control/audit/match/partner; **ops outbox + Spanner dead-letters**; no mobile by design |

## Cross-role spine status

| Hop | Status |
|-----|--------|
| Checkout → reserve → create | Wired |
| Dispatch → manifest LOADED | Wired |
| Seal → depart IN_TRANSIT | Wired |
| scan-qr → collect-cash → fiscal → COMPLETED | Wired |
| Claim file → approve → chargeback + WS | Wired |
| Shop-closed cancel inventory release | Wired (2026-07-31) |
| Factory loading-bay ↔ payload | Wired (W2) |

## Platform / realtime

| Surface | Status |
|---------|--------|
| Outbox → Kafka → notification dispatcher | Wired |
| Twin consumer (`void-digital-twin`) | Wired (W1) |
| CLAIM_* fanout | Wired |
| FCM / device-token | Env-dependent (owner SMS/APNs residual) |
| Partner webhooks / AS2 / EDI-lite | Wired (cert residual) |

## Legend

**Wired** = Class A path exists in code on the clients listed. Does **not** mean owner keys, legal OFD, or prod optimizer pods are live — see [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md).
