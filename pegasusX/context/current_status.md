# PegasusX Migration & Staging Status

*Last Updated: 2026-08-04 (Substance Gate API pass)*

## 1. Code completeness (this closure)

Closed in monorepo (see `artifacts/PegasusX_Ecosystem_Status_Report.md` for detail):

- **2026-08-04 Substance Gate (backend-first)** SSMR preflight green; full `ssmr-smokecheck e2e` + `ssmr-ecosystem-marker-gate-ok` on image `ssmr-substance-gate-a66868b8-084112` (worker replicas=1). Claims spine required markers green (`CLAIM_ELIGIBILITY` / `CLAIM_WINDOW_SNAPSHOT` / media GCS / file / reverse). Sign-off: [`artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md`](../artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md). Client UI walks DEFERRED; client-policy HTTP 200 all roles×platforms. Ops still: GP card SUCCESS, Firebase SMS.
- **2026-08-04 Gate-0 hygiene** `Claims`/`ClaimEvidences` in `spanner.ddl`; iOS driver/payload drop `convertFromSnakeCase`; supplier Android OrgFleet wreckage deleted (Enterprise+Store Kotlin compile green); optimizer Time dimension in minutes + empty-route fallback; worker `replicas:1` (SSMR scaled); `AUTO_CONFIRM_PREORDERS_ENABLED` sweeper; orphan `ledger/` package deleted. Deferred: multi-tenant seed, partner API/ML, Spanner backup TF, outbox leases.
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
