# PegasusX Migration & Staging Status

*Last Updated: 2026-08-01 (SSMR e2e + marker gate green)*

## 1. Code completeness (this closure)

Closed in monorepo (see `artifacts/PegasusX_Ecosystem_Status_Report.md` for detail):

- **P0** Shop-closed CANCELLED/RETURN paths release inventory in-txn
- **P1** `LogisticsException` contracts + CLAIM_* WS/inbox fanout; gen-contracts strict green
- **P1/P2** Retailer credit mobile; supplier finance mutations; driver rescue/earnings/scan-qr; warehouse rescue URLs; factory staff create + exception resolve
- **P2** BillingTierWorker wired; AnalyticsStreamProcessor dummy removed; spatial topic env removed
- **2026-08-01** No-mock / prod-ready: removed retailer empty-list demo injection; portal seeds default off; demand density stub writes removed; staging `FISCAL_PROVIDER=PEGASUS`; tracking/UI shells empty or API-driven; Pegasus branded HTML/PDF receipts
- **2026-08-01** SSMR e2e hardened for no-seed cloud: factory lifecycle via dispatch API, payloader JWT scoped to `ssmr-warehouse-1`, return-gate `IN_TRANSIT` before arrive

## 2. SSMR cloud reality (`pegasus-503013`)

| Layer | Status |
|-------|--------|
| GKE `pegasusx-ssmr-gke` ns `pegasusx-ssmr` | Live |
| Spanner migrations | **Applied** 2026-08-01 (`DONE_FAIL=0`) |
| ConfigMap | `FISCAL_PROVIDER=PEGASUS`, seeds/`ALLOW_DRIVER_DEMO_FALLBACK=false`, `REQUIRE_INFRA_ADAPTERS=true`, `PAYLOAD_DEMO_WAREHOUSE_ID=ssmr-warehouse-1` |
| Backend image | `…/backend-go:ssmr-gap-closure-nomock3` (API + worker) |
| Ingress LB `api-ssmr.pegasusx.app` → `136.69.43.141` | Live; ManagedCert still **Provisioning** (no public DNS A) |
| Health / cloud smoke | OK (`cloud_smoke_ssmr.sh` → `PX11_CLOUD_SMOKE_OK` via port-forward) |
| SSMR e2e + marker gate | **PASS** 2026-08-01 (`ssmr-e2e.log`, `ssmr-ecosystem-marker-gate-ok`) via `svc/backend-go` port-forward `:18180` |
| Firebase Auth/FCM backend | Live; app configs / real SMS still needed |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` |
| Global Pay | Card path still needs real merchant password; e2e uses cash fallback (`PX_E2E_PAYMENT_CASH_FALLBACK_OK`) |

## 3. Remaining ops checklist (owner)

See [`artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md`](../artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md):

1. DNS A `api-ssmr.pegasusx.app` → `136.69.43.141` → ManagedCert Active + HTTPS redirect  
2. Firebase app configs + OTP SMS  
3. Global Pay merchant password → SUCCESS (not cash fallback)  
4. Optional: redeploy image with latest tree fixes (`auth_login` warehouse default); Soliq/OFD; flag rollout; `void-494000` teardown  

**Do not** flip `PEGASUSX_ENV=production` until DNS/TLS, GP SUCCESS, and `ValidateProductionProfile` pass.

## 4. Local / cloud verification green

- `go test ./retailer ./driver ./demand ./order` (mock-kill + receipts)  
- Empty API → empty UI policy on tracking/analytics shells  
- `/tmp/ssmr-smokecheck e2e` → exit 0; marker gate exit 0 (115 markers printed; negotiation skipped by design)

## 5. Not blocked on Spanner quota

Migrations applied. Remaining work is DNS/TLS, GP password, Firebase clients — not mock kill or e2e marker proof.
