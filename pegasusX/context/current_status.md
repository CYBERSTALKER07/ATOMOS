# PegasusX Migration & Staging Status

*Last Updated: 2026-08-01 (prod-ready no-mocks gap closure)*

## 1. Code completeness (this closure)

Closed in monorepo (see `artifacts/PegasusX_Ecosystem_Status_Report.md` for detail):

- **P0** Shop-closed CANCELLED/RETURN paths release inventory in-txn
- **P1** `LogisticsException` contracts + CLAIM_* WS/inbox fanout; gen-contracts strict green
- **P1/P2** Retailer credit mobile; supplier finance mutations; driver rescue/earnings/scan-qr; warehouse rescue URLs; factory staff create + exception resolve
- **P2** BillingTierWorker wired; AnalyticsStreamProcessor dummy removed; spatial topic env removed
- **2026-08-01** No-mock / prod-ready: removed retailer empty-list demo injection; portal seeds default off; demand density stub writes removed; staging `FISCAL_PROVIDER=PEGASUS`; tracking/UI shells empty or API-driven; Pegasus branded HTML/PDF receipts

## 2. SSMR cloud reality (`pegasus-503013`)

| Layer | Status |
|-------|--------|
| GKE `pegasusx-ssmr-gke` ns `pegasusx-ssmr` | Live — pods Restarted after ConfigMap patch |
| Spanner migrations | **Applied** 2026-08-01 (`DONE_FAIL=0`, 46 DDL files) |
| ConfigMap | `FISCAL_PROVIDER=PEGASUS`, seeds/`ALLOW_DRIVER_DEMO_FALLBACK=false`, `REQUIRE_INFRA_ADAPTERS=true` |
| Ingress LB `api-ssmr.pegasusx.app` → `136.69.43.141` | Live; ManagedCert still **Provisioning** (no public DNS A) |
| Health smoke | OK via `curl --resolve …:80:136.69.43.141` |
| Backend image with no-mock code | **Not pushed** — Cloud Build / Docker Hub blocked from this workstation (see handoff) |
| Firebase Auth/FCM backend | Live; app configs / real SMS still needed |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` |
| Global Pay | CAPTURED path; SUCCESS needs real merchant password |

**ADC note:** `application-default set-quota-project pegasus-503013` may fail with `USER_PROJECT_DENIED` on `v-o-i-d`. Spanner apply worked with `GOOGLE_CLOUD_QUOTA_PROJECT=pegasus-503013` env; Cloud Build still needs IAM fix.

## 3. Remaining ops checklist (owner)

See [`artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md`](../artifacts/OWNER_SECRETS_HANDOFF_2026-08-01.md):

1. Fix ADC/Cloud Build IAM → build + rollout no-mock image tag  
2. DNS A `api-ssmr.pegasusx.app` → `136.69.43.141` → ManagedCert Active + HTTPS redirect  
3. Firebase app configs + OTP SMS  
4. Global Pay merchant password → SUCCESS  
5. SSMR e2e → `ssmr_ecosystem_marker_gate.sh`  
6. Optional: Soliq/OFD, flag rollout, `void-494000` teardown  

## 4. Local verification already green

- `go test ./retailer ./driver ./demand ./order` (mock-kill + receipts)  
- Empty API → empty UI policy on tracking/analytics shells  

## 5. Not blocked on Spanner quota

Migrations applied. Remaining work is image push, DNS/TLS, GP password, Firebase clients, e2e marker proof.
