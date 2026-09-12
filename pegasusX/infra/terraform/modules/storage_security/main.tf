/**
 * Module: Storage & Security
 * GCS Buckets, Cloud Armor WAF Policies, Secret Manager Secrets, and Workload Identity IAM.
 */

# 1. Google Cloud Storage Buckets

# 1.1 Media & Cryptographic Evidence Dossiers Bucket (POD, Damage Photos, Signed Contracts)
resource "google_storage_bucket" "media" {
  name                        = var.media_bucket_name
  location                    = var.location
  project                     = var.project_id
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  cors {
    origin          = var.cors_origins
    method          = ["GET", "HEAD", "PUT", "POST", "OPTIONS"]
    response_header = ["*"]
    max_age_seconds = 3600
  }

  lifecycle_rule {
    condition {
      age = 365 # Retain active evidence for 1 year before nearline transition
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }

  labels = var.labels
}

# 1.2 App Updates & Distribution Bucket (OTA Mobile APKs, Tauri Desktop Installers)
resource "google_storage_bucket" "updates" {
  name                        = var.updates_bucket_name
  location                    = var.location
  project                     = var.project_id
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "inherited" # Allows public read distribution for mobile/desktop updates

  versioning {
    enabled = true
  }

  cors {
    origin          = ["*"]
    method          = ["GET", "HEAD", "OPTIONS"]
    response_header = ["*"]
    max_age_seconds = 3600
  }

  labels = var.labels
}

# 1.3 Bulk Import & Export Staging Bucket (Supplier Catalogs, Inventory CSV/XLSX)
resource "google_storage_bucket" "imports_exports" {
  name                        = var.imports_bucket_name
  location                    = var.region
  project                     = var.project_id
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 30 # Auto-purge temporary import/export spreadsheets after 30 days
    }
    action {
      type = "Delete"
    }
  }

  labels = var.labels
}

# 1.4 Terraform Remote State Storage Bucket
resource "google_storage_bucket" "tf_state" {
  name                        = var.tf_state_bucket_name
  location                    = var.region
  project                     = var.project_id
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 10
    }
    action {
      type = "Delete"
    }
  }

  labels = var.labels
}

# 2. Cloud Armor Enterprise WAF Security Policy (OWASP Top 10 + Rate Limiting)
resource "google_compute_security_policy" "edge_waf" {
  name        = "pegasusx-edge-waf-policy"
  description = "PegasusX Cloud Armor Enterprise WAF policy with OWASP Top 10 rules and IP rate limiting"
  project     = var.project_id

  # 2.1 OWASP Top 10 Preconfigured WAF Rules
  rule {
    action      = "deny(403)"
    priority    = 1000
    description = "OWASP SQL Injection Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1001
    description = "OWASP Cross-Site Scripting (XSS) Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('xss-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1002
    description = "OWASP Local File Inclusion (LFI) Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('lfi-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1003
    description = "OWASP Remote File Inclusion (RFI) Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('rfi-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1004
    description = "OWASP Remote Code Execution (RCE) Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('rce-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1005
    description = "OWASP Method Enforcement Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('methodenforcement-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1006
    description = "OWASP Scanner Detection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('scannerdetection-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1007
    description = "OWASP Protocol Attack Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('protocolattack-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1008
    description = "OWASP PHP Injection Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('php-v33-stable')"
      }
    }
  }

  rule {
    action      = "deny(403)"
    priority    = 1009
    description = "OWASP Session Fixation Protection"
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sessionfixation-v33-stable')"
      }
    }
  }

  # 2.2 Rate Limiting Rule: 500 requests per 10s per Client IP
  rule {
    action      = "rate_based_ban"
    priority    = 2000
    description = "Rate limit aggressive clients to 500 requests per 10 seconds"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      enforce_on_key = "IP"
      rate_limit_threshold {
        count        = var.rate_limit_threshold_count
        interval_sec = var.rate_limit_interval_sec
      }
      ban_duration_sec = 600
    }
  }

  # 2.3 Default Allow Rule
  rule {
    action      = "allow"
    priority    = 2147483647
    description = "Default allow rule for legitimate traffic"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
  }
}

# 3. Secret Manager Secrets Inventory
locals {
  secrets = toset([
    "pegasusx-db-credentials",
    "pegasusx-redis-auth-token",
    "pegasusx-soliq-ofd-cert",
    "pegasusx-soliq-ofd-api-key",
    "pegasusx-soliq-ofd-pinfl",
    "pegasusx-global-pay-secret",
    "pegasusx-payme-secret",
    "pegasusx-click-secret",
    "pegasusx-firebase-sa-key",
    "pegasusx-jwt-signing-key",
    "pegasusx-sms-playmobile-key"
  ])
}

resource "google_secret_manager_secret" "app_secrets" {
  for_each  = local.secrets
  secret_id = each.value
  project   = var.project_id

  replication {
    auto {}
  }

  labels = var.labels
}

# 4. IAM Service Accounts for Workload Identity
locals {
  service_accounts = {
    "backend-go-sa" = {
      display_name = "PegasusX Core Backend REST API & Asynchronous Worker Service Account"
      description  = "GSA bound to K8s backend-go-sa for Spanner, Redis, Kafka, Secrets, and GCS"
      roles = [
        "roles/spanner.databaseUser",
        "roles/secretmanager.secretAccessor",
        "roles/storage.objectAdmin",
        "roles/managedkafka.client"
      ]
    }
    "ai-worker-sa" = {
      display_name = "PegasusX AI Demand Forecasting & Ingestion Service Account"
      description  = "GSA bound to K8s ai-worker-sa for Spanner, Kafka, and GCS spreadsheet imports"
      roles = [
        "roles/spanner.databaseUser",
        "roles/storage.objectViewer",
        "roles/managedkafka.client"
      ]
    }
    "optimizer-sa" = {
      display_name = "PegasusX OR-Tools Vehicle Routing Optimization Service Account"
      description  = "GSA bound to K8s optimizer-sa for Spanner ledger and dispatch operations"
      roles = [
        "roles/spanner.databaseUser"
      ]
    }
  }
}

# 4.1 Create Google Service Accounts
resource "google_service_account" "workload_sa" {
  for_each     = local.service_accounts
  account_id   = each.key
  display_name = each.value.display_name
  description  = each.value.description
  project      = var.project_id
}

# 4.2 Bind Workload Identity User Role to K8s Service Accounts
resource "google_service_account_iam_member" "workload_identity_binding" {
  for_each           = local.service_accounts
  service_account_id = google_service_account.workload_sa[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.k8s_namespace}/${each.key}]"
}

# 4.3 Assign Project-Level IAM Roles to Service Accounts
locals {
  sa_role_pairs = flatten([
    for sa_name, sa_cfg in local.service_accounts : [
      for role in sa_cfg.roles : {
        sa_name = sa_name
        role    = role
        key     = "${sa_name}-${role}"
      }
    ]
  ])
}

resource "google_project_iam_member" "sa_project_roles" {
  for_each = { for pair in local.sa_role_pairs : pair.key => pair }
  project  = var.project_id
  role     = each.value.role
  member   = "serviceAccount:${google_service_account.workload_sa[each.value.sa_name].email}"
}
