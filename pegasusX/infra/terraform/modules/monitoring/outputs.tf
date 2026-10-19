/**
 * Module: Monitoring
 * Output declarations
 */

output "notification_channel_ids" {
  description = "List of created notification channel IDs."
  value       = local.notification_channels
}

output "alert_policy_ids" {
  description = "Map of created alert policy IDs."
  value = {
    spanner_high_cpu            = google_monitoring_alert_policy.spanner_high_cpu.id
    spanner_storage             = google_monitoring_alert_policy.spanner_storage_utilization.id
    gke_container_restart       = google_monitoring_alert_policy.gke_container_restart.id
    redis_memory                = google_monitoring_alert_policy.redis_memory_utilization.id
    redis_rejected_connections  = google_monitoring_alert_policy.redis_rejected_connections.id
    nat_dropped_packets         = google_monitoring_alert_policy.nat_dropped_packets.id
    http_5xx_error_rate         = google_monitoring_alert_policy.http_5xx_error_rate.id
    kafka_produce_latency       = google_monitoring_alert_policy.kafka_produce_latency.id
    kafka_consumer_lag          = google_monitoring_alert_policy.kafka_consumer_lag.id
    gcs_api_errors              = google_monitoring_alert_policy.gcs_api_errors.id
    cloud_armor_ddos            = google_monitoring_alert_policy.cloud_armor_ddos.id
    gke_node_memory_pressure    = google_monitoring_alert_policy.gke_node_memory_pressure.id
    uptime_check_failure        = google_monitoring_alert_policy.uptime_check_failure.id
    ai_worker_down              = google_monitoring_alert_policy.ai_worker_down.id
    ai_worker_not_ready         = google_monitoring_alert_policy.ai_worker_not_ready.id
    ai_worker_consumer_lag      = google_monitoring_alert_policy.ai_worker_consumer_lag.id
    outbox_lag_high             = google_monitoring_alert_policy.outbox_lag_high.id
    fiscal_success_low          = google_monitoring_alert_policy.fiscal_success_low.id
    capture_success_low         = google_monitoring_alert_policy.capture_success_low.id
    outbox_relay_restarts_high  = google_monitoring_alert_policy.outbox_relay_restarts_high.id
    outbox_dlq_depth_nonzero    = google_monitoring_alert_policy.outbox_dlq_depth_nonzero.id
    partner_webhook_success_low = google_monitoring_alert_policy.partner_webhook_success_low.id
  }
}

output "uptime_check_id" {
  description = "The ID of the synthetic uptime check config."
  value       = google_monitoring_uptime_check_config.api_healthz.id
}

output "dashboard_id" {
  description = "The ID of the unified platform telemetry dashboard."
  value       = google_monitoring_dashboard.platform_overview.id
}
