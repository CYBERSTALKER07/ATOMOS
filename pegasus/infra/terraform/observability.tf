# ──────────────────────────────────────────────────────────────────────────────
# observability.tf — GCP Monitoring dashboards, uptime checks, and alert
# policies for the V.O.I.D. logistics platform.
#
# Alert thresholds are deliberately conservative (matching the doctrine):
#   - Kafka consumer lag > 10 s sustained 1 min
#   - Outbox relay lag > 60 s
#   - Redis circuit-breaker OPEN > 5 min
#   - HTTP error rate > 1 % over 5 min
#   - HTTP p99 latency > 500 ms over 5 min
# ──────────────────────────────────────────────────────────────────────────────

# ── Prometheus / Managed Grafana workspace ─────────────────────────────────

resource "google_monitoring_monitored_project" "void" {
  metrics_scope = "locations/global/metricsScopes/${local.project_id}"
  name          = "locations/global/metricsScopes/${local.project_id}/projects/${local.project_id}"
}

# ── Kafka consumer lag alert ───────────────────────────────────────────────

resource "google_monitoring_alert_policy" "kafka_consumer_lag" {
  display_name = "VOID — Kafka Consumer Lag > 10 s"
  combiner     = "OR"

  conditions {
    display_name = "kafka lag high"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_GT"
      threshold_value = 10
      duration        = "60s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  alert_strategy {
    auto_close = "3600s"
  }

  documentation {
    content   = "Kafka consumer group is falling behind. Check ai-worker pod health and Kafka broker disk. See COMMS-HARDENING-DOCTRINE §3."
    mime_type = "text/markdown"
  }
}

# ── Outbox relay lag alert ─────────────────────────────────────────────────

resource "google_monitoring_alert_policy" "outbox_relay_lag" {
  display_name = "VOID — Outbox Relay Lag > 60 s"
  combiner     = "OR"

  conditions {
    display_name = "outbox stuck-event"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_outbox_relay_lag_seconds/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_GT"
      threshold_value = 60
      duration        = "60s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "OutboxEvents row unprocessed > 60 s. Root cause: Kafka topic shape mismatch or broker outage. See High-Performance-Code-Standards §1 stuck-event watchdog."
    mime_type = "text/markdown"
  }
}

# ── Redis circuit-breaker OPEN alert ──────────────────────────────────────

resource "google_monitoring_alert_policy" "redis_circuit_breaker_open" {
  display_name = "VOID — Redis Circuit Breaker OPEN > 5 min"
  combiner     = "OR"

  conditions {
    display_name = "redis cb open"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_redis_circuit_breaker_state/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_GT"
      threshold_value = 1.5 # 2.0 = OPEN; threshold 1.5 avoids float comparison drift
      duration        = "300s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Redis circuit breaker in OPEN state > 5 min. Cache reads are failing fast. Operator action required. See cache-redis-correctness doctrine."
    mime_type = "text/markdown"
  }
}

# ── HTTP error rate > 1 % alert ────────────────────────────────────────────

resource "google_monitoring_alert_policy" "http_error_rate" {
  display_name = "VOID — HTTP 5xx Error Rate > 1%"
  combiner     = "OR"

  conditions {
    display_name = "backend 5xx"
    condition_threshold {
      filter          = "metric.type=\"kubernetes.io/container/request_count\" resource.type=\"k8s_container\" metric.labels.response_code_class=\"5xx\" resource.labels.cluster_name=\"${var.gke_cluster_name}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.01
      duration        = "300s"
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Backend HTTP 5xx rate exceeds 1% over 5 min. Check pod logs for trace_id correlation."
    mime_type = "text/markdown"
  }
}

# ── HTTP p99 latency > 500 ms alert ───────────────────────────────────────

resource "google_monitoring_alert_policy" "http_p99_latency" {
  display_name = "VOID — HTTP p99 Latency > 500 ms"
  combiner     = "OR"

  conditions {
    display_name = "backend p99"
    condition_threshold {
      filter          = "metric.type=\"kubernetes.io/container/request_latencies\" resource.type=\"k8s_container\" resource.labels.cluster_name=\"${var.gke_cluster_name}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 500
      duration        = "300s"
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_PERCENTILE_99"
        cross_series_reducer = "REDUCE_MAX"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "HTTP p99 latency above 500 ms SLO. Check Spanner stale read age and outbox relay load."
    mime_type = "text/markdown"
  }
}

# ── Uptime check — backend health endpoint ─────────────────────────────────

resource "google_monitoring_uptime_check_config" "backend_health" {
  display_name = "VOID Backend /healthz"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path         = "/healthz"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = local.project_id
      host       = var.backend_hostname
    }
  }
}

# ── Uptime check — ai-worker gRPC health ──────────────────────────────────

resource "google_monitoring_uptime_check_config" "ai_worker_grpc" {
  display_name = "VOID ai-worker gRPC :8082"
  timeout      = "10s"
  period       = "60s"

  tcp_check {
    port = 8082
  }

  monitored_resource {
    type = "k8s_pod"
    labels = {
      project_id     = local.project_id
      location       = local.region
      cluster_name   = var.gke_cluster_name
      namespace_name = "void"
      pod_name       = "ai-worker"
    }
  }
}

# ── Variables used by this file ────────────────────────────────────────────

variable "alert_notification_channels" {
  type        = list(string)
  description = "List of google_monitoring_notification_channel IDs to receive alerts."
  default     = []
}

variable "backend_hostname" {
  type        = string
  description = "Hostname for the backend uptime check (e.g. api.void.thelab.uz)."
  default     = "api.void.thelab.uz"
}
