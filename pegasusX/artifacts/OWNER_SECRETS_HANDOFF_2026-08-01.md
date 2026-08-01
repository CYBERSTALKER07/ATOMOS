# Owner secrets / ops handoff — 2026-08-01

Prod-ready gap-closure (no mocks) landed in monorepo and is live on SSMR. Spanner migrations applied. Backend image **`ssmr-gap-closure-nomock3`** deployed. Full SSMR e2e + ecosystem marker gate **passed** via cluster port-forward (2026-08-01).

## Agent completed

| Step | Result |
|------|--------|
| Kill backend mocks / seeds defaults | Done; ConfigMap: `FISCAL_PROVIDER=PEGASUS`, all `*_SEED=false`, `ALLOW_DRIVER_DEMO_FALLBACK=false` |
| Kill UI static fleet fixtures | Supplier/warehouse tracking → live API or empty; chart shells → empty-state |
| Spanner migrations | `DONE_FAIL=0` on `pegasusx-ssmr-db` |
| Cloud Build + rollout | **Done** — `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-gap-closure-nomock3` |
| ConfigMap follow-ups | `PAYLOAD_DEMO_WAREHOUSE_ID=ssmr-warehouse-1`, `WAREHOUSE_DEMO_ID=ssmr-warehouse-1` |
| Cloud smoke | `PX11_CLOUD_SMOKE_OK` |
| SSMR e2e + marker gate | **PASS** — `ssmr-e2e.log` → `ssmr-ecosystem-marker-gate-ok` |
| ManagedCert | Still **Provisioning** (no public A record) |

## Owner actions (still blocking prod flip)

1. **DNS** — create A record:  
   `api-ssmr.pegasusx.app. 300 IN A 136.69.43.141`  
   Wait: `kubectl -n pegasusx-ssmr get mcrt pegasusx-ssmr-api-cert` → **Active**.  
   Then enable HTTPS redirect on FrontendConfig (see `artifacts/step11-ingress-ssmr.md`).

2. **Global Pay** — real staging merchant password → GSM secrets → restart API → SUCCESS path (today e2e proves cash fallback only).

3. **Firebase client configs** — ship `google-services.json` / `GoogleService-Info.plist` for apps; enable SMS if real OTP required.

4. **Optional image bump** — tree has payloader warehouse default + e2e-only fixes; redeploy if you want login path aligned without ConfigMap override:  
   tag e.g. `ssmr-gap-closure-nomock4` via `cloudbuild.backend.yaml`.

## Interim smoke (no DNS)

```bash
curl -fsS --resolve api-ssmr.pegasusx.app:80:136.69.43.141 \
  http://api-ssmr.pegasusx.app/v1/health

# Full e2e (port-forward)
kubectl -n pegasusx-ssmr port-forward svc/backend-go 18180:80
PUBLIC_BASE_URL=http://127.0.0.1:18180 JWT_SECRET=… GLOBAL_PAY_WEBHOOK_SECRET=… \
  /tmp/ssmr-smokecheck e2e | tee ssmr-e2e.log
bash scripts/parity/ssmr_ecosystem_marker_gate.sh ssmr-e2e.log
```

**Do not** set `PEGASUSX_ENV=production` until DNS/TLS + GP SUCCESS + production profile validation pass.
