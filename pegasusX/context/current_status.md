# PegasusX Migration & Staging Status

*Last Updated: 2026-08-02 (nomock4 + DNS/TLS + Firebase clients)*

## 1. Code completeness (this closure)

Closed in monorepo (see `artifacts/PegasusX_Ecosystem_Status_Report.md` for detail):

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
| Backend image | `…/backend-go:ssmr-gap-closure-nomock4` (API + worker) |
| Ingress LB `api-ssmr.pegasusx.app` → `136.69.43.141` | Live; ManagedCert **Active** (Google Trust Services WR3); HTTPS redirect on |
| Health / cloud smoke | OK (`PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app` → `PX11_CLOUD_SMOKE_OK`) |
| SSMR e2e + marker gate | **PASS** 2026-08-01 (`ssmr-e2e.log`, `ssmr-ecosystem-marker-gate-ok`) via port-forward |
| Firebase Auth/FCM backend | Live; iOS plists + Android JSON/plugin applied; real SMS / SHA-1 still owner |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` |
| Global Pay | Card path still needs real merchant password; e2e uses cash fallback (`PX_E2E_PAYMENT_CASH_FALLBACK_OK`) |

## 3. Remaining ops checklist (owner)

See [`artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md`](../artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md):

1. **Global Pay** — real staging merchant password → GSM → SUCCESS (not cash fallback); register webhook in GP portal  
2. **Firebase** — Phone SMS / Blaze; Android debug SHA-1; APNs for iOS push if required  
3. Optional: Soliq/OFD; flag rollout; unused Spanner teardown  

**Do not** flip `PEGASUSX_ENV=production` until GP SUCCESS and `ValidateProductionProfile` pass.

## 4. Local / cloud verification green

- `go test ./retailer ./driver ./demand ./order` (mock-kill + receipts)  
- Empty API → empty UI policy on tracking/analytics shells  
- `/tmp/ssmr-smokecheck e2e` → exit 0; marker gate exit 0 (115 markers printed; negotiation skipped by design)  
- 2026-08-02: `https://api-ssmr.pegasusx.app/healthz` + `cloud_smoke_ssmr.sh` → `PX11_CLOUD_SMOKE_OK`

## 5. Not blocked on Spanner quota

Migrations applied. DNS/TLS and Firebase client files are done. Remaining blocker for card SUCCESS is Global Pay merchant credentials.
