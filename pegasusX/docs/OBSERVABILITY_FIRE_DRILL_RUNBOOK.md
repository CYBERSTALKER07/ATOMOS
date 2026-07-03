# Observability Fire Drill — PX-PROD-4

Staging exercise before production go-live. Goal: on-call resolves a simulated incident from alert → runbook → green dashboard in **≤ 30 minutes** without tribal knowledge.

**Prerequisites:** staging cluster live, Cloud Monitoring dashboards imported, on-call roster assigned.

**Local SSMR path (no GCP billing):** use `bash scripts/fire_drill_ssmr.sh` or `make fire-drill-ssmr` against docker-compose SSMR. Artifacts land in `artifacts/fire-drill-{a,b,c,d}.log`. Drill C reuses `scripts/planning_export_local_cron.sh`.

---

## Drill A — Kafka consumer lag (SEV2)

| Step | Action | Pass criteria |
|------|--------|---------------|
| 1 | Pause `backend-go-worker` consumer deployment: `kubectl scale deployment/backend-go-worker -n pegasusx --replicas=0` | `void_kafka_consumer_lag_seconds` rises within 5m |
| 2 | Confirm alert fires (email/PagerDuty) | Alert received by on-call |
| 3 | Follow [`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md) § Kafka consumer lag | Root cause identified (scaled to zero) |
| 4 | `kubectl scale deployment/backend-go-worker -n pegasusx --replicas=2` | Lag drains to < 10s within 15m |
| 5 | Verify WS fanout: place test order in SSMR or staging | Retailer/supplier inbox updates without manual refresh |

**Evidence:** screenshot of lag spike + recovery; log excerpt with `trace_id`.

---

## Drill B — Planning brain DLQ replay

| Step | Action | Pass criteria |
|------|--------|---------------|
| 1 | Publish malformed `planning.signal.ingest.v1` to staging Kafka (missing `signal_id`) | Message lands on DLQ topic |
| 2 | Confirm supplier `GET /v1/supplier/analytics/demand/today` still returns 200 with math-only `baseline_source` | No 5xx; no `ml` in response |
| 3 | Fix projector / skip bad message; replay DLQ per [`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md) § Planning brain DLQ | `PlanningSignalProjections` row count resumes growth |
| 4 | Check `DemandForecastBaseline` write rate in logs | Baseline projector healthy |

---

## Drill C — Planning export job failure

| Step | Action | Pass criteria |
|------|--------|---------------|
| 1 | Manually trigger CronJob: `kubectl create job --from=cronjob/pegasusx-planning-training-export planning-export-drill -n pegasusx` | Job completes |
| 2 | If Spanner creds revoked temporarily, confirm job fails and alert fires | Failed job visible in Cloud Monitoring |
| 3 | Restore creds; re-run job | `planning training export complete` log with `rows` > 0 |
| 4 | Run `make planning-export-validate FILE=/tmp/export.jsonl` on captured output | Validator prints `OK` |

---

## Drill D — API rollback

| Step | Action | Pass criteria |
|------|--------|---------------|
| 1 | Deploy intentionally broken API tag to staging (or use `kubectl rollout pause`) | `/health` or readiness fails |
| 2 | `kubectl rollout undo deployment/backend-go -n pegasusx` | Pods healthy within 5m |
| 3 | `make validate-launch-readiness` | Pass |

---

## Sign-off

| Role | Name | Date | Drills completed |
|------|------|------|------------------|
| On-call lead | | | A / B / C / D |
| Platform | | | |
| Product | | | |

Update `PX-PROD-4` anchor in [`context/plan_production_scale.md`](../context/plan_production_scale.md) to **shipped** after all four drills pass on staging. For local-only proof without GCP billing, mark `PX-LC-3` **shipped** in [`context/plan_local_closure.md`](../context/plan_local_closure.md) when `make fire-drill-ssmr` exits 0.
