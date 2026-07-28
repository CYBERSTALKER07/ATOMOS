# GCP Migration & Wire-Ready Checklist

*Status: Steps 0–14 complete. Receipts: **PEGASUS platform** default (no Soliq). Step 15 OFD deferred until Soliq API; Global Pay receipt API plug-in ready.*

- [x] **0** Local make wire-ready (re-check)
- [x] **1** GCP identity + project
- [x] **2** Terraform: Spanner, Redis, GKE, VPC, AR, GSM, budget
- [x] **3** Spanner full schema + fiscal migrations (SSMR: preorder DDL applied during Step 10)
- [x] **4** Redis AUTH → GSM + in-VPC PING (Memorystore TLS+AUTH)
- [x] **5** Kafka + topics (Strimzi in-cluster on SSMR; dual-write off)
- [x] **6** Real secrets + External Secrets (ESO SecretSynced)
- [x] **7** kubectl credentials (`pegasusx-ssmr-gke`)
- [x] **8** Build/push images to AR (`ssmr-4a0796fd` / ai glibc)
- [x] **9** Deploy API + worker + ai-worker (1/1 Ready)
- [x] **10** Cloud smoke: order → FISCALIZING → COMPLETED (FAKE) — see `artifacts/step10-smoke-ssmr.md`
- [x] **11** Ingress + DNS + TLS — GCE LB `136.69.43.141` / host `api-ssmr.pegasusx.app` (managed cert pending DNS; see `artifacts/step11-ingress-ssmr.md`)
- [x] **12** Firebase phone OTP + FCM — Auth verifier + FCM ADC online (`ssmr-s12-firebase`); phone test number set; real SMS/device push needs client apps (see `artifacts/step12-firebase-ssmr.md`)
- [x] **13** Maps API key + geocode — real key `pegasusx-ssmr-maps` in GSM; reverse/forward/place/autocomplete green (see `artifacts/step13-maps-ssmr.md`)
- [x] **14** Global Pay staging webhooks — endpoint + Basic auth + CAPTURED accept/replay green; SUCCESS verify waits on real GP staging merchant password (see `artifacts/step14-globalpay-ssmr.md`)
- [~] **15** OFD sandbox — **deferred**: live path is `FISCAL_PROVIDER=PEGASUS` (platform receipts); Soliq/MY_SOLIQ when creds arrive; GP payment receipts optional secondary (see `artifacts/receipts-multi-provider.md`)
- [ ] **16** Point apps/portals at staging
- [ ] **17** HPA / observability polish
- [ ] **18** Production promotion (later)
- [ ] **19** Destroy old void-494000 (stop double bill)
