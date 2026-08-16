# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-08-14 (ecosystem flexibility: factory solver dispatch, topology hybrid, country catalog, loyalty live, Payload/Load; **not cloud, not store**)  
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
| SUPPLIER | portal (Tauri desktop), Android, iOS | supplierroutes + finance/claims/pulse + return-policy + planning | **Wired** | Desktop = `supplier-portal` Tauri; `/planning` web; CT scored+playbooks typed native; payout-policy thin UI, rail `no_live_rail`; negotiations product-deferred |
| RETAILER | desktop, Android, iOS | retailerroutes, order, payment, credit + Retail OS packs 0–6 | **Wired** | HQ / Credit-AR / CT on all 3; CT tiles navigate (P13-E); AUTHORIZE_BYPASS photo wired |
| DRIVER | Android, iOS | driverroutes, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave; §8.8 kit |
| WAREHOUSE | portal, Android, iOS | warehouseroutes + WMS + return-policy | **Wired** | Portal: bins/pick-waves/cycle/cold/labor; mobile pick/cycle under Transfer Actions; Control Tower typed scored list (P13-C) + portal |
| FACTORY | portal, Android, iOS | factoryroutes | **Wired** | Loading-bay start/seal **REAL** ↔ payload Class A; factory **Payload/Load** factory-plane only. **G7 SLA board + badges on portal** (mobile shows supply list `sla_*` when present). `POST /v1/factory/dispatch` **live Spanner** = warehouse solver class → `FactoryTruckManifests` only (empty no invent); nil-Spanner tests still `pick_n_created_v1`. Staff POST + exception resolve are **Class A persist + outbox** (P3); resolve lookup **Spanner-first** (P9-B). Staff `PasswordHash` is bcrypt/invite (P9-A), not `unset`. |
| PAYLOAD | Expo terminal + Android + iOS | payloaderoutes + factory manifests bridge | **Wired** | Seal/inject/reassign/returns; **seal-all** on terminal+Android+iOS (P13-A); factory loading-bay APIs on all three. Capacity 410. |
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

**Wired = happy path, not every FEATURES row.** A Wired role row does not make saved cards, `/v1/ai/predictions` alias, inventory audit, request-cancel, or negotiations live. Cards/AI-alias/audit are **GONE** 410 (P1). Factory dispatch **live Spanner** is warehouse solver class → `FactoryTruckManifests` only; nil-Spanner tests still `pick_n_created_v1`. Loyalty is live `{enrolled:false}` (never fake Bronze). P15 not cloud. P16 not store. See [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) and [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md).
