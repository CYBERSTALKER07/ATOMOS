# Cloud end-to-end implementation plan

**Goal:** Take pegasusX from **infra-up** on GCP to a **staging spine** (order → FISCALIZING → COMPLETED with FAKE OFD), then layer auth, maps, pay, OFD, clients, and optional prod promotion.

**Project:** `pegasus-503013` · **Account:** `blackfoxenterprise3697@gmail.com` · **Region:** `asia-south1`  
**Budget:** ~$1,500/mo pilot ([`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md))

**Companion docs**
| Doc | Use |
|-----|-----|
| [`AGENT_CLOUD_CONTEXT.md`](./AGENT_CLOUD_CONTEXT.md) | Live applied vs not |
| [`CLOUD_E2E_WIRING_STEPS.md`](./CLOUD_E2E_WIRING_STEPS.md) | Command-level checklist |
| [`CLOUD_SERVICES_WIRING_PLAN.md`](./CLOUD_SERVICES_WIRING_PLAN.md) | Phase map / money order |
| [`CLOUD_DEVOPS_DEEP_DIVE_PLAN.md`](./CLOUD_DEVOPS_DEEP_DIVE_PLAN.md) | Per-tech deep dives D0–D17 |
| [`PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md`](./PHASE_0_CLOUD_FOUNDATION_RUNBOOK.md) | Make targets |

---

## 1. Principles (do not violate)

1. **No product logic rewrites** — wire env, secrets, deploy, prove only.  
2. **Money order:** infra → secrets → deploy → smoke (FAKE fiscal) → Firebase → Maps → Pay staging → OFD sandbox → clients → prod.  
3. **Never** production OFD/Pay before step 10 green on staging.  
4. **Never** target `void-494000` for new work.  
5. **One step’s exit criteria green** before treating the next as “done.”  
6. **Parallel only** when there is no data/runtime dependency (see §3).

---

## 2. Current baseline (start of plan)

| Step | Status | Evidence |
|------|--------|----------|
| 0 | ⬜ optional re-run | Local gates |
| 1 | ✅ | Project `pegasus-503013` |
| 2 | ✅ | Spanner + Redis + GKE + VPC + AR + GSM + budget |
| 3 | ✅ *base schema* | ~71 tables incl. `OrderFiscalReceipts`, `PaymentSessions`, `OutboxEvents` |
| 4 | 🔄 | Redis READY; GSM auth may exist; PING prove optional |
| 5–18 | ⬜ | Not done |
| 19 | ⬜ | Old project may still bill |

**Critical path starts at step 5 (Kafka)** for a real outbox-driven cloud spine, then 6 → 8 → 9 → 10.

---

## 3. Dependency graph

```text
[0 wire-ready] ─────────────────────────────────────────┐
                                                        │
[1 identity] ✅                                         │
[2 terraform] ✅ ──┬── [3 schema] ✅ ──┐                 │
                   ├── [4 redis prove] ──┤                 │
                   └── [7 kubectl] ──────┤                 │
                                        │                 │
[5 Confluent Kafka] ────────────────────┤                 │
[6 secrets + ESO] ──────────────────────┼──► [8 images]   │
                                        │         │       │
                                        └─────────┼──► [9 deploy]
                                                  │       │
                                                  └──► [10 smoke FAKE fiscal]
                                                          │
                    ┌─────────────────────────────────────┤
                    ▼                     ▼               ▼
              [11 ingress]          [12 Firebase]    [13 Maps]
                    │                     │               │
                    └──────────┬──────────┴───────────────┘
                               ▼
                        [14 Pay staging]
                               ▼
                        [15 OFD sandbox]
                               ▼
                        [16 point clients]
                               ▼
                        [17 HPA / obs]
                               ▼
                        [18 prod] (optional later)
                               │
[19 destroy void-494000] ◄─────┴── anytime after 10 stable (cost)
```

**Hard dependencies**
- 9 needs 2, 3, 5, 6, 7, 8  
- 10 needs 9 (+ working Kafka + Spanner + Redis)  
- 14 needs 10 + 11 (webhook URL)  
- 15 needs 10 + 14  
- 16 needs 11 + 12 (and usually 10)  
- 18 needs all staging green  

**Can run in parallel**
- 0 with anything non-blocking  
- 4 with 5  
- 12 with 5–9 (Firebase project setup offline)  
- 13 with 5–9 (Maps key enablement)  
- 19 after 10 (independent of 11–18)

---

## 4. Implementation waves

### Wave A — Finish platform data plane (1–2 days)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **0** | Eng | `make wire-ready` | `wire-ready-ok` |
| **3** | Eng | Confirm schema; if migrations lag: `bash scripts/d3_apply_schema_gcloud.sh` | 71+ tables; `OrderFiscalReceipts` exists |
| **4** | Platform | AUTH in GSM; GKE Job PING | `PONG`; secrets `pegasusx-staging-redis-auth` / `-addr` |
| **5** | Platform | Confluent Basic cluster + topics + API key | Real bootstrap (not `pkc-xxxxx`); `phase0_wire_kafka_confluent.sh` OK |
| **7** | Eng | `gcloud container clusters get-credentials` + auth plugin | `kubectl get nodes` Ready |

**Commands (Wave A)**

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013

# 0
make wire-ready

# 3 (only if needed)
bash scripts/d3_apply_schema_gcloud.sh

# 4 — AUTH to GSM (then Job prove)
AUTH=$(gcloud redis instances get-auth-string pegasusx-staging-redis \
  --region=asia-south1 --format='value(authString)')
# create/add secret pegasusx-staging-redis-auth + addr
# kubectl Job redis:7-alpine PING with --tls --insecure

# 5 — after Confluent UI create
export KAFKA_BOOTSTRAP="pkc-….gcp.confluent.cloud:9092"
export KAFKA_SASL_USERNAME="…"
export KAFKA_SASL_PASSWORD="…"
bash scripts/phase0_wire_kafka_confluent.sh

# 7
export USE_GKE_GCLOUD_AUTH_PLUGIN=True
gcloud container clusters get-credentials pegasusx-staging-gke \
  --region=asia-south1 --project=pegasus-503013
kubectl get nodes
```

**Wave A done when:** Spanner schema complete, Redis proven, Kafka real, kubectl works.

---

### Wave B — Secrets + images + first deploy (1–2 days)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **6** | Platform + Eng | Fill `.env.staging.secrets`; `make phase0-sync-secrets`; apply External Secrets | Pods can mount JWT, Kafka, Redis, Spanner via WI/ESO |
| **8** | Eng | Build + push backend + ai-worker | Images in AR with `IMAGE_TAG` |
| **9** | Eng | Render + apply staging overlay: API, worker, ai-worker, OSRM, migrate if needed | All key Deployments Ready; logs no emulator |

**Secrets minimum for spine (step 6)**

| Secret | Required for step 10 |
|--------|----------------------|
| `JWT_SECRET` | Yes |
| `INTERNAL_API_KEY` | Yes (optimizer/ai-worker) |
| Kafka bootstrap + SASL | Yes |
| Redis password | Yes |
| Spanner via WI (no key file) | Yes |
| `FISCAL_PROVIDER=FAKE` | Yes |
| Firebase / Maps / Pay / OFD | No for step 10 |

```bash
# 6
cp .env.staging.secrets.example .env.staging.secrets  # fill, never commit
make phase0-sync-secrets
kubectl apply -f infra/k8s/external-secrets/   # + ClusterSecretStore if not present

# 8
export IMAGE_TAG="staging-$(git rev-parse --short HEAD)"
make docker-build-backend docker-build-ai-worker
GAR=$(cd infra/terraform && terraform output -raw artifact_registry_url)
docker tag pegasusx-backend:local "${GAR}/backend-go:${IMAGE_TAG}"
docker tag pegasusx-ai-worker:local "${GAR}/ai-worker:${IMAGE_TAG}"
gcloud auth configure-docker "$(echo $GAR | cut -d/ -f1)" --quiet
docker push "${GAR}/backend-go:${IMAGE_TAG}"
docker push "${GAR}/ai-worker:${IMAGE_TAG}"

# 9
make render-k8s-from-terraform IMAGE_TAG="${IMAGE_TAG}"
kubectl apply -k infra/k8s/overlays/staging --load-restrictor=LoadRestrictionsNone
kubectl get pods -n pegasusx-staging   # or namespace from overlay
kubectl logs deploy/backend-go -n pegasusx-staging --tail=100
```

**Config hard requirements on pods**

```text
PEGASUSX_ENV=staging
REQUIRE_INFRA_ADAPTERS=true
SPANNER_EMULATOR_HOST=          # unset
FISCAL_PROVIDER=FAKE
REDIS_TLS_ENABLED=true
KAFKA_SECURITY_PROTOCOL=SASL_SSL
```

**Wave B done when:** API + worker Ready; health endpoints OK; no emulator fallbacks in logs.

---

### Wave C — Staging spine prove (0.5–1 day)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **10** | Eng + QA | Order/cash path on staging API | Order → **FISCALIZING** → **COMPLETED** with FAKE OFD; outbox → Kafka; Redis used for idempotency |

```bash
# Temporary: port-forward if no ingress yet
kubectl port-forward -n pegasusx-staging svc/backend-go 8080:8080
export PUBLIC_BASE_URL=http://127.0.0.1:8080
make cloud-smoke-ssmr
# Plus manual fiscal path if smoke does not cover full hard-gate
```

**Wave C done when:** fiscal hard-gate proven on **cloud** Spanner (not local SSMR only).  
**This is the “staging platform is real” gate.** Do not start Pay/OFD before this.

---

### Wave D — Edge access + product integrations (2–4 days)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **11** | Platform | Ingress, DNS, managed cert, `PUBLIC_BASE_URL`, WS origins | HTTPS health; WSS connects |
| **12** | Client + Eng | Firebase project, phone auth, Admin JSON → GSM, FCM | Staging OTP on driver/portal test numbers |
| **13** | Platform | Enable Geocoding/Places; key in GSM; restrict key | Geocode non-empty; OSRM routes |
| **14** | Finance + Eng | Global Pay staging merchant, webhook URL, HMAC | Webhook → PAYMENT_CLEARED → FISCALIZING → COMPLETED (still FAKE OFD); no double charge |
| **15** | Finance + Eng | MY_SOLIQ sandbox after 10+14 | Real `fiscal_receipt_id` + QR; fail path + force audited |

**Order inside Wave D:** prefer **11 early** so webhooks (14) and mobile (12/16) have a stable URL.  
**12/13** can start during Wave B offline (account setup) but **prove** after 11.

---

### Wave E — Clients + polish (1–2 days)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **16** | Client + Eng | Point portals/apps at staging API + Firebase + WS | Role logins work; live updates |
| **17** | Platform | HPA max (e.g. 4), PDB, re-enable obs after metrics | Scale under load test; alerts fire test |

---

### Wave F — Cost + production (optional / later)

| Step | Owner | Work | Exit criteria |
|------|-------|------|---------------|
| **19** | Platform | Destroy void-494000 Spanner/Redis/GKE | No unexpected bill on old project |
| **18** | Release | Prod project or isolation; prod secrets; hypercare 72h | Sign-off table; rollback = deploy undo |

---

## 5. Suggested calendar

| Day | Wave | Focus |
|-----|------|--------|
| **D1** | A | wire-ready, Redis prove, Confluent create + wire, kubectl |
| **D2** | B | secrets file, ESO, build/push, deploy |
| **D3** | C | cloud smoke + fiscal FAKE hard-gate |
| **D4–D5** | D | ingress/TLS, Firebase, Maps |
| **D6** | D | Global Pay staging |
| **D7** | D | OFD sandbox (if D3+C green) |
| **D8** | E | clients + HPA/obs |
| **D9+** | F | destroy old project; prod plan |

Adjust for Confluent/Firebase/Pay credential lag (external owners).

---

## 6. RACI (compact)

| Step | Eng | Platform | Finance | Client | Release |
|------|-----|----------|---------|--------|---------|
| 0–3, 8–10 | A | C | I | I | I |
| 4–7, 11, 17, 19 | C | A | I | I | I |
| 5 Kafka account | C | A | I | I | I |
| 12, 16 | C | C | I | A | I |
| 13 | C | A | I | C | I |
| 14–15 | C | C | A | I | I |
| 18 | C | C | C | C | A |

A = accountable · C = contributor · I = informed

---

## 7. Risk register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Kafka still placeholder | Deploy fails or silent no-events | Block step 9/10 until real bootstrap |
| Spanner 100 PU cost always-on | Bill | Destroy non-prod when idle; don’t raise PU without need |
| Double project (void + pegasus) | Double bill | Step 19 after spine green |
| External Secrets miswired | Crash loops | Dry-run secrets; `kubectl describe externalsecret` |
| Observability TF before metrics | Apply failures | Keep `enable_observability_resources=false` until apps scrape |
| Webhook URL before TLS | Pay/OFD fail | Finish step 11 before 14–15 |
| Production keys early | Compliance/money risk | FAKE fiscal until 15 intentionally |

---

## 8. Definition of done

### Staging platform (must ship first)
- [ ] Steps **0–10** green  
- [ ] `REQUIRE_INFRA_ADAPTERS=true` on cloud pods  
- [ ] FAKE fiscal hard-gate on real Spanner  
- [ ] Budget alerts configured  

### Staging product (pilot users)
- [ ] Steps **11–16** green  
- [ ] Pay staging + OFD sandbox as required by pilot SOP  

### Production
- [ ] Step **18** + hypercare  
- [ ] Step **19** done or accepted residual cost  

---

## 9. Agent execution protocol

When user says **“implement step N”** or **“continue E2E”**:

1. Read this plan + `AGENT_CLOUD_CONTEXT.md`.  
2. Confirm account/project: `blackfoxenterprise3697@gmail.com` / `pegasus-503013`.  
3. Execute **only** the current wave’s incomplete steps.  
4. Prove exit criteria before advancing.  
5. Update status tables in `AGENT_CLOUD_CONTEXT.md` and this plan’s §2.  
6. Never destroy resources without explicit user approval.

**Next recommended user command:**  
`implement wave A` or `start step 5` (Kafka is the usual critical-path blocker).

---

## 10. One-page status board (update as you go)

```text
0  wire-ready          [~]  failed spanner stale-read gate (local); not blocking cloud Wave A
1  GCP identity        [x]
2  Terraform stack     [x]
3  Spanner schema      [x]  71 tables; OrderFiscalReceipts + PaymentSessions + OutboxEvents
4  Redis prove         [x]  GSM auth/addr + WAVE_A_REDIS_OK (PING/SET/GET)
5  Kafka GCP Managed   [x]  ACTIVE + 7 topics; GSM; IAM; Go kafkautil SASL PLAIN+token
6  Secrets + ESO       [~]  JWT + internal key + redis/kafka in GSM; ESO install still open
7  kubectl             [x]  nodes Ready
8  Images AR           [ ]
9  Deploy              [ ]  staging CM patched for Spanner/Kafka/Redis auth
10 Cloud smoke FAKE    [ ]  ← platform gate
11 Ingress TLS         [ ]
12 Firebase            [ ]
13 Maps                [ ]
14 Pay staging         [ ]
15 OFD sandbox         [ ]
16 Clients             [ ]
17 HPA/obs             [ ]
18 Prod                [ ]
19 Destroy void-494000 [ ]
```
