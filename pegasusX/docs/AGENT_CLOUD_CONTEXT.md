# Agent + human cloud context (read this first)

**Last updated:** 2026-07-20  
**Purpose:** Single source of truth for what is **already applied**, what to **apply next**, and what **not** to touch.  
**Repo path:** `pegasusX/` under V.O.I.D monorepo.

---

## Active target (use this only)

| Field | Value |
|-------|--------|
| Google account | `blackfoxenterprise3697@gmail.com` |
| Project name | pegasus |
| Project ID | **`pegasus-503013`** |
| Project number | `1002695564567` |
| Region | `asia-south1` |
| Billing account (tfvars) | `01BFC8-0FA416-0BBA18` |
| Budget | $1500 → `blackfoxenterprise3697@gmail.com` |
| Tenant slug | `staging` |

```bash
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013
```

**Do not** apply Terraform or migrations to `void-494000` (old pilot).

---

## Old project (paused / still may bill)

| Field | Value |
|-------|--------|
| Project | `void-494000` |
| Account that owned it | `cyberstalkerx7@gmail.com` |
| Status | **Paused cutover** — Spanner/Redis/GKE may still exist and **bill** |
| TF state archive | `pegasusX/artifacts/tfstate-archive/` |
| Destroy | Only when human says **“destroy void-494000 stack”** |

New account **cannot** list resources on `void-494000` (expected IAM deny).

---

## Phase status (D0–D17)

| Phase | Status | Notes |
|-------|--------|--------|
| **D0** Local SSMR | Optional / prior | `make test-ssmr-*` — local Docker, not cloud |
| **D1** GCP project/IAM | **DONE** | APIs enabled on `pegasus-503013` |
| **D2** Terraform apply | **DONE** | Spanner + Redis + GKE + AR + GSM + budget + VPC |
| **D3** Spanner schema | **IN PROGRESS / partial** | Instance READY; **schema incomplete** (~16 tables). Full DDL + migrations **not** finished. Fiscal table missing. |
| **D4** Redis prove | **NOT DONE** on new project | Instance READY; need AUTH+TLS prove + GSM auth secret |
| **D5** Kafka Confluent | **NOT DONE** | Topic names planned; bootstrap still placeholder |
| **D6** Secrets ESO | **NOT DONE** | GSM secret shells exist; External Secrets not on cluster |
| **D7** GKE ready | **MOSTLY DONE** | Cluster RUNNING + WI SA from D2 |
| **D8** Images | **NOT DONE** | AR empty (0 MB) |
| **D9** Deploy apps | **NOT DONE** | No backend pods yet |
| **D10–D17** | **NOT DONE** | HPA polish, Firebase, obs, ingress, maps, pay, OFD, prod |

---

## Already applied on `pegasus-503013` (live)

### Infrastructure (Terraform D2)

| Resource | Name | Status |
|----------|------|--------|
| Spanner instance | `pegasusx-staging-spanner` | READY · **100 PU** · regional-asia-south1 |
| Spanner database | `pegasusx-staging-db` | READY |
| Redis | `pegasusx-staging-redis` | READY · STANDARD_HA · 1 GB · port **6378** · host private VPC |
| GKE Autopilot | `pegasusx-staging-gke` | RUNNING · asia-south1 |
| VPC | `pegasusx-staging-vpc` | live |
| Artifact Registry | `pegasusx-staging-images` | live · empty |
| GCS app updates | `pegasus-503013-pegasusx-staging-app-updates` | live (global name includes project id) |
| Billing budget | pegasusX monthly cap $1500 | created |
| Runtime SA | `staging-backend@pegasus-503013.iam.gserviceaccount.com` | WI bound to `pegasusx/backend-go` |
| GSM secrets | Kafka topics, JWT/Maps shells, etc. | created (many values empty/placeholder) |

**Spanner URI:**
```text
projects/pegasus-503013/instances/pegasusx-staging-spanner/databases/pegasusx-staging-db
```

**AR URL:**
```text
asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-staging-images
```

### Schema (D3) — partial only

Last check:

- Table count ≈ **16** (full app needs many more from `schema/spanner.ddl` + migrations)
- Present examples: `Orders`, `Suppliers`, `Retailers`
- **Missing (critical):** `OrderFiscalReceipts`, likely `OutboxEvents` and most domain tables

