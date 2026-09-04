/**
 * Module: Messaging
 * Google Managed Service for Apache Kafka Cluster & 10 Canonical Partitioned Topics
 */

# 1. Google Managed Service for Apache Kafka Cluster (3 AZ Brokers with Dedicated vCPU / Memory)
resource "google_managed_kafka_cluster" "events" {
  cluster_id = var.kafka_cluster_id
  location   = var.region
  project    = var.project_id

  capacity_config {
    vcpu_count   = var.kafka_vcpu_count
    memory_bytes = var.kafka_memory_bytes
  }

  gcp_config {
    access_config {
      network_configs {
        subnet = var.subnet_id
      }
    }
  }

  rebalance_config {
    mode = "AUTO_REBALANCE_ON_SCALE_UP"
  }

  labels = var.labels
}

# 2. Canonical Partitioned Kafka Topics
resource "google_managed_kafka_topic" "canonical_topics" {
  for_each           = var.topics
  topic_id           = each.key
  cluster            = google_managed_kafka_cluster.events.cluster_id
  location           = var.region
  project            = var.project_id
  partition_count    = each.value.partition_count
  replication_factor = each.value.replication_factor
  configs            = lookup(each.value, "configs", {})

  depends_on = [
    google_managed_kafka_cluster.events
  ]
}
