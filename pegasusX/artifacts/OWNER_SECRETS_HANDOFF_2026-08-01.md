# Owner secrets / ops handoff — 2026-08-01

Prod-ready gap-closure (no mocks) landed in monorepo. Cluster ConfigMap patched; Spanner migrations applied (46 DDL files, idempotent). **New backend image not yet pushed** (Cloud Build bucket forbidden + local Docker Hub timeout). Redeploy of mock-kill code still required once build credentials work.

## Agent completed

| Step | Result |
|------|--------|
| Kill backend mocks / seeds defaults | Done in repo; SSMR ConfigMap: `FISCAL_PROVIDER=PEGASUS`, all `*_SEED=false`, `ALLOW_DRIVER_DEMO_FALLBACK=false` |
| Kill UI static fleet fixtures | Supplier/warehouse tracking → live API or empty; chart shells → empty-state |
| Spanner migrations | `DONE_FAIL=0` — all `schema/migrations/*.ddl` applied to `pegasusx-ssmr-db` |
| ConfigMap + rollout restart | Pods healthy; `GET /v1/health` ok via `--resolve` |
| Cloud Build submit | **Blocked** — ADC quota project stuck on `v-o-i-d` (`USER_PROJECT_DENIED` / Cloud Build bucket forbidden) |
| Local docker build/push | **Blocked** — `auth.docker.io` i/o timeout |
| ManagedCert | Still **Provisioning** (no public A record) |

## Owner actions (blocking)

1. **Fix ADC / Cloud Build access**
   - Grant caller `roles/serviceusage.serviceUsageConsumer` on `pegasus-503013` (and stop ADC defaulting to `v-o-i-d`).
   - `gcloud auth application-default login` then `gcloud auth application-default set-quota-project pegasus-503013`
   - Rebuild:  
     `gcloud builds submit --config=cloudbuild.backend.yaml --substitutions=_TAG=ssmr-gap-closure-nomock,_REPO=pegasusx-ssmr-images --project=pegasus-503013 .`
   - Rollout:  
     `kubectl -n pegasusx-ssmr set image deploy/backend-go backend-go=asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-gap-closure-nomock`  
     (same tag for `backend-go-worker`)

2. **DNS** — create A record:  
   `api-ssmr.pegasusx.app. 300 IN A 136.69.43.141`  
   Wait: `kubectl -n pegasusx-ssmr get mcrt pegasusx-ssmr-api-cert` → **Active**.  
   Then enable HTTPS redirect on FrontendConfig (see `artifacts/step11-ingress-ssmr.md`).

3. **Global Pay** — real staging merchant password → GSM secrets → restart API → SUCCESS path smoke (`artifacts/step14-globalpay-ssmr.md`).

4. **Firebase client configs** — ship `google-services.json` / `GoogleService-Info.plist` for apps; enable SMS if real OTP required.

5. **After image + DNS** — run e2e and marker gate:  
   `PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app bash scripts/staging_smoke.sh 2>&1 | tee ssmr-e2e.log`  
   `bash scripts/parity/ssmr_ecosystem_marker_gate.sh ssmr-e2e.log`

## Interim smoke (no DNS)

```bash
curl -fsS --resolve api-ssmr.pegasusx.app:80:136.69.43.141 \
  http://api-ssmr.pegasusx.app/v1/health
```
