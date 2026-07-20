# Cloud account cutover — full reconnect

**Decision (2026-07-20):** Abandon pilot wiring on `void-494000` and reconnect **everything** on a **completely different Google account / project**.

## Old stack (leave or destroy later)

| Item | Value |
|------|--------|
| Project | `void-494000` |
| Billing | `01444D-F6DDEC-B7DC05` |
| Spanner | `pegasusx-staging-spanner` (100 PU — **billing**) |
| Redis | `pegasusx-staging-redis` |
| GKE | `pegasusx-staging-gke` |
| TF state | local `infra/terraform/terraform.tfstate` (old project) |

**Until destroy is explicit:** old resources may keep charging. Prefer destroy soon after new stack is healthy.

## New stack — restart from D1

Do **not** reuse old Terraform state for the new account.

### 1. Auth as the new account

```bash
# Log out of mental model of void-494000
gcloud auth login                          # NEW Google account
gcloud auth application-default login
gcloud auth application-default set-quota-project NEW_PROJECT_ID
gcloud config set project NEW_PROJECT_ID
gcloud config unset api_endpoint_overrides/spanner   # no local emulator for cloud

gcloud projects list
gcloud billing accounts list
```

### 2. Fresh project prerequisites (D1)

- [ ] Project created + billing linked  
- [ ] You can `gcloud projects describe NEW_PROJECT_ID`  
- [ ] Budget alert email decided  
- [ ] APIs enabled (or let Terraform enable): Spanner, Redis, GKE, Secret Manager, Artifact Registry, Compute, Monitoring, IAM, Service Networking, Billing Budgets  

### 3. Fresh Terraform inputs (D2)

```bash
cd pegasusX
# Archive old state so apply cannot mutate void-494000 by accident
mkdir -p artifacts/tfstate-archive
mv infra/terraform/terraform.tfstate \
   artifacts/tfstate-archive/terraform.tfstate.void-494000.$(date +%Y%m%d) 2>/dev/null || true
mv infra/terraform/terraform.tfstate.backup \
   artifacts/tfstate-archive/ 2>/dev/null || true

cp infra/terraform/staging.tfvars.example infra/terraform/staging.tfvars
# Edit staging.tfvars:
#   project_id          = "NEW_PROJECT_ID"
#   billing_account_id  = "XXXXXX-XXXXXX-XXXXXX"
#   budget_alert_emails = ["you@…"]
#   region              = "asia-south1"   # or your choice
#   kafka still placeholder until Confluent
```

```bash
make terraform-init
make phase0-plan    # review
# apply only when you say go
```

### 4. Phase order on the new account

| Phase | What |
|-------|------|
| D1 | Project + billing + APIs + ADC |
| D2 | Terraform plan (then apply) |
| D3 | Spanner + migrations |
| D4 | Redis prove |
| D5 | Confluent wire → GSM |
| D6–D9 | Secrets ESO, images, deploy |

### 5. Destroy old account stack (only when you confirm)

```bash
# ONLY after re-auth to void-494000 intentionally, with archived state restored
# OR Console: delete Spanner instance, Redis, GKE cluster on void-494000
```

Never destroy while ADC points at the **new** project using the **old** state file.

## Kafka / Firebase / Pay

Unchanged product-wise; re-create secrets in the **new** project’s Secret Manager after Confluent / Firebase are ready.

## New target (filled 2026-07-20)

| Field | Value |
|-------|--------|
| Project name | pegasus |
| Project number | `1002695564567` |
| Project ID | **`pegasus-503013`** |
| Owner login | **`blackfoxenterprise3697@gmail.com`** |
| Billing (live) | **`01BFC8-0FA416-0BBA18`** (linked) |
| `staging.tfvars` | filled |

### D1 status — **complete** (2026-07-20)

- [x] CLI account `blackfoxenterprise3697@gmail.com`  
- [x] Project ACTIVE · number `1002695564567`  
- [x] Billing enabled  
- [x] APIs enabled: Spanner, Redis, GKE, Secret Manager, Artifact Registry, Compute, Monitoring, IAM, …  
- [x] Budget email → `blackfoxenterprise3697@gmail.com` · $1500  

**Next:** **“start d2”** (terraform plan) then apply when ready.

## Signal to resume automation

Say **“start d2”** for plan-only, or **“apply d2”** for full stack on `pegasus-503013`.
