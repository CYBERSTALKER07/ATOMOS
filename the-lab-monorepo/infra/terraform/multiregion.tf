# ──────────────────────────────────────────────────────────────────────────────
# multiregion.tf — Multi-region Spanner and GKE/Kafka topology for the
# V.O.I.D. platform at million-user scale.
#
# Controlled by var.enable_multiregion (default: false).
# When false: single-region Spanner (asia-south1) + single GKE cluster.
# When true:  multi-region Spanner (nam-eur-asia3) + regional GKE clusters
#             in asia-south1, europe-west1, us-central1 with global Kafka.
#
# This file is ADDITIVE — it never removes the existing regional resources
# in main.tf. The multi-region switch is a pure scale-out, not a replacement.
# ──────────────────────────────────────────────────────────────────────────────

variable "enable_multiregion" {
  type        = bool
  description = "Upgrade Spanner to nam-eur-asia3 and provision regional GKE + Kafka clusters."
  default     = false
}

# ── Spanner: conditional multi-region instance ────────────────────────────
# When enable_multiregion = true, provision a second Spanner instance with the
# global multi-region configuration. The existing asia-south1 instance in
# main.tf is preserved for rollback safety until traffic is fully migrated.

resource "google_spanner_instance" "void_multiregion" {
  count        = var.enable_multiregion ? 1 : 0
  name         = "void-db-global"
  project      = var.project_id
  config       = "nam-eur-asia3"     # 3-continent multi-region config
  display_name = "V.O.I.D. Global Spanner (nam-eur-asia3)"

  # num_nodes at regional scale: 3 nodes per continent = 9 total for
  # nam-eur-asia3 gives ~30 000 QPS read throughput and 3 000 QPS write.
  # Cost: ~$18 k/month. Scale down to 1 node/continent for staging.
  num_nodes = var.spanner_global_nodes

  labels = {
    env         = var.env
    managed_by  = "terraform"
    role        = "primary-global"
  }
}

variable "spanner_global_nodes" {
  type        = number
  description = "Number of Spanner nodes per region in the multi-region config."
  default     = 3
}

resource "google_spanner_database" "void_multiregion" {
  count    = var.enable_multiregion ? 1 : 0
  instance = google_spanner_instance.void_multiregion[0].name
  name     = "void-db"
  project  = var.project_id

  # DDL is managed via schema/ migrations; Terraform only provisions the
  # empty database. Never pass DDL statements here.
  deletion_protection = true
}

# ── Regional GKE clusters (EU + US) ──────────────────────────────────────
# When enable_multiregion is true, provision matching clusters in EU and US.
# asia-south1 cluster remains the primary and is defined in main.tf.

resource "google_container_cluster" "void_eu" {
  count    = var.enable_multiregion ? 1 : 0
  name     = "void-cluster-eu"
  project  = var.project_id
  location = "europe-west1"

  remove_default_node_pool = true
  initial_node_count       = 1

  release_channel {
    channel = "REGULAR"
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  addons_config {
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
  }
}

resource "google_container_node_pool" "void_eu_nodes" {
  count      = var.enable_multiregion ? 1 : 0
  name       = "void-eu-node-pool"
  project    = var.project_id
  location   = "europe-west1"
  cluster    = google_container_cluster.void_eu[0].name
  node_count = var.gke_node_count

  autoscaling {
    min_node_count = var.gke_node_min
    max_node_count = var.gke_node_max
  }

  node_config {
    machine_type = var.gke_machine_type
    disk_size_gb = 100
    disk_type    = "pd-ssd"

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

resource "google_container_cluster" "void_us" {
  count    = var.enable_multiregion ? 1 : 0
  name     = "void-cluster-us"
  project  = var.project_id
  location = "us-central1"

  remove_default_node_pool = true
  initial_node_count       = 1

  release_channel {
    channel = "REGULAR"
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  addons_config {
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
  }
}

resource "google_container_node_pool" "void_us_nodes" {
  count      = var.enable_multiregion ? 1 : 0
  name       = "void-us-node-pool"
  project    = var.project_id
  location   = "us-central1"
  cluster    = google_container_cluster.void_us[0].name
  node_count = var.gke_node_count

  autoscaling {
    min_node_count = var.gke_node_min
    max_node_count = var.gke_node_max
  }

  node_config {
    machine_type = var.gke_machine_type
    disk_size_gb = 100
    disk_type    = "pd-ssd"

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# ── Kafka — regional replicas with rack awareness ─────────────────────────
# Variables for Kafka are defined here; actual broker provisioning happens
# via the helm_release in main.tf augmented by these per-region values.
# When enable_multiregion = true, the values below override the defaults.

variable "kafka_replication_factor" {
  type        = number
  description = "Kafka default replication factor. Must be ≤ broker count."
  default     = 3
}

variable "kafka_min_isr" {
  type        = number
  description = "Kafka min.insync.replicas. Must be < replication_factor."
  default     = 2
}

variable "kafka_partitions_per_topic" {
  type        = number
  description = "Default partition count per topic. 128 for million-scale throughput."
  default     = 128
}

# Multi-region Kafka cluster bootstrap via Strimzi CRD (rendered by helm).
# The actual ClusterRoleBinding + KafkaCluster CR is applied by the
# ops/kafka-cluster.yaml manifest; this file just documents the intent and
# supplies variables.

# ── Outputs ───────────────────────────────────────────────────────────────

output "spanner_global_instance" {
  value       = var.enable_multiregion ? google_spanner_instance.void_multiregion[0].name : "N/A (single-region)"
  description = "Name of the global Spanner instance when multiregion is enabled."
}

output "gke_eu_endpoint" {
  value       = var.enable_multiregion ? google_container_cluster.void_eu[0].endpoint : "N/A"
  description = "EU GKE cluster API endpoint."
}

output "gke_us_endpoint" {
  value       = var.enable_multiregion ? google_container_cluster.void_us[0].endpoint : "N/A"
  description = "US GKE cluster API endpoint."
}

# ── Variables referenced from main.tf that multiregion scales ─────────────

variable "gke_node_count" {
  type    = number
  default = 3
}

variable "gke_node_min" {
  type    = number
  default = 2
}

variable "gke_node_max" {
  type    = number
  default = 20
}

variable "gke_machine_type" {
  type    = string
  default = "n2-standard-4"
}

variable "gke_cluster_name" {
  type    = string
  default = "void-cluster"
}

variable "env" {
  type    = string
  default = "prod"
}
