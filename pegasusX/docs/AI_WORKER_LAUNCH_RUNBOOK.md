# AI Worker Launch Runbook (PX7-A3)

## Purpose
Provide a concrete launch and hypercare procedure for the pegasusX ai-worker so support can prove worker health, detect consumer drift quickly, and escalate with the right evidence.

## Scope
- Worker monitoring surface:
  - `/healthz`
  - `/ready`
  - `/metrics`
- Kubernetes packaging:
  - `infra/k8s/ai-worker/configmap.yaml`
  - `infra/k8s/ai-worker/deployment.yaml`
  - `infra/k8s/ai-worker/service.yaml`
- Local SSMR runtime:
  - `infra/docker-compose.ssmr.yml`
  - host port `8181 -> HEALTH_PORT=8081`
- Worker runtime authority:
  - `apps/ai-worker/main.go`
  - Kafka consume path on `events.TopicMain`
  - Spanner writes for `AIPredictions` and `OutboxEvents`

## Ownership
1. First response: launch owner or platform support.
2. Co-owner: backend/worker engineering for Kafka, Spanner, or runtime shutdown anomalies.
3. Escalation owner: platform engineering when health checks fail or consumer lag grows without recovery.

## Pre-launch checklist
1. Confirm the worker image placeholder is replaced in `infra/k8s/ai-worker/deployment.yaml` for the target environment.
2. Confirm the config map values for Spanner and Kafka match the target tenant or are overridden by the deployment system.
3. Run `make validate-ai-worker-k8s` or `pnpm infra:k8s:validate` and require a clean pass before rollout.
4. If cloud launch is in scope, set Terraform observability vars before apply:
  - `enable_observability_resources=true`
  - `ai_worker_monitoring_host=<worker-host-or-ip>` when uptime checks should be created
  - `alert_notification_channels=[...]` for launch paging targets
5. Verify the deployment applies with one healthy pod and the `ai-worker` ClusterIP service exposes port `8081`.
6. Start the sandbox or deployment with `HEALTH_PORT` or `AI_WORKER_HTTP_PORT` set if a non-default bind is required.
7. Verify `GET /healthz` returns `200` with `{"status":"ok"}`.
8. Verify `GET /ready` returns `200` with `{"status":"ready"}`.
9. Verify `GET /metrics` exposes:
  - `void_ai_worker_up 1`
  - `void_ai_worker_ready 1`
10. After live or synthetic traffic, verify `void_kafka_consumer_lag_seconds` is present for the worker consumer.
11. If observability resources are enabled, confirm the ai-worker launch dashboard and alert policies appear in Cloud Monitoring.
12. Confirm worker logs show startup with topic, brokers, and health port.

## Local proof steps
1. Start the isolated sandbox and wait for the ai-worker container to become healthy.
2. Run:

```bash
curl -sf http://localhost:8181/healthz
curl -sf http://localhost:8181/ready
curl -sf http://localhost:8181/metrics | grep 'void_ai_worker_\|void_kafka_consumer_lag_seconds'
```

3. If no lag metric appears yet, generate or replay a small amount of Kafka traffic before treating that as a fault.

## Failure handling matrix

| Surface | Signal | Meaning | Support action |
|---|---|---|---|
| `/healthz` | `503 {"status":"shutting_down"}` | worker is draining or unhealthy | stop rollout, check recent deploy/restart activity, capture logs |
| `/ready` | `503 {"status":"not_ready"}` | worker should not receive traffic yet or is draining | do not treat as healthy; verify boot completion or shutdown intent |
| `/metrics` | `void_ai_worker_up 0` | process is not healthy | escalate immediately with logs and recent deploy context |
| `/metrics` | `void_ai_worker_ready 0` while process stays up | worker is intentionally draining or stuck before ready | confirm shutdown intent; if not intentional, restart and escalate |
| `/metrics` | lag gauge rises on the same topic/partition without recovery | consumer backlog is growing | inspect Kafka connectivity, fetch/commit errors, and Spanner write failures |
| logs | repeated `failed to fetch message` or `failed to commit message` | Kafka path is unstable | capture broker target, timestamps, and the first failing partition/offset |
| logs | repeated `failed to create AIPrediction` | Spanner write path is failing | capture order id / trace id and escalate to backend engineering |

## Rollback / containment
1. If health or readiness fails after rollout, stop the rollout and return the worker to the previously known-good image or revision.
2. If only the worker is affected, contain at the worker layer first; do not roll back unrelated backend services unless evidence shows a shared regression.
3. After rollback, re-run the pre-launch checks before restoring confidence.

## Evidence package
1. UTC timestamps for the first bad probe result and the latest known-good result.
2. Output from `/healthz`, `/ready`, and the relevant `/metrics` lines.
3. Recent ai-worker logs including broker target, fetch/commit errors, and any trace ids.
4. If available, affected `order_id`, `prediction_id`, and Kafka topic/partition/offset.
5. Release identifier or local revision under test.