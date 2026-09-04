# Platform SLOs (Phase 3)

Target SLIs for Cloud Monitoring dashboards / alert policies
(`infra/terraform/observability.tf`, `observability_pilot.tf`).

| SLI | Target | Error budget signal |
|-----|--------|---------------------|
| Outbox publish lag (p99 claim→publish) | < 30s | `void_outbox_lag_seconds` |
| Outbox relay watchdog restarts | < 1 / hour | `void_outbox_relay_restarts_total` |
| Outbox / Kafka DLQ depth | = 0 sustained 5m | `void_outbox_dlq_depth` (OutboxDeadLetters) |
| Fiscal submit success rate | ≥ 99% / 24h | `void_fiscal_success_ratio` |
| Capture success rate (GP) | ≥ 99% / 24h | `void_capture_success_ratio` |
| Partner webhook delivery success | ≥ 99% / 1h | `void_partner_webhook_success_ratio` |

Enable with `enable_observability_resources=true` in tfvars. Notification channels via
`alert_notification_channels`.

Collector: `apps/backend-go/telemetry/slo_metrics.go` (+ `outbox` counters for relay restarts).

Phase 3 gates assert Terraform files declare the outbox / fiscal / capture / relay /
DLQ / partner-webhook alert stubs; live paging requires ops to flip the enable flag.
