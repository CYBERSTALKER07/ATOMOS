# GCP Managed Service for Apache Kafka — the HA event backbone.
#
# Rationale: the in-cluster Strimzi manifest (infra/k8s/kafka.yaml) is dev/local
# only. Staging already runs on Managed Kafka (GCP_MANAGED_OAUTH in kafkautil);
# production follows the same path. Provision with:
#   terraform apply -var="enable_managed_kafka=true"
# then point KAFKA_BROKERS at the output bootstrap and KAFKA_AUTH_MODE at
# GCP_MANAGED_OAUTH (see overlays/staging for the reference wiring).

variable "enable_managed_kafka" {
  description = "Provision a GCP Managed Service for Apache Kafka cluster (HA, multi-zone)."
  type        = bool
  default     = false
}

variable "managed_kafka_subnet_name" {
  description = "Subnet (in the workload VPC, same region) Managed Kafka attaches to. Defaults to the VPC's auto-created subnet named after the VPC."
  type        = string
  default     = ""
}

variable "managed_kafka_vcpu" {
  description = "vCPU capacity for the Managed Kafka cluster (3 brokers minimum for HA)."
  type        = number
  default     = 3
}

variable "managed_kafka_memory_gib" {
  description = "Memory in GiB per broker for the Managed Kafka cluster."
  type        = number
  default     = 16
}

locals {
  managed_kafka_subnet = trimspace(var.managed_kafka_subnet_name) != "" ? var.managed_kafka_subnet_name : local.vpc_name
}

resource "google_managed_kafka_cluster" "events" {
  count      = var.enable_managed_kafka ? 1 : 0
  cluster_id = "${local.resource_prefix}-kafka"
  location   = var.region

  capacity_config {
    vcpu_count   = var.managed_kafka_vcpu
    memory_bytes = var.managed_kafka_memory_gib * 1024 * 1024 * 1024
  }

  gcp_config {
    access_config {
      network_configs {
        subnet = "projects/${var.project_id}/regions/${var.region}/subnetworks/${local.managed_kafka_subnet}"
      }
    }
  }

  labels = local.labels

  depends_on = [google_project_service.required_apis]
}

resource "google_managed_kafka_topic" "orders" {
  count              = var.enable_managed_kafka ? 1 : 0
  cluster            = google_managed_kafka_cluster.events[0].cluster_id
  location           = var.region
  topic_id           = var.kafka_topic_main
  partition_count    = 12
  replication_factor = 3

  configs = {
    "min.insync.replicas" = "2"
    "cleanup.policy"      = "delete"
  }
}

output "managed_kafka_bootstrap" {
  description = "Bootstrap host:port for the Managed Kafka cluster (empty when disabled)."
  value       = var.enable_managed_kafka ? "bootstrap.${google_managed_kafka_cluster.events[0].cluster_id}.${var.region}.managedkafka.${var.project_id}.cloud.goog:9092" : ""
}
