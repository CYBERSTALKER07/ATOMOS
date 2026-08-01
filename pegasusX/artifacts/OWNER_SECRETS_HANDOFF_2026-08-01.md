# Owner secrets / ops handoff — 2026-08-01 (updated 2026-08-02)

Prod-ready gap-closure (no mocks) landed in monorepo and is live on SSMR. Spanner migrations applied. Backend image **`ssmr-gap-closure-nomock4`** deployed (2026-08-02). Full SSMR e2e + ecosystem marker gate **passed** via cluster port-forward (2026-08-01). DNS/TLS and Firebase client configs applied in-session.

## Agent completed

| Step | Result |
|------|--------|
| Kill backend mocks / seeds defaults | Done; ConfigMap: `FISCAL_PROVIDER=PEGASUS`, all `*_SEED=false`, `ALLOW_DRIVER_DEMO_FALLBACK=false` |
| Kill UI static fleet fixtures | Supplier/warehouse tracking → live API or empty; chart shells → empty-state |
| Spanner migrations | `DONE_FAIL=0` on `pegasusx-ssmr-db` |
| Cloud Build + rollout | **Done** — `…/backend-go:ssmr-gap-closure-nomock4` (API + worker) |
| ConfigMap follow-ups | `PAYLOAD_DEMO_WAREHOUSE_ID=ssmr-warehouse-1`, `WAREHOUSE_DEMO_ID=ssmr-warehouse-1` |
| DNS A + ManagedCert | **Active** — `api-ssmr.pegasusx.app` → `136.69.43.141`; HTTPS redirect enabled |
| Firebase iOS plists | In all 6 app targets; helpers load plist (emulator opt-in only) |
| Firebase Android JSON + plugin | In all 6 apps; `google-services` 4.4.2; helpers use real JSON (emulator via `local.properties`) |
| Cloud smoke | `PX11_CLOUD_SMOKE_OK` via `https://api-ssmr.pegasusx.app` (2026-08-02) |
| SSMR e2e + marker gate | **PASS** — `ssmr-e2e.log` → `ssmr-ecosystem-marker-gate-ok` (2026-08-01) |

## Owner actions (still blocking prod flip)

1. **Global Pay** — real staging `service_id` / username / **password** → GSM (`pegasusx-ssmr-global-pay-*`) → ESO refresh / restart API → SUCCESS path (today e2e proves cash fallback only).  
   Register webhook: `https://api-ssmr.pegasusx.app/v1/webhooks/global-pay`

2. **Firebase SMS / device trust** — Blaze + Phone provider; Android SHA-1/SHA-256 for release/debug; APNs for iOS FCM if required.

3. Optional: Soliq/OFD; feature-flag rollout; delete unused `pegasusx-staging-spanner` to cut burn.

## Smoke (public HTTPS)

```bash
curl -fsS https://api-ssmr.pegasusx.app/healthz
PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app bash scripts/cloud_smoke_ssmr.sh

# Full e2e (port-forward still fine)
kubectl -n pegasusx-ssmr port-forward svc/backend-go 18180:80
PUBLIC_BASE_URL=http://127.0.0.1:18180 JWT_SECRET=… GLOBAL_PAY_WEBHOOK_SECRET=… \
  /tmp/ssmr-smokecheck e2e | tee ssmr-e2e.log
bash scripts/parity/ssmr_ecosystem_marker_gate.sh ssmr-e2e.log
```

**Do not** set `PEGASUSX_ENV=production` until GP SUCCESS + production profile validation pass.
