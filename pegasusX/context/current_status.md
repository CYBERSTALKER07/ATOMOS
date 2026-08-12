# PegasusX Migration & Staging Status

> **Re-aligned 2026-08-12** against code. Prefer [`docs/DOCS_SOURCE_OF_TRUTH.md`](../docs/DOCS_SOURCE_OF_TRUTH.md) + [`docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md) when this snapshot conflicts.


*Last Updated: 2026-08-07 (Gate 5 Phase 1 ADR + Theatre #13 FX Wave 2+ order currency picker + Theatre #8 seasonality + Gate-0 CI + FX Wave 1+2 + partner OAuth + CoA + EDI DESADV SSCC + theatre leftovers + billing + P0-8 + §8.5)*

## 1. Code completeness (this closure)

**Gate 5 / §8.10 (2026-08-12 re-verify):** Phase 1 **Wired** (`TenantContext`, `RequireTenant`, `PreferTenantSupplierID`, outbox `SupplierId`, tenant rate limits; seed = bootstrap fallback). Phase 2 **Wired (backend)** ParentOrders + `MULTI_SUPPLIER_CHECKOUT_ENABLED`. Phase 3 **Wired (backend)** GlobalProducts (flag). Residual: seed cleanup, retailer multi-partner UI, marketplace Phases 4–5. Docs: [`MULTI_TENANCY_GATE5_PHASE1.md`](../docs/MULTI_TENANCY_GATE5_PHASE1.md) / [`PHASE2`](../docs/MULTI_TENANCY_GATE5_PHASE2.md) / [`PHASE3`](../docs/MULTI_TENANCY_GATE5_PHASE3.md).

**Theatre #13 FX Wave 2+:** Flag-gated `ORDER_CURRENCY_PICKER_ENABLED` + `ORDER_CURRENCY_ALLOWLIST`; `GET /v1/order/currencies`; Create/UnifiedCheckout stamp allowlisted currency (422 `currency_not_allowed`); desktop/Android/iOS picker when enabled. Marker `PX_E2E_ORDER_CURRENCY_PICKER_OK` / `_SKIPPED`. Residual: AR multi-currency ledger, Airwallex live FX. Docs: [`docs/FX_RATES.md`](../docs/FX_RATES.md).

**Theatre #13 FX Wave 2:** Billing GMV `ConvertMinor` into operating currency (skip on missing rate); settlement authority `operating_currency_total_minor` + `operating_conversion_partial`; portal `/settings/fx-rates` + `GET /v1/supplier/fx-rates`; marker `PX_E2E_FX_SETTLEMENT_CONVERT_OK` / `_SKIPPED`.

**Theatre #8 Seasonality:** `SeasonalTemplateOverrides.Multiplier` persisted (clamp [0.5, 2.5]; inherit builtin; else 1.2) — kills read-time 1.2 hardcode. Shared `seasonalcore` builtins for planning + replenishment; Spanner override reader on suggested qty. YoY/month estimate drafts via `POST /v1/supplier/planning/seasonal-estimate` + optional `planning-forecast` hook (`FORECAST_SEASONAL_ESTIMATE_ENABLED`, default off; inactive drafts only). Apply `20260806_seasonal_override_multiplier.ddl`. Portal/Android/iOS multiplier field. Marker: `PX_E2E_SEASONAL_OVERRIDE_OK` / `_SKIPPED`. Residual: full HW annual library / weather/POS.

**Gate-0 CI:** Root workflows `pegasusx-ci.yml` (backend race/golangci/gitleaks/govulncheck + desktop) and `pegasusx-native-mobile-build.yml` (all 12 Android/iOS apps). Scripts: `scripts/ci_android_apps.sh`, `scripts/ci_ios_apps.sh`. Config: `.golangci.yml`, `.gitleaks.toml`. Docs: [`docs/GATE0_CI.md`](../docs/GATE0_CI.md). Fixed driver iOS `mobile-ios-kit` package path. Residual: ESLint a11y hard gate; SwiftLint/detekt.

**Theatre #13 Multi-currency FX (Wave 1+2+):** `FxRates` + `fxrates.ConvertMinor` (fail closed); payment `currency_mismatch`; bootstrap UZS identity (+ optional `FX_SEED_USD_UZS_SCALED`); admin `GET/PUT /v1/admin/fx-rates` + supplier GET + portal `/settings/fx-rates`; billing GMV ConvertMinor → operating; settlement `operating_currency_total_minor`; flag-gated order currency picker (`ORDER_CURRENCY_PICKER_*`, `GET /v1/order/currencies`, desktop/Android/iOS). Apply `20260806_fx_rates.ddl`. Docs: [`docs/FX_RATES.md`](../docs/FX_RATES.md). Markers: `PX_E2E_FX_RATE_SEEDED_OK` / `_SKIPPED`, `PX_E2E_CURRENCY_MISMATCH_DENIED` / `_SKIPPED`, `PX_E2E_FX_SETTLEMENT_CONVERT_OK` / `_SKIPPED`, `PX_E2E_ORDER_CURRENCY_PICKER_OK` / `_SKIPPED`. Residual: multi-currency AR ledger, Airwallex FX.

**§8.9 OAuth2 client_credentials:** `POST /partner/v1/oauth/token` reuses `PartnerApiKeys` as clients; short-lived HS256 JWT (`token_use=partner_access`); dual-accept with `pxk_` on `/partner/v1/*`; live revoke via key status. Env: `PARTNER_JWT_SECRET` (or derived from `JWT_SECRET`). Marker: `PX_E2E_PARTNER_OAUTH_OK` / `_SKIPPED`.

**§8.9 configurable CoA:** `PartnerCoaMaps` + GET/PUT `/partner/v1/coa` (+ supplier/admin JWT); journals export uses tenant AR/revenue/bank accounts (defaults `62.01`/`90.01`/`51.01`). Portal Integrations CoA fields. Apply `20260806_partner_coa.ddl`. Residual: certified 1C exchange package.

**§8.9 EDI DESADV SSCC:** Outbound DESADV emits CPS/PAC/GIN+BJ (optional GIN+BN GTIN) from `ManifestShipUnits` when present; hydrated in `EdiOutboundWorker.loadSnapshot`. Still open: certified EDIFACT.

**§8.9 AS2 transport:** `POST /partner/v1/as2` + outbound push over EDI-lite bytes; sync MDN; `PartnerAs2Configs`. Apply `20260806_partner_as2.ddl`. Flags: `PARTNER_AS2_ENABLED`, `PARTNER_AS2_INSECURE_PLAIN` (SSMR only). **Not Drummond-certified.** Docs: [`docs/PARTNER_AS2.md`](../docs/PARTNER_AS2.md). Markers: `PX_E2E_PARTNER_AS2_ORDERS_OK` / `_SKIPPED`, `PX_E2E_PARTNER_AS2_ORDRSP_OK` / `_SKIPPED`.

**Theatre leftovers (honesty + promo):** AI confidence gate + touchless `MinConfidenceScore` already Gate-0 **WIRED** (audit/SUBSTANCE_GATE refreshed). Promo sandbox: caller `elasticity` (default 0.5) + `elasticity_used`; closed-loop actuals from `LineItemsJson.promotion_id` (units + line totals), empty promo → zeros. Cold-chain breach auto-raise + band hydrate **wired** (Bluetooth/cumulative minutes residual). i18n / marketplace fee+invoice remain **partial residuals**.

**Billing meter (§2.2):** Schema columns already matched Spanner; **FIXED** Kafka decode of live `ORDER_FINALIZED` (`amount_minor` / nested `total.amount`); non-positive amounts skipped; `Idx_BillingMeterEvents_ByOrderId`. Residual: fee schedule + invoices.

**P0-8 prod overlay:** Images + ManagedCertificate TLS **closed (Gate-0)**. Secrets path **repo-WIRED** — TF shells for all 12 ExternalSecret GSM names; `phase0_sync` stubs unused PSP rails; prod overlay includes SecretStore+ExternalSecret; SSMR `redis-password` aligned. Residual ops: real GSM versions, overlay apply, ManagedCert DNS. Checklist: [`docs/CLOUD_CREDENTIALS_CHECKLIST.md`](../docs/CLOUD_CREDENTIALS_CHECKLIST.md).

**Runtime SoT (optimizer + maps):** [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) — §8.5 constraint fidelity + multi-depot + OSRM `/table` matrix wired; **SSMR overlay patches optimizer-core `replicas: 1`** + image remap; **prod keeps `replicas: 0`** until AR image publish. Live `"optimizer_source":"optimizer"` still needs healthy pod. Geometry: Google Routes → OSRM → dense. Marker: `PX_E2E_OPTIMIZER_CONSTRAINT_OK` / `_SKIPPED`.

**WMS Gate 4 (§8.7):** Waves **1A–1C + PR-4–7** coded — lots/FEFO, pick+seal, cycle apply-on-approve + ABC + accuracy, S-shape/LIFO + soft-warn, cold-chain quarantine, inventory reconcile. Ops apply: [`docs/WMS_GATE4_OPS.md`](../docs/WMS_GATE4_OPS.md). Docs: [`WMS_LOTS_FEFO.md`](../docs/WMS_LOTS_FEFO.md), [`WMS_PICK_WAVES.md`](../docs/WMS_PICK_WAVES.md), [`WMS_CYCLE_COUNTS.md`](../docs/WMS_CYCLE_COUNTS.md), [`WMS_COLD_CHAIN.md`](../docs/WMS_COLD_CHAIN.md), [`WMS_GATE4_HARDENING.md`](../docs/WMS_GATE4_HARDENING.md). Residual: native scan UX, forbid all non-rollup V2 writes, serials.

**Partner Integration (§8.9 Wave 1 + 2A + 2B + 2C + journals + DESADV SSCC + CoA + OAuth + AS2):** Machine API keys, **OAuth2 client_credentials**, `/partner/v1`, HMAC webhooks, bulk export/SFTP, **EDI-lite** (DESADV CPS/PAC/GIN+BJ from ship units), **AS2 transport** (not Drummond — [`docs/PARTNER_AS2.md`](../docs/PARTNER_AS2.md)), **GS1 GLN/SSCC/ZPL**, **1C journals** + **configurable CoA** (`PartnerCoaMaps` — [`docs/PARTNER_JOURNALS_1C.md`](../docs/PARTNER_JOURNALS_1C.md)), portal knobs, OpenAPI. Apply partner DDLs + `20260806_gs1_labels.ddl` + `20260806_partner_coa.ddl` + `20260806_partner_as2.ddl`. Flags: `PARTNER_*`, `GS1_*`. **JWT core OpenAPI WIRED** — [`contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml), `make jwt-openapi-gate` ([`docs/JWT_CORE_OPENAPI.md`](../docs/JWT_CORE_OPENAPI.md)). Residual: certified 1C exchange package, certified EDIFACT, expand JWT OpenAPI coverage + SDK replace of ApiClient.

**Collections substance:** AR dunning step machine + `DelinquencyCount` bump + CREDIT_HOLD auto-freeze + inbox/FCM; **SMS/email/WhatsApp transports code-wired** (`DUNNING_*_PROVIDER`); owner keys + WhatsApp Content SID residual. Flags `AR_INVOICES_ENABLED` + `AR_DUNNING_ENABLED`; marker `PX_E2E_COLLECTIONS_DUNNING_OK` / `_SKIPPED`.

**Forecast accuracy (§8.4):** `ForecastAccuracyDaily` + `planning/accuracy` nightly job + training-export SKU-day actuals + supplier `GET .../demand/accuracy` (WAPE/bias/TS); flag `FORECAST_ACCURACY_ENABLED`; migration `20260806_forecast_accuracy_daily.ddl`; marker `PX_E2E_FORECAST_ACCURACY_OK` / `_SKIPPED`.

**Forecast algo (§8.1):** Croston SBA / SES / Holt–Winters → `DemandForecastBaseline` via `cmd/planning-forecast`; seasonal Multiplier via `seasonalcore` + persisted overrides; residual bands; predictive-push diverted when on; fake weather/POS stubs removed; docs [`docs/FORECAST_ALGO.md`](../docs/FORECAST_ALGO.md); flags `FORECAST_ALGO_ENABLED`, `FORECAST_SEASONAL_ESTIMATE_ENABLED`; markers `PX_E2E_FORECAST_ALGO_OK` / `_SKIPPED`, `PX_E2E_SEASONAL_OVERRIDE_OK` / `_SKIPPED`.

**Safety stock (§8.2):** Service-level `SS = z_α·√(L·σ_d² + d̄²·σ_L²)` + ROP; `ReceivedAt` on receive; `InTransitQty` populated; policy knobs GET/PATCH + supplier portal; MEIO/echelon share helper; retailer `ReorderSuggestions` batch uses same SS helper when flag on (else `demand·0.15`); 90-day fill-rate replay (`cmd/safety-stock-replay`, `POST /v1/admin/planning/safety-stock/replay`); docs [`docs/SAFETY_STOCK.md`](../docs/SAFETY_STOCK.md); flag `SAFETY_STOCK_V2_ENABLED`; migration `20260806_safety_stock_v2.ddl`; markers `PX_E2E_SAFETY_STOCK_OK` / `_SKIPPED`, `PX_E2E_SAFETY_STOCK_REPLAY_OK` / `_SKIPPED`.

**Auto-order (§8.3):** Inventory-grounded `(R,s,S)` + scoped policy fix + `execution_mode` `off|shadow|draft|place`; shadow ledger + acceptance stats; synthesis `/2` diverted when `AUTO_ORDER_INVENTORY_GROUNDED`; desktop/Android/iOS mode + shadow inbox; docs [`docs/AUTO_ORDER.md`](../docs/AUTO_ORDER.md); migration `20260806_retailer_auto_order_shadow.ddl`; markers `PX_E2E_AUTO_ORDER_SHADOW_OK` / `_SKIPPED`. Residual: auto-flip place at ≥80% acceptance.

Closed in monorepo (Gate-0 Track A — see `artifacts/GATE0_BLAST_RADIUS_2026-08-05.md`, `artifacts/GATE0_SPANNER_BACKUP_RESTORE_2026-08-05.md`):

- **2026-08-05 Google Routes world-scale** Geometry provider chain Google Routes → OSRM → dense; `ROUTING_PROVIDER`; Terraform Maps APIs + client key GSM shells; retailer tracking `route_geometry`; factory `GET /v1/factory/fleet/live-map` + portal/Android/iOS; payload inbound `driver_lat`/`driver_lng`. Artifact: `artifacts/GOOGLE_ROUTES_WORLD_SCALE_2026-08-05.md`. Display SDKs unchanged (Google/MapLibre/MapKit render backend polylines).
- **2026-08-05 Gate-0 Track A** Payment bypass / early-complete → `FISCALIZING` (ADR-009); ops driver PIN bcrypt; nil Spanner fail-loud; SchemaMigrations + narrow DDL benign; outbox UUID + ClaimedBy lease + Kafka `event_id` dedupe; idempotency SHA-256 + principal/route scope; AI confidence gate rejectable + touchless `MinConfidenceScore`; seasonal multiplier on suggested qty; `H3_BINPACK` rename; VelocityGauge honest empty; binaries untracked; gen-contracts vendored; HPA CPU 250m; OSRM PVC; prod digest pins + ManagedCertificate; Spanner PITR 7d + restore rehearsal RTO ~30 min + TF backup + GCS remote state migrated; Firebase Android debug SHA-1 on all 6 apps; iOS `aps-environment` + driver FCM/location. Residual owner ops: live OTP SMS proof, GP card SUCCESS e2e, OSRM PVC map extract (optional), **optimizer-core AR image + deploy (replicas ≥ 1)**. Artifacts: `GATE0_BLAST_RADIUS_*`, `GATE0_SPANNER_BACKUP_RESTORE_*`, `GATE0_BATCH_B_GCP_*`, `GATE0_BATCH_C_CREDS_*`.
- Closed in monorepo (see `artifacts/SUBSTANCE_GATE_CLIENT_SIGNOFF_2026-08-05.md`):

- **2026-08-05 Client Parity Closure** P0-4 iOS offline classifier + GPS fail-closed; AUTHORIZE_BYPASS photo ×3 retailer; supplier/WH return-policy portal+mobile; driver PoD credit gate; empty chart unmount + SpendAnalytics thin wire; portal i18n bootstrap (supplier/warehouse). Interactive UI walks READY_FOR_WALK (human PASS pending). Product-deferred unchanged: negotiations / Soliq / offline POS.
- **2026-08-04 Substance Gate (backend-first)** SSMR preflight green; full `ssmr-smokecheck e2e` + `ssmr-ecosystem-marker-gate-ok` on image `ssmr-substance-gate-a66868b8-084112` (worker replicas=1). Claims spine required markers green (`CLAIM_ELIGIBILITY` / `CLAIM_WINDOW_SNAPSHOT` / media GCS / file / reverse). Sign-off: [`artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md`](../artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md). Client UI walks were DEFERRED; client-policy HTTP 200 all roles×platforms. Ops still: GP card SUCCESS, Firebase SMS.
- **2026-08-04 Gate-0 hygiene** `Claims`/`ClaimEvidences` in `spanner.ddl`; iOS driver/payload drop `convertFromSnakeCase`; supplier Android OrgFleet wreckage deleted (Enterprise+Store Kotlin compile green); optimizer Time dimension in minutes + empty-route fallback; worker `replicas:1` (SSMR scaled); `AUTO_CONFIRM_PREORDERS_ENABLED` sweeper; orphan `ledger/` package deleted. Closed/superseded: partner API shipped; Spanner backup TF + outbox leases Gate-0; multi-tenant Phase 1–3 wired (seed fallback residual).
- **2026-08-04 G3 (backend)** Supplier/WH return-policy tables + resolve_window; immutable `ClaimWindowHours`/`EndsAt`/`PolicySource` on COMPLETED; eligibility/FileClaim prefer snapshot; GET/PUT `/v1/supplier/return-policy` + `/v1/warehouse/return-policy`. Portal/mobile settings UX still open. E2e asserts non-empty `policy_source` (`PX_E2E_CLAIM_WINDOW_SNAPSHOT_OK`).
- **2026-08-04 G2** `GET /v1/orders/{id}/claim-eligibility` (shared window math with file-claim); retailer countdown + CTA hide on desktop/Android/iOS; e2e `PX_E2E_CLAIM_ELIGIBILITY_OK`.
- **2026-08-04 G1** Stock-first Request return / chargeback on retailer desktop, Android, iOS: COMPLETED/DELIVERED_ON_CREDIT order picker → existing FileClaim UI (`initialSku` / `preferredSku` prefill); reuses `POST /v1/orders/{id}/claims`.
- **2026-08-04 Phase B leftovers** G11/G25 claim file Redis idempotency + stable client `claim-file:` keys; G12 returns Kafka consumer on `REVERSE_LOGISTICS_REQUIRED` + `claim_reverse_open_fail_total`; G22 `warehouse_id` on `CLAIM_FILED` WH fanout; G20 e2e receive→CONCEALED→QUARANTINE→inbound markers (`PX_E2E_CLAIMS_CONCEALED_OK`, `PX_E2E_STORE_STOCK_CLAIM_HOLD_OK`, `PX_E2E_CLAIMS_REVERSE_OK`, `PX_E2E_CLAIMS_IDEMPOTENCY_OK`) required in marker gate.
- **2026-08-04 Phase A/B** GCS evidence fail-closed (`REQUIRE_INFRA_ADAPTERS` / prod|ssmr|staging — no `placehold.co`; `invalid_evidence_uri`; e2e `PX_E2E_CLAIM_MEDIA_GCS_OK`). RS0 `claim.file` + `ResolveRetailerOrgID`. G8 hold fail-closed + compensate REJECTED. G9 `ReceivableQty` excludes residual/open claims. E2 design doc + `PerimeterKeyForSupplier` (prod still global key). Ops: WI SA needs `roles/iam.serviceAccountTokenCreator` for signBlob — see owner secrets handoff.
- **2026-08-04 Phase A** Credit risk scoring removed (no score worker / `RiskTier` gates / suggested-limit desk); CREDIT_LEAVE = status + available. E1 session-scoped CT/Compliance; E3 shop-closed DDL wired on SSMR+staging + CI schema-drift gate.
- **P0** Shop-closed CANCELLED/RETURN paths release inventory in-txn
- **P1** `LogisticsException` contracts + CLAIM_* WS/inbox fanout; gen-contracts strict green
- **P1/P2** Retailer credit mobile; supplier finance mutations; driver rescue/earnings/scan-qr; warehouse rescue URLs; factory staff create + exception resolve
- **P2** BillingTierWorker wired; AnalyticsStreamProcessor dummy removed; spatial topic env removed
- **2026-08-01** No-mock / prod-ready: removed retailer empty-list demo injection; portal seeds default off; demand density stub writes removed; staging `FISCAL_PROVIDER=PEGASUS`; tracking/UI shells empty or API-driven; Pegasus branded HTML/PDF receipts
- **2026-08-01** SSMR e2e hardened for no-seed cloud: factory lifecycle via dispatch API, payloader JWT scoped to `ssmr-warehouse-1`, return-gate `IN_TRANSIT` before arrive
- **2026-08-02** Firebase client configs in apps; Android google-services plugin + real JSON init; iOS plists wired

## 2. SSMR cloud reality (`pegasus-503013`)

| Layer | Status |
|-------|--------|
| GKE `pegasusx-ssmr-gke` ns `pegasusx-ssmr` | Live |
| Spanner migrations | **Applied** 2026-08-01 (`DONE_FAIL=0`) |
| ConfigMap | `FISCAL_PROVIDER=PEGASUS`, seeds/`ALLOW_DRIVER_DEMO_FALLBACK=false`, `REQUIRE_INFRA_ADAPTERS=true`, `PAYLOAD_DEMO_WAREHOUSE_ID=ssmr-warehouse-1` |
| Backend image | `…/backend-go:ssmr-substance-gate-a66868b8-084112` (API + worker) |
| Ingress LB `api-ssmr.pegasusx.app` → `136.69.43.141` | Live; ManagedCert **Active** (Google Trust Services WR3); HTTPS redirect on |
| Health / cloud smoke | OK (`PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app` → `PX11_CLOUD_SMOKE_OK`) |
| SSMR e2e + marker gate | **PASS** 2026-08-04 (`artifacts/ssmr-e2e-substance-gate-2026-08-04.log`, `ssmr-ecosystem-marker-gate-ok`) |
| Firebase Auth/FCM backend | Live; iOS plists + Android JSON/plugin applied; real SMS / SHA-1 still owner |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` |
| Global Pay | Card path still needs real merchant password; e2e uses cash fallback (`PX_E2E_PAYMENT_CASH_FALLBACK_OK`) |

## 3. Remaining ops checklist (owner)

See [`artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md`](../artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md) and [`docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md`](../docs/L1_FIELD_UNLOCK_RELEASE_CHECKLIST.md):

1. **Global Pay** — real staging merchant password → GSM → SUCCESS (`PX_E2E_PAYMENT_CARD_SUCCESS_OK`); register webhook in GP portal  
2. **Firebase** — Phone SMS / Blaze; Android debug SHA-1; APNs for iOS push if required  
3. Optional: Soliq/OFD ([`docs/SOLIQ_SANDBOX_READINESS.md`](../docs/SOLIQ_SANDBOX_READINESS.md)); `QUANTITY_NEGOTIATION_ENABLED` after client UX; unused Spanner teardown  

**Engineering (2026-08-04 next-layer remaining):** CT sim hard-blocked on ssmr/prod; auto-order worker flag; negotiation env gate; claim/local-sku docs; mobile local SKUs; e2e markers for card SUCCESS / auto-order / quarantine / Soliq skip.

**Do not** flip `PEGASUSX_ENV=production` until GP SUCCESS and `ValidateProductionProfile` pass.

## 4. Local / cloud verification green

- `go test ./retailer ./driver ./demand ./order` (mock-kill + receipts)  
- Empty API → empty UI policy on tracking/analytics shells  
- `/tmp/ssmr-smokecheck e2e` → exit 0; marker gate exit 0 (115 markers printed; negotiation skipped by design)  
- 2026-08-02: `https://api-ssmr.pegasusx.app/healthz` + `cloud_smoke_ssmr.sh` → `PX11_CLOUD_SMOKE_OK`

## 5. Not blocked on Spanner quota

Migrations applied. DNS/TLS and Firebase client files are done. Remaining blocker for card SUCCESS is Global Pay merchant credentials.
