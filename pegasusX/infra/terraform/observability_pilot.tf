# Pilot operations dashboard + alerts (P1 weeks 1–8)

resource "google_monitoring_alert_policy" "backend_5xx_rate" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — Backend 5xx rate > 1%"
  combiner     = "OR"

  conditions {
    display_name = "http 5xx fraction high"
    condition_monitoring_query_language {
      query = <<-EOT
        fetch prometheus_target
        | metric 'prometheus.googleapis.com/void_http_requests_total/counter'
        | filter (metric.status_class = '5xx')
        | group_by 5m, [value_requests_total_aggregate: aggregate(value.requests_total)]
        | every 5m
        | condition val() > 10
      EOT
      duration = "300s"
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Backend 5xx rate elevated for 5 minutes. Check recent deploy, Spanner health, and Kafka lag. Roll back if SEV1."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "backend_kafka_lag" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — Backend Kafka lag > 30 s"
  combiner     = "OR"

  conditions {
    display_name = "backend consumer lag high"
    condition_threshold {
      filter          = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\""
      comparison      = "COMPARISON_GT"
      threshold_value = 30
      duration        = "120s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Kafka consumer lag exceeded 30 seconds on backend-go worker. Retailers may see stale order status until lag clears."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "spanner_cpu_high" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — Spanner CPU > 65%"
  combiner     = "OR"

  conditions {
    display_name = "spanner instance cpu high"
    condition_threshold {
      filter          = "metric.type=\"spanner.googleapis.com/instance/cpu/utilization\" resource.type=\"spanner_instance\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.65
      duration        = "600s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Spanner CPU above 65% for 10 minutes. Review hot queries in docs/SPANNER_HOT_PATH_REVIEW.md and dashboard stale-read usage."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "optimizer_fallback_rate" {
  count        = local.observability_enabled ? 1 : 0
  display_name = "pegasusX — Optimizer fallback_phase1 > 5% (5m)"
  combiner     = "OR"

  conditions {
    display_name = "fallback fraction high"
    condition_monitoring_query_language {
      query = <<-EOT
        fetch prometheus_target
        | {
            metric 'prometheus.googleapis.com/void_optimizer_source_total/counter'
            | filter (metric.source = 'fallback_phase1')
            | align rate(5m)
            | every 5m
            | group_by [], [fallback_rate: aggregate(value.counter)]
          ;
            metric 'prometheus.googleapis.com/void_optimizer_source_total/counter'
            | align rate(5m)
            | every 5m
            | group_by [], [total_rate: aggregate(value.counter)]
        }
        | join
        | value fallback_rate / total_rate
        | condition val() > 0.05
      EOT
      duration = "300s"
    }
  }

  notification_channels = var.alert_notification_channels

  documentation {
    content   = "Dispatch optimizer fallback_phase1 exceeds 5% for 5 minutes. Check ai-worker health, OPTIMIZER_BASE_URL, and solver timeouts."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_dashboard" "pilot_launch" {
  count = local.observability_enabled ? 1 : 0

  dashboard_json = jsonencode({
    displayName = "pegasusX — Pilot Launch (P1)"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "Kafka consumer lag (max)"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"prometheus.googleapis.com/void_kafka_consumer_lag_seconds/gauge\" resource.type=\"prometheus_target\""
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
          xPos   = 4
          yPos   = 0
          width  = 4
          height = 4
          widget = {
            title = "WS connections (sum)"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"prometheus.googleapis.com/void_ws_connections/gauge\" resource.type=\"prometheus_target\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MAX"
                    crossSeriesReducer = "REDUCE_SUM"
                  }
                }
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
            title = "Spanner CPU %"
            scorecard = {
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"spanner.googleapis.com/instance/cpu/utilization\" resource.type=\"spanner_instance\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                  }
                }
              }
              sparkChartView = {}
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 6
          height = 4
          widget = {
            title = "HTTP 5xx (rate)"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"prometheus.googleapis.com/void_http_requests_total/counter\" resource.type=\"prometheus_target\" metric.labels.status_class=\"5xx\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
              }]
            }
          }
        },
        {
          xPos   = 6
          yPos   = 4
          width  = 6
          height = 4
          widget = {
            title = "WS connections by hub"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"prometheus.googleapis.com/void_ws_connections/gauge\" resource.type=\"prometheus_target\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_MAX"
                      groupByFields    = ["metric.label.hub"]
                    }
                  }
                }
              }]
            }
          }
        },
        {
          xPos   = 0
          yPos   = 8
          width  = 6
          height = 4
          widget = {
            title = "Optimizer fallback_phase1 rate"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"prometheus.googleapis.com/void_optimizer_source_total/counter\" resource.type=\"prometheus_target\" metric.labels.source=\"fallback_phase1\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
              }]
            }
          }
        },
        {
          xPos   = 6
          yPos   = 8
          width  = 6
          height = 4
          widget = {
            title = "Redis geocode cache hits (rate)"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"prometheus.googleapis.com/void_redis_cache_hit_total/counter\" resource.type=\"prometheus_target\" metric.labels.prefix=\"geo\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
              }]
            }
          }
        }
      ]
    }
  })
}
