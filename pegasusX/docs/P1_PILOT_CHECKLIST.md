# P1 Pilot Checklist — Weeks 1–8

Operational follow-up after [`P0_LAUNCH_CHECKLIST.md`](./P0_LAUNCH_CHECKLIST.md). Run weekly during the controlled pilot.

## Weekly automated gate

```bash
cd pegasusX
make p1-pilot-weekly
PUBLIC_BASE_URL=https://api.prod.example.com make p1-pilot-weekly  # includes cloud smoke
```

## 1. Manual QA on real devices

| Item | Doc | Owner | Week 1 [ ] | Week 4 [ ] | Week 8 [ ] |
|------|-----|-------|------------|------------|------------|
| Role-row sign-off | [`qa/PX12_ROLE_ROW_QA.md`](./qa/PX12_ROLE_ROW_QA.md) | Client lead | | | |
| Manual runbook | [`qa/PX12_MANUAL_QA_RUNBOOK.md`](./qa/PX12_MANUAL_QA_RUNBOOK.md) | QA | | | |
| Multi-pod WS check | Open retailer on 2 phones; dispatch; both see update within 5s | Backend | | | |

**Minimum real-device matrix:** retailer checkout + tracking, driver geofence delivery, warehouse dispatch preview + execute, payload manifest scan.

## 2. Spanner hot-path review

See [`SPANNER_HOT_PATH_REVIEW.md`](./SPANNER_HOT_PATH_REVIEW.md).

| Checkpoint | Target | Action if breached |
|------------|--------|-------------------|
| Spanner CPU (10m mean) | &lt; 60% | Enable stale reads on dashboards; cap list limits |
| P99 order list | &lt; 300ms | Verify `FORCE_INDEX` hints; run migration `20250622_pilot_hot_path_indexes` |
| Dispatch preview | &lt; 2s for 300 orders | Check optimizer cache hit rate; reduce preview limit |
| Inventory list | &lt; 500ms | Warehouse SKU count audit; paginate if &gt; 500 SKUs |

**Apply new indexes (once per environment):**

```bash
# via migrate job or apply-migration
apps/backend-go/schema/migrations/20250622_pilot_hot_path_indexes.ddl
```

## 3. Monitoring dashboards

Terraform: `infra/terraform/observability_pilot.tf` → dashboard **pegasusX — Pilot Launch (P1)**.

| Signal | Metric / alert | Threshold |
|--------|----------------|-----------|
| Kafka lag | `void_kafka_consumer_lag_seconds` | Alert &gt; 30s (2 min) |
| WS load | `void_ws_connections{hub}` | Investigate if sum &gt; 500 |
| Spanner CPU | `spanner.googleapis.com/instance/cpu/utilization` | Alert &gt; 65% (10 min) |
| 5xx rate | `void_http_requests_total{status_class="5xx"}` | Alert if sustained spike |
| AI worker | Existing `observability.tf` dashboard | Lag &lt; 10s |

**GKE:** ensure `/metrics` is scraped from `backend-go` and `backend-go-worker` pods (Prometheus or Google Managed Prometheus).

## 4. Support SOP staffing

Humans must exist — docs alone are insufficient.

| Queue | Runbook | Primary | Backup | Hours |
|-------|---------|---------|--------|-------|
| Payment exceptions | [`PAYMENT_EXCEPTION_SOP.md`](./PAYMENT_EXCEPTION_SOP.md) | Finance | Release owner | 09:00–21:00 |
| Shop closed / offload | [`DELIVERY_ESCALATION_POLICY.md`](./DELIVERY_ESCALATION_POLICY.md) | Driver support | Warehouse lead | 08:00–22:00 |
| Stuck driver / no GPS | [`DRIVER_SUPPORT_PLAYBOOK.md`](./DRIVER_SUPPORT_PLAYBOOK.md) | Driver support | Backend on-call | 24/7 SEV1 |
| Warehouse dispatch lock | [`WAREHOUSE_EXCEPTION_SOP.md`](./WAREHOUSE_EXCEPTION_SOP.md) | Warehouse lead | Supplier ops | 08:00–20:00 |
| Incidents | [`INCIDENT_RESPONSE_RUNBOOK.md`](./INCIDENT_RESPONSE_RUNBOOK.md) | Release owner | Backend owner | 24/7 SEV1/2 |

Fill roster names in your ops wiki; link from this table before week 1.

## 5. Pilot expansion gates (week 4 / week 8)

Only expand retailer or driver caps when **all** hold for 7 consecutive days:

- No SEV1 incidents
- Kafka lag p95 &lt; 10s
- Spanner CPU p95 &lt; 55%
- PX12 manual QA pass on latest build
- Finance dispute queue &lt; 5 open items

## Related

- Scale roadmap (post-pilot): [`P2_SCALE_ROADMAP.md`](./P2_SCALE_ROADMAP.md)
- Launch runbook: [`LAUNCH_READINESS_RUNBOOK.md`](./LAUNCH_READINESS_RUNBOOK.md)
