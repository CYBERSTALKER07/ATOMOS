# Infra (Terraform / GKE / K8s / cells) — audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/`  
**Kind:** catalog on disk. Files ≠ applied GCP. **Do not terraform apply. Do not kubectl apply `cells/eu`.**

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`GLOBAL_SCALE_BACKEND_INFRA.md`](./GLOBAL_SCALE_BACKEND_INFRA.md) · [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md)

---

## 0. Verdict

```
VERDICT: NOT READY FOR LAYER B
EVIDENCE: checkout_reads_this false; PSP honesty stubs; payout always unimplemented; Kafka prod overlay comments
DOCS vs CODE: terraform README Autopilot→Standard + Redis STANDARD_HA — live gke.tf Autopilot + Redis BASIC
NEXT: Layer A code (pack law, honest PSP/fiscal/payout). Residual Redis HA / managed Kafka / OSRM extract are OPS after that.
```

Autopilot (`gke.tf:44`) is **OK** vs GKE golden path. Do not re-provision Standard as a “fix.”

---

## 1. What exists on disk (not “live GCP”)

| Piece | Files say | Gate |
|-------|-----------|------|
| GKE Autopilot | `enable_gke` default **false**; WI pool set | `infra/terraform/gke.tf:1-5`, `:39-64` |
| Spanner | Regional 100 PU, drop protection | `main.tf:119-133` |
| Redis | **BASIC**, AUTH+TLS defaults | `main.tf:103-117` |
| Managed Kafka | `enable_managed_kafka` default **false** | `kafka.tf:10-14` |
| Ingress | GCE, host **`api.pegasusx.app` only** (`/v1`, `/v1/ws`, `/partner`, `/healthz`) | `infra/k8s/ingress/ingress.yaml` |
| ExternalSecrets | JWT, Redis password, Maps, PSP webhooks. Fiscal/SMS **commented** | `k8s/external-secrets/` |
| Strimzi | “Do NOT deploy to prod”; **not** in overlay resources | `k8s/kafka.yaml:1-6` |

`REQUIRE_INFRA_ADAPTERS=true` in base ConfigMap (`backend-go/configmap.yaml:38`). Kafka brokers still `kafka.pegasusx.svc.cluster.local:9092` (`:14`).

---

## 2. Overlays

| Overlay | Fact |
|---------|------|
| **base** | Namespace `pegasusx`, API+worker, Ingress, planning/place **off** |
| **prod** | `FISCAL_PROVIDER=MY_SOLIQ`; optimizer **replicas 0**; OSRM included; Managed Kafka **comments only** (`overlays/prod/kustomization.yaml:50`, `:61-84`) |
| **staging** | Inherits prod; `PEGASUSX_ENV=staging`, `FISCAL_PROVIDER=PEGASUS`, `FCM_ALLOW_NOOP=true`; **does** set Managed Kafka brokers; optimizer/OSRM replicas 1 |
| **ssmr** | Does **not** include `../../base`. Host `api-ssmr.pegasusx.app` |
| **sandbox** | Same host `api-ssmr`; `supplier-portal.yaml` exists, **not** in `resources:` |
| **cells/eu** | Header: plan/render **only**. Merges **base not prod**. No Ingress, no ESO |
| **cells/uz** | Merges prod; comment do not apply from catalog |

`make cell-plan` → `scripts/cell_plan.sh` — **never apply** (`:2-5`). EU tfvars: `enable_observability_resources = false`, `jwt_secret = ""`.

---

## 3. Product-spine blockers (Layer A — before any apply)

| # | Blocker | Evidence |
|---|---------|----------|
| 1 | `CheckoutReadsThis: false` | `auth/market_pack.go:138-139` |
| 2 | STRIPE/ADYEN/PAYME/CLICK honesty executors | `payment/execution.go` → `no_live_keys` 501 |
| 3 | Live payout always false | `auth/payout_pack.go:45-48` |
| 4 | Only `cell-uz` `live: true` | `auth/cell_directory.go` |
| 5 | Place / factory planning off | ConfigMap + prod overlay |
| 6 | Prod fiscal MY_SOLIQ vs commented ESO fiscal keys | unkeyed ≠ success |
| 7 | Prod Kafka still in-cluster DNS; Strimzi not mounted | ConfigMap vs overlay comments |

Cloud day must not invent those features.

---

## 4. Portals vs API

- Ingress hosts: API only. No `supplier.pegasusx.app` rules.
- `WS_ALLOWED_ORIGINS` lists `https://{supplier,warehouse,factory,retailer}.pegasusx.app` — `configmap.yaml:33` — **no matching Ingress**.
- Only portal Dockerfile: `apps/supplier-portal/deploy/Dockerfile`. Warehouse/factory/retailer/admin: none.
- SSMR `supplier-portal.yaml` ClusterIP, omitted from kustomize `resources`.

Class A hosting = GKE Ingress → `backend-go`. Portals are Tauri static / local Next until a **code** hosting slice (not Firebase Hosting).

---

## 5. Residuals (ops, not proof apps are done)

- OSRM init exits if `/data/region.osrm` missing — `osrm/deployment.yaml:24-27`.
- Optimizer prod replicas 0 until AR image.
- Observability TF resources off in staging + cell tfvars.
- Empty `jwt_secret` in tfvars → no GSM version (`main.tf` skip when empty).
- Golden-path gaps: no private nodes / ADVANCED_DATAPATH / GSM addon in `gke.tf` — P2, not this program’s apply.

---

## 6. How it should be (skills)

| Skill | Keep / improve |
|-------|----------------|
| GKE Autopilot | **Keep** |
| Terraform cells | Plan-only until Layer A REAL |
| Memorystore HA | Ops **after** Redis is cache-only (see [`REDIS_AUDIT.md`](./REDIS_AUDIT.md)) |
| Managed Kafka | Overlay env like staging **after** worker+outbox proven; TF flag is not a product feature |
| ExternalSecrets | Uncomment fiscal/SMS only when executors are REAL and fail-closed |

---

## 7. Next

**Do not apply.** Next code: GS-M leftovers are **not** this file’s job unless asked. Infra next if asked: document-only — prod Kafka ConfigMap honesty (in-cluster vs managed), not `terraform apply`.
