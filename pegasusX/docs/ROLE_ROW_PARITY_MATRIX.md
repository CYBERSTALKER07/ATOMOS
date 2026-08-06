# pegasusX Role-Row Parity Matrix

**Last updated:** 2026-08-06 (Partner Integration Wave 2C GS1 labels)  
**SoT for feature inventory:** [`ECOSYSTEM_FEATURES_BY_ROLE.md`](./ECOSYSTEM_FEATURES_BY_ROLE.md)  
**Optimizer + maps runtime:** [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md)  
**Partner API:** [`PARTNER_API.md`](./PARTNER_API.md) · [`../contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml) · **JWT core OpenAPI:** [`JWT_CORE_OPENAPI.md`](./JWT_CORE_OPENAPI.md) · [`../contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml)  
**Divergences:** [`../context/parity-ledger.md`](../context/parity-ledger.md)  
**Full status report:** [`../artifacts/PegasusX_Ecosystem_Status_Report.md`](../artifacts/PegasusX_Ecosystem_Status_Report.md)  
**Client walks:** [`../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md`](../artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md)  
**Retail OS gate:** [`RETAILER_OS_PRODUCTION_GATE.md`](./RETAILER_OS_PRODUCTION_GATE.md) · [`RETAILER_OS_E2E_MATRIX.md`](./RETAILER_OS_E2E_MATRIX.md)

## Summary

