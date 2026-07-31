# PegasusX Migration & Staging Status

*Last Updated: 2026-07-31 (ecosystem gap-closure pass)*

## 1. Code completeness (this closure)

Closed in monorepo (see `artifacts/PegasusX_Ecosystem_Status_Report.md` for detail):

- **P0** Shop-closed CANCELLED/RETURN paths release inventory in-txn
- **P1** `LogisticsException` contracts + CLAIM_* WS/inbox fanout; gen-contracts strict green
- **P1/P2** Retailer credit mobile; supplier finance mutations; driver rescue/earnings/scan-qr; warehouse rescue URLs; factory staff create + exception resolve
- **P2** BillingTierWorker wired; AnalyticsStreamProcessor dummy removed; spatial topic env removed

## 2. SSMR cloud reality (`pegasus-503013`)

Per `artifacts/PROD_WIRING_AND_THIRD_PARTIES.md` (2026-07-28) + live kubectl (2026-07-31):

| Layer | Status |
|-------|--------|
| GKE `pegasusx-ssmr-gke` ns `pegasusx-ssmr` | Live — `backend-go`, `backend-go-worker`, `ai-worker` Running |
| Spanner / Redis / Strimzi Kafka | Live (provisioned) |
| GSM + External Secrets | Live |
| Ingress LB `api-ssmr.pegasusx.app` → `136.69.43.141` | Live; ManagedCert pending DNS |
| Firebase Auth/FCM backend | Live; app configs / real SMS still needed |
| Maps geocode | Live |
| Fiscal | `FISCAL_PROVIDER=PEGASUS` (Soliq deferred) |
| Global Pay | Webhook CAPTURED smoke; SUCCESS needs real merchant password |

**Note:** `gcloud spanner …` from this workstation may fail with `USER_PROJECT_DENIED` on quota project `v-o-i-d` even when kubectl to GKE works. Fix ADC/quota project before CLI migrations:

```bash
gcloud config set project pegasus-503013
gcloud auth application-default set-quota-project pegasus-503013
```

## 3. Remaining ops checklist (owner actions)

1. DNS A `api-ssmr.pegasusx.app` → `136.69.43.141` → ManagedCert Active  
2. Apply any pending Spanner migrations (incl. shop-closed guards) via `scripts/apply_ssmr_spanner_migrations.sh` with correct ADC  
3. Redeploy API/worker images after this gap-closure commit  
4. Firebase app configs + real OTP SMS on ≥1 mobile role  
5. Global Pay merchant password → SUCCESS path  
6. Optional: Soliq/OFD sandbox when legal receipts required  
7. Run SSMR e2e → `scripts/parity/ssmr_ecosystem_marker_gate.sh <ssmr-e2e.log>` (includes `PX_E2E_SHOP_CLOSED_CANCEL_RELEASE_OK`)  
8. Flags per `docs/gap-closure/STAGING_FLAGS.md`  
9. Destroy legacy `void-494000` when billing-safe  

## 4. Local verification already green

- `go test ./order ./kafka ./notifications ./events` (touched packages)  
- `scripts/parity/gen_contracts_gate.sh`  
- `scripts/parity/role_row_contract_check(_full).sh` (baseline)

## 5. Not blocked on Spanner quota

The 2026-07-21 “Spanner quota blocker” narrative is **obsolete** for `pegasus-503013` SSMR. Remaining work is DNS, credentials, redeploy, and e2e proof — not greenfield Terraform.
