# Cloud & DevOps deep-dive plan (enterprise, phased)

> **Purpose:** Configure **Kubernetes, Terraform, Docker, Firebase, Kafka, Redis, Spanner, GKE, secrets, observability** to enterprise pilot standard — **one layer at a time**, deep, with exit criteria.  
> **Rule:** No product logic changes. Wire infra + config only.  
> **Budget:** ~$1,500/mo pilot envelope — [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md).  
> **Parent sequence:** [`CLOUD_SERVICES_WIRING_PLAN.md`](./CLOUD_SERVICES_WIRING_PLAN.md) · **Software gate:** [`PRE_CLOUD_THIRD_PARTY_GATE.md`](./PRE_CLOUD_THIRD_PARTY_GATE.md)  
**Official vendor docs:** [Appendix A](#appendix-a--official-documentation-source-of-truth) (Spanner · GKE · Redis · Kafka · Secrets · Terraform · Docker · Firebase · Maps · Billing)

**Repo anchors (already exist — we configure & operate, not rewrite):**

| Layer | Path |
|-------|------|
| Terraform | `infra/terraform/` |
| K8s | `infra/k8s/` (+ `backend-go`, `backend-go-worker`, `ai-worker`, `osrm`, `external-secrets`, overlays) |
| Docker local | `infra/docker-compose.ssmr.yml`, `docker-compose.yml` |
| Secrets | GSM + `infra/k8s/external-secrets/`, `phase0-sync-secrets` |
| App config | `bootstrap/`, `.env.ssmr.example`, k8s ConfigMaps |

**Agent skills to use while diving (install via `npx skills`):**  
`terraform-gcp`, `terraform-engineer`, `terraform-skill`, `gke-basics`, `gke-golden-path`, `gke-reliability`, `kubernetes-specialist`, `devops`, `docker-compose-orchestration`, `firebase-basics`, `firebase-auth-basics`, `firebase-security-rules-auditor`, `kafka-development` + local `kafka-event-contracts`, `redis-development` + local Redis suite, `spanner-discipline`.

---

## How to use this document

1. Finish **Phase D0** (local green) before any paid cloud.  
2. For each phase: **Understand → Configure → Prove → Document → Sign-off**.  
3. Do **not** start Phase Dn+1 until Dn exit criteria are checked.  
4. Dive sessions = one technology deep (e.g. “today is only Kafka”).

---

## Master phase index

| Phase | Technology focus | Depends on | Cloud $ |
|-------|------------------|------------|---------|
| **D0** | Local Docker SSMR + proofs | — | $0 |
| **D1** | GCP org / project / billing / IAM | D0 | ~$0 |
| **D2** | Terraform foundation (VPC, budget) | D1 | low |
| **D3** | Cloud Spanner | D2 | **high** (main bill) |
| **D4** | Memorystore Redis | D2 | medium |
| **D5** | Managed Kafka (Confluent) | D2 | medium |
| **D6** | Secret Manager + External Secrets | D2–D5 | low |
| **D7** | GKE Autopilot + Artifact Registry | D2 | medium |
| **D8** | Docker images + CI push | D7 | low |
| **D9** | Kubernetes deploy (API + worker + ai-worker + OSRM) | D3–D8 | medium |
| **D10** | HPA / PDB / reliability | D9 | low delta |
| **D11** | Firebase Auth + FCM | D6, D9 | low |
| **D12** | Observability + budgets | D7–D9 | low |
| **D13** | Ingress / DNS / TLS | D9 | low |
| **D14** | Maps / OSRM hardening | D9 | usage |
| **D15** | Payments env (staging) | D6, D9 | usage |
| **D16** | Fiscal OFD env (sandbox) | D9, D15 optional | usage |
| **D17** | Production promotion | All above green | full envelope |

---

## Phase D0 — Local Docker SSMR (always first)

### Deep topics
- Compose isolation (`COMPOSE_PROJECT_NAME=pegasusx-ssmr`)
- Spanner emulator vs real Spanner
- Kafka topics creation (`kafka-init`)
- backend-setup migrations
- Marker contracts (`contracts/ssmr_*.json`)

### Configure / prove
```bash
cd pegasusX
make test-ssmr-lifecycle   # spine
make test-ssmr-fiscal      # money hard-gate
make test-ssmr-infra       # full e2e → __SSMR_OK__
```

### Exit
- [ ] `__SSMR_OK__` and `__SSMR_FISCAL_OK__` within last 7 days  
- [ ] Team can explain: API vs worker, outbox → Kafka → fiscal consumer  

### Dive checklist
- [ ] Read `infra/docker-compose.ssmr.yml` end-to-end  
- [ ] Map each port (8180 API, 9110 Spanner, 6389 Redis, 9094 Kafka)  
- [ ] Know how to `ssmr-infra-up` and leave stack running for debugging  

---

## Phase D1 — GCP project, billing, IAM (enterprise baseline)

### Deep topics
- Billing account linkage  
- Project isolation (`staging` vs `prod` projects preferred)  
- IAM least privilege: Terraform SA, GKE WI, human roles  
- Org policies (optional): disable external IPs where possible, require OS Login  

### Configure
1. Create GCP project (e.g. `pegasusx-staging`)  
2. Link billing  
3. Enable APIs: Spanner, Redis, GKE, Secret Manager, Artifact Registry, Compute, Monitoring, IAM  
4. Create Terraform deploy identity (user ADC for first apply, later WIF for CI)  

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### Exit
- [x] Billing active — `void-494000` → billing account `01444D-F6DDEC-B7DC05` (void, open)  
- [x] You can `gcloud projects describe void-494000`  
- [x] Budget alert emails decided — `cyberstalkerx7@gmail.com` (`monthly_budget_usd=1500`)  
- [x] Required APIs enabled (Spanner, Redis, GKE, Secret Manager, Artifact Registry, Compute, Monitoring, IAM, Service Networking, Billing Budgets)  
- [x] `staging.tfvars` `project_id` corrected to `void-494000` (was invalid `v-o-i-d`)  
- [x] Local Spanner emulator `api_endpoint_overrides/spanner` **unset** for cloud work  
- [x] gcloud account `cyberstalkerx7@gmail.com` + ADC + quota project aligned  

**D1 signed:** 2026-07-20 · project `void-494000` · no paid Spanner/GKE/Redis yet (API enable only ≈ $0)

### Cost
Near $0 until resources are created.

---

## Phase D2 — Terraform foundation (IaC is source of truth)

### Deep topics
- State: local → GCS backend (`backend.gcs.tf.example`) for team  
- Modules vs flat TF in `infra/terraform/`  
- `staging.tfvars` vs secrets via `TF_VAR_*`  
- Plan → apply discipline; never hand-edit cloud for managed resources  

### Configure
```bash
cd pegasusX
cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
# Fill: project_id, billing_account_id, monthly_budget_usd=1500, region, kafka placeholders

make terraform-init
make phase0-plan    # review every resource
# Do NOT apply until D1 complete and costs understood
```

### Dive checklist
- [x] Read `main.tf`, `gke.tf`, `budget.tf`, `variables.tf`, `outputs.tf`  
- [x] Decide remote state bucket — **defer** until multi-operator (still local state; `backend.gcs.tf.example` ready)  
- [x] List every resource that will incur cost on apply — see `artifacts/terraform-d2-plan-review.md`  

### Exit
- [x] `terraform plan` succeeds with **expected** resource set — **59 to add, 0 change, 0 destroy** (2026-07-20)  
- [x] Written plan review artifact — `artifacts/terraform-d2-plan-review.md` + `.tfplan`  
- [x] D2 hardening: budget emails wired; GKE Autopilot IP policy fixed for auto-mode VPC  
- [x] **Human apply go-ahead** — applied 2026-07-20 (`apply d2`)  

**D2 signed (applied):** 2026-07-20 · project `void-494000` · live stack  

**Live (cost drivers):** Spanner 100 PU · Redis STANDARD_HA 1GB · GKE Autopilot · AR · VPC · budget $1500 · GSM secrets. Observability alerts deferred (`enable_observability_resources=false` until apps export Prometheus metrics — D12).

### Agent skill
`terraform-gcp`, `terraform-engineer`, HashiCorp `terraform-test` / `terraform-skill`

---

## Phase D3 — Cloud Spanner (largest cost; dive deep)

### Deep topics
- Regional instance, **100 PU** pilot cap  
- Database + IAM (workload identity for GKE)  
- Migrations: base `schema/spanner.ddl` + `schema/migrations/*` incl. fiscal  
- Stale reads (15s) for dashboards vs strong for money  
- Backup / PITR for enterprise  
- Emulator parity: what SSMR already proved  

### Configure
1. Terraform apply **Spanner only** if possible, or full apply knowing Spanner dominates bill  
2. `make phase0-migrate` against cloud  
3. Verify `OrderFiscalReceipts` exists  

### Prove
```bash
# From bastion/local with SPANNER_* cloud env
go run ./apps/backend-go/cmd/setup   # or migrate job
# Simple read: INFORMATION_SCHEMA or health path that hits Spanner
```

### Exit
- [ ] Real Spanner URI (not emulator)  
- [ ] Migrations through `20260720_order_fiscal_receipts.ddl`  
- [ ] Budget dashboard shows Spanner line  

### Cost control
- Do **not** raise PU until CPU >70% sustained  
- Destroy non-prod instance when idle for weeks  

### Agent skill
Local: `spanner-discipline`, `test-with-spanner`

---

## Phase D4 — Memorystore Redis

### Deep topics
- Cache + idempotency + WS Pub/Sub  
- AUTH + transit encryption (`redis_auth_enabled`, TLS)  
- Memory 1 GB pilot  
- Key naming / hash tags for future cluster  
- What breaks if Redis down (strict mode vs degrade)  

### Configure
- Terraform Memorystore — **live** from D2 apply  
- App env: `REDIS_ADDR`, password from GSM  
- `REQUIRE_INFRA_ADAPTERS=true` ⇒ must connect  

### Live (2026-07-20)
| Field | Value |
|-------|-------|
| Instance | `pegasusx-staging-redis` |
| Tier | STANDARD_HA · Redis 7.0 · 1 GB |
| Endpoint | `10.42.205.148:6378` (private VPC only) |
| AUTH | enabled |
| TLS | `SERVER_AUTHENTICATION` |
| Network | `pegasusx-staging-vpc` |
| GSM auth | `pegasusx-staging-redis-auth` |
| GSM addr | `pegasusx-staging-redis-addr` |

### Prove
- [x] In-VPC Job: `PING` → **PONG**, `SET`/`GET` key with AUTH+TLS (`D4_REDIS_PROVE_OK`)  
- [ ] Backend health / ping (after D9 deploy)  
- [ ] Idempotency store is Redis, not in-memory (after D9)  
- [ ] Multi-pod WS only after GKE multi-replica  

### Exit
- [x] Redis instance READY with AUTH + TLS  
- [x] AUTH material in Secret Manager  
- [x] In-cluster AUTH prove (GKE Job)  
- [ ] App wired with `REDIS_ADDR` + password secret (D9 External Secrets)  

**D4 signed:** 2026-07-20 · private IP only (not reachable from laptop without VPC)

### Agent skill
`redis-development`, `redis-core`, `redis-connections`, `redis-security`, `cache-redis-correctness`

---

## Phase D5 — Managed Kafka (Confluent or compatible)

### Deep topics
- Bootstrap servers, SASL/TLS  
- Topics: main, DLQ, orders, spatial, realtime, webhooks, freeze-locks, inventory-import  
- Partitions (pilot: 3 on main)  
- Consumer groups: `void-order-mutator`, `void-notification-dispatcher`, warehouse, ai-worker  
- Outbox relay: Spanner → Kafka (exactly-once intent via outbox)  
- Dual-write flags **off** on pilot  

### Configure
1. Create Confluent cluster in same region as GKE  
2. Create topics matching `staging.tfvars`  
3. Store bootstrap (+ API key secret) in GSM  
4. Terraform already wires secret names  

### Prove
- Outbox event appears on topic after order create  
- Notification dispatcher consumes  
- Fiscal: `FISCAL_RECEIPT_REQUESTED` → order consumer  

### Exit
- [ ] Topics exist; app connects with `REQUIRE_INFRA_ADAPTERS`  
- [ ] No silent fallback to log-only publisher  

### Agent skill
`kafka-development` + local **`kafka-event-contracts`** (authoritative for this monorepo)

---

## Phase D6 — Secrets (GSM + External Secrets)

### Deep topics
- Secret Manager as system of record  
- External Secrets Operator on GKE  
- Rotation without rebuild  
- Never commit `.env.staging.secrets`  
- Separation: staging vs prod secret names  

### Configure
```bash
# Fill .env.staging.secrets from example
make phase0-sync-secrets
# Apply infra/k8s/external-secrets/
```

### Secret matrix (minimum staging)

| Secret | Phase needed |
|--------|----------------|
| JWT_SECRET | D9 |
| INTERNAL_API_KEY | D9 |
| KAFKA_* | D5–D9 |
| REDIS auth | D4–D9 |
| FIREBASE credentials | D11 |
| GLOBAL_PAY_* | D15 |
| FISCAL_MY_SOLIQ_* | D16 (keep FAKE until then) |
| GOOGLE_MAPS_API_KEY | D14 |

### Exit
- [ ] Pods mount secrets; `kubectl get externalsecret` Ready  
- [ ] Rotation test: update GSM → pod refresh  

---

## Phase D7 — GKE Autopilot + Artifact Registry

### Deep topics
- Autopilot vs Standard  
- Workload Identity  
- Private nodes / network  
- Namespaces (`pegasusx-staging`)  
- Artifact Registry + image signing (optional enterprise)  

### Configure
- `enable_gke = true` in tfvars  
- `gcloud container clusters get-credentials …`  
- Create AR repos for `backend-go`, `ai-worker`  

### Prove
```bash
kubectl get nodes
kubectl get ns
```

### Exit
- [ ] Cluster reachable  
- [ ] WI can pull images and access GSM/Spanner  

### Agent skill
`gke-basics`, `gke-golden-path`, `gke-reliability`, `kubernetes-specialist`

---

## Phase D8 — Docker images (build once, promote many)

### Deep topics
- Multi-stage Dockerfiles  
- Non-root, minimal base  
- Tagging: `staging-gitsha`, never only `latest` in prod  
- CI: build on main, scan (optional Trivy)  

### Configure
```bash
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make docker-build-backend docker-build-ai-worker
# push to Artifact Registry (see PHASE_0 runbook)
```

### Exit
- [ ] Images in AR for backend + ai-worker  
- [ ] Same tag referenced by k8s render  

### Agent skill
`docker-compose-orchestration`, `devops`

---

## Phase D9 — Kubernetes deploy (enterprise topology)

### Deep topics
- **Two processes:** `backend-go` (API) + `backend-go-worker` (`PEGASUSX_RUN_MODE=worker`)  
- ai-worker, OSRM, optimizer-core  
- ConfigMaps vs Secrets  
- Probes: `/healthz`, `/ready`  
- PDB / HPA (pilot max 4)  
- Overlays: `dev` / `staging` / `pilot` / `prod`  

### Configure
```bash
make render-k8s-from-terraform IMAGE_TAG=...
kubectl apply -k infra/k8s/overlays/staging
# migrate Job
kubectl wait --for=condition=complete job/backend-go-migrate -n <ns> --timeout=600s
```

### Prove
```bash
kubectl get pods -n <ns>
curl -sf https://$PUBLIC_BASE_URL/v1/health
# Worker: orders leave FISCALIZING
```

### Exit
- [ ] API + worker + ai-worker Ready  
- [ ] `REQUIRE_INFRA_ADAPTERS=true`  
- [ ] `FISCAL_PROVIDER=FAKE`  
- [ ] Cash path → COMPLETED on cloud  

### Critical enterprise rule
**Worker is not optional.** Down worker = stuck fiscal + cash bag freeze.

---

## Phase D10 — Autoscaling & resilience (concurrency)

### Deep topics
- HPA: CPU → scale pods up/down when concurrency drops (cooldown)  
- Min replicas ≥ 2 for API (HA)  
- PDB: maxUnavailable  
- Priority shedding / rate limits (app middleware)  
- Spanner stays fixed PU; do not auto-scale PU blindly  

### Configure
- Use pilot overlay defaults; document min/max  
- Load test optional: `make load-cert-cloud` later  

### Exit
- [ ] HPA object exists; max capped  
- [ ] Scale-down tested (or accepted default cooldown)  

---

## Phase D11 — Firebase (Auth + FCM)

### Deep topics
- Separate Firebase project per env  
- Phone Auth, App Check (enterprise later)  
- Admin SDK on backend  
- Client: `google-services.json` / `GoogleService-Info.plist` per app  
- Supplier portal: cookie JWT (no Firebase client)  

### Configure
1. Create Firebase project  
2. Enable Phone Auth  
3. Service account → GSM  
4. `FIREBASE_AUTH_ENABLED=true`  
5. Device OTP for driver first  

### Prove
- Login issues backend JWT  
- `POST /v1/user/device-token` + test notification  

### Exit
- [ ] At least one role row OTP green on staging  
- [ ] FCM delivery verified  

### Agent skill
`firebase-basics`, `firebase-auth-basics`, `firebase-security-rules-auditor`

---

## Phase D12 — Observability & cost governance

### Deep topics
- Cloud Monitoring dashboards (ai-worker lag, pod restarts)  
- Log retention  
- Budget alerts 50/80/100%  
- SLOs from `LOAD_TEST_SLO.md` (later)  

### Configure
- `enable_observability_resources = true`  
- Notification channels for alerts  
- `ai_worker_monitoring_host` after ingress  

### Exit
- [ ] Budget alerts fire test  
- [ ] Can see pod restarts + Kafka consumer lag  

---

## Phase D13 — Ingress, DNS, TLS

### Deep topics
- HTTPS only  
- `PUBLIC_BASE_URL`  
- WS upgrade paths  
- CORS / `WS_ALLOWED_ORIGINS`  

### Prove
- Health + client-policy over HTTPS  
- Mobile reaches API  

---

## Phase D14 — Maps + OSRM

### Deep topics
- Server key: Geocoding + Places only  
- Android Maps SDK keys restricted by package/SHA  
- OSRM sidecar, not Google Directions  
- MapLibre for portals  

### Exit
- [ ] Geocode works  
- [ ] Driver route geometry non-empty  

---

## Phase D15 — Payments (staging merchants)

### Deep topics
- `GLOBAL_PAY_ENV=staging`  
- Webhook URL + HMAC  
- Idempotent replay  
- Still `FISCAL_PROVIDER=FAKE`  

### Exit
- [ ] Webhook → PAYMENT_CLEARED → FISCALIZING → COMPLETED (fake OFD)  
- [ ] Double webhook no double money  

---

## Phase D16 — Fiscal OFD sandbox

### Deep topics
- Adapter already in `order/fiscal_provider.go`  
- Misconfig hard-fails  
- TIN, sandbox URL, API key  

### Exit
- [ ] Real `fiscal_receipt_id` + QR  
- [ ] Fail path → FISCAL_FAILED; force audited  

---

## Phase D17 — Production promotion

### Deep topics
- Separate project or strict secret isolation  
- Pilot overlay: dual-write off, HPA cap  
- Hypercare 72h  
- Rollback: deployment undo, not Spanner wipe  

### Exit
- [ ] Sign-off table in `CLOUD_SERVICES_WIRING_PLAN.md` complete  
- [ ] Pilot caps documented  

---

## Suggested calendar (deep dives)

| Week | Phases | Focus sessions |
|------|--------|----------------|
| W0 | D0 | SSMR mastery |
| W1 | D1–D2 | GCP + Terraform plan review |
| W2 | D3–D5 | Spanner, Redis, Kafka deep |
| W3 | D6–D9 | Secrets, GKE, first deploy |
| W4 | D10–D13 | HPA, Firebase, ingress, obs |
| W5 | D14–D16 | Maps, Pay, OFD |
| W6 | D17 | Prod pilot |

---

## Per-technology “done” definition (enterprise)

| Tech | Done means |
|------|------------|
| **Docker** | SSMR green; prod images multi-stage, tagged, in AR |
| **Terraform** | Remote state; plan reviewed; apply = only path for infra |
| **Spanner** | Migrations applied; WI access; PU capped; backups set |
| **Redis** | AUTH+TLS; idempotency+cache on Redis; no in-memory prod |
| **Kafka** | Topics + consumers; outbox publishing; DLQ defined |
| **K8s/GKE** | API+worker HA; probes; HPA max; PDB; WI |
| **Secrets** | GSM + External Secrets; no secrets in git/images |
| **Firebase** | Staging OTP + FCM for pilot roles |
| **Obs** | Budget + lag + restart alerts |

---

## What we will **not** do in these phases

- Rewrite backend topology for “enterprise” marketing  
- Multi-region Spanner  
- Helm rewrite of working kustomize (unless you later choose)  
- Auto-scale Spanner PU without metrics  
- Production OFD/Pay before D9 spine green  

---

## Sign-off grid

| Phase | Date | Owner | Evidence | OK |
|-------|------|-------|----------|----|
| D0 Local SSMR | | | `__SSMR_OK__` log | ☐ |
| D1 GCP/IAM | 2026-07-20 | ops | **`pegasus-503013`** + billing + APIs (cutover) | ☑ |
| D2 Terraform plan | 2026-07-20 | ops | **applied** on `pegasus-503013` (Spanner/Redis/GKE live) | ☑ |
| D3 Spanner | | | migrate schema on new Spanner | ☐ |
| D4 Redis | | | AUTH prove on new Redis | ☐ |
| D5 Kafka | | | consume event | ☐ |
| D6 Secrets | | | ExternalSecret Ready | ☐ |
| D7 GKE | | | nodes Ready | ☐ |
| D8 Images | | | AR digests | ☐ |
| D9 Deploy | | | health + fiscal complete | ☐ |
| D10 HPA | | | hpa describe | ☐ |
| D11 Firebase | | | OTP + push | ☐ |
| D12 Obs | | | budget alert test | ☐ |
| D13 Ingress | | | HTTPS health | ☐ |
| D14 Maps | | | geocode | ☐ |
| D15 Pay | | | webhook replay | ☐ |
| D16 OFD | | | real receipt | ☐ |
| D17 Prod | | | hypercare | ☐ |

---

## Immediate next action

### Account cutover (2026-07-20) — D1 done on new project

| Field | Value |
|-------|--------|
| Account | `blackfoxenterprise3697@gmail.com` |
| Project | **`pegasus-503013`** (pegasus · `1002695564567`) |
| Billing | `01BFC8-0FA416-0BBA18` (enabled) |
| Old project | `void-494000` paused (may still bill until destroy) |

**D1 + D2 applied** on `pegasus-503013` — Spanner/Redis/GKE **live** (billing on).  
Summary: `artifacts/terraform-d2-apply-summary-pegasus-503013.md`  

**Next:** **“start d3”** (migrations) · **“start d4”** · **“start d5”** · or **“destroy void-494000 stack”** to stop old project charges.

---

## Appendix A — Official documentation (source of truth)

> **Rule:** Prefer vendor docs over blog posts. Links below map 1:1 to phases D0–D14.  
> **Checked against official sites:** 2026-07-20. Re-verify pricing pages before apply (rates change).

### A.0 Local Docker (D0)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Compose overview | https://docs.docker.com/compose/ | Multi-container stack model |
| Compose file reference | https://docs.docker.com/reference/compose-file/ | `infra/docker-compose.ssmr.yml` semantics |
| Services / ports / env | https://docs.docker.com/reference/compose-file/services/ | API 8180, Spanner emu, Redis, Kafka |
| `docker compose` CLI | https://docs.docker.com/reference/cli/docker/compose/ | `make test-ssmr-*` under the hood |

**Dive note:** Compose Spec (not legacy v2/v3 file versions) is the current format. Project isolation = `COMPOSE_PROJECT_NAME=pegasusx-ssmr`.

---

### A.1 GCP project / billing / IAM (D1)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Create budgets & alerts | https://docs.cloud.google.com/billing/docs/how-to/budgets | `$1500` envelope + email thresholds |
| Budget API / programmatic notify | https://docs.cloud.google.com/billing/docs/how-to/budget-api-overview | Optional auto-actions later |
| Cost management overview | https://cloud.google.com/cost-management | Reports + alerts toolkit |
| gcloud ADC | https://cloud.google.com/docs/authentication/application-default-credentials | Local `terraform` / `gcloud` auth |

**Dive note:** Budgets **alert** by default; they do **not** stop spend unless you wire Pub/Sub + automation. Plan email alerts first.

---

### A.2 Terraform (D2)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Terraform on Google Cloud | https://docs.cloud.google.com/docs/terraform/terraform-overview | IaC model for this stack |
| Google provider (registry) | https://registry.terraform.io/providers/hashicorp/google/latest/docs | Resource schemas |
| Provider config reference | https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference | `project` / `region` / ADC |
| GCS remote state backend | https://developer.hashicorp.com/terraform/language/backend/gcs | Team state + locking |
| Getting started (Google provider) | https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started | First plan/apply patterns |

**Repo:** `infra/terraform/` · start with `make phase0-plan` only (no apply until review).

**Dive note:** State belongs in a **pre-existing** GCS bucket (`backend "gcs"`). Do not hand-edit resources that Terraform owns.

---

### A.3 Cloud Spanner (D3 — main fixed cost)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Compute capacity (nodes & PUs) | https://docs.cloud.google.com/spanner/docs/compute-capacity | **100 PU pilot cap**; 1000 PU = 1 node |
| Create / manage instances | https://docs.cloud.google.com/spanner/docs/create-manage-instances | Instance lifecycle |
| Instances overview | https://docs.cloud.google.com/spanner/docs/instances | Regional config choice |
| Spanner pricing | https://cloud.google.com/spanner/pricing | Hourly compute while instance exists |

**Facts from official compute-capacity docs (must internalize):**

- Capacity is **processing units (PUs)** or **nodes**; **1000 PUs = 1 node**.  
- PU steps are **multiples of 100** (100, 200, …).  
- Billing tracks compute over time even when QPS is idle — **teardown non-prod when idle**.  
- Granular instances allow **&lt; 1 node** (e.g. 100–900 PU).

**Repo skills:** `spanner-discipline`, `test-with-spanner`.  
**Migrations:** `schema/spanner.ddl` + `schema/migrations/*` (incl. fiscal receipts).

---

### A.4 Memorystore Redis (D4)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Memorystore for Redis docs | https://docs.cloud.google.com/memorystore/docs/redis | Managed Redis product home |
| Product overview | https://cloud.google.com/memorystore | HA / use cases |
| Pricing | https://cloud.google.com/memorystore/docs/redis/pricing | 1 GB pilot sizing |

**App uses:** cache, idempotency keys, WS Pub/Sub fanout. Prefer AUTH + transit encryption in staging/prod.  
**Repo skills:** `redis-core`, `redis-connections`, `redis-security`, `cache-redis-correctness`.

---

### A.5 Managed Kafka / Confluent (D5)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Confluent Cloud overview | https://docs.confluent.io/cloud/current/overview.html | Managed Kafka on GCP |
| Confluent Cloud quick start | https://docs.confluent.io/cloud/current/get-started/index.html | Cluster → topic → produce |
| Apache Kafka documentation | https://kafka.apache.org/documentation/ | Consumer groups, partitions, offsets |

**Repo:** topics/env from `staging.tfvars.example` (`kafka_bootstrap_servers`, `kafka_topic_*`).  
**Contract skill:** local **`kafka-event-contracts`** (outbox, `WriteMessages`, DLQ, partition keys).  
**Prove:** Spanner outbox → Kafka → `void-notification-dispatcher` / fiscal consumer.

---

### A.6 Secret Manager + External Secrets (D6)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Secret Manager overview | https://docs.cloud.google.com/secret-manager/docs/overview | SoR for API keys / JWT / Pay / OFD |
| Create & access secrets | https://docs.cloud.google.com/secret-manager/docs/creating-and-accessing-secrets | Versioned secret data |
| Create secret quickstart | https://docs.cloud.google.com/secret-manager/docs/create-secret-quickstart | Console / gcloud path |
| External Secrets Operator | https://external-secrets.io/ | Sync GSM → K8s Secrets |
| ESO Google Secret Manager provider | https://external-secrets.io/latest/provider/google-secrets-manager/ | Workload Identity auth order |

**Repo:** `infra/k8s/external-secrets/`, `make phase0-sync-secrets`.  
**Auth preference on GKE:** Workload Identity → avoid static SA JSON in pods.

---

### A.7 GKE Autopilot + HPA (D7, D9, D10)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Autopilot overview | https://docs.cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview | Google manages nodes; pay for pods |
| Create Autopilot cluster | https://docs.cloud.google.com/kubernetes-engine/docs/how-to/creating-an-autopilot-cluster | First cluster |
| Autopilot vs Standard | https://docs.cloud.google.com/kubernetes-engine/docs/resources/autopilot-standard-feature-comparison | Why Autopilot for pilot |
| Horizontal Pod autoscaling | https://docs.cloud.google.com/kubernetes-engine/docs/concepts/horizontalpodautoscaler | Scale on CPU/memory/custom metrics |
| Configure HPA | https://docs.cloud.google.com/kubernetes-engine/docs/how-to/horizontal-pod-autoscaling | `maxReplicas` pilot cap (e.g. 4) |
| HPA + cluster scale tutorial | https://docs.cloud.google.com/kubernetes-engine/docs/learn/scalable-apps-autoscale | HPA pods ↔ Autopilot nodes |

**Facts from official Autopilot / HPA docs:**

- Autopilot provisions/scales **nodes** when HPA adds pods; idle pod scale-down → node reclaim.  
- You pay Autopilot **pod** CPU/memory (+ cluster management fee), not idle VMs you own.  
- Always set **HPA max** so concurrency spikes cannot blow the $1500 envelope.

**Repo:** `infra/k8s/` · API + `backend-go-worker` + `ai-worker` + OSRM.

---

### A.8 Firebase Auth / phone OTP (D11)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Firebase Authentication | https://firebase.google.com/docs/auth | Product overview |
| Phone auth — Android | https://firebase.google.com/docs/auth/android/phone-auth | Driver / warehouse mobile |
| Phone auth — iOS | https://firebase.google.com/docs/auth/ios/phone-auth | iOS clients |
| Phone auth — Web | https://firebase.google.com/docs/auth/web/phone-auth | Portals / desktop |
| Auth limits (incl. phone) | https://firebase.google.com/docs/auth/limits | Staging rate limits; test numbers |

**Dive note:** Enable Phone provider in Firebase console; use **test/fictional numbers** in staging to avoid SMS burn and IP hourly limits.

---

### A.9 Maps / geocoding (D14)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Geocoding API overview | https://developers.google.com/maps/documentation/geocoding/guides-v3/overview | Address ↔ lat/lng |
| Geocoding API docs hub | https://developers.google.com/maps/documentation/geocoding | Keys, quotas, errors |
| Maps Platform pricing | https://mapsplatform.google.com/pricing/ | Usage-based; key in GSM |

**Repo secret:** `GOOGLE_MAPS_API_KEY` via GSM (D6). OSRM remains in-cluster for routes.

---

### A.10 Cross-cutting cost model (all paid phases)

| Topic | Official doc | Why we care |
|-------|--------------|-------------|
| Spanner pricing | https://cloud.google.com/spanner/pricing | Dominant fixed cost while on |
| Memorystore pricing | https://cloud.google.com/memorystore/docs/redis/pricing | Always-on cache |
| GKE product / Autopilot billing model | https://cloud.google.com/kubernetes-engine | Pod-based compute |
| Cloud Billing budgets | https://docs.cloud.google.com/billing/docs/how-to/budgets | Alert at % of $1500 |

**Internal:** [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md) · [`COST_GOVERNANCE_RUNBOOK.md`](./COST_GOVERNANCE_RUNBOOK.md).

---

### A.11 How to use this appendix in a dive session

1. Open the **phase** section (D0–D17) above.  
2. Open the matching **A.x** rows in a browser tab set.  
3. Configure only what the phase lists; prove exit criteria.  
4. If a vendor page disagrees with this appendix, **trust the vendor page** and update this file.  

**Do not** apply Terraform or enable Spanner until D0 is green and D1–D2 plan is reviewed.
