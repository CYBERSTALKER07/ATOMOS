/**
 * Module: Compute
 * GKE Autopilot / Standard Cluster, Node Service Account, IAM Bindings, and Workload Identity
 */

# 1. GKE Node Service Account (Used for Standard Node Pool & Base Node Daemonsets)
resource "google_service_account" "gke_node_sa" {
  account_id   = "${var.cluster_name}-node-sa"
  display_name = "GKE Node Service Account for ${var.cluster_name}"
  project      = var.project_id
  description  = "Dedicated least-privilege service account for PegasusX GKE nodes"
}

# 2. IAM Roles for GKE Nodes
locals {
  gke_node_iam_roles = [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/monitoring.viewer",
    "roles/stackdriver.resourceMetadata.writer",
    "roles/storage.objectViewer",
    "roles/artifactregistry.reader"
  ]
}

resource "google_project_iam_member" "gke_node_iam" {
  for_each = toset(local.gke_node_iam_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.gke_node_sa.email}"
}

# 3. GKE Cluster (Autopilot or Standard Multi-Zone Regional Cluster)
resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.region
  project  = var.project_id

  network    = var.network_name
  subnetwork = var.subnetwork_name

  enable_autopilot = var.enable_autopilot

  # IP Allocation Policy for VPC-Native Cluster (Secondary Subnet Ranges)
  ip_allocation_policy {
    cluster_secondary_range_name  = var.pod_ip_range_name
    services_secondary_range_name = var.service_ip_range_name
  }

  # Private Cluster Configuration (Private Nodes + Managed Endpoint)
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = var.master_ipv4_cidr_block
  }

  # Master Authorized Networks Config
  master_authorized_networks_config {
    dynamic "cidr_blocks" {
      for_each = var.authorized_ipv4_cidr_blocks
      content {
        cidr_block   = cidr_blocks.value.cidr_block
        display_name = cidr_blocks.value.display_name
      }
    }
  }

  # Release Channel Configuration
  release_channel {
    channel = var.release_channel
  }

  # Maintenance Window Policy (Low-traffic window at 03:00 UTC)
  maintenance_policy {
    daily_maintenance_window {
      start_time = "03:00"
    }
  }

  # Workload Identity Configuration (GCP IAM to K8s Service Account Federation)
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Standard Mode Overrides (Only applicable when enable_autopilot = false)
  remove_default_node_pool = var.enable_autopilot ? null : true
  initial_node_count       = var.enable_autopilot ? null : 1

  # Resource labels
  resource_labels = var.labels

  # Lifecycle protection for production cluster
  lifecycle {
    ignore_changes = [
      # Ignore node_config and other Autopilot-managed attributes
      initial_node_count
    ]
  }
}

# 4. Standard Node Pool (Only created if enable_autopilot = false)
resource "google_container_node_pool" "primary_nodes" {
  count      = var.enable_autopilot ? 0 : 1
  name       = "${var.cluster_name}-primary-pool"
  location   = var.region
  cluster    = google_container_cluster.primary.name
  project    = var.project_id
  node_count = var.standard_node_min_count

  autoscaling {
    min_node_count = var.standard_node_min_count
    max_node_count = var.standard_node_max_count
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  node_config {
    machine_type    = var.standard_node_machine_type
    service_account = google_service_account.gke_node_sa.email
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = merge(var.labels, {
      pool = "primary"
    })

    tags = ["gke-node", "${var.cluster_name}-node"]
  }
}
