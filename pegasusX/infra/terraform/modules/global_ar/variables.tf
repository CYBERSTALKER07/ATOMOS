variable "project_id" {
  description = "Google Cloud project ID hosting the shared multi-region Artifact Registry."
  type        = string
}

variable "location" {
  description = "Multi-region or region for the shared image repo (not a cell stack)."
  type        = string
  default     = "europe"
}

variable "repository_id" {
  description = "Artifact Registry Docker repository identifier."
  type        = string
  default     = "pegasusx"
}
