# Cloud Run to GKE Cutover Runbook

## Objective

Migrate production API traffic from Cloud Run (`lab-go-gateway`) to the global external application load balancer fronting GKE backends with zero data-loss risk and fast rollback.

## Scope

- In scope:
  - DNS and TLS cutover for `backend_hostname` to the global forwarding IP (`google_compute_global_address.void_frontend`).
  - Traffic switch from Cloud Run ingress path to GKE-backed LB path.
  - Runtime validation gates for API, WebSocket, Kafka relay, cache invalidation, and payments.
- Out of scope:
  - Spanner schema migrations.
  - Kafka topic contract changes.
  - Client app release changes.

## Preconditions

1. Terraform apply completed successfully for:
   - `google_compute_backend_service.void_backend`
   - `google_compute_target_https_proxy.void_https`
   - `google_compute_global_forwarding_rule.void_https`
   - `google_dns_managed_zone.void_public_zone`
   - `google_dns_record_set.void_backend_a`
2. Managed certificate `google_compute_managed_ssl_certificate.void_cert` is ACTIVE.
3. GKE deployment has at least 2 healthy pods per critical workload (`backend-go`, `ai-worker`).
4. `/healthz`, `/metrics`, and `/v1/metrics` return success from inside cluster and via LB endpoint.
5. On-call owner and rollback operator are assigned for the window.

## Cutover Stages

### Stage 0 - Freeze and Snapshot

1. Announce change freeze for backend deploys and Terraform applies.
2. Capture baseline:
   - API p50/p95/p99 latency (5m)
   - 5xx rate
   - Kafka consumer lag
   - outbox relay lag
   - Redis circuit breaker state
3. Record active Cloud Run revision and traffic split:
   - `gcloud run services describe lab-go-gateway --region asia-south1 --format='value(status.traffic)'`

Rollback checkpoint A:

- If baseline already violates SLO, abort cutover.

### Stage 1 - LB Readiness Gate

1. Verify backend health checks are green in Compute backend service.
2. Validate public HTTPS endpoint against forwarding IP + Host header:
   - `curl -sS --resolve api.void.pegasus.uz:443:<GLOBAL_IP> https://api.void.pegasus.uz/healthz`
3. Validate representative APIs:
   - auth login (non-mutating smoke)
   - catalog read
   - order list read
4. Validate WebSocket handshake against supplier and driver hubs.

Rollback checkpoint B:

- If health checks flap or critical endpoint error rate > 1%, fix before DNS switch.

### Stage 2 - DNS Cutover

1. Apply DNS record (`A`) pointing `backend_hostname` to global LB IP.
2. Confirm authoritative DNS answer and propagation at two external resolvers.
3. Monitor 10 minutes:
   - 5xx rate <= baseline + 0.3%
   - p99 latency <= baseline + 20%
   - Kafka lag < 10s sustained
   - Outbox lag < 60s

Rollback checkpoint C:

- If any gate fails for more than 5 minutes, execute rollback stage immediately.

### Stage 3 - Functional Validation Gate

Execute smoke matrix while traffic is on LB/GKE:

1. Supplier portal:
   - login
   - control-center actions (broadcast/reconcile/replenishment trigger)
2. Warehouse portal and mobile:
   - manifests list
   - dispatch preview
3. Factory portal:
   - payload override manifest load and transfer move
4. Retailer desktop:
   - checkout path (GLOBAL_PAY + CASH)
5. Background integrity:
   - outbox publish/consume path
   - notification dispatcher receipts

Rollback checkpoint D:

- Any high-consequence mutation failure (payment, manifest transition, completion flow) triggers rollback.

### Stage 4 - Stabilization Window

1. Hold release freeze for 30-60 minutes post-cutover.
2. Watch alerts:
   - HTTP 5xx
   - HTTP p99
   - Kafka lag
   - outbox lag
   - Redis breaker state
3. If stable, close change window and lift freeze.

## Rollback Procedure (Fast Path)

1. DNS rollback:
   - Repoint `backend_hostname` record to the previous Cloud Run ingress endpoint strategy used before cutover.
2. Runtime rollback:
   - Route traffic back to the previous Cloud Run revision (100%).
   - Keep GKE stack running; do not destroy infra during incident window.
3. Validate:
   - Cloud Run `/healthz` and core read endpoints.
   - 5xx and p99 return to pre-cutover baseline.
4. Communicate incident status and start root-cause analysis.

Rollback success criteria:

- Error budget burn returns to normal trend.
- No queue growth in Kafka lag and no outbox stuck events.
- Payment and order transitions succeed from portals.

## Post-Cutover Exit Criteria

1. 24h stability on LB/GKE path with no P0/P1 regressions.
2. No sustained SLO alert breach.
3. Verified event integrity:
   - no producer/consumer orphan regressions in versionscan.
4. Formal sign-off from backend owner + operations owner.

## Required Commands

Run from repo root (`pegasus`) where applicable.

```bash
# Terraform static and plan gates
cd infra/terraform
terraform init -upgrade
terraform fmt -check
terraform validate
terraform plan -var="gcp_project_id=<PROJECT_ID>" -var="project_id=<PROJECT_ID>" -var="backend_hostname=api.void.pegasus.uz"

# Backend compile + targeted tests
cd ../../apps/backend-go
go build ./...
go test ./factory ./warehouse ./kafka

# Guardrail gate
cd ../..
python3 scripts/versionscan.py enforce --changed-only
```

## Notes

- Keep Cloud Run service in warm standby until the 24h post-cutover gate is complete.
- Do not perform schema or event-contract changes during the same window.
- If Terraform is unavailable on operator host, run all Terraform gates in CI before the window and attach artifacts to the change ticket.
