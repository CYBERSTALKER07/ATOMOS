/**
 * Module: Monitoring
 * Cloud Monitoring Notification Channels, 12 Production Alert Policies, Synthetic Uptime Check,
 * and Unified SRE Platform Telemetry Dashboard.
 */

# 1. Notification Channels

# 1.1 Email Notification Channel
resource "google_monitoring_notification_channel" "email_channel" {
  count        = length(var.alert_email_endpoints) > 0 ? 1 : 0
  display_name = "PegasusX SRE On-Call Email Alerts (${var.environment})"
  type         = "email"
  project      = var.project_id

  labels = {
    email_address = var.alert_email_endpoints[0]
  }

  user_labels = var.labels
}

# 1.2 Webhook Notification Channel (Slack / PagerDuty)
resource "google_monitoring_notification_channel" "webhook_channel" {
  count        = var.slack_webhook_url != "" ? 1 : 0
  display_name = "PegasusX Slack Incident Alerts (${var.environment})"
  type         = "webhook_tokenauth"
  project      = var.project_id

  labels = {
    url = var.slack_webhook_url
  }

  user_labels = var.labels
}

locals {
  notification_channels = compact([
    length(google_monitoring_notification_channel.email_channel) > 0 ? google_monitoring_notification_channel.email_channel[0].name : "",
    length(google_monitoring_notification_channel.webhook_channel) > 0 ? google_monitoring_notification_channel.webhook_channel[0].name : ""
  ])
}

# 2. Production Alert Policies (12 Mission-Critical Rules)

