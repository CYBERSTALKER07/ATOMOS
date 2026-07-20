# Cloud services wiring plan (step by step)

> **Goal:** Connect real cloud + third-party services **without** changing product logic.  
> Software Layer A is complete; this plan is **Layer B only** (provision → secret → deploy → prove → promote).  
> Budget envelope: **~$1,500/mo** pilot — see [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md).

**Authority chain:**  
[`PRE_CLOUD_THIRD_PARTY_GATE.md`](./PRE_CLOUD_THIRD_PARTY_GATE.md) → this plan →  
[`CLOUD_DEVOPS_DEEP_DIVE_PLAN.md`](./CLOUD_DEVOPS_DEEP_DIVE_PLAN.md) (per-tech deep dives D0–D17) ·  
[`PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](./PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md) ·  
[`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md) ·  
[`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md) ·  
[`WIRE_READY_STAGING_RUNBOOK.md`](./WIRE_READY_STAGING_RUNBOOK.md)

**Rule:** Never flip `PEGASUSX_ENV=production` or real merchant/OFD keys until the matching **Prove** step is green on **staging**.

---

## Owners

| Role | Owns |
|------|------|
| **Platform** | GCP project, Terraform, GKE, Spanner, Redis, Kafka, GSM, ingress, monitoring |
| **Finance** | Global Pay / Payme / Click / OFD TIN + merchant contracts |
| **Client** | Firebase `google-services.json` / `GoogleService-Info.plist`, Maps SDK keys, store signing |
| **Eng** | Migrations, image build, deploy verify, smoke scripts, worker health |
| **Release** | Sign-off table, hypercare roster, go-live |

---

## Phase map (do not reorder)

```
0  Local software green (already done)
1  GCP foundation (VPC, Spanner, Redis, Kafka, GSM, GKE)
2  Secrets baseline (JWT, internal key — no money yet)
3  Deploy API + worker + ai-worker + OSRM (FAKE fiscal, no live PSP)
4  Prove platform spine on staging
5  Firebase Auth + FCM
6  Maps geocode + Android Maps SDK + OSRM
7  Payments: Global Pay staging (+ optional Payme/Click)
8  Fiscal OFD sandbox (MY_SOLIQ) — only after spine + cash path green on cloud
9  Portals / clients pointed at staging
10 Production promotion + 72h hypercare
```

Money order is intentional: **infra → auth → maps → PSP → OFD**.  
Never wire production OFD before staging cash path completes under FAKE, then sandbox OFD.

---

## Phase 0 — Local software gate (precondition)

**Owner:** Eng  
**Status expectation:** Already green after fiscal hard-gate work.

| Step | Action | Pass criteria |
|------|--------|---------------|
| 0.1 | `make wire-ready` | `wire-ready-ok` |
| 0.2 | `make test-ssmr-fiscal` | `__SSMR_FISCAL_OK__` |
| 0.3 | `make px12-preflight` | `px12-preflight-ok` |
| 0.4 | Confirm no soft `COMPLETED` without fiscal | unit + prior gap-hunt |

**Do not start Phase 1 if 0.x fails.**

---

## Phase 1 — GCP foundation (no app traffic)

**Owner:** Platform  
**Docs:** `PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md` §1, `infra/terraform/`

| Step | Action | Pass criteria |
|------|--------|---------------|
| 1.1 | Create/select GCP project; link billing | Billing active |
| 1.2 | `gcloud auth application-default login` | ADC works |
| 1.3 | Copy `infra/terraform/staging.tfvars.example` → `staging.tfvars` | `project_id`, `billing_account_id`, `monthly_budget_usd=1500`, alert emails set |
| 1.4 | Provision **managed Kafka** (e.g. Confluent `asia-south1`); create topics per terraform README | Bootstrap servers known |
| 1.5 | `make terraform-init` → `make phase0-plan` → `make phase0-apply` | Outputs: Spanner URI, GKE name, Artifact Registry, GSM |
| 1.6 | Confirm budget alerts at 80% / 100% | Policies exist in Monitoring |

**Services wired:** Spanner instance/DB, Memorystore Redis, GKE Autopilot, Secret Manager, Artifact Registry, VPC, budget.

**Still OFF:** live PSP, OFD, Firebase prod, Maps keys (placeholders OK).

---

## Phase 2 — Secret Manager baseline

**Owner:** Platform (+ Finance later for payment/OFD)  
**Docs:** `CLOUD_CREDENTIALS_CHECKLIST.md`, `make phase0-sync-secrets`

| Step | Secret / env | When |
|------|--------------|------|
| 2.1 | `JWT_SECRET`, `JWT_ISSUER` | Now |
| 2.2 | `INTERNAL_API_KEY` (optimizer / ai-worker) | Now |
| 2.3 | `KAFKA_BROKERS` (+ SASL if required) | Now |
| 2.4 | Webhook **dev/staging** HMAC placeholders for Global Pay / Payme / Click | Now (real values Phase 7) |
| 2.5 | `FISCAL_PROVIDER=FAKE` | Now (stay FAKE through Phase 4–6) |
| 2.6 | Firebase Admin JSON | Phase 5 |
| 2.7 | `GOOGLE_MAPS_API_KEY` | Phase 6 |
| 2.8 | `GLOBAL_PAY_*` real staging | Phase 7 |
| 2.9 | `FISCAL_MY_SOLIQ_*` | Phase 8 |

```bash
# Boss fills .env.staging.secrets (never commit)
make phase0-sync-secrets
```

**Pass:** GSM versions exist; External Secrets can mount them (after operator install if needed).

---

## Phase 3 — First deploy (platform only)

**Owner:** Eng + Platform  
**Docs:** Phase 0 §3–5, `infra/k8s/overlays/staging`

| Step | Action | Pass criteria |
|------|--------|---------------|
| 3.1 | `make phase0-migrate` (or migrate Job) through **all** migrations incl. `20260720_order_fiscal_receipts.ddl` | `OrderFiscalReceipts` + fiscal columns on Orders |
| 3.2 | Build/push images: `backend-go`, `ai-worker` | Tags in Artifact Registry |
| 3.3 | `make render-k8s-from-terraform IMAGE_TAG=…` | Rendered manifests |
| 3.4 | `kubectl apply -k infra/k8s/overlays/staging` | Pods: **backend-go**, **backend-go-worker**, **ai-worker**, **osrm**, External Secrets |
| 3.5 | Confirm `REQUIRE_INFRA_ADAPTERS=true` | No emulator fallback |
| 3.6 | Confirm `PEGASUSX_RUN_MODE`: API = api/all; worker = **worker** | Two deployments healthy |
| 3.7 | Ingress / `PUBLIC_BASE_URL` | TLS HTTPS reachable |

**Critical:** Worker down ⇒ orders stuck `FISCALIZING`. Treat as P0 ops.

---

## Phase 4 — Prove platform spine (still FAKE fiscal, no live money)

**Owner:** Eng  
**Docs:** `cloud_smoke_ssmr.sh`, credential validation §1

| Step | Action | Pass criteria |
|------|--------|---------------|
| 4.1 | `PUBLIC_BASE_URL=https://… make cloud-smoke-ssmr` | Health + client-policy |
| 4.2 | `bash scripts/validate_staging_credentials.sh` | `staging-credentials-ok` (non-secret names) |
| 4.3 | Manual or scripted: create order → dispatch → seal → depart → arrive → cash collect | Order enters `FISCALIZING` then `COMPLETED` under **FAKE** OFD |
| 4.4 | Kill worker 2 min → open-fiscal / return-complete blocked → restore worker → completes | Worker topology proven |
| 4.5 | Optional: `make load-cert-cloud` | SLO profile against real Spanner |

**Exit Phase 4:** Staging runs full logistics spine with cloud Spanner/Kafka/Redis; fiscal still FAKE.

---

## Phase 5 — Firebase (Auth + FCM)

**Owner:** Client + Platform  
**Docs:** credential validation §4, `CLOUD_CREDENTIALS_CHECKLIST` Firebase section

| Step | Action | Pass criteria |
|------|--------|---------------|
| 5.1 | Create Firebase project (staging); enable Phone Auth | Console ready |
| 5.2 | Generate Admin service account → GSM → mount `FIREBASE_CREDENTIALS_PATH` | Backend can verify tokens |
| 5.3 | Set `FIREBASE_AUTH_ENABLED=true` on staging | Flag on |
| 5.4 | Ship `google-services.json` / `GoogleService-Info.plist` per app (driver, warehouse, factory, payload, retailer iOS as needed) | Builds with correct project |
| 5.5 | Device OTP login per role row (driver first) | Token accepted; session issued |
| 5.6 | `POST /v1/user/device-token` + test push | FCM delivery |

**Supplier portal** stays cookie/JWT OTP — no Firebase client.

---

## Phase 6 — Maps + routing

**Owner:** Platform + Client  
**Docs:** credential validation §5 / maps table

| Step | Action | Pass criteria |
|------|--------|---------------|
| 6.1 | Enable Geocoding + Places APIs; restrict key | `GOOGLE_MAPS_API_KEY` in GSM |
| 6.2 | Backend topology geocode on staging | Address → lat/lng |
| 6.3 | Confirm OSRM sidecar `ROUTING_OSRM_URL` | Route geometry on driver/supplier maps |
| 6.4 | Android Maps SDK keys (driver + retailer) with package/SHA restrictions | Map tiles on device |
| 6.5 | iOS MapKit (no GCP Maps JS) | Fleet maps render |

**Do not enable:** Maps JavaScript API (portals use MapLibre), Google Directions (use OSRM).

---

## Phase 7 — Payments (staging merchants)

**Owner:** Finance + Eng  
**Docs:** credential validation §2–3; code: `payment/global_pay_executor.go`, webhooks

| Step | Action | Pass criteria |
|------|--------|---------------|
| 7.1 | Global Pay **staging** credentials → GSM | `GLOBAL_PAY_ENV=staging`, service id, user, password, webhook secret |
| 7.2 | Register webhook URL `https://…/v1/webhooks/global-pay` | Provider console configured |
| 7.3 | Checkout / card-at-delivery path on staging | Session → perform → webhook → `PAYMENT_CLEARED` |
| 7.4 | Confirm order → `FISCALIZING` → FAKE fiscal → `COMPLETED` | Hard-gate intact with real PSP |
| 7.5 | Replay same webhook | Idempotent; no double capture |
| 7.6 | (Optional) Payme / Click sandbox secrets + fixture-like live POST | 401 without signature; 200 with |
| 7.7 | Cancelled order + late pay | Lands `RECONCILIATION_REQUIRED`; complete only via **force audit** (not soft COMPLETED) |

**Still:** `FISCAL_PROVIDER=FAKE`.

---

## Phase 8 — Fiscal OFD sandbox (MY_SOLIQ)

**Owner:** Finance + Eng  
**Pre-req:** Phase 4 cash path + Phase 7 webhook path green; `make test-ssmr-fiscal` was green locally.

| Step | Action | Pass criteria |
|------|--------|---------------|
| 8.1 | Obtain sandbox base URL, API key, supplier **TIN (STIR)** | Contract signed |
| 8.2 | GSM: `FISCAL_MY_SOLIQ_BASE_URL`, `API_KEY`, `TIN`, optional PATH/TIMEOUT | Versions present |
| 8.3 | Set staging `FISCAL_PROVIDER=MY_SOLIQ` (API + worker) | Both pods restarted |
| 8.4 | Cash collect on staging | Attempt `SUCCESS`, real `fiscal_receipt_id` + QR on tracking |
| 8.5 | Force OFD down / bad key | `FISCAL_FAILED` (never invent SUCCESS) |
| 8.6 | Force-complete with reason | `FORCE_SKIPPED` + audit event |
| 8.7 | Re-run open-fiscal / shift freeze with live provider | Same product behavior |

**Only after 8.x green:** consider production OFD credentials in a later promote step.

---

## Phase 9 — Clients + portals at staging URL

**Owner:** Client + Eng

| Step | Action | Pass criteria |
|------|--------|---------------|
| 9.1 | Point portals: `NEXT_PUBLIC_*` / backend base URL → staging | Login + order list |
| 9.2 | Point native apps to staging API/WS | Driver cash + fiscal wait UI |
| 9.3 | Desktop Tauri updater staging bucket if used | Optional |
| 9.4 | Role-row manual QA | [`docs/qa/PX12_ROLE_ROW_QA.md`](./qa/PX12_ROLE_ROW_QA.md) |

---

## Phase 10 — Production promotion

**Owner:** Release + all  
**Docs:** `P0_LAUNCH_CHECKLIST.md`, `P1_PILOT_CHECKLIST.md`, incident runbook

| Step | Action | Pass criteria |
|------|--------|---------------|
| 10.1 | Pilot overlay: dual-write OFF, HPA caps | `infra/k8s/overlays/pilot` |
| 10.2 | Prod secrets (new GSM versions; never reuse staging keys) | Versions isolated |
| 10.3 | `GLOBAL_PAY_ENV=production` only after staging LC-02 sign-off | Finance written OK |
| 10.4 | `FISCAL_PROVIDER=MY_SOLIQ` prod URL/TIN | One real receipt in pilot shop |
| 10.5 | `PEGASUSX_ENV=production` | Profile validation scripts green |
| 10.6 | Hypercare 72h roster | Named on-call for API, worker, OFD, PSP |
| 10.7 | Pilot caps: 1 warehouse, 10–30 drivers, 50–150 retailers | Documented |

**Rollback:**

```bash
kubectl rollout undo deployment/backend-go -n <ns>
kubectl rollout undo deployment/backend-go-worker -n <ns>
# Keep Spanner/Kafka data; flip FISCAL_PROVIDER or payment env if provider outage
```

---

## Service checklist (quick reference)

| # | Service | Phase | Env / secret highlights |
|---|---------|-------|-------------------------|
| 1 | Cloud Spanner | 1, 3 | `SPANNER_*` |
| 2 | Memorystore Redis | 1, 3 | `REDIS_*` |
| 3 | Managed Kafka | 1, 3 | `KAFKA_BROKERS`, topics |
| 4 | GKE + Artifact Registry | 1, 3 | images, WI |
| 5 | Secret Manager + External Secrets | 2 | all secrets |
| 6 | backend-go API | 3 | `HTTP_PORT`, `REQUIRE_INFRA_ADAPTERS` |
| 7 | backend-go-worker | 3 | `PEGASUSX_RUN_MODE=worker` |
| 8 | ai-worker | 3 | `INTERNAL_API_KEY`, Kafka |
| 9 | OSRM sidecar | 3, 6 | `ROUTING_OSRM_URL` |
| 10 | Cloud Monitoring / budget | 1 | alerts |
| 11 | GCS | 3+ | catalog / imports / updates |
| 12 | Firebase Auth + FCM | 5 | Admin + client configs |
| 13 | Google Maps Geocode/Places | 6 | `GOOGLE_MAPS_API_KEY` |
| 14 | Maps SDK Android | 6 | per-app keys |
| 15 | Global Pay | 7 | `GLOBAL_PAY_*`, webhook |
| 16 | Payme / Click (optional) | 7 | webhook secrets |
| 17 | OFD / my.soliq | 8 | `FISCAL_PROVIDER`, `FISCAL_MY_SOLIQ_*` |
| 18 | Cloud Run portals (if used) | 9 | min-instances=0 |
| 19 | Store signing | 10 | Apple / Play / Tauri |

---

## Week-by-week suggestion (pilot)

| Week | Phases | Outcome |
|------|--------|---------|
| W0 | 0 | Local green locked |
| W1 | 1–2 | GCP + secrets baseline |
| W2 | 3–4 | Staging spine under FAKE fiscal |
| W3 | 5–6 | Auth + maps |
| W4 | 7 | Real PSP staging |
| W5 | 8–9 | OFD sandbox + clients |
| W6 | 10 | Limited pilot traffic + hypercare |

Compress only if Platform + Finance capacity allows; **do not skip 4.x or 8.5**.

---

## Sign-off table (fill as you go)

| Phase | Date | Owner | Evidence | OK |
|-------|------|-------|----------|----|
| 0 Local | | Eng | wire-ready + fiscal log | ☐ |
| 1 Terraform | | Platform | `terraform output` | ☐ |
| 2 Secrets | | Platform | GSM versions | ☐ |
| 3 Deploy | | Eng | kubectl get pods | ☐ |
| 4 Spine | | Eng | cloud smoke + cash complete | ☐ |
| 5 Firebase | | Client | OTP + push | ☐ |
| 6 Maps | | Platform | geocode + OSRM | ☐ |
| 7 PSP | | Finance | webhook + no double charge | ☐ |
| 8 OFD | | Finance | real receipt_id + QR | ☐ |
| 9 Clients | | Client | PX12 QA sheet | ☐ |
| 10 Prod | | Release | hypercare + pilot caps | ☐ |

---

## What this plan deliberately does **not** do

- Change fiscal/payment state machines during wiring  
- Put production OFD keys in local Docker  
- Enable multi-supplier OFD legs  
- Use Maps JS API or Google Directions  
- Soft-complete orders to “unblock” cloud testing  

If something fails, fix **config / secrets / deploy topology** first. If product logic is wrong, stop wiring and reopen Layer A — do not invent behavior under live keys.