| Role | Clients | Backend | UI parity | E2E / notes |
|------|---------|---------|-----------|-------------|
| SUPPLIER | portal, Android, iOS | supplierroutes + finance/claims/pulse + return-policy | **Wired** | Return-policy settings on all 3; negotiations product-deferred |
| RETAILER | desktop, Android, iOS | retailerroutes, order, payment, credit + Retail OS packs 0–6 | **Wired** | AUTHORIZE_BYPASS photo wired; CT pulse honest empty/live |
| DRIVER | Android, iOS | driverroutes, delivery, telemetry | **Wired** | P0-4 offline classifier fixed; PoD required for credit leave; §8.8 kit + capture-time coords |
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
| Partner API keys + `/partner/v1` | **Wired** (apply Spanner migration for cloud) |
| Outbound webhooks (HMAC) | **Wired** (Kafka enqueue + delivery worker + list/deactivate/replay) |
| Partner bulk export + optional SFTP | **Wired** (`exports:read`, `20260806_partner_exports.ddl`, flags) |
| Partner 1C journals export | **Wired** (`resource=journals`, CSV/JSON/XML + configurable CoA — [`PARTNER_JOURNALS_1C.md`](./PARTNER_JOURNALS_1C.md)) |
| Partner EDI-lite (ORDERS/ORDRSP/DESADV/INVOIC) | **Wired** (`20260806_partner_edi.ddl`, SFTP/local root; DESADV SSCC CPS/PAC/GIN; AS2 transport Wired — not Drummond) |
| Partner AS2 transport | **Wired** (`20260806_partner_as2.ddl`, sync MDN; not Drummond-certified — [`PARTNER_AS2.md`](./PARTNER_AS2.md)) |
| GS1 GLN + SSCC + ZPL labels | **Wired** (`20260806_gs1_labels.ddl`, `docs/GS1_LABELS.md`; DESADV GIN+BJ from ship units) |
| Supplier portal Integrations | **Wired** (`/settings/integrations` — keys/webhooks/exports/SFTP/EDI) |
| OpenAPI partner contract | [`partner.openapi.yaml`](../contracts/partner.openapi.yaml) |
| OpenAPI JWT core (~45 ops) | **Wired** — [`jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml); `make jwt-openapi-gate`; residual = full catalog + SDK replace of ApiClient |
| WMS bins/lots/FEFO (§8.7 Wave 1A) | **Wired** behind `WMS_LOTS_ENABLED` — [`WMS_LOTS_FEFO.md`](./WMS_LOTS_FEFO.md); portal + Android/iOS putaway |
| WMS pick waves + seal gate (§8.7 Wave 1B) | **Wired** behind `WMS_PICK_WAVES_ENABLED` — [`WMS_PICK_WAVES.md`](./WMS_PICK_WAVES.md); portal `/pick-waves` + Android/iOS |
| WMS cycle counts (§8.7 Wave 1C/PR-4) | **Wired** behind `WMS_CYCLE_COUNTS_ENABLED` — apply-on-approve + ABC + accuracy; portal + Android/iOS |
| WMS S-shape / LIFO + soft seal (PR-5) | **Wired** behind `WMS_PICK_SSHAPE_ENABLED` / `WMS_SEAL_SOFT_WARN` |
| WMS cold chain (PR-6) | **Wired** behind `WMS_COLD_CHAIN_ENABLED` — [`WMS_COLD_CHAIN.md`](./WMS_COLD_CHAIN.md) |
| WMS Gate 4 ops / harden (PR-7) | Reconcile endpoint + [`WMS_GATE4_OPS.md`](./WMS_GATE4_OPS.md); scan UX residual |
| AR dunning / DelinquencyCount / CREDIT_HOLD | **Wired** behind `AR_DUNNING_ENABLED` |
| Off-app SMS/email dunning | Deferred (FCM + inbox only) |
| Forecast accuracy (WAPE/bias/TS) | **Wired** behind `FORECAST_ACCURACY_ENABLED` (supplier portal) |
| Croston / SES / Holt–Winters baselines | **Wired** behind `FORECAST_ALGO_ENABLED` |
| Safety stock (service level / lead σ) | **Wired** behind `SAFETY_STOCK_V2_ENABLED` (supplier portal knobs; Android/iOS read policy; retailer reorder batch shares SS helper) |
| Shadow auto-order (§8.3) | **Wired** — `off\|shadow\|draft\|place`; inventory `(R,s,S)`; desktop/Android/iOS; `AUTO_ORDER_INVENTORY_GROUNDED` diverts synthesis `/2` |

## Maps / route geometry (world-scale)

| Surface | Status |
|---------|--------|
| Backend geometry (Google Routes → OSRM → dense) | **Wired** (`ROUTING_PROVIDER=auto`) |
| Persist on seal / driver geometry GET | Wired (stored polyline preferred) |
| Supplier / warehouse live-map + dispatch preview | Wired (existing clients) |
| Driver map polyline | Wired |
| Retailer tracking planned route overlay | **Wired** (`route_geometry` on tracking) |
| Factory fleet live-map (pins) | **Wired** portal + Android + iOS |
| Payload inbound truck lat/lng | **Wired** (thin; terminal + Android) |
| OSRM PVC regional extract | Ops optional fallback only |
| Factory polyline columns | Deferred (pins-first) |

## Dispatch optimizer (OR-Tools)

| Surface | Status |
|---------|--------|
| Code path (`optimizerclient` → `optimizer-core`) | **Wired** (supplier + warehouse dispatch) |
| Constraint fields + multi-depot + OSRM `/table` matrix | **Wired** (§8.5; haversine fallback) |
| Local compose (`docker-compose.ssmr.yml`) | OR-Tools available when service up |
| SSMR GKE overlay | **Manifest included, `replicas: 0`** until AR image; heuristic until then |
| Prod GKE overlay | Manifest present, **`replicas: 0`**, no real AR image |
| Client apps call solver | **No** — `optimizer_source` on API only |
| Exit criteria for cloud OR-Tools | Publish image + replicas ≥ 1 + `"optimizer_source":"optimizer"` |

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
| optimizer-core live in SSMR/prod | Cloud ops | AR image + replicas ≥ 1 (see runtime SoT) |
| Interactive UI walk PASS cells | Operator | Flip READY_FOR_WALK → PASS in client signoff |

Historical snapshot (pre-delete, not current truth): [`../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md`](../artifacts/ROLE_ROW_PARITY_MATRIX_SNAPSHOT_2026-07-07.md)
