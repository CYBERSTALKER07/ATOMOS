# Platform SLOs (Phase 3)

Target SLIs for Cloud Monitoring dashboards / alert policies
(`infra/terraform/observability.tf`, `observability_pilot.tf`).

| SLI | Target | Error budget signal |
|-----|--------|---------------------|
| Outbox publish lag (p99 claim→publish) | < 30s | `void_outbox_lag_seconds` |
| Outbox relay watchdog restarts | < 1 / hour | relay process restart counter |
| Kafka DLQ depth | = 0 sustained 5m | DLQ topic lag |
| Fiscal submit success rate | ≥ 99% / 24h | fiscal success / attempts |
| Capture success rate (GP) | ≥ 99% / 24h | CAPTURED / CAPTURE_PENDING |
| Partner webhook delivery success | ≥ 99% / 1h | webhook attempts 2xx |

Enable with `enable_observability_resources=true` in tfvars. Notification channels via
`alert_notification_channels`.

Phase 3 gates assert Terraform files declare the outbox / fiscal / capture alert
stubs; live paging requires ops to flip the enable flag.
