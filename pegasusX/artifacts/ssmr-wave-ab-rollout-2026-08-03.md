# SSMR Wave A/B backend image roll — 2026-08-03


> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`../docs/DOCS_SOURCE_OF_TRUTH.md`](../docs/DOCS_SOURCE_OF_TRUTH.md) · [`../docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md) · [`../context/current_status.md`](../context/current_status.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.

## Image

| Field | Value |
|-------|--------|
| Tag | `ssmr-wave-ab-a1fafaa0` |
| Registry | `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go` |
| Cloud Build | `b960bc7a-7b97-4c60-bf89-6482b8ddeeee` **SUCCESS** (~9m26s) |
| Prior image | `ssmr-ao-place-a1fafaa0-sup` |

## Rollout

| Deploy | Result |
|--------|--------|
| `backend-go` | rolled out, ready 1/1 |
| `backend-go-worker` | rolled out, ready 1/1 |

## Live checks

| Check | Result |
|-------|--------|
| `GET /healthz` | **200** `status=ok` |
| `GET /ready` | **200** |
| `negotiation-isolation` | **PASS** (`PX_E2E_NEGOTIATION_ISOLATION_OK`) |
| `GET /v1/retailer/local-skus` (no auth) | **401** (mounted) |
| `GET /v1/retailer/pos/catalog` | **401** (mounted) |
| `GET /v1/supplier/analytics/demand/flywheel` | **401** (mounted) |
| `GET /v1/retailer/control-tower/pulse` | **401** (mounted) |

401 without JWT = route present in new image (404 would mean missing).

## Code now live on SSMR (from this tree)

- Wave A: CT honesty, notif dual-read, durable family flags  
- B1: claims → store quarantine bridge  
- B3: local POS catalog (`local:` SKUs)  
- B4 + B4.4: `DEMAND_SIGNAL` emit + `FlywheelDemandFeed` + supplier flywheel API  
- B2: quantity negotiations **still product-disabled** (410 / empty)

## DDL already applied (prior steps)

- `20260809_retailer_local_catalog.ddl`  
- `20260809_flywheel_demand_feed.ddl`  
- (plus earlier auto-order / org-flags migrations)

## Re-verify

```bash
export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app
export JWT_SECRET="$(kubectl -n pegasusx-ssmr get secret backend-go-secrets -o jsonpath='{.data.jwt-secret}' | base64 -d)"
cd apps/backend-go && go run ./cmd/ssmr-smokecheck negotiation-isolation
```