Stuck slow path was `go run ./cmd/setup` (one DDL at a time, timeouts). **Stopped.** Prefer gcloud batch script below.

### Local repo state

| Path | Role |
|------|------|
| `infra/terraform/staging.tfvars` | **pegasus-503013** config |
| `infra/terraform/terraform.tfstate` | **Live** state for pegasus-503013 |
| `artifacts/tfstate-archive/` | Old void-494000 state/tfvars |
| `.env.k8s.generated` | Cloud Spanner env for pegasus-503013 |
| `scripts/d3_apply_schema_gcloud.sh` | **Preferred D3 apply** from IDE |
| `scripts/stop_d3_migrate.sh` | Kill stuck setup/migrate |
| `docs/CLOUD_ACCOUNT_CUTOVER.md` | Cutover notes |
| `docs/CLOUD_DEVOPS_DEEP_DIVE_PLAN.md` | Full phase plan D0–D17 |

---

## What you should apply next (ordered)

### 1. Finish D3 — Spanner schema (do this first)

**From IDE terminal** (Cloud extension logged into same account):

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013
bash scripts/d3_apply_schema_gcloud.sh
```

Success marker: `d3-gcloud-schema-ok`

Verify in Spanner Studio:

```sql
SELECT COUNT(*) AS tables
FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '';

SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = '' AND TABLE_NAME = 'OrderFiscalReceipts';
```

**Do not** re-run old `make phase0-migrate` / `go run ./cmd/setup` unless gcloud path fails.

### 2. D4 — Redis prove (instance already exists)

- Store AUTH in GSM: `pegasusx-staging-redis-auth`
- Prove PING from GKE Job (private IP only)
- Wire later: `REDIS_ADDR`, `REDIS_TLS_ENABLED=true`

### 3. D5 — Kafka (Confluent) — needs your Confluent account

- Create cluster + topics (`staging.events.*`)
- Wire GSM: `scripts/phase0_wire_kafka_confluent.sh`
- Bootstrap currently placeholder `pkc-xxxxx…`

### 4. D8–D9 — Build images + deploy to GKE

- Push to Artifact Registry
- `kubectl apply` staging overlays
- External Secrets, backend + worker

### 5. Optional cost cleanup

- **Destroy void-494000** stack when human confirms (stops double billing)

---

## What NOT to apply / redo

| Action | Why |
|--------|-----|
| `terraform apply` full stack again without plan | Already applied; only for deltas |
| Create second Spanner/GKE on same project | Duplicates cost |
| Use project `void-494000` for new work | Abandoned cutover target |
| Login as `cyberstalkerx7@gmail.com` for pegasus-503013 | No access |
| `enable_observability_resources = true` before apps | Prometheus metrics missing → TF alerts fail |
| Production OFD/Pay | Later phases (D15–D16) |

---

## Cost reality

**Billing now on pegasus-503013:**

- Spanner 100 PU (main cost)
- Redis STANDARD_HA 1 GB
- GKE Autopilot control plane / pods when scheduled

**May still bill on void-494000** if resources not destroyed.

---

## Agent instructions (short)

1. Always target **`pegasus-503013`** + **`blackfoxenterprise3697@gmail.com`**.
2. Read this file + `CLOUD_DEVOPS_DEEP_DIVE_PLAN.md` before cloud changes.
3. Prefer **plan before apply**; never destroy without explicit user yes.
4. D3 incomplete → run / help user run **`scripts/d3_apply_schema_gcloud.sh`**.
5. After D3: D4 Redis prove → D5 Kafka → D8/D9 deploy.
6. Terraform state is local at `infra/terraform/terraform.tfstate` (pegasus).

---

## Human: Google Cloud IDE extension

1. Sign in as **blackfoxenterprise3697@gmail.com**
2. Select project **pegasus-503013**
3. Use Terminal for `d3_apply_schema_gcloud.sh`
4. Use Spanner / GKE / Memorystore browsers to inspect, not recreate

---

## One-line status

**D1+D2 live on pegasus-503013; D3 schema partial (~16 tables) — finish with gcloud script; D4–D9 not done; void-494000 abandoned but may still cost money.**
