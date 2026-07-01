# pegasusX Incident Response Runbook

## Severity

| Level | Examples | Response |
|-------|----------|----------|
| SEV1 | Payment webhooks failing, Spanner unavailable | All-hands, rollback API revision |
| SEV2 | Kafka lag > 60s, WS pubsub failures sustained | Scale pods, inspect relay |
| SEV3 | Single-role client regression | Patch + hotfix release |

## Kafka consumer lag

1. Check `void_kafka_consumer_lag_seconds` in Cloud Monitoring.
2. Inspect notification dispatcher logs: `trace_id`, `event_type`, DLQ topic.
3. Scale `ai-worker` / fix stuck partition; replay DLQ after root-cause fix.

## Planning brain DLQ replay

1. Symptom: `planning.signal.ingest.v1` lag or warehouse/supplier forecast stale.
2. Inspect ai-worker `planningingest` consumer logs and `PlanningSignalProjections` row growth.
3. Replay DLQ only after fixing projector idempotency; use `signal_id` dedup on `PlanningSignalProjections` PK.
4. Set `PLANNING_BRAIN_SHADOW=true` on ai-worker to validate baseline writes without touchless side effects.

## WebSocket / Redis pubsub

1. Symptom: clients on different pods see different state.
2. Check `ws_pubsub_failures_total` — fail-open keeps local delivery; fix Redis Memorystore connectivity.
3. Verify all pods subscribe to `cache:invalidate` and hub relay channels.

## Payment webhook replay

1. Never mutate ledger rows in place — append reversing entries.
2. Use gateway transaction id with `idempotency.Guard` — replays must be no-ops.
3. Escalate mismatches to Treasury exception queue.

## Stuck order playbooks

- **Driver**: verify order status, geofence, manifest gate — `docs/DRIVER_SUPPORT_PLAYBOOK.md`
- **Supplier**: negotiation / shop-closed — `docs/PAYMENT_EXCEPTION_SOP.md`
- **Retailer**: tracking + payment — `docs/RETAILER_ONBOARDING_SUPPORT_FLOWS.md`

## Rollback

1. `kubectl rollout undo deployment/backend-go -n pegasusx`
2. Re-point load balancer if needed (`docs/CLOUD_CUTOVER_RUNBOOK.md`)
3. Re-run `make validate-launch-readiness`
