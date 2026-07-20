# P0 Launch Checklist — Real-Client Pilot

Use before onboarding real shops, drivers, and retailers. Complements [`LAUNCH_READINESS_RUNBOOK.md`](./LAUNCH_READINESS_RUNBOOK.md).

## Automated (run locally or CI)

```bash
cd pegasusX
make wire-ready              # full gate before staging wire (includes SSMR)
make test-ssmr-fiscal        # fiscal hard-gate markers (fakes only — see PRE_CLOUD_THIRD_PARTY_GATE.md)
make p0-preflight              # subset; use P0_SKIP_SSMR=1 if Docker unavailable
P0_SKIP_SSMR=1 make p0-preflight   # skip Docker SSMR when infra not up
PUBLIC_BASE_URL=https://api.staging.example.com make p0-preflight  # + cloud smoke
```

| Gate | What it proves |
|------|----------------|
| `go test ./... -short` | Backend compiles; unit + short integration tests |
| `validate_production_profile.sh` | Prod ConfigMap: `REQUIRE_INFRA_ADAPTERS`, no kafka migration flags |
| `validate-backend-k8s` | API + worker split, PDB, HPA, probes |
| `validate-ai-worker-k8s` | Optimizer worker manifests |
| `gen-contracts-gate` | Event schema ↔ generated stubs |
| `gap-hunter-gate` | Producer/consumer event shape parity |
| `test-ssmr-infra` | Cross-role E2E (`PX_E2E_*` markers) |
| `test-ssmr-fiscal` | ADR-009 money hard-gate (`PX_E2E_FISCAL_*` → `__SSMR_FISCAL_OK__`) — **before real PSP/OFD** |
| `kubectl kustomize overlays/prod` | Renderable prod stack (ingress, worker, API) |

## GCP / staging (before prod traffic)

- [ ] Apply `infra/k8s/base` or `overlays/prod` with real image tags and secrets
- [ ] Run Spanner migrate job (`infra/k8s/backend-go/migrate-job.yaml`)
- [ ] Deploy **both** `backend-go` (api) and `backend-go-worker` (worker)
- [ ] `PUBLIC_BASE_URL=... make load-cert-cloud` — production SLO profile on real Spanner
- [ ] Terraform: set `billing_account_id`, `monthly_budget_usd = 1500`, alert emails
- [ ] Confirm monitoring: `void_kafka_consumer_lag_seconds`, `void_ai_worker_ready`

## Production env contract

- [ ] `PEGASUSX_ENV=production`
- [ ] `REQUIRE_INFRA_ADAPTERS=true`
- [ ] Webhook secrets **not** prefixed with `dev-`
- [ ] `GLOBAL_PAY_USERNAME` / `PASSWORD` / `SERVICE_ID` set when `GLOBAL_PAY_ENV=production`
- [ ] Firebase production project + driver OTP enabled
- [ ] Portal seed flags **unset** (`WAREHOUSE_PORTAL_SEED`, `FACTORY_PORTAL_SEED`, etc.)
- [ ] `KAFKA_TOPIC_DUAL_WRITE` and `KAFKA_TOPIC_CONSUME_DOMAIN` **off** for pilot (default)

## Human / ops (non-code)

- [ ] [`PX12_MANUAL_QA_RUNBOOK.md`](./qa/PX12_MANUAL_QA_RUNBOOK.md) on real devices
- [ ] Finance: [`PAYMENT_EXCEPTION_SOP.md`](./PAYMENT_EXCEPTION_SOP.md) staffed
- [ ] Driver: [`DRIVER_SUPPORT_PLAYBOOK.md`](./DRIVER_SUPPORT_PLAYBOOK.md) staffed
- [ ] **72h hypercare** roster after go-live ([`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md))
- [ ] **Observability fire-drill** on staging ([`OBSERVABILITY_FIRE_DRILL_RUNBOOK.md`](./OBSERVABILITY_FIRE_DRILL_RUNBOOK.md)) — PX-PROD-4
- [ ] Planning export CronJob deployed; 7 consecutive `planning-export-validate` green days — PX-PROD-3
- [ ] Pilot cap documented (e.g. max retailers/week, one warehouse)

## Pilot scope (recommended)

| Dimension | Suggested cap |
|-----------|----------------|
| Warehouses | 1 |
| Drivers | 10–30 |
| Active retailers | 50–150 |
| Budget | $1,500/mo target; alert at 80% |

## Rollback

```bash
kubectl rollout undo deployment/backend-go -n pegasusx
kubectl rollout undo deployment/backend-go-worker -n pegasusx
make validate-launch-readiness
```

See [`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md).
