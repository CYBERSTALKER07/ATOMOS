locals {
  observability_enabled   = var.enable_observability_resources
  ai_worker_uptime_host   = trimspace(var.ai_worker_monitoring_host)
  ai_worker_uptime_checks = local.observability_enabled && local.ai_worker_uptime_host != ""
}

resource "google_monitoring_alert_policy" "ai_worker_down" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — AI Worker Down"
  combiner     = "OR"

  conditions {
    display_name = "ai-worker up metric low"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_ai_worker_up/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_LT"
      threshold_value = 0.5
      duration        = "300s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "ai-worker process health is below 1 for 5 minutes. Check `/healthz`, container restarts, and recent fetch/commit failures."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "ai_worker_not_ready" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — AI Worker Not Ready"
  combiner     = "OR"

  conditions {
    display_name = "ai-worker ready metric low"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_ai_worker_ready/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_LT"
      threshold_value = 0.5
      duration        = "300s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "ai-worker readiness stayed low for 5 minutes. Check rollout state, shutdown drains, and probe reachability."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "ai_worker_consumer_lag" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — AI Worker Kafka Lag > 10 s"
  combiner     = "OR"

  conditions {
    display_name = "ai-worker consumer lag high"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\" metric.labels.consumer=\"pegasusx-ai-worker\""
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

  documentation {
    content   = "ai-worker Kafka lag exceeded 10 seconds. Check broker reachability, commit errors, and Spanner write failures before traffic is allowed to accumulate further."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_uptime_check_config" "ai_worker_health" {
  count        = local.ai_worker_uptime_checks ? 1 : 0
  display_name = "pegasusX ai-worker /healthz"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path         = "/healthz"
    port         = var.ai_worker_monitoring_port
    use_ssl      = var.ai_worker_monitoring_use_ssl
    validate_ssl = var.ai_worker_monitoring_use_ssl
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = local.ai_worker_uptime_host
    }
  }
}

resource "google_monitoring_uptime_check_config" "ai_worker_ready" {
  count        = local.ai_worker_uptime_checks ? 1 : 0
  display_name = "pegasusX ai-worker /ready"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path         = "/ready"
    port         = var.ai_worker_monitoring_port
    use_ssl      = var.ai_worker_monitoring_use_ssl
    validate_ssl = var.ai_worker_monitoring_use_ssl
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = local.ai_worker_uptime_host
    }
  }
}

resource "google_monitoring_dashboard" "ai_worker_launch" {
  count = local.observability_enabled ? 1 : 0

  dashboard_json = jsonencode({
    displayName = "pegasusX — AI Worker Launch"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "AI Worker Up"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"prometheus.googleapis.com/void_ai_worker_up/gauge\" resource.type=\"prometheus_target\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                  }
                }
              }
              gaugeView = {
                lowerBound = 0
                upperBound = 1
              }
            }
          }
        },
        {
          xPos   = 4
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "AI Worker Ready"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"prometheus.googleapis.com/void_ai_worker_ready/gauge\" resource.type=\"prometheus_target\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                  }
                }
              }
              gaugeView = {
                lowerBound = 0
                upperBound = 1
              }
            }
          }
        },
        {
          xPos   = 8
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "AI Worker Kafka Lag"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\" metric.labels.consumer=\"pegasusx-ai-worker\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MAX"
                    crossSeriesReducer = "REDUCE_MAX"
                  }
                }
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 12
          height = 4
          widget = {
            title = "AI Worker Kafka Lag Trend"
            xyChart = {
              dataSets = [
                {
                  plotType   = "LINE"
                  targetAxis = "Y1"
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\" metric.labels.consumer=\"pegasusx-ai-worker\""
                      aggregation = {
                        alignmentPeriod  = "60s"
                        perSeriesAligner = "ALIGN_MAX"
                      }
                    }
                  }
                }
              ]
              yAxis = {
                label = "seconds"
                scale = "LINEAR"
              }
            }
          }
        }
      ]
    }
  })
}