/**
 * Module: Monitoring
 * Variable declarations for Cloud Monitoring Alert Policies, Notification Channels, and Dashboards.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "alert_email_endpoints" {
  description = "List of email addresses for on-call SRE alert notifications."
  type        = list(string)
  default     = ["sre-alerts@pegasusx.io"]
}

variable "slack_webhook_url" {
  description = "Optional Slack incoming webhook URL for incident notifications."
  type        = string
  default     = ""
}

variable "api_hostname" {
  description = "Hostname for external synthetic uptime probes."
  type        = string
  default     = "api.pegasusx.io"
}

variable "labels" {
  description = "Resource labels applied to monitoring resources."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "monitoring"
  }
}