# 2.1 Spanner High CPU Utilization (>80% for 5 mins)
resource "google_monitoring_alert_policy" "spanner_high_cpu" {
  display_name = "[P1] Spanner High CPU Utilization (>80%) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Cloud Spanner CPU utilization > 80% for 5m"
    condition_threshold {
      filter          = "resource.type = \"spanner_instance\" AND metric.type = \"spanner.googleapis.com/instance/cpu/utilization\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.2 Spanner Storage Utilization (>80% for 15 mins)
resource "google_monitoring_alert_policy" "spanner_storage_utilization" {
  display_name = "[P2] Spanner High Storage Utilization (>80%) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Cloud Spanner storage utilization > 80% for 15m"
    condition_threshold {
      filter          = "resource.type = \"spanner_instance\" AND metric.type = \"spanner.googleapis.com/instance/storage/utilization\""
      duration        = "900s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.3 GKE Pod CrashLoopBackOff / High Container Restart Rate
resource "google_monitoring_alert_policy" "gke_container_restart" {
  display_name = "[P1] GKE Container Restart / CrashLoopBackOff - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Container restart count > 3 in 5m"
    condition_threshold {
      filter          = "resource.type = \"k8s_container\" AND metric.type = \"kubernetes.io/container/restart_count\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 3
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["resource.label.pod_name", "resource.label.container_name"]
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.4 Cloud Memorystore Redis High Memory Utilization (>80%)
resource "google_monitoring_alert_policy" "redis_memory_utilization" {
  display_name = "[P1] Redis Memory Utilization (>80%) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Redis memory usage ratio > 80% for 5m"
    condition_threshold {
      filter          = "resource.type = \"redis_instance\" AND metric.type = \"redis.googleapis.com/stats/memory/usage_ratio\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.5 Cloud NAT Dropped Sent Packets (Port Exhaustion)
resource "google_monitoring_alert_policy" "nat_dropped_packets" {
  display_name = "[P1] Cloud NAT Dropped Packets (Port Exhaustion) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "NAT dropped sent packets > 0 in 5m"
    condition_threshold {
      filter          = "resource.type = \"nat_gateway\" AND metric.type = \"compute.googleapis.com/nat/dropped_sent_packets_count\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_DELTA"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.6 HTTP 5xx Server Error Rate (>1% of Total Requests)
resource "google_monitoring_alert_policy" "http_5xx_error_rate" {
  display_name = "[P1] HTTP 5xx Error Rate > 1% - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "HTTP 5xx error responses > 10 in 5m"
    condition_threshold {
      filter          = "resource.type = \"https_lb_rule\" AND metric.type = \"loadbalancing.googleapis.com/https/request_count\" AND metric.label.response_code_class = \"500\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 10
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.7 Kafka High Produce Request Latency (>500ms)
resource "google_monitoring_alert_policy" "kafka_produce_latency" {
  display_name = "[P2] Kafka High Produce Request Latency (>500ms) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Managed Kafka produce request latency > 500ms for 5m"
    condition_threshold {
      filter          = "resource.type = \"managedkafka.googleapis.com/Cluster\" AND metric.type = \"managedkafka.googleapis.com/cluster/produce_request_latency\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 500
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.8 Kafka Consumer Lag Alert
resource "google_monitoring_alert_policy" "kafka_consumer_lag" {
  display_name = "[P1] Kafka High Consumer Lag (>1000 messages) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Kafka consumer lag messages > 1000 for 5m"
    condition_threshold {
      filter          = "resource.type = \"managedkafka.googleapis.com/Cluster\" AND metric.type = \"managedkafka.googleapis.com/consumer_group/consumer_lag_messages\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 1000
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_SUM"
        group_by_fields      = ["metric.label.consumer_group_id", "metric.label.topic_id"]
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.9 GCS Storage API 4xx/5xx Errors
resource "google_monitoring_alert_policy" "gcs_api_errors" {
  display_name = "[P2] GCS Storage API Elevated Error Rate - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "GCS request error count > 20 in 5m"
    condition_threshold {
      filter          = "resource.type = \"gcs_bucket\" AND metric.type = \"storage.googleapis.com/api/request_count\" AND metric.label.response_code != \"OK\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 20
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.10 Cloud Armor High Block Rate / DDoS Attack Alert
resource "google_monitoring_alert_policy" "cloud_armor_ddos" {
  display_name = "[P1] Cloud Armor High Block Rate (DDoS Surge) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Cloud Armor blocked request count > 500 in 1m"
    condition_threshold {
      filter          = "resource.type = \"https_lb_rule\" AND metric.type = \"loadbalancing.googleapis.com/https/backend_request_count\" AND metric.label.response_code = \"403\""
      duration        = "60s"
      comparison      = "COMPARISON_GT"
      threshold_value = 500
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_RATE"
        cross_series_reducer = "REDUCE_SUM"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.11 GKE Node High Memory Pressure (>85%)
resource "google_monitoring_alert_policy" "gke_node_memory_pressure" {
  display_name = "[P2] GKE Node Memory Pressure (>85%) - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Node allocatable memory utilization > 85% for 10m"
    condition_threshold {
      filter          = "resource.type = \"k8s_node\" AND metric.type = \"kubernetes.io/node/memory/allocatable_utilization\""
      duration        = "600s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.85
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 2.12 Synthetic Uptime Check Probe Failure
resource "google_monitoring_alert_policy" "uptime_check_failure" {
  display_name = "[P1] API Synthetic Uptime Check Failing - ${var.environment}"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Uptime check passed < 1.0 for 2m"
    condition_threshold {
      filter          = "resource.type = \"uptime_url\" AND metric.type = \"monitoring.googleapis.com/uptime_check/check_passed\""
      duration        = "120s"
      comparison      = "COMPARISON_LT"
      threshold_value = 1.0
      trigger {
        count = 1
      }
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_FRACTION_TRUE"
        cross_series_reducer = "REDUCE_MEAN"
      }
    }
  }

  notification_channels = local.notification_channels
  user_labels           = var.labels
}

# 3. Synthetic Uptime Check Configuration (Public Health Endpoint /healthz)
resource "google_monitoring_uptime_check_config" "api_healthz" {
  display_name = "PegasusX API Healthz Synthetic Probe (${var.environment})"
  project      = var.project_id
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
      project_id = var.project_id
      host       = var.api_hostname
    }
  }

  content_matchers {
    content = "ok"
    matcher = "CONTAINS_STRING"
  }

  user_labels = var.labels
}

# 4. Cloud Monitoring Dashboard (Unified SRE Platform Telemetry)
resource "google_monitoring_dashboard" "platform_overview" {
  dashboard_json = jsonencode({
    displayName = "PegasusX FMCG Platform Overview (${var.environment})"
    gridLayout = {
      columns = "2"
      widgets = [
        {
          title = "Cloud Spanner CPU Utilization"
          xyChart = {
            dataSets = [
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"spanner_instance\" AND metric.type=\"spanner.googleapis.com/instance/cpu/utilization\""
                    aggregation = {
                      perSeriesAligner   = "ALIGN_MEAN"
                      crossSeriesReducer = "REDUCE_MEAN"
                      alignmentPeriod    = "60s"
                    }
                  }
                }
                plotType = "LINE"
              }
            ]
            yAxis = {
              label = "CPU Ratio"
              scale = "LINEAR"
            }
          }
        },
        {
          title = "Cloud Memorystore Redis Memory Ratio"
          xyChart = {
            dataSets = [
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"redis_instance\" AND metric.type=\"redis.googleapis.com/stats/memory/usage_ratio\""
                    aggregation = {
                      perSeriesAligner   = "ALIGN_MEAN"
                      crossSeriesReducer = "REDUCE_MEAN"
                      alignmentPeriod    = "60s"
                    }
                  }
                }
                plotType = "LINE"
              }
            ]
            yAxis = {
              label = "Memory Usage Ratio"
              scale = "LINEAR"
            }
          }
        },
        {
          title = "GKE Ingress HTTP Request Rate by Status"
          xyChart = {
            dataSets = [
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"https_lb_rule\" AND metric.type=\"loadbalancing.googleapis.com/https/request_count\""
                    aggregation = {
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                      groupByFields      = ["metric.label.response_code_class"]
                      alignmentPeriod    = "60s"
                    }
                  }
                }
                plotType = "STACKED_BAR"
              }
            ]
            yAxis = {
              label = "Requests / sec"
              scale = "LINEAR"
            }
          }
        },
        {
          title = "GKE Container Restart Rate"
          xyChart = {
            dataSets = [
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"k8s_container\" AND metric.type=\"kubernetes.io/container/restart_count\""
                    aggregation = {
                      perSeriesAligner   = "ALIGN_DELTA"
                      crossSeriesReducer = "REDUCE_SUM"
                      groupByFields      = ["resource.label.pod_name"]
                      alignmentPeriod    = "300s"
                    }
                  }
                }
                plotType = "LINE"
              }
            ]
            yAxis = {
              label = "Restarts / 5min"
              scale = "LINEAR"
            }
          }
        }
      ]
    }
  })
  project = var.project_id
}
