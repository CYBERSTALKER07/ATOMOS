# Gate-0: Spanner PITR + automated backup schedule (IaC-owned).
# Live SSMR already had a console schedule; this brings it under Terraform.

variable "spanner_version_retention_period" {
  description = "Spanner database version retention for PITR (e.g. 7d)."
  type        = string
  default     = "7d"
}

variable "spanner_backup_retention_days" {
  description = "Full backup retention in days for the daily schedule."
  type        = number
  default     = 7
}

variable "spanner_backup_cron" {
  description = "Cron (UTC) for daily full backup schedule."
  type        = string
  default     = "0 3 * * *"
}

# version_retention_period is set on google_spanner_database.main in main.tf.

resource "google_spanner_backup_schedule" "daily_full" {
  instance = google_spanner_instance.ledger.name
  database = google_spanner_database.main.name
  name     = "daily-full-backup"

  retention_duration = "${var.spanner_backup_retention_days * 24 * 3600}s"
  spec {
    cron_spec {
      text = var.spanner_backup_cron
    }
  }
  full_backup_spec {}
  encryption_config {
    encryption_type = "GOOGLE_DEFAULT_ENCRYPTION"
  }

  depends_on = [google_spanner_database.main]
}
