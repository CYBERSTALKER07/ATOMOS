terraform {
  required_version = "~> 1.10"

  backend "gcs" {
    # Bucket is provisioned once via `gsutil mb gs://void-terraform-state` and is
    # outside Terraform management to avoid the chicken-and-egg bootstrap problem.
    bucket = "void-terraform-state"
    prefix = "terraform/state"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.15"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
  }
}

provider "google" {
  project = var.gcp_project_id
  region  = "asia-south1"
}

# Helm and kubernetes providers share the same GKE credentials so both blocks
# reference the cluster output from google_container_cluster.void_gke.
# The data source resolves the short-lived access token at plan/apply time.
provider "kubernetes" {
  host                   = "https://${google_container_cluster.void_gke.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(google_container_cluster.void_gke.master_auth[0].cluster_ca_certificate)
}

variable "gcp_project_id" {
  type        = string
  description = "The Google Cloud Project ID for The Lab Industries"
}
