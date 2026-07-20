# Cloud end-to-end wiring — all steps

**Target:** staging on GCP project **`pegasus-503013`**  
**Account:** `blackfoxenterprise3697@gmail.com`  
**Region:** `asia-south1`  
**Budget:** ~$1,500/mo pilot  

**Related:**  
[`AGENT_CLOUD_CONTEXT.md`](./AGENT_CLOUD_CONTEXT.md) (live status) ·  
[`CLOUD_SERVICES_WIRING_PLAN.md`](./CLOUD_SERVICES_WIRING_PLAN.md) ·  
[`CLOUD_DEVOPS_DEEP_DIVE_PLAN.md`](./CLOUD_DEVOPS_DEEP_DIVE_PLAN.md) ·  
[`PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](./PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md)

**Rules**
- Software logic is done; this is **wire secrets + infra + deploy only**.
- Stay on **staging** until sign-off: `FISCAL_PROVIDER=FAKE`, `GLOBAL_PAY_ENV=staging` until later steps.
- Money order: **infra → auth → maps → payments → OFD**. Never production OFD first.
- Never use old project **`void-494000`** for new work (destroy separately when ready).

---

## Snapshot legend

| Marker | Meaning |
|--------|---------|
| ✅ | Done on `pegasus-503013` |
| 🔄 | Partial / in progress |
| ⬜ | Not done — you/agent still apply |

---

## STEP 0 — Local software gate (precondition)

| # | Action | Pass | Status |
|---|--------|------|--------|
| 0.1 | `cd pegasusX && make wire-ready` | `wire-ready-ok` | ⬜ re-check anytime |
| 0.2 | `make test-ssmr-fiscal` | `__SSMR_FISCAL_OK__` | ⬜ optional if recently green |
| 0.3 | Fiscal hard-gate still enforced in code | no soft COMPLETED | ✅ (prior work) |

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
make wire-ready
```

---

## STEP 1 — Identity & project

| # | Action | Status |
|---|--------|--------|
| 1.1 | Google account owns project | ✅ `blackfoxenterprise3697@gmail.com` |
| 1.2 | Project | ✅ `pegasus-503013` (pegasus / 1002695564567) |
| 1.3 | Billing linked | ✅ `01BFC8-0FA416-0BBA18` |
| 1.4 | CLI login | ✅ (re-run if session expires) |

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013
gcloud auth application-default set-quota-project pegasus-503013
gcloud projects describe pegasus-503013
```

**IDE:** Cloud extension → same account + project.

---

## STEP 2 — GCP foundation (Terraform)

| # | Action | Status |
|---|--------|--------|
| 2.1 | `staging.tfvars` filled | ✅ |
| 2.2 | `terraform init` + plan | ✅ |
| 2.3 | `terraform apply` | ✅ |
| 2.4 | Spanner 100 PU | ✅ READY |
| 2.5 | Redis STANDARD_HA 1GB | ✅ READY |
| 2.6 | GKE Autopilot | ✅ RUNNING |
| 2.7 | VPC, AR, GSM shells, budget, WI SA | ✅ |

**Live names**

| Resource | Name |
|----------|------|
| Spanner | `pegasusx-staging-spanner` / `pegasusx-staging-db` |
| Redis | `pegasusx-staging-redis` · private IP · :6378 |
| GKE | `pegasusx-staging-gke` |
| AR | `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-staging-images` |
| SA | `staging-backend@pegasus-503013.iam.gserviceaccount.com` |

```bash
# Only if you need to re-plan/re-apply deltas later:
cd pegasusX
make phase0-plan
# make phase0-apply   # careful — live state exists
```

---

## STEP 3 — Spanner schema (D3) 🔄 YOU ARE HERE

| # | Action | Status |
|---|--------|--------|
| 3.1 | Instance + empty DB | ✅ |
| 3.2 | Base DDL (`schema/spanner.ddl`) | 🔄 partial (~16 tables) |
| 3.3 | Migrations through `20260720_order_fiscal_receipts.ddl` | ⬜ |
| 3.4 | Prove `OrderFiscalReceipts` exists | ⬜ |

**Preferred (IDE terminal — batch gcloud, faster than Go setup):**

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013
bash scripts/d3_apply_schema_gcloud.sh
# expect: d3-gcloud-schema-ok
```

**If stuck Go migrate is running:**

```bash
bash scripts/stop_d3_migrate.sh
```

**Prove (Spanner Studio or gcloud):**

```sql
SELECT COUNT(*) AS tables
FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '';

SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = '' AND TABLE_NAME = 'OrderFiscalReceipts';
```

Also expect planning tables after full migrate: `DemandForecastBaseline`, etc.

**Env for later tools:**

```bash
# already roughly in .env.k8s.generated
export SPANNER_PROJECT=pegasus-503013
export SPANNER_INSTANCE=pegasusx-staging-spanner
export SPANNER_DATABASE=pegasusx-staging-db
export SPANNER_EMULATOR_HOST=
```

---

## STEP 4 — Redis prove (D4)

| # | Action | Status |
|---|--------|--------|
| 4.1 | Memorystore instance | ✅ READY |
| 4.2 | AUTH string → GSM | ⬜ |
| 4.3 | In-VPC PING (GKE Job) | ⬜ |
| 4.4 | App env `REDIS_ADDR` + TLS + password | ⬜ (at deploy) |

```bash
# Save AUTH (never commit)
AUTH=$(gcloud redis instances get-auth-string pegasusx-staging-redis \
  --region=asia-south1 --project=pegasus-503013 --format='value(authString)')
HOST=$(gcloud redis instances describe pegasusx-staging-redis \
  --region=asia-south1 --project=pegasus-503013 --format='value(host)')

printf '%s' "$AUTH" | gcloud secrets create pegasusx-staging-redis-auth \
  --project=pegasus-503013 --replication-policy=automatic --data-file=- \
  2>/dev/null || printf '%s' "$AUTH" | gcloud secrets versions add pegasusx-staging-redis-auth \
  --project=pegasus-503013 --data-file=-

printf '%s' "${HOST}:6378" | gcloud secrets create pegasusx-staging-redis-addr \
  --project=pegasus-503013 --replication-policy=automatic --data-file=- \
  2>/dev/null || true

# Prove from GKE (private IP) — same pattern as prior D4 Job with redis:7-alpine + TLS
gcloud container clusters get-credentials pegasusx-staging-gke \
  --region=asia-south1 --project=pegasus-503013
# then Job with redis-cli --tls --insecure PING → PONG
```

App later:

```text
REDIS_ADDR=<host>:6378
REDIS_PASSWORD=<from GSM>
REDIS_TLS_ENABLED=true
```

---

## STEP 5 — Managed Kafka / Confluent (D5)

| # | Action | Status |
|---|--------|--------|
| 5.1 | Create Confluent cluster (GCP `asia-south1` if possible) | ⬜ |
| 5.2 | Create topics (3 partitions) | ⬜ |
| 5.3 | API key (username/password) | ⬜ |
| 5.4 | Wire GSM via script | ⬜ |
| 5.5 | Update `staging.tfvars` bootstrap (optional re-apply secret only) | ⬜ |

**Topics (staging names)**

```text
staging.events.orders
staging.events.orders-dlq
staging.events.spatial
staging.events.realtime
staging.events.webhooks
staging.events.freeze-locks
staging.events.inventory-import
```

Optional planning: `planning.signal.ingest.v1`, `planning.forecast.request.v1`, `planning.forecast.result.v1`

```bash
export KAFKA_BOOTSTRAP="pkc-REAL....:9092"
export KAFKA_SASL_USERNAME="<api-key>"
export KAFKA_SASL_PASSWORD="<api-secret>"
cd pegasusX
bash scripts/phase0_wire_kafka_confluent.sh
```

Runbook: `artifacts/d5-kafka-confluent-runbook.md`

**Consumer groups (app creates):**  
`void-order-mutator`, `void-notification-dispatcher`, `void-warehouse-mutator`

---

## STEP 6 — Secrets baseline (D6)

| # | Secret / env | Status |
|---|--------------|--------|
| 6.1 | GSM secret **ids** from Terraform | ✅ shells |
| 6.2 | Real `JWT_SECRET`, `INTERNAL_API_KEY` | ⬜ |
| 6.3 | Kafka real bootstrap + SASL | ⬜ (step 5) |
| 6.4 | Redis AUTH | ⬜ (step 4) |
| 6.5 | Maps / Firebase / Pay / OFD | ⬜ later steps |
| 6.6 | External Secrets Operator on GKE | ⬜ |
| 6.7 | `make phase0-sync-secrets` from `.env.staging.secrets` | ⬜ |

```bash
cd pegasusX
cp .env.staging.secrets.example .env.staging.secrets
# fill values — NEVER commit
make phase0-sync-secrets

# Install/apply External Secrets (see infra/k8s/external-secrets/)
kubectl apply -k infra/k8s/external-secrets/   # adjust path/overlay as in repo
```

**Runtime flags for first cloud spine**

```text
PEGASUSX_ENV=staging
REQUIRE_INFRA_ADAPTERS=true
FISCAL_PROVIDER=FAKE
GLOBAL_PAY_ENV=staging   # when pay wired
SPANNER_EMULATOR_HOST=   # empty / unset
```

---

## STEP 7 — GKE access (D7 polish)

| # | Action | Status |
|---|--------|--------|
| 7.1 | Cluster RUNNING | ✅ |
| 7.2 | Workload Identity pool | ✅ (from apply) |
| 7.3 | `kubectl` credentials + auth plugin | ⬜ on each machine |
| 7.4 | Namespace / overlays ready | ⬜ at deploy |

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/share/google-cloud-sdk/bin:$PATH"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
gcloud container clusters get-credentials pegasusx-staging-gke \
  --region=asia-south1 --project=pegasus-503013
kubectl get nodes
```

---

## STEP 8 — Container images (D8)

| # | Action | Status |
|---|--------|--------|
| 8.1 | Build backend + ai-worker | ⬜ |
| 8.2 | Push to Artifact Registry | ⬜ |

```bash
cd pegasusX
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make docker-build-backend docker-build-ai-worker

GAR="$(cd infra/terraform && terraform output -raw artifact_registry_url)"
# asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-staging-images

docker tag pegasusx-backend:local "${GAR}/backend-go:${IMAGE_TAG}"
docker tag pegasusx-ai-worker:local "${GAR}/ai-worker:${IMAGE_TAG}"
gcloud auth configure-docker "$(echo "$GAR" | cut -d/ -f1)" --quiet
docker push "${GAR}/backend-go:${IMAGE_TAG}"
docker push "${GAR}/ai-worker:${IMAGE_TAG}"
```

---

## STEP 9 — Deploy API + worker + ai-worker (D9)

| # | Action | Status |
|---|--------|--------|
| 9.1 | Render k8s from terraform outputs | ⬜ |
| 9.2 | Apply staging overlay | ⬜ |
| 9.3 | Migrate Job (if not already from step 3) | ⬜ |
| 9.4 | `backend-go` + `backend-go-worker` + `ai-worker` Ready | ⬜ |
| 9.5 | OSRM / optimizer as required by overlay | ⬜ |

```bash
cd pegasusX
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make render-k8s-from-terraform IMAGE_TAG="${IMAGE_TAG}"

kubectl apply -k infra/k8s/overlays/staging --load-restrictor=LoadRestrictionsNone
# migrate job if used:
# kubectl apply -f artifacts/k8s-rendered/backend-go-migrate-job.yaml -n pegasusx-staging
# kubectl wait --for=condition=complete job/backend-go-migrate -n pegasusx-staging --timeout=600s

kubectl get pods -A | grep -E 'backend|ai-worker|osrm'
kubectl logs -n pegasusx-staging deploy/backend-go --tail=50
```

**Topology:** two processes — API (`backend-go`) and worker (`PEGASUSX_RUN_MODE=worker`).

---

## STEP 10 — Platform spine prove (cloud smoke)

| # | Action | Status |
|---|--------|--------|
| 10.1 | Health / ready | ⬜ |
| 10.2 | Order path → FISCALIZING → COMPLETED with **FAKE** OFD | ⬜ |
| 10.3 | Outbox → Kafka → consumers | ⬜ |
| 10.4 | Redis idempotency (not in-memory) | ⬜ |

```bash
export PUBLIC_BASE_URL=https://<staging-api-host>   # after ingress
make cloud-smoke-ssmr
# or scripts/cloud_smoke_ssmr.sh
```

Pass markers: fiscal / lifecycle OK without soft complete.

---

## STEP 11 — Ingress / DNS / TLS (D13)

| # | Action | Status |
|---|--------|--------|
| 11.1 | Ingress IP | ⬜ |
| 11.2 | DNS `api.staging.<domain>` | ⬜ |
| 11.3 | Managed cert / TLS | ⬜ |
| 11.4 | `PUBLIC_BASE_URL`, `WS_ALLOWED_ORIGINS` | ⬜ |

```bash
kubectl get ingress -n pegasusx-staging
# point DNS A/AAAA → LB IP; attach cert per WS_INGRESS_AFFINITY.md
```

---

## STEP 12 — Firebase Auth + FCM (D11)

| # | Action | Status |
|---|--------|--------|
| 12.1 | Firebase project (link or same GCP) | ⬜ |
| 12.2 | Enable Phone auth | ⬜ |
| 12.3 | Admin credentials → GSM | ⬜ |
| 12.4 | Client `google-services.json` / iOS plist | ⬜ |
| 12.5 | Staging OTP test numbers | ⬜ |

---

## STEP 13 — Maps + OSRM (D14)

| # | Action | Status |
|---|--------|--------|
| 13.1 | Enable Geocoding (+ Places if needed) | ⬜ |
| 13.2 | Server key → GSM `pegasusx-staging-google-maps-api-key` | ⬜ |
| 13.3 | Restrict key by IP / API | ⬜ |
| 13.4 | OSRM pod healthy; geocode works | ⬜ |

---

## STEP 14 — Payments staging (D15)

| # | Action | Status |
|---|--------|--------|
| 14.1 | Global Pay **staging** merchants | ⬜ |
| 14.2 | Webhook URL + HMAC secret → GSM | ⬜ |
| 14.3 | `GLOBAL_PAY_ENV=staging` | ⬜ |
| 14.4 | Webhook → PAYMENT_CLEARED → FISCALIZING → COMPLETED (FAKE OFD) | ⬜ |
| 14.5 | Double webhook no double money | ⬜ |

**Still FAKE fiscal.**

---

## STEP 15 — Fiscal OFD sandbox (D16)

| # | Action | Status |
|---|--------|--------|
| 15.1 | MY_SOLIQ sandbox credentials | ⬜ |
| 15.2 | `FISCAL_PROVIDER=MY_SOLIQ` (or env switch) | ⬜ |
| 15.3 | Real `fiscal_receipt_id` + QR | ⬜ |
| 15.4 | Fail path → FISCAL_FAILED; force audited | ⬜ |

Only after step 10 + cash/pay paths green on cloud with FAKE.

---

## STEP 16 — Clients → staging (D11–D14 clients)

| # | Action | Status |
|---|--------|--------|
| 16.1 | Portals `NEXT_PUBLIC_API_URL` / desktop API base | ⬜ |
| 16.2 | Driver apps staging API + Firebase | ⬜ |
| 16.3 | WebSocket origins allowlist | ⬜ |

---

## STEP 17 — Observability + HPA (D10 / D12)

| # | Action | Status |
|---|--------|--------|
| 17.1 | HPA max cap (pilot e.g. 4) | ⬜ |
| 17.2 | PDB | ⬜ |
| 17.3 | Re-enable TF observability **after** metrics scrape | ⬜ |
| 17.4 | Budget alert email received (test) | ⬜ |

`enable_observability_resources = false` until apps export Prometheus metrics.

---

## STEP 18 — Production promotion (D17) — later

| # | Action | Status |
|---|--------|--------|
| 18.1 | Separate prod project or hard secret isolation | ⬜ |
| 18.2 | Prod secrets / merchants / OFD | ⬜ |
| 18.3 | Dual-write off; HPA cap | ⬜ |
| 18.4 | Hypercare 72h | ⬜ |
| 18.5 | Rollback = deploy undo, **not** Spanner wipe | ⬜ |

---

## STEP 19 — Cost hygiene (anytime)

| # | Action | Status |
|---|--------|--------|
| 19.1 | Confirm budget 80%/100% emails | ⬜ confirm |
| 19.2 | Destroy **void-494000** stack when cutover stable | ⬜ explicit only |
| 19.3 | Idle staging: scale Spanner PU / destroy non-prod | optional |

---

## Ordered “do next” checklist (minimal path to cloud spine)

Copy this for daily standups:

```text
[ ] 0  make wire-ready (local gate)
[x] 1  gcloud project pegasus-503013
[x] 2  Terraform stack live
[ ] 3  bash scripts/d3_apply_schema_gcloud.sh  → OrderFiscalReceipts
[ ] 4  Redis AUTH → GSM + GKE PING
[ ] 5  Confluent cluster + phase0_wire_kafka_confluent.sh
[ ] 6  .env.staging.secrets + phase0-sync-secrets + External Secrets
[ ] 7  kubectl get-credentials
[ ] 8  docker build/push IMAGE_TAG
[ ] 9  render-k8s + kubectl apply staging
[ ] 10 cloud-smoke / fiscal FAKE complete on staging URL
[ ] 11 ingress + TLS + PUBLIC_BASE_URL
[ ] 12 Firebase phone OTP
[ ] 13 Maps key + geocode
[ ] 14 Global Pay staging webhooks
[ ] 15 OFD sandbox (only after 10+14)
[ ] 16 Point clients at staging
[ ] 17 HPA/obs polish
[ ] 19 destroy void-494000 (optional cost)
```

---

## Key commands cheat sheet

```bash
# Always
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013

# D3 schema
bash scripts/d3_apply_schema_gcloud.sh

# Secrets
make phase0-sync-secrets

# Images
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make docker-build-backend docker-build-ai-worker
# tag + push to AR (see step 8)

# Deploy
make render-k8s-from-terraform IMAGE_TAG="${IMAGE_TAG}"
kubectl apply -k infra/k8s/overlays/staging --load-restrictor=LoadRestrictionsNone

# Smoke
export PUBLIC_BASE_URL=https://api.staging.example.com
make cloud-smoke-ssmr
```

---

## Success definition (staging “wired”)

1. Real Spanner (full schema) + Redis AUTH + Kafka real bootstrap  
2. API + worker pods Ready on GKE with `REQUIRE_INFRA_ADAPTERS=true`  
3. HTTPS health  
4. Cash/order path: **FISCALIZING → COMPLETED** with **FAKE** OFD  
5. Outbox events appear on Kafka  
6. Budget alerts configured  
7. No dependency on emulators  

Production OFD/Pay and store clients are **after** that spine.

---

## Agent note

Before any cloud action, re-read **`AGENT_CLOUD_CONTEXT.md`** and update this file’s ✅/🔄/⬜ when a step completes.
