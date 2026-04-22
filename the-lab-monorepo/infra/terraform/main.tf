variable "image_url" {
  type        = string
  description = "The GAR Docker image URL injected by GitHub Actions"
}

variable "backend_image" {
  type        = string
  description = "backend-go GAR image (e.g. asia-south1-docker.pkg.dev/PROJECT/lab/backend-go:SHA)"
}

variable "ai_worker_image" {
  type        = string
  description = "ai-worker GAR image (e.g. asia-south1-docker.pkg.dev/PROJECT/lab/ai-worker:SHA)"
}

variable "gke_node_count" {
  type        = number
  default     = 2
  description = "Initial node count per zone in the default node pool"
}

# 1. Enable Required GCP APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "run.googleapis.com",
    "spanner.googleapis.com",
    "redis.googleapis.com",
    "compute.googleapis.com",
    "vpcaccess.googleapis.com",
    "container.googleapis.com",       # GKE
    "artifactregistry.googleapis.com" # GAR for container images
  ])
  service            = each.key
  disable_on_destroy = false
}

# 2. The Private VPC Network
resource "google_compute_network" "lab_vpc" {
  name                    = "lab-industries-vpc"
  auto_create_subnetworks = true
  depends_on              = [google_project_service.required_apis]
}

# 3. Google Cloud Memorystore (Redis)
resource "google_redis_instance" "cache" {
  name           = "lab-memory-layer"
  tier           = "STANDARD_HA" # Active-passive HA — required once Kafka lag-based autoscaling is live
  memory_size_gb = 1
  region         = "asia-south1"
  network        = google_compute_network.lab_vpc.id
  redis_version  = "REDIS_7_0"
}

# 4. Google Cloud Spanner (The Ledger)
resource "google_spanner_instance" "ledger" {
  name         = "lab-ledger-instance"
  config       = "regional-asia-south1"
  display_name = "The Lab Industries Ledger"
  num_nodes    = 1
  depends_on   = [google_project_service.required_apis]
}

resource "google_spanner_database" "main_db" {
  instance = google_spanner_instance.ledger.name
  name     = "lab_logistics_db"
  # Note: DDL statements are applied via your Go migration scripts, not here.
}

# 5. Google Cloud Run (The Go Backend)
resource "google_cloud_run_v2_service" "backend" {
  name     = "lab-go-gateway"
  location = "asia-south1"
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 100
    }
    
    vpc_access {
      network = google_compute_network.lab_vpc.id
      egress  = "PRIVATE_RANGES_ONLY" # Secures Redis connection
    }

    containers {
      # Placeholder image until your CI/CD builds the real one
      image = var.image_url
      
      env {
        name  = "REDIS_ADDRESS"
        value = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
      }
      env {
        name  = "SPANNER_DATABASE_URI"
        value = "projects/${var.gcp_project_id}/instances/${google_spanner_instance.ledger.name}/databases/${google_spanner_database.main_db.name}"
      }
      # Other env vars (Kafka, Google Maps API, FCM) injected via Secret Manager later
    }
  }
}

# 6. Expose Cloud Run to the Public
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  project  = google_cloud_run_v2_service.backend.project
  location = google_cloud_run_v2_service.backend.location
  name     = google_cloud_run_v2_service.backend.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ─────────────────────────────────────────────────────────────────────────────
# 7. GKE Standard Cluster (production target for backend-go + ai-worker)
# ─────────────────────────────────────────────────────────────────────────────

resource "google_service_account" "void_backend_sa" {
  account_id   = "void-backend-sa"
  display_name = "V.O.I.D backend workload identity SA"
}

# Grant Spanner + Secret Manager access to the workload SA.
resource "google_project_iam_member" "void_backend_spanner" {
  project = var.gcp_project_id
  role    = "roles/spanner.databaseUser"
  member  = "serviceAccount:${google_service_account.void_backend_sa.email}"
}

resource "google_project_iam_member" "void_backend_secrets" {
  project = var.gcp_project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.void_backend_sa.email}"
}

resource "google_container_cluster" "void_gke" {
  name     = "void-cluster"
  location = "asia-south1"

  # We manage the node pool separately so it can be updated without forcing
  # a cluster recreation.
  remove_default_node_pool = true
  initial_node_count       = 1

  network    = google_compute_network.lab_vpc.id
  subnetwork = "default"

  # Workload Identity — pods authenticate to GCP APIs as the bound SA,
  # no service-account key files needed.
  workload_identity_config {
    workload_pool = "${var.gcp_project_id}.svc.id.goog"
  }

  release_channel {
    # REGULAR keeps the cluster on the latest tested GKE version automatically.
    channel = "REGULAR"
  }

  depends_on = [google_project_service.required_apis]
}

resource "google_container_node_pool" "void_default" {
  name       = "void-default-pool"
  location   = "asia-south1"
  cluster    = google_container_cluster.void_gke.name
  node_count = var.gke_node_count

  autoscaling {
    min_node_count = 1
    max_node_count = 10
  }

  node_config {
    machine_type = "e2-standard-4" # 4 vCPU, 16 GB — handles 8–10 backend pods
    disk_size_gb = 50
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# Bind the Kubernetes ServiceAccount (void-system/void-backend-sa) to the GCP SA
# so Workload Identity can issue tokens without key files.
resource "google_service_account_iam_member" "void_workload_identity" {
  service_account_id = google_service_account.void_backend_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.gcp_project_id}.svc.id.goog[void-system/void-backend-sa]"
}

# ─────────────────────────────────────────────────────────────────────────────
# 8. KEDA — installed via Helm into the void-cluster
# ─────────────────────────────────────────────────────────────────────────────

data "google_client_config" "default" {}

provider "helm" {
  kubernetes {
    host                   = "https://${google_container_cluster.void_gke.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(google_container_cluster.void_gke.master_auth[0].cluster_ca_certificate)
  }
}

resource "helm_release" "keda" {
  name             = "keda"
  repository       = "https://kedacore.github.io/charts"
  chart            = "keda"
  version          = "2.16.0"
  namespace        = "keda"
  create_namespace = true

  set {
    name  = "resources.operator.requests.cpu"
    value = "100m"
  }
  set {
    name  = "resources.operator.requests.memory"
    value = "128Mi"
  }
  set {
    name  = "resources.operator.limits.cpu"
    value = "500m"
  }
  set {
    name  = "resources.operator.limits.memory"
    value = "512Mi"
  }

  depends_on = [google_container_node_pool.void_default]
}
