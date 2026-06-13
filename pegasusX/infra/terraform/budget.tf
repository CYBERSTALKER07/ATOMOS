variable "monthly_budget_usd" {
  description = "Monthly GCP spend cap for pegasusX full-GCP-minimal footprint. Alerts fire at 80% and 100%."
  type        = number
  default     = 1500
}

variable "billing_account_id" {
  description = "GCP billing account ID (XXXXXX-XXXXXX-XXXXXX). When empty, budget resource is skipped."
  type        = string
  default     = ""
}

variable "budget_alert_emails" {
  description = "Email addresses for billing budget notifications."
  type        = list(string)
  default     = []
}

locals {
  budget_enabled = trimspace(var.billing_account_id) != "" && var.monthly_budget_usd > 0
}

resource "google_billing_budget" "pegasusx_monthly" {
  count           = local.budget_enabled ? 1 : 0
  billing_account = var.billing_account_id
  display_name    = "pegasusX monthly cap (${var.tenant_slug})"

  budget_filter {
    projects = ["projects/${var.project_id}"]
  }

  amount {
    specified_amount {
      currency_code = "USD"
      units         = tostring(floor(var.monthly_budget_usd))
    }
  }

  threshold_rules {
    threshold_percent = 0.8
    spend_basis       = "CURRENT_SPEND"
  }

  threshold_rules {
    threshold_percent = 1.0
    spend_basis       = "CURRENT_SPEND"
  }
}
