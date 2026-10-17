variable "project_id" {
  description = "New GCP project id for the EU cell. Must not be pegasus-503013."
  type        = string
  default     = "pegasusx-cell-eu"
}

variable "project_name" {
  description = "Display name for the EU cell project."
  type        = string
  default     = "PegasusX cell-eu"
}

variable "cell_id" {
  type    = string
  default = "eu"
}

variable "region" {
  type    = string
  default = "europe-west1"
}

variable "org_id" {
  description = "GCP organization id. Required to apply; empty is catalog-plan only."
  type        = string
  default     = ""
}

variable "folder_id" {
  description = "Optional folder (alternative to org_id)."
  type        = string
  default     = ""
}

variable "billing_account_id" {
  description = "Billing account to attach. Empty is catalog-plan only."
  type        = string
  default     = ""
}
